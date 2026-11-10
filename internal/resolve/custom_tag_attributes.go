package resolve

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// A custom tag's attributes scope is what the tag calling it passes:
//
//	<cf_fileselector name="newfile" bean="#rc.contentBean#" …>
//
// MasaCMS's admin calls its file picker this way, and the picker's
// attributes.bean was typed nowhere. customTagAttribute reads every
// `<cf_name …>`, and `<cfmodule template="…/name.cfm" …>`, in the workspace
// and types an attribute given as one `#expr#` where the call is made, as
// viewArgs does a ColdBox view's args; the calls are the alternatives, and a
// tag handing an attribute on (`<cf_filetools bean="#attributes.bean#">`)
// passes what it holds. Which file a tag name runs is decided at run time
// (customTagPaths, the caller's directory), so a name two templates in the
// workspace answer to is not claimed. A call passing the attribute as text,
// or an attributeCollection, leaves it untyped, as does a tag that sets the
// attribute itself; a call omitting it says nothing, and a cfparam default
// fills only what was not passed. The calls are found in the batch file list,
// so this runs in a batch scan only.

// customTagSite is one call of a custom tag: where it is, the attributes it
// passes by lowercased name with their raw values, and opaque when it passes
// an attributeCollection.
type customTagSite struct {
	file   string
	line   int
	attrs  map[string]string
	opaque bool
}

// customTagAttribute is the component the custom tag pr holds as variable
// (`attributes.x`), as the tags calling it pass it, or "".
func (r *Resolver) customTagAttribute(variable string, pr *parser.ParseResult, tr *callTrace, ctx lookupCtx) string {
	rest, ok := strings.CutPrefix(strings.ToLower(variable), "attributes.")
	if !ok || rest == "" || strings.ContainsAny(rest, ".[(") || pr == nil || !pr.URI.IsFile() || ctx.depth >= maxAssignedDepth || len(r.owner().InferArgsFiles) == 0 {
		return ""
	}

	file := cfpath.FromURI(string(pr.URI))
	if !strings.EqualFold(filepath.Ext(file), ".cfm") {
		return ""
	}

	tag := strings.ToLower(strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)))

	sites, templates := r.customTagSites()
	if templates[tag] != 1 || len(sites[tag]) == 0 {
		return ""
	}

	// A tag that sets the attribute reads its own value; a cfparam only
	// fills one that was not passed.
	if setsName(withoutTags(pr.Content, "cfparam"), "attributes."+rest) {
		return ""
	}

	var comps []string

	for i := range sites[tag] {
		s := &sites[tag][i]
		if s.opaque {
			return ""
		}

		raw, passed := s.attrs[rest]
		if !passed {
			continue
		}

		expr, whole := wholeHashExpr(raw)
		if !whole {
			return ""
		}

		comp := r.renderedValueD(significantTokens(expr), viewArgSite{file: s.file, line: s.line}, lookupCtx{depth: ctx.depth, callers: ctx.callers})
		if comp == "" {
			return ""
		}

		for alt := range strings.SplitSeq(comp, "|") {
			if !lex.ContainsFold(comps, alt) {
				comps = append(comps, alt)
			}
		}
	}

	if len(comps) == 0 {
		return ""
	}

	slices.Sort(comps)

	answer := strings.Join(comps, "|")
	tr.addf("%q is what the tags calling this one pass it: %q", variable, answer)

	return answer
}

// customTagSites are the workspace's custom tag calls by tag name, and how
// many templates answer to each name, read once per scan from the batch file
// list. One goroutine builds it while the others wait: every scan worker asks
// at once, and each reading the whole workspace for itself took 5s of tassweb's
// scan. The files are read in parallel.
func (r *Resolver) customTagSites() (map[string][]customTagSite, map[string]int) {
	o := r.owner()

	if sites, templates, ok := o.cachedTagSites(); ok {
		return sites, templates
	}

	o.tagSitesBuild.Lock()
	defer o.tagSitesBuild.Unlock()

	if sites, templates, ok := o.cachedTagSites(); ok {
		return sites, templates
	}

	files := o.InferArgsFiles
	found := make([]map[string][]customTagSite, len(files))

	failed := forEachFile(files, func(i int, file string) { found[i] = r.customTagCallsInFile(file) })

	sites, templates := map[string][]customTagSite{}, map[string]int{}

	// A file whose calls could not be read is an opaque caller of each tag
	// it names, which makes that tag's attributes decline. When even its tag
	// names cannot be read it may call any tag, and no tag's callers are
	// known: every attribute declines, and no include is read from them.
	opaque := map[string][]string{}

	for _, file := range failed {
		names, ok := r.tagsNamedIn(file)
		if !ok {
			found = nil

			break
		}

		for _, name := range names {
			opaque[name] = append(opaque[name], file)
		}
	}

	for i, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".cfm") {
			templates[strings.ToLower(strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)))]++
		}

		if found == nil {
			continue
		}

		for tag, calls := range found[i] {
			sites[tag] = append(sites[tag], calls...)
		}
	}

	if found != nil {
		for tag, from := range opaque {
			for _, file := range from {
				sites[tag] = append(sites[tag], customTagSite{file: file, opaque: true})
			}
		}
	}

	o.mu.Lock()
	o.tagSites, o.tagTemplates = sites, templates
	o.mu.Unlock()

	return sites, templates
}

// customTagCallsInFile is customTagCallsIn over one file, nil when it holds
// none. A file indexed in this scan was read then (IndexCallerFile), and its
// calls are taken from that read.
func (r *Resolver) customTagCallsInFile(file string) map[string][]customTagSite {
	o := r.owner()

	o.mu.RLock()
	calls, read := o.tagCallsByFile[file]
	o.mu.RUnlock()

	if read {
		return calls
	}

	text, err := r.readText(file)
	if err != nil {
		return nil
	}

	return customTagCallsIn(text, file)
}

// recordTagCalls keeps the custom tag calls in path, whose text indexing has
// in hand, for customTagSites, which would otherwise read and walk the file
// again. Only a batch scan reads them (InferArgsFiles). pr, when not nil, is
// the file's parse, whose regions are reused rather than split again.
//
// A read that panics records nothing and leaves the file indexed: the build
// reads the file again, and its panic is caught and reported there, where a
// failed file is an opaque caller (customTagSites).
func (r *Resolver) recordTagCalls(path, content string, pr *parser.ParseResult) {
	if len(r.InferArgsFiles) == 0 {
		return
	}

	defer func() { _ = recover() }()

	var calls map[string][]customTagSite

	if mayCallCustomTag(content) {
		var found []parser.CustomTagCall
		if pr != nil && pr.Content == content {
			found = pr.CustomTagCalls(nil)
		} else {
			found = parser.CustomTagCalls(content, nil)
		}

		calls = customTagSitesOf(content, path, found)
	}

	o := r.owner()

	o.mu.Lock()
	if o.tagCallsByFile == nil {
		o.tagCallsByFile = map[string]map[string][]customTagSite{}
	}

	o.tagCallsByFile[path] = calls
	o.mu.Unlock()
}

// ShareTagCalls gives every resolver in rs the custom tag calls any of them
// recorded while indexing. A scan indexes each file with the resolver of the
// config governing it, but every resolver searches all the scan's files for a
// tag's callers (InferArgsFiles), and without the others' records it would
// read and walk each of their files again. A file's calls do not depend on
// its config, so the union serves them all.
func ShareTagCalls(rs []*Resolver) {
	all := map[string]map[string][]customTagSite{}

	for _, r := range rs {
		o := r.owner()

		o.mu.RLock()
		maps.Copy(all, o.tagCallsByFile)
		o.mu.RUnlock()
	}

	for _, r := range rs {
		o := r.owner()

		o.mu.Lock()
		o.tagCallsByFile = maps.Clone(all)
		o.mu.Unlock()
	}
}

// tagsNamedIn is the custom tag names file's text spells, `cf_name` read by
// the plainest search there is, for a file whose full read of its calls
// panicked: `<cf_name` and the script call `cf_name(` both count, and so may
// a name in a comment, which only makes a tag decline. ok is false when it
// cannot say: the text cannot be read, it calls cfmodule (whose tag is in an
// attribute), or this read panics too.
func (r *Resolver) tagsNamedIn(file string) (names []string, ok bool) {
	defer func() {
		if recover() != nil {
			names, ok = nil, false
		}
	}()

	text, err := r.readText(file)
	if err != nil || callsCFModule(text) {
		return nil, false
	}

	for from := 0; ; {
		i := parser.IndexFold(text[from:], "cf_")
		if i < 0 {
			return names, true
		}

		i += from
		j := i + len("cf_")

		k := j
		for k < len(text) && lex.IsWord(text[k]) {
			k++
		}

		if k > j && (i == 0 || !lex.IsWord(text[i-1])) {
			if name := strings.ToLower(text[j:k]); !slices.Contains(names, name) {
				names = append(names, name)
			}
		}

		from = max(k, i+1)
	}
}

func (r *Resolver) cachedTagSites() (map[string][]customTagSite, map[string]int, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.tagSites, r.tagTemplates, r.tagSites != nil
}

// customTagCallsIn reads the custom tag calls in content by tag name: each
// `<cf_name …>` and `<cfmodule template="…/name.cfm" …>`, with the attributes
// it passes. A call from cfscript, cf_name(…) or cfmodule(template=…), is an
// opaque caller: its arguments are expressions, not the `#expr#` attribute
// values customTagAttribute reads. The calls are found by
// parser.CustomTagCalls, which skips comments.
func customTagCallsIn(content, file string) map[string][]customTagSite {
	if !mayCallCustomTag(content) {
		return nil
	}

	return customTagSitesOf(content, file, parser.CustomTagCalls(content, nil))
}

// customTagSitesOf is customTagCallsIn over calls already read from content.
// What it keeps is copied out of content, so a site held for the scan does
// not keep the whole file's text alive.
func customTagSitesOf(content, file string, calls []parser.CustomTagCall) map[string][]customTagSite {
	var out map[string][]customTagSite

	for i := range calls {
		c := &calls[i]

		tag := customTagName(c)
		if tag == "" {
			continue
		}

		site := customTagSite{file: file, line: c.Line, opaque: c.Script}
		if !c.Script {
			attrs := parser.TagAttributes(content[c.AttrStart:c.AttrEnd])

			site.attrs = make(map[string]string, len(attrs))
			for k, v := range attrs {
				site.attrs[strings.Clone(k)] = strings.Clone(v)
			}

			_, site.opaque = site.attrs["attributecollection"]
		}

		if out == nil {
			out = map[string][]customTagSite{}
		}

		tag = strings.Clone(tag)
		out[tag] = append(out[tag], site)
	}

	return out
}

// mayCallCustomTag reports whether content could hold a custom tag call at
// all: it spells cf_ or cfmodule, or a Lucee `module template=` or `module
// name=` statement. Most files do not, and are not walked.
func mayCallCustomTag(content string) bool {
	return parser.IndexFold(content, "cf_") >= 0 || parser.IndexFold(content, "cfmodule") >= 0 || hasModuleStatement(content)
}

// hasModuleStatement reports whether text holds Lucee's `module template=…`
// or `module name=…` statement: the word module, space, and the attribute
// with its =. Prose ("the module name is") has no =.
func hasModuleStatement(text string) bool {
	for from := 0; ; {
		i := parser.IndexFold(text[from:], "module")
		if i < 0 {
			return false
		}

		i += from
		from = i + len("module")

		if i > 0 && lex.IsWord(text[i-1]) {
			continue
		}

		j := lex.SkipSpace(text, from)
		if j == from {
			continue
		}

		for _, attr := range []string{"template", "name"} {
			if lex.HasPrefixFold(text[j:], attr) {
				if k := lex.SkipSpace(text, j+len(attr)); k < len(text) && text[k] == '=' {
					return true
				}
			}
		}
	}
}

// callsCFModule reports whether text calls a tag by its path or dotted name,
// which only the call's attribute says: <cfmodule, cfmodule( from cfscript,
// or Lucee's module statement. A mention in prose or a longer name does not
// count.
func callsCFModule(text string) bool {
	if hasModuleStatement(text) {
		return true
	}

	for from := 0; ; {
		i := parser.IndexFold(text[from:], "cfmodule")
		if i < 0 {
			return false
		}

		i += from
		from = i + len("cfmodule")

		if from < len(text) && lex.IsWord(text[from]) {
			continue
		}

		if i > 0 && text[i-1] == '<' {
			return true
		}

		if i > 0 && lex.IsWord(text[i-1]) {
			continue
		}

		if j := lex.SkipSpace(text, from); j < len(text) && text[j] == '(' {
			return true
		}
	}
}

// customTagName is the lowercased name of the template the call c runs, as
// customTagSites counts templates by name: a cf_ tag's name, or the file name
// of a literal cfmodule template ending .cfm; "" for an end tag and any
// other call.
func customTagName(c *parser.CustomTagCall) string {
	switch {
	case c.Closing || c.Dynamic:
		return ""
	case c.Kind == parser.CustomTagUnderscore:
		return strings.ToLower(c.Name)
	case c.Kind == parser.CustomTagModuleTemplate && strings.EqualFold(filepath.Ext(c.Name), ".cfm"):
		return strings.ToLower(strings.TrimSuffix(filepath.Base(filepath.FromSlash(c.Name)), filepath.Ext(c.Name)))
	default:
		return ""
	}
}

// wholeHashExpr is the expression of raw when raw is one #expr# and nothing
// else but space.
func wholeHashExpr(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 3 || raw[0] != '#' || raw[len(raw)-1] != '#' {
		return "", false
	}

	expr := raw[1 : len(raw)-1]

	return expr, !strings.Contains(expr, "#")
}

// withoutTags is content with every <name …> tag cut out, name matched
// without case and ending at a non-name byte, the tag running to its first >.
func withoutTags(content, name string) string {
	open := "<" + name

	i := parser.IndexFold(content, open)
	if i < 0 {
		return content
	}

	var b strings.Builder

	for i >= 0 {
		end := i + len(open)
		if end < len(content) && lex.IsWord(content[end]) {
			b.WriteString(content[:end])
			content = content[end:]
		} else {
			b.WriteString(content[:i])

			gt := strings.IndexByte(content[end:], '>')
			if gt < 0 {
				// A tag never closed is no tag: its text stays.
				b.WriteString(content[i:])

				content = ""

				break
			}

			content = content[end+gt+1:]
		}

		i = parser.IndexFold(content, open)
	}

	b.WriteString(content)

	return b.String()
}

// attributeIncludeTargets is what `<cfinclude template="#attributes.name#">`
// in the custom tag tagFile includes: the template each call of the tag
// passes as name, written as a literal and resolved as a cfinclude in the tag
// is. RustCFML's test runner calls `<cf_runtest file="core/test_x.cfm">` six
// hundred times, and runtest.cfm includes its harness and then the file, so
// each test calls the harness's assert() bare. A call passing the attribute
// computed, or an attributeCollection, names nothing it can be read from, and
// a tag name two templates answer to is not claimed, as customTagAttribute
// does not. The calls are found in the batch file list, so this runs in a
// batch scan only.
func (r *Resolver) attributeIncludeTargets(name, tagFile string) []string {
	if len(r.owner().InferArgsFiles) == 0 || !strings.EqualFold(filepath.Ext(tagFile), ".cfm") {
		return nil
	}

	tag := strings.ToLower(strings.TrimSuffix(filepath.Base(tagFile), filepath.Ext(tagFile)))

	sites, templates := r.customTagSites()
	if templates[tag] != 1 {
		return nil
	}

	var out []string

	for i := range sites[tag] {
		raw, passed := sites[tag][i].attrs[name]
		if !passed || strings.Contains(raw, "#") {
			continue
		}

		if target := r.IncludePath(raw, tagFile); target != "" && !slices.Contains(out, target) {
			out = append(out, target)
		}
	}

	slices.Sort(out)

	return out
}
