package parser

import (
	"path"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// includeWindow bounds how far past the keyword a match may reach. A tag with
// its attributes or a script include with its path is far shorter; the bound
// only has to stop a stray keyword costing a scan to the end of the file.
const includeWindow = 512

// ExtractIncludes returns the static paths a file cfincludes, in source order
// and without duplicates, exactly as written.
//
// An included file runs in its includer's variables scope, so a function one
// declares is callable unqualified from the other. That is the relationship the
// resolver needs, and it is why this is separate from ExtractLinks: a link is
// anything that names a file — href, action, <cfmodule template> — and a
// cfmodule runs in a scope of its own, so reading one as an include would
// resolve calls through a file they can never reach.
//
// Only a path naming a CFML file counts, which is also what keeps the script
// form off prose that happens to say `include "…"`. An include inside a
// <!--- ---> comment is skipped, since a commented-out include is not a live one.
//
// Includes are found by the walk document links use (fileRefs), with only
// what an include needs read: one reading of what is a statement, a comment
// and a tag. This scan used to find the keyword anywhere, so a commented-out
// `// include "x.cfm"`, one in a string and one in a page's prose were each
// read as live.
func ExtractIncludes(content string) []string {
	return includePaths(IncludeSites(content))
}

// includePaths is the paths of sites, in order and without duplicates.
func includePaths(sites []IncludeSite) []string {
	var out []string

	for _, site := range sites {
		if !slices.ContainsFunc(out, func(s string) bool { return strings.EqualFold(s, site.Path) }) {
			out = append(out, site.Path)
		}
	}

	return out
}

// IncludeSite is one include statement: the path as ExtractIncludes reports
// it, and the byte offset the statement starts at.
type IncludeSite struct {
	Path   string
	Offset int
}

// IncludeSites is every include ExtractIncludes reads, one per statement and
// with where it is, so a template can be read as its includer sees it at
// that line: a template included inside a function sees the function's locals.
// The script and tag forms come first, in source order, then the directory
// listings' globs.
func IncludeSites(content string) []IncludeSite {
	if IndexFold(content, "include") < 0 {
		return nil
	}

	regions, _ := ClassifyRegionsIdx(content)

	return includeSitesIn(content, regions)
}

// includeSitesIn is IncludeSites over regions already split from content.
func includeSitesIn(content string, regions []Region) []IncludeSite {
	if IndexFold(content, "include") < 0 {
		return nil
	}

	var (
		out      []IncludeSite
		listed   map[string]string // query name → directory, lowercased name
		fromList []IncludeSite     // includes of a listing's names: Path is the template as written
		wrappers []IncludeSite     // includes of an argument: Path is the argument's name
	)

	emit := func(kind fileRefKind, at, start, end int) {
		switch kind {
		case refListing:
			if dir, name, ok := qualifyingListing(content[start:end]); ok {
				listed = withListing(listed, name, dir)
			}
		case refInclude:
			raw := content[start:end]
			if _, _, ok := listingIncludePath(raw); ok {
				fromList = append(fromList, IncludeSite{Path: raw, Offset: at})
			}

			if site, ok := includeSite(content, raw, at); ok {
				out = append(out, site)
			} else if param, ok := argumentInclude(raw); ok {
				wrappers = append(wrappers, IncludeSite{Path: param, Offset: at})
			}
		case refTemplate, refHref, refAction:
		}
	}

	regions = fileRegions(content, regions)
	for i := range regions {
		regionFileRefs(&regions[i], true, emit)
	}

	if len(wrappers) > 0 {
		out = append(out, wrapperIncludes(content, regions, wrappers)...)
	}

	if len(listed) > 0 {
		out = append(out, listingIncludes(fromList, listed)...)
	}

	return out
}

// argumentInclude reports the lowercased argument whose value a function
// includes, when raw is that one argument, bare or through one call:
// `#arguments.template#`, `#LCase(arguments.template)#`.
func argumentInclude(raw string) (string, bool) {
	if len(raw) < len("#arguments.x#") || raw[0] != '#' || raw[len(raw)-1] != '#' || strings.Count(raw, "#") != 2 {
		return "", false
	}

	expr := strings.TrimSpace(raw[1 : len(raw)-1])

	// One wrapping call, LCase(...) or trim(...).
	if open := strings.IndexByte(expr, '('); open > 0 && expr[len(expr)-1] == ')' && isIdentName(expr[:open]) {
		expr = strings.TrimSpace(expr[open+1 : len(expr)-1])
	}

	const prefix = "arguments."
	if !lex.HasPrefixFold(expr, prefix) || !isIdentName(expr[len(prefix):]) {
		return "", false
	}

	return strings.ToLower(expr[len(prefix):]), true
}

// isIdentName reports whether s is one CFML identifier.
func isIdentName(s string) bool {
	if s == "" || !isIdentStart(s[0]) {
		return false
	}

	for i := 1; i < len(s); i++ {
		if !isIdentPart(s[i]) {
			return false
		}
	}

	return true
}

// IncludeWrapper is a function that includes the template one of its
// arguments names: Param, lowercased, at Position among its arguments.
type IncludeWrapper struct {
	Param    string
	Position int
}

// IncludeWrappers is the functions content declares that include the
// template an argument names, `include "#arguments.template#"` or through one
// call (`#LCase(arguments.template)#`), by lowercased name. Wheels' Global.cfc
// declares $include so; a call of one is an include into the variables scope
// of the component whose method runs it. Finding them takes a parse, which
// only a file with such an include pays for.
func IncludeWrappers(content string) map[string]IncludeWrapper {
	if IndexFold(content, "arguments.") < 0 || IndexFold(content, "include") < 0 {
		return nil
	}

	var argIncludes []IncludeSite

	emit := func(kind fileRefKind, at, start, end int) {
		if kind != refInclude {
			return
		}

		if param, ok := argumentInclude(content[start:end]); ok {
			argIncludes = append(argIncludes, IncludeSite{Path: param, Offset: at})
		}
	}

	regions, _ := ClassifyRegionsIdx(content)

	regions = fileRegions(content, regions)
	for i := range regions {
		regionFileRefs(&regions[i], true, emit)
	}

	return includeWrappersIn(content, argIncludes)
}

// includeWrappersIn is IncludeWrappers from argIncludes, each include of an
// argument in content with the argument's name as its Path.
func includeWrappersIn(content string, argIncludes []IncludeSite) map[string]IncludeWrapper {
	if len(argIncludes) == 0 {
		return nil
	}

	scopes := FindFuncScopes(content)

	var (
		defs     []FunctionDef
		wrappers map[string]IncludeWrapper
	)

	for _, inc := range argIncludes {
		line := strings.Count(content[:inc.Offset], "\n")

		var fn string

		for _, sc := range scopes {
			if sc.Start <= line && line <= sc.End {
				fn = sc.Name // the innermost, which comes last
			}
		}

		if fn == "" {
			continue
		}

		if defs == nil {
			defs = ParseFunctionDefs("", content)
		}

		for i := range defs {
			if !strings.EqualFold(defs[i].Name, fn) {
				continue
			}

			for k, a := range defs[i].Arguments {
				if strings.EqualFold(a.Name, inc.Path) {
					if wrappers == nil {
						wrappers = map[string]IncludeWrapper{}
					}

					wrappers[strings.ToLower(fn)] = IncludeWrapper{Param: inc.Path, Position: k}
				}
			}
		}
	}

	return wrappers
}

// wrapperIncludes is the includes made through an include wrapper content
// declares (IncludeWrappers): Wheels' Global.cfc loads the application's
// routes with $include(template = "/config/routes.cfm"), which runs them in
// Global's own variables scope. A call of the function in the same file,
// unqualified and with a literal path for that argument, named or in its
// place, is an include there. A call through another object runs the template
// in that object's scope, which the resolver decides
// (wheels_include_wrapper.go).
//
// argIncludes holds each include of an argument, the argument's name as its
// Path.
func wrapperIncludes(content string, regions []Region, argIncludes []IncludeSite) []IncludeSite {
	params := includeWrappersIn(content, argIncludes)
	if len(params) == 0 {
		return nil
	}

	var out []IncludeSite

	for i := range regions {
		r := &regions[i]
		if r.Kind != RegionScript {
			continue
		}

		sc := NewScriptScanner(r.Text)
		prev := Token{Kind: TokSemicolon}

		for {
			t := sc.NextSkipComments()
			if t.Kind == TokEOF {
				break
			}

			before := prev
			prev = t

			p, ok := params[strings.ToLower(t.Value)]
			if t.Kind != TokIdent || !ok || before.Kind == TokDot || sc.PeekSkipComments().Kind != TokLParen {
				continue
			}

			sc.NextSkipComments()

			if tmpl, ok := wrapperArgument(sc, p.Param, p.Position); ok {
				if site, ok := includeSite(content, tmpl, r.Offset+t.Offset); ok {
					out = append(out, site)
				}
			}
		}
	}

	return out
}

// wrapperArgument reads a call's arguments from just past its '(' for the
// literal string given for the argument called name, at position among them:
// by name, or in its place when the call names none.
func wrapperArgument(sc *Scanner, name string, position int) (string, bool) {
	for k := 0; ; k++ {
		t := sc.NextSkipComments()

		var value Token

		if t.Kind == TokIdent {
			if op := sc.PeekSkipComments(); op.Kind == TokEquals || op.Kind == TokColon {
				sc.NextSkipComments()

				if v := sc.NextSkipComments(); strings.EqualFold(t.Value, name) && v.Kind == TokString {
					value = v
				}
			}
		} else if t.Kind == TokString && k == position {
			value = t
		}

		// The argument ends at a ',' or the call's ')'; anything more makes
		// it an expression, not a literal.
		end := sc.NextSkipComments()
		if value.Kind == TokString && len(value.Value) >= 2 && (end.Kind == TokComma || end.Kind == TokRParen) {
			return value.Value[1 : len(value.Value)-1], true
		}

		if end.Kind != TokComma {
			return "", false
		}
	}
}

// withListing records a listing, making the map on first use: most files have
// none.
func withListing(listed map[string]string, name, dir string) map[string]string {
	if listed == nil {
		listed = map[string]string{}
	}

	listed[name] = dir

	return listed
}

// includeSite reads raw, the path an include statement at gives, as the site
// it is: a literal template, or a computed path the resolver can read
// (isComputedPath), as a glob where that is what it names.
func includeSite(content, raw string, at int) (IncludeSite, bool) {
	if raw == "" || strings.ContainsAny(raw, `"'`) || strings.Contains(raw, "#") && !isComputedPath(raw) {
		return IncludeSite{}, false
	}

	p := strings.TrimSpace(raw)
	if glob, ok := computedNameGlob(p); ok {
		p = glob
	} else if glob, dir, ok := listedNameGlob(p); ok {
		if !listsTemplates(content, dir) {
			return IncludeSite{}, false
		}

		p = glob
	} else if attr, ok := AttributeInclude(p); ok {
		return IncludeSite{Path: "#attributes." + attr + "#", Offset: at}, true
	}

	if p == "" || strings.Contains(p, "://") || !isIncludable(p) {
		return IncludeSite{}, false
	}

	return IncludeSite{Path: p, Offset: at}, true
}

// includeFormAt decides which form the "include" found at i begins, and where
// that form starts: "<cfinclude" at the '<', "cfinclude" or "include" at the
// keyword. It returns a nil reader for an occurrence that is neither — the
// tail of a longer identifier or a member call such as arr.include(…). The
// Application.cfc walk (mapping_sources.go) reads its includes with it.
func includeFormAt(content string, i int) (int, func(string) (int, int, bool)) {
	start := i
	if i >= 2 && strings.EqualFold(content[i-2:i], "cf") {
		start = i - 2
	}

	if start >= 1 && content[start-1] == '<' {
		if start == i {
			return 0, nil // "<include" is not a CFML tag
		}

		return start - 1, includeTagAt
	}

	if start >= 1 {
		if c := content[start-1]; c == '.' || c == '_' || isAlnum(c) {
			return 0, nil
		}
	}

	return start, includeScriptAt
}

func isAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// isIncludable reports whether p names a template cfinclude can run. It is not
// cfpath.IsCFMLFile, which imports this package.
func isIncludable(p string) bool {
	ext := strings.ToLower(path.Ext(p))

	return ext == ".cfm" || ext == ".cfml"
}

// qualifyingListing reads one <cfdirectory> tag and reports the directory
// and lowercased query name of a listing listingIncludes can follow: a
// literal directory beside the listing file, not recursive, and filtered to
// templates. Anything computed is not a static answer.
func qualifyingListing(tag string) (dir, name string, ok bool) {
	attrs := tagAttrMap(tag, cfAttrs)

	dir, ok = listingDir(attrs["directory"])
	if !strings.EqualFold(attrs["action"], "list") || !ok || attrs["name"] == "" ||
		strings.EqualFold(attrs["recurse"], "true") || strings.EqualFold(attrs["recurse"], "yes") ||
		!strings.EqualFold(attrs["filter"], "*.cfm") || strings.Contains(dir, "..") {
		return "", "", false
	}

	return dir, strings.ToLower(attrs["name"]), true
}

// listingIncludes is the glob includes of a file that lists a directory of
// templates beside itself and includes each one it finds. Mura applies its
// database updates this way: configBean lists dbUpdates/*.cfm and includes
// every one, so each runs in configBean's variables scope.
//
// The include is a glob, `sub/*.cfm`, which the resolver expands. includes
// is the includes of a listing's names (listingIncludePath), with the
// template as written, and listed the qualifying listings by query name.
func listingIncludes(includes []IncludeSite, listed map[string]string) []IncludeSite {
	var out []IncludeSite

	for _, inc := range includes {
		prefix, query, _ := listingIncludePath(inc.Path)

		dir, ok := listed[strings.ToLower(query)]
		if !ok || !strings.EqualFold(strings.TrimSuffix(prefix, "/"), dir) {
			continue
		}

		out = append(out, IncludeSite{Path: dir + "/*.cfm", Offset: inc.Offset})
	}

	return out
}

// computedNameGlob reads an include whose file name is computed and whose
// directory is not — `#current.action#.cfm`, `layouts/#docFormat#.cfm` — as
// every template in that directory, the glob a directory listing gives. It
// is how a dispatcher picks a page by name: Lucee's admin web.cfm includes
// `#url.action#.cfm`, so every page beside it runs there and calls the
// helpers web_functions.cfm declares. The name must be one #...# span and
// nothing else, the extension .cfm, and the directory relative and literal:
// a mapping, a computed directory or a name with a literal part is not read.
func computedNameGlob(p string) (string, bool) {
	dir, name := path.Split(p)
	if len(name) < len("#x#.cfm") || !strings.EqualFold(name[len(name)-len(".cfm"):], ".cfm") {
		return "", false
	}

	if !computedName(dir, name[:len(name)-len(".cfm")]) {
		return "", false
	}

	if dir == "" {
		return "./*.cfm", true
	}

	return dir + "*.cfm", true
}

// computedName reports whether stem is one #...# span and dir a relative,
// literal directory, as computedNameGlob and listedNameGlob need.
func computedName(dir, stem string) bool {
	return len(stem) >= len("#x#") && stem[0] == '#' && stem[len(stem)-1] == '#' && strings.Count(stem, "#") == 2 &&
		!strings.ContainsAny(dir, "#*\\") && !strings.HasPrefix(dir, "/")
}

// listedNameGlob reads an include whose whole file name is computed, extension
// and all, in a written directory — `specs/#file#` — as the glob of that
// directory's templates, and returns the directory as written. A test suite
// lists its spec directory and includes each file it finds:
//
//	var specs = DirectoryList( ExpandPath( "specs" ), false, "name", "*.cfm" );
//	for( var file in specs ){ include "specs/#file#"; }
//
// The name says nothing about the extension, so the glob holds only when the
// file lists that directory for templates as well (listsTemplates); without
// a directory, `#attributes.file#`, it says nothing about where the file is.
func listedNameGlob(p string) (glob, dir string, ok bool) {
	dir, name := path.Split(p)
	if dir == "" || !computedName(dir, name) {
		return "", "", false
	}

	return dir + "*.cfm", dir, true
}

// listsTemplates reports whether content calls directoryList on dir, written
// as the include writes it and not recursively, filtered to `*.cfm`: the
// listing that makes listedNameGlob's glob the files the include can name.
// A <cfdirectory> listing is listingIncludes' to read.
func listsTemplates(content, dir string) bool {
	dir = strings.TrimSuffix(strings.TrimPrefix(dir, "./"), "/")
	if dir == "" {
		return false
	}

	if listsTemplatesByTag(content, dir) {
		return true
	}

	const call = "directoryList"

	for i := indexFold(content, call); i >= 0; i = indexFoldFrom(content, call, i+len(call)) {
		args, ok := callArgs(content, i+len(call))
		if ok && (strings.Contains(args, `"*.cfm"`) || strings.Contains(args, `'*.cfm'`)) &&
			quotesDir(args, dir) && !hasWordFold(args, "true") && !hasWordFold(args, "yes") {
			return true
		}
	}

	return false
}

// callArgs is the text between the parentheses of a call whose name ends at
// i, bounded by includeWindow; quotes are not stepped over, which a listing's
// arguments do not need.
func callArgs(content string, i int) (string, bool) {
	i = skipRegexpSpace(content, i)
	if i >= len(content) || content[i] != '(' {
		return "", false
	}

	depth := 0

	for j := i; j < min(len(content), i+includeWindow); j++ {
		switch content[j] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return content[i+1 : j], true
			}
		}
	}

	return "", false
}

// quotesDir reports whether args holds dir as a string literal, with or
// without a leading "./" and a trailing slash.
func quotesDir(args, dir string) bool {
	for _, q := range []string{`"`, `'`} {
		for _, d := range []string{dir, dir + "/", "./" + dir, "./" + dir + "/"} {
			if indexFold(args, q+d+q) >= 0 {
				return true
			}
		}
	}

	return false
}

// hasWordFold reports whether word appears in s as a whole word.
func hasWordFold(s, word string) bool {
	for i := indexFold(s, word); i >= 0; i = indexFoldFrom(s, word, i+len(word)) {
		if wordStartAt(s, i) && wordEndAt(s, i+len(word)) {
			return true
		}
	}

	return false
}

// AttributeInclude reports the lowercased attribute name of an include whose
// path is exactly one custom tag attribute, `#attributes.file#`. A custom tag
// that runs the template it is handed (`<cf_runtest file="core/test_x.cfm">`)
// includes it into the tag's own variables scope, so what the tag includes
// beside it is callable from the template. Which templates those are is what
// the tag's callers pass, which the resolver reads; IncludeSites reports the
// path in this one spelling so it can.
func AttributeInclude(p string) (string, bool) {
	const prefix = "#attributes."

	if len(p) <= len(prefix)+1 || !lex.HasPrefixFold(p, prefix) || p[len(p)-1] != '#' {
		return "", false
	}

	name := p[len(prefix) : len(p)-1]
	for i := range len(name) {
		if !lex.IsWord(name[i]) {
			return "", false
		}
	}

	return strings.ToLower(name), true
}

// listsTemplatesByTag reports whether content lists templates with a
// <cfdirectory action="list" filter="*.cfm"> and spells the include's
// directory, less any leading `../`, somewhere in its text: Galaxie's
// updater builds the directory from a variable (`root & "installer/updates/"`)
// and includes `../../installer/updates/#migration.file#` for each file found.
func listsTemplatesByTag(content, dir string) bool {
	tail := strings.TrimLeft(dir, "./")
	if tail == "" {
		return false
	}

	const tag = "<cfdirectory"

	for i := indexFold(content, tag); i >= 0; i = indexFoldFrom(content, tag, i+len(tag)) {
		end := tagEndIndex(content[i:])
		if end < 0 {
			continue
		}

		attrs := content[i : i+end]
		if hasWordFold(attrs, "list") && (strings.Contains(attrs, `"*.cfm"`) || strings.Contains(attrs, `'*.cfm'`)) &&
			!hasWordFold(attrs, "recurse") && spellsDirectory(content, tail) {
			return true
		}
	}

	return false
}

// spellsDirectory reports whether content writes tail as the start of a string
// outside a <cfinclude> or <cfdirectory> tag: the line that builds the
// directory the listing reads, not the include or the listing themselves.
func spellsDirectory(content, tail string) bool {
	for i := indexFold(content, tail); i >= 0; i = indexFoldFrom(content, tail, i+1) {
		if i == 0 || (content[i-1] != '"' && content[i-1] != '\'') {
			continue
		}

		open := strings.LastIndexByte(content[:i], '<')
		if open >= 0 && (lex.HasPrefixFold(content[open:], "<cfinclude") || lex.HasPrefixFold(content[open:], "<cfdirectory")) {
			continue
		}

		return true
	}

	return false
}
