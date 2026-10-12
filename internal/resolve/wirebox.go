package resolve

import (
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// WireBox registers a model under an id, and most ids are not the path to
// the component: `UserService@users` is the UserService in the users
// module's models, `RequestStorage@cbstorages` one in a module the
// workspace may not have installed, and `binder.map( "Mailer" ).to(
// "#moduleMapping#.models.MailService" )` an id that names no file at all.
// This file is what the workspace's ModuleConfig.cfc and config/WireBox.cfc
// say about ids, read the way ColdBox reads them:
//
//   - A module registers its models under `this.modelNamespace`, which
//     defaults to the module's name — its directory's.
//   - `this.cfmapping` is a mapping to the module's directory, so a dot-path
//     starting with it is inside the module.
//   - `map( id ).to( path )` is an id's component, `#moduleMapping#` being
//     the module's own directory.

// wireboxWorkspace is that knowledge, for one set of configuration files.
type wireboxWorkspace struct {
	key        string
	namespaces map[string][]string // lowercased namespace -> module directories
	modelRoots map[string][]string // eligible auto-mapped roots in those modules
	cfmappings map[string]string   // lowercased cfmapping -> module directory
	ids        map[string]idTarget // lowercased id -> what it is mapped to
}

// idTarget is a `map( id ).to( path )`: path is a dot-path, read from dir.
type idTarget struct {
	path, dir string
}

var binderMapRe = regexp.MustCompile(`(?is)\bmap\(\s*["']([^"'#]+)["']\s*\)\s*\.\s*to\(\s*["']([^"']+)["']\s*\)`)

// wirebox reads every ModuleConfig.cfc and config/WireBox.cfc the index
// holds, remembered until that set of files changes, as applicationHelpers
// is.
func (r *Resolver) wirebox() *wireboxWorkspace {
	if r.Index == nil || r.FS == nil {
		return nil
	}

	modules := r.Index.FindFilesByBasename("ModuleConfig")

	var binders []string

	for _, p := range r.Index.FindFilesByBasename("WireBox") {
		if strings.EqualFold(filepath.Base(filepath.Dir(p)), "config") {
			binders = append(binders, p)
		}
	}

	key := strings.Join(modules, "\x00") + "\x01" + strings.Join(binders, "\x00")

	r.mu.RLock()
	cached := r.wb
	r.mu.RUnlock()

	if cached != nil && cached.key == key {
		return cached
	}

	wb := &wireboxWorkspace{
		key:        key,
		namespaces: map[string][]string{},
		modelRoots: map[string][]string{},
		cfmappings: map[string]string{},
		ids:        map[string]idTarget{},
	}

	for _, m := range modules {
		data, err := r.fs().ReadFile(m)
		if err != nil {
			continue
		}

		src := string(data)
		dir := filepath.Dir(m)
		constructor := parser.ApplicationConstructor(src)
		properties, propertiesKnown := moduleAssignments(constructor, true)

		ns := filepath.Base(dir)
		if value := diString(properties["modelnamespace"]); propertiesKnown && value != "" {
			ns = strings.TrimSpace(value)
		}

		wb.namespaces[strings.ToLower(ns)] = append(wb.namespaces[strings.ToLower(ns)], dir)
		if root := moduleModelRoot(src, dir); root != "" {
			wb.modelRoots[strings.ToLower(ns)] = append(wb.modelRoots[strings.ToLower(ns)], root)
		}

		disabled, disabledSet := properties["disabled"]

		registered := propertiesKnown && (!disabledSet || len(disabled) == 1 && strings.EqualFold(disabled[0].Value, "false"))
		if registered {
			mapping := diString(properties["cfmapping"])
			if value, exists := properties["classmapping"]; exists {
				if class := diString(value); class != "" {
					mapping = class
				} else if len(value) != 1 || value[0].Kind != parser.TokString {
					mapping = ""
				}
			}

			if mapping != "" {
				wb.cfmappings[strings.ToLower(strings.Trim(strings.TrimSpace(mapping), "/"))] = dir
			}
		}

		wb.addMaps(src, dir)
	}

	for _, b := range binders {
		if text, err := r.readText(b); err == nil {
			// config/WireBox.cfc is the application's: its paths are read
			// from the application's root, the directory above config/.
			wb.addMaps(text, filepath.Dir(filepath.Dir(b)))
		}
	}

	r.mu.Lock()
	r.wb = wb
	r.mu.Unlock()

	return wb
}

// addMaps records every `map( id ).to( path )` in src.
func (wb *wireboxWorkspace) addMaps(src, dir string) {
	for _, m := range binderMapRe.FindAllStringSubmatch(src, -1) {
		id := strings.ToLower(strings.TrimSpace(m[1]))
		if _, ok := wb.ids[id]; !ok {
			wb.ids[id] = idTarget{path: strings.TrimSpace(m[2]), dir: dir}
		}
	}
}

// wireboxID is ComponentPath's answer for a WireBox id: a mapped one's
// target, and `Name@namespace` the Name in that module's models. It reports
// whether component was an id it answers for; a plain dot-path is not, and
// is left to the lookups after it.
func (r *Resolver) wireboxID(component, baseDir string) (string, bool) {
	wb := r.wirebox()

	// Ids are looked up only for a component with no dot: a mapped target is
	// a dot-path, so resolving one cannot come back here, and `map( "X" ).to(
	// "X" )` is not an id worth the loop.
	if wb != nil && !strings.Contains(component, ".") {
		if t, ok := wb.ids[strings.ToLower(component)]; ok && !strings.EqualFold(t.path, component) {
			return r.idTargetPath(t, wb), true
		}
	}

	name, ns, qualified := strings.Cut(component, "@")
	if !qualified {
		return "", false
	}

	if wb != nil {
		if _, installed := wb.namespaces[strings.ToLower(ns)]; installed {
			return r.inModuleModels(name, wb.modelRoots[strings.ToLower(ns)], baseDir), true
		}
	}

	// A namespace is part of the registration identity. Stripping it can
	// borrow an unrelated application's component of the same basename.
	return "", true
}

// idTargetPath is the file a `map( id ).to( path )` names.
func (r *Resolver) idTargetPath(t idTarget, wb *wireboxWorkspace) string {
	path := t.path

	for _, token := range []string{"#moduleMapping#.", "#moduleMapping#/"} {
		if len(path) > len(token) && strings.EqualFold(path[:len(token)], token) {
			return r.ComponentPath(path[len(token):], t.dir)
		}
	}

	if first, rest, ok := strings.Cut(path, "."); ok {
		if dir, mapped := wb.cfmappings[strings.ToLower(first)]; mapped {
			if p := r.ComponentPath(rest, dir); p != "" {
				return p
			}
		}
	}

	return r.ComponentPath(path, t.dir)
}

// inModuleModels is the component called name among the models of any of
// the module directories: ColdBox maps a module's models directory, every
// component in it under its file name. A workspace holding several
// applications holds a copy of a module in each (coldbox-samples has a
// cbstorages per sample app), and the copy that answers is the one in the
// caller's application (sameApplication); two or more there, or none and
// several elsewhere, is ambiguous.
func (r *Resolver) inModuleModels(name string, dirs []string, baseDir string) string {
	if len(dirs) == 0 || name == "" || strings.ContainsAny(name, "./\\") {
		return ""
	}

	var hits []string

	for _, c := range r.Index.FindFilesByBasename(name) {
		for _, d := range dirs {
			if underAny([]string{d}, c) {
				hits = append(hits, c)
			}
		}
	}

	if len(hits) == 0 {
		return ""
	}

	slices.Sort(hits)

	hits = slices.Compact(hits)
	if len(hits) > 1 && baseDir != "" {
		hits = slices.DeleteFunc(hits, func(hit string) bool { return !sameApplication(hit, baseDir) })
	}

	if len(hits) != 1 {
		return ""
	}

	return hits[0]
}

// sameApplication reports whether the module file hit belongs to the
// application baseDir is in: the directory holding hit's nearest `modules`
// folder is an ancestor of baseDir (or baseDir itself).
func sameApplication(hit, baseDir string) bool {
	dir := filepath.Dir(hit)
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}

		if strings.EqualFold(filepath.Base(dir), "modules") {
			return underAny([]string{parent}, baseDir)
		}

		dir = parent
	}
}

// moduleModelRoot follows the two ColdBox configuration stages: public
// properties are captured before configure(), and variables.conventions after.
// Unknown control flow and child injector ownership are withheld.
func moduleModelRoot(content, dir string) string {
	constructor, methods := parser.ApplicationSections(content)

	flags, ok := moduleAssignments(constructor, true)
	if !ok {
		return ""
	}

	for key, required := range map[string]bool{"automapmodels": true, "activate": true, "disabled": false, "moduleinjector": false} {
		if value, exists := flags[key]; exists {
			if len(value) != 1 || !strings.EqualFold(value[0].Value, "true") && !strings.EqualFold(value[0].Value, "false") || strings.EqualFold(value[0].Value, "true") != required {
				return ""
			}
		}
	}

	if value, exists := flags["modelnamespace"]; exists && diString(value) == "" {
		return ""
	}

	cfg, ok := moduleAssignments(constructor, false)
	if !ok {
		return ""
	}

	if configure, exists := methods["configure"]; exists {
		if configure == "" {
			return ""
		}

		writes, valid := moduleAssignments(configure, false)
		if !valid {
			return ""
		}

		maps.Copy(cfg, writes)
	}
	// Environment methods execute after configure when selected at runtime. An
	// unknown environment must not turn a potentially changed root into a default.
	for name, method := range methods {
		if name == "configure" || name == "onload" || name == "onunload" {
			continue
		}

		for _, token := range allTokens(method) {
			if strings.EqualFold(token.Value, "conventions") {
				return ""
			}
		}
	}

	folder := "models"
	if value, exists := cfg["conventions.modelslocation"]; exists {
		folder = diString(value)
	}

	if folder == "" || strings.ContainsAny(folder, "#\\") {
		return ""
	}

	root := filepath.Join(dir, filepath.FromSlash(folder))
	if !underAny([]string{dir}, root) || root == dir {
		return ""
	}

	return root
}

// moduleAssignments reads literal dot/bracket writes in source order. A whole
// conventions struct replaces earlier fields, including the default location.
func moduleAssignments(source string, public bool) (map[string][]parser.Token, bool) {
	result := map[string][]parser.Token{}
	sc := parser.NewScanner(source)

	var prefix []parser.Token

	guarded := false

	for {
		token := sc.NextSkipComments()
		if token.Kind == parser.TokEOF {
			if !public && moduleConventionsReference(prefix) {
				return nil, false
			}

			break
		}

		if token.Kind == parser.TokIdent && slices.Contains([]string{"if", "else", "switch", "for", "while", "try", "catch"}, strings.ToLower(token.Value)) {
			guarded = true
		}

		if token.Kind != parser.TokEquals {
			prefix = append(prefix, token)

			continue
		}

		target, from := diConfigAssignmentTarget(prefix)
		if !public && (moduleConventionsReference(prefix[:max(0, from)]) || from > 0 && strings.EqualFold(prefix[from-1].Value, "var") && moduleConventionsReference(prefix[from:])) {
			return nil, false
		}

		lhs := prefix
		target = strings.ToLower(target)

		value := diReadValue(sc)
		if !public && moduleConventionsReference(value) {
			return nil, false
		}

		prefix = nil

		var key string

		switch {
		case public && strings.HasPrefix(target, "this."):
			key = strings.TrimPrefix(target, "this.")
			if !slices.Contains([]string{"automapmodels", "modelnamespace", "moduleinjector", "activate", "disabled", "cfmapping", "classmapping"}, key) {
				continue
			}
		case !public:
			key = strings.TrimPrefix(target, "variables.")
			if key != "conventions" && key != "conventions.modelslocation" {
				if moduleConventionsReference(lhs) {
					return nil, false
				}

				continue
			}
		default:
			continue
		}

		if guarded {
			return nil, false
		}

		if key == "conventions" {
			fields, ok := diStruct(value)
			if !ok {
				return nil, false
			}

			result["conventions.modelslocation"] = []parser.Token{{Kind: parser.TokString, Value: `"models"`}}
			if location, exists := fields["modelslocation"]; exists {
				result["conventions.modelslocation"] = location
			}
		} else {
			result[key] = value
		}
	}

	return result, true
}

// Only bare/variables conventions are the configuration value. A public or
// other object's member of the same name is unrelated.
func moduleConventionsReference(tokens []parser.Token) bool {
	for i, token := range tokens {
		if token.Kind != parser.TokIdent || !strings.EqualFold(token.Value, "conventions") {
			continue
		}

		if i > 0 && tokens[i-1].Kind == parser.TokDot && (i < 2 || !strings.EqualFold(tokens[i-2].Value, "variables")) {
			continue
		}

		return true
	}

	return false
}

// inModuleMapping resolves a dot-path whose first segment is a module's
// cfmapping, inside the module.
func (r *Resolver) inModuleMapping(component string) string {
	first, rest, ok := strings.Cut(component, ".")
	if !ok || rest == "" {
		return ""
	}

	wb := r.wirebox()
	if wb == nil {
		return ""
	}

	dir, mapped := wb.cfmappings[strings.ToLower(first)]
	if !mapped {
		return ""
	}

	return r.ComponentPath(rest, dir)
}

// uninstalledModule reports whether component is an id of a module the
// workspace does not hold — `RequestStorage@cbstorages` with no cbstorages
// module in sight — which is the module not being installed, not a
// component that is missing.
func (r *Resolver) uninstalledModule(component string) bool {
	_, ns, ok := strings.Cut(component, "@")
	if !ok || ns == "" {
		return false
	}

	wb := r.wirebox()

	return wb == nil || len(wb.namespaces[strings.ToLower(ns)]) == 0
}
