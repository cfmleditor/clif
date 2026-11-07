package resolve

import (
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// An abstract component calls what its subclasses supply: ContentBox's
// baseContentHandler uses variables.ormService and never declares it, and each
// concrete handler injects a different service under that name. A receiver is
// looked up in the file and then up its extends chain, never down, so every
// call on it was "has no component ref".
//
// subclassComponent asks the components that extend the file what they hold
// for the name. It answers only when every concrete subclass (a leaf of the
// tree under the file) types it, by a ref of its own or of a component between
// it and the file: one that does not leaves the name to a pre-handler, a mixin
// or a runtime set, and the others' answer would be a guess. The components
// are returned as alternatives, as withSubclasses spells them, so a method any
// of them declares is found.
//
// Only a name the file never types, only variables scope or unscoped, and only
// where the file is extended within the workspace; the walk is capped at
// maxSubclasses.

const maxSubclasses = 16

func (r *Resolver) subclassComponent(variable string, line uint32, pr *parser.ParseResult, tr *callTrace, leaf string) string {
	if r.Index == nil || pr == nil || variable == "" || strings.ContainsAny(variable, "[(") {
		return ""
	}

	name := parser.StripReceiverScope(variable)
	if name == "" || strings.Contains(name, ".") {
		return ""
	}

	root, _, qualified := strings.Cut(variable, ".")
	if qualified && !strings.EqualFold(root, "variables") {
		return ""
	}

	// An unscoped name a function declares is its local, not the component's.
	if !qualified {
		if scope := pr.FuncScopeAt(int(line)); scope.Start != -1 {
			if pr.FuncDeclares(scope.Start, scope.End, name) {
				return ""
			}
		}
	}

	base := pr.URI.Path()
	leaves := r.leafSubclasses(base)

	if leaf != "" {
		leaves = slices.DeleteFunc(leaves, func(l string) bool { return !cfpath.SamePath(l, leaf) })
	}

	if len(leaves) == 0 {
		return ""
	}

	scope := parser.ReceiverRefScope(variable)

	var comps []string

	for _, leaf := range leaves {
		comp := r.subclassRef(leaf, base, name, scope)
		if comp == "" {
			tr.addf("subclass %s does not type %q, so no subclass answer is given", filepath.Base(leaf), name)

			return ""
		}

		if !slices.ContainsFunc(comps, func(c string) bool { return strings.EqualFold(c, comp) }) {
			comps = append(comps, comp)
		}
	}

	slices.Sort(comps)

	answer := strings.Join(comps, "|")

	tr.addf("resolved %q to %q: what every subclass of %s holds for it", variable, answer, filepath.Base(base))

	return answer
}

// leafSubclasses are the components under base, directly or not, that nothing
// extends, in path order.
func (r *Resolver) leafSubclasses(base string) []string {
	if r.Index == nil {
		return nil
	}

	// Every bare call no file or chain answers asks this of its file, and the
	// walk scans the extends map once per component under the base. Kept on
	// the owner under the mappings the walk resolved with: descendsFrom
	// resolves each extends through r.ComponentPath, and a caller-context view
	// resolves with its application's mappings (mappingsKey), so views whose
	// mappings agree share an answer and the owner, which resolves by
	// directory, keeps its own.
	o := r.owner()
	scope := "owner"

	if r.indexer != nil {
		scope = "view:" + r.mappingsKey
	}

	key := pathKey(base) + "\x00" + strconv.FormatUint(r.Index.ShapeGeneration(), 10)

	o.mu.RLock()
	leaves, ok := o.leafSubs[scope][key]
	o.mu.RUnlock()

	// A copy: a caller narrows the list in place (subclassComponent).
	if ok {
		return slices.Clone(leaves)
	}

	leaves = r.leafSubclassesUncached(base)

	// Each scope has its own cap, so a workspace of many applications does
	// not empty every view's answers each time one view fills up.
	o.mu.Lock()
	if o.leafSubs == nil {
		o.leafSubs = map[string]map[string][]string{}
	}

	if o.leafSubs[scope] == nil || len(o.leafSubs[scope]) >= 8192 {
		o.leafSubs[scope] = map[string][]string{}
	}

	o.leafSubs[scope][key] = leaves
	o.mu.Unlock()

	return slices.Clone(leaves)
}

func (r *Resolver) leafSubclassesUncached(base string) []string {
	// One pass over the extends map answers every name below. Asked per
	// component, FilesExtendingName scanned the whole map each time, and the
	// walk checked each file against the list so far by cleaning both paths:
	// 98% of a Lucee scan (110 s), whose test cases are thousands of leaves
	// under one base.
	extenders := r.Index.ExtendersByName()

	extending := func(path string) []index.Extender {
		return extenders[strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))]
	}

	all := []string{base}
	seen := map[string]bool{pathKey(base): true}

	for i := 0; i < len(all) && len(all) <= maxSubclasses; i++ {
		parent := all[i]

		for _, e := range extending(parent) {
			sub := cfpath.FromURI(e.URI)

			if seen[pathKey(sub)] {
				continue
			}

			if r.descendsFromDeclared(sub, e.Extends, parent) {
				all = append(all, sub)
				seen[pathKey(sub)] = true
			}
		}
	}

	var leaves []string

	for _, c := range all[1:] {
		extended := false

		for _, e := range extending(c) {
			if r.descendsFromDeclared(cfpath.FromURI(e.URI), e.Extends, c) {
				extended = true

				break
			}
		}

		if !extended {
			leaves = append(leaves, c)
		}
	}

	return leaves
}

// subclassRef is the component the file at path, or one of the components
// between it and base, holds for name at file level, nearest first.
func (r *Resolver) subclassRef(path, base, name string, scope parser.RefScope) string {
	for range 16 {
		if cfpath.SamePath(path, base) {
			return ""
		}

		for _, ref := range r.Index.RefsForFile(cfpath.ToURI(path)) {
			if ref != nil && ref.Component != "" && strings.EqualFold(ref.Variable, name) && scope.Admits(ref) {
				return ref.Component
			}
		}

		ext, ok := r.extendsOf(path, cfpath.ToURI(path))
		if !ok || ext == "" {
			return ""
		}

		path = r.ComponentPath(ext, filepath.Dir(path))
		if path == "" {
			return ""
		}
	}

	return ""
}

// leafLiteral is the literal string a component assigns `variables.name` in
// its own source, or in the nearest component it extends below base: the
// value a subclass gives a variable its base reads, as ContentBox's handlers
// give entityPlural and handler. "" when none writes one.
func (r *Resolver) leafLiteral(leaf, base, name string) string {
	re := compiledPattern(`(?i)\bvariables\.` + regexp.QuoteMeta(name) + `\s*=\s*["']([^"'#]*)["']`)

	path := leaf

	for range 16 {
		if path == "" || cfpath.SamePath(path, base) {
			return ""
		}

		if text, err := r.readText(path); err == nil && mentions(text, "variables."+name) {
			if m := re.FindStringSubmatch(text); m != nil && m[1] != "" {
				return m[1]
			}
		}

		ext, ok := r.extendsOf(path, cfpath.ToURI(path))
		if !ok || ext == "" {
			return ""
		}

		path = r.ComponentPath(ext, filepath.Dir(path))
	}

	return ""
}

// mappingsKey is a set of mappings as a string, the same for any two sets
// that map alike.
func mappingsKey(m map[string]string) string {
	keys := slices.Sorted(maps.Keys(m))

	var b strings.Builder

	for _, k := range keys {
		b.WriteString(strings.ToLower(k))
		b.WriteByte('=')
		b.WriteString(m[k])
		b.WriteByte(0)
	}

	return b.String()
}
