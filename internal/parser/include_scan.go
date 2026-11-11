package parser

import (
	"path"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// The scanners here read an include form by hand at a place a caller has
// found, which the Application.cfc walk does (mapping_sources.go); and the
// shapes of path an include may name. They were regular expressions, and each
// mirrors the expression it replaced exactly:
// TestIncludeScannersMatchTheirExpressions holds them to it. IncludeSites
// finds includes with the document walk instead (includes.go).

// isRegexpSpace is \s in Go's regexp syntax: no vertical tab.
func isRegexpSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\f' || c == '\r'
}

func skipRegexpSpace(s string, i int) int {
	for i < len(s) && isRegexpSpace(s[i]) {
		i++
	}

	return i
}

// wordStartAt is \b before s[i] when s[i] is a word byte.
func wordStartAt(s string, i int) bool {
	return i == 0 || !lex.IsWord(s[i-1])
}

// wordEndAt is \b after a word ending before s[i].
func wordEndAt(s string, i int) bool {
	return i >= len(s) || !lex.IsWord(s[i])
}

func isQuote(c byte) bool { return c == '"' || c == '\'' }

// tagAttrValues calls value for each place, from i up to the tag's first '>',
// that `\bname\s*=\s*` stands, with the offset just past it, until value
// reports a match. It is the `[^>]*?\bname\s*=\s*` of a tag pattern: the
// earliest occurrence whose remainder matches wins, as a lazy repeat's would.
func tagAttrValues(s string, i int, name string, value func(j int) bool) bool {
	end := strings.IndexByte(s[i:], '>')
	if end < 0 {
		end = len(s)
	} else {
		end += i
	}

	for p := i; p < end; p++ {
		// The attribute name holds no '>', so the name may run past end only
		// as the tail of a lazy repeat would: it starts before the '>'.
		if !lex.HasPrefixFold(s[p:], name) || !wordStartAt(s, p) {
			continue
		}

		j := skipRegexpSpace(s, p+len(name))
		if j >= len(s) || s[j] != '=' {
			continue
		}

		if value(skipRegexpSpace(s, j+1)) {
			return true
		}
	}

	return false
}

// quotedPath reads `["']([^"'#]+)["']` at j: the path's bounds.
func quotedPath(s string, j int) (start, end int, ok bool) {
	if j >= len(s) || !isQuote(s[j]) {
		return 0, 0, false
	}

	k := j + 1
	for k < len(s) && !isQuote(s[k]) && s[k] != '#' {
		k++
	}

	if k == j+1 || k >= len(s) || !isQuote(s[k]) {
		return 0, 0, false
	}

	return j + 1, k, true
}

// includeTagAt reads `^(?i)<cfinclude\b[^>]*?\btemplate\s*=\s*` and an
// includePathAt
// and returns the path's bounds.
func includeTagAt(s string) (start, end int, ok bool) {
	const open = "<cfinclude"
	if !lex.HasPrefixFold(s, open) || !wordEndAt(s, len(open)) {
		return 0, 0, false
	}

	found := tagAttrValues(s, len(open), "template", func(j int) bool {
		start, end, ok = includePathAt(s, j)

		return ok
	})

	return start, end, found
}

// includeScriptAt reads
// `^(?i)(?:cf)?include\s*\(?\s*(?:template\s*=\s*)?` and an includePathAt.
func includeScriptAt(s string) (start, end int, ok bool) {
	i := 0
	if lex.HasPrefixFold(s, "cf") {
		i = 2
	}

	if !lex.HasPrefixFold(s[i:], "include") {
		return 0, 0, false
	}

	i = skipRegexpSpace(s, i+len("include"))
	if i < len(s) && s[i] == '(' {
		i++
	}

	i = skipRegexpSpace(s, i)

	// The optional template= is taken only when it is whole: a "template"
	// without its '=' leaves the quote test to fail on its 't', which is
	// where the pattern's other branch fails too.
	if lex.HasPrefixFold(s[i:], "template") {
		if j := skipRegexpSpace(s, i+len("template")); j < len(s) && s[j] == '=' {
			i = skipRegexpSpace(s, j+1)
		}
	}

	return includePathAt(s, i)
}

// includePathAt is the path an include names at j: a literal one
// (quotedPath), one whose file name alone is computed, which
// computedNameGlob reads as a glob, or one custom tag attribute
// (AttributeInclude). It is
// `([^"'#]+|(?:[^"'#*\\/][^"'#*\\]*/)?#[^"'#/]+#\.cfm|[^"'#*\\/][^"'#*\\]*/#[^"'#/]+#|#attributes\.\w+#)`
// between quotes.
func includePathAt(s string, j int) (start, end int, ok bool) {
	if start, end, ok = quotedPath(s, j); ok {
		return start, end, true
	}

	if j >= len(s) || !isQuote(s[j]) {
		return 0, 0, false
	}

	k := j + 1
	for k < len(s) && !isQuote(s[k]) {
		k++
	}

	if k >= len(s) {
		return 0, 0, false
	}

	if !isComputedPath(s[j+1 : k]) {
		return 0, 0, false
	}

	return j + 1, k, true
}

// RootVariablePath is the rooted path left of p when it begins with one plain
// root variable and a slash, `#application.baseUrl#/common/function/page.cfm`:
// the variable holds the application's root (see namesARoot), which is how a site installed under a
// path (Galaxie Blog) writes every include, and what follows names the template
// from that root. Only a plain variable counts, not a call or an expression;
// the rest of the path is literal.
func RootVariablePath(p string) (string, bool) {
	if len(p) < len("#x#/y") || p[0] != '#' {
		return "", false
	}

	end := strings.IndexByte(p[1:], '#')
	if end < 1 {
		return "", false
	}

	name, rest := p[1:1+end], p[2+end:]
	if len(rest) < 2 || rest[0] != '/' || strings.ContainsAny(rest, "#*\\") {
		return "", false
	}

	for i := range len(name) {
		if c := name[i]; !isIdentPart(c) && c != '.' {
			return "", false
		}
	}

	if !namesARoot(name) {
		return "", false
	}

	return rest, true
}

// namesARoot reports whether the last segment of a variable's name says it
// holds the application's root: baseUrl, basePath, appRoot, webRoot, rootDir.
// `#dynamicDir#/page.cfm` is some directory a variable computes, which is not
// read as a root.
func namesARoot(name string) bool {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		name = name[i+1:]
	}

	name = strings.ToLower(name)
	for _, suffix := range []string{"url", "path", "dir", "uri"} {
		if trimmed := strings.TrimSuffix(name, suffix); trimmed != "" {
			name = trimmed

			break
		}
	}

	return strings.HasSuffix(name, "root") || strings.HasSuffix(name, "base") && !strings.HasSuffix(name, "database")
}

// isComputedPath reports whether p is a computed path an include may name:
// computedNameGlob's, listedNameGlob's or AttributeInclude's.
func isComputedPath(p string) bool {
	if _, ok := RootVariablePath(p); ok {
		return true
	}

	if _, ok := computedNameGlob(p); ok {
		return true
	}

	if _, _, ok := listedNameGlob(p); ok {
		return true
	}

	if _, ok := ComputedPrefixTemplate(p); ok {
		return true
	}

	_, ok := AttributeInclude(p)

	return ok
}

// listingDir reads
// `(?i)^#\s*getDirectoryFromPath\s*\(\s*getCurrentTemplatePath\s*\(\s*\)\s*\)\s*#([\w./-]+?)/?$`
// and returns the directory it names.
func listingDir(s string) (string, bool) {
	i := 0

	for _, part := range []string{"#", "getDirectoryFromPath", "(", "getCurrentTemplatePath", "(", ")", ")", "#"} {
		if part != "#" || i > 0 {
			i = skipRegexpSpace(s, i)
		}

		if !lex.HasPrefixFold(s[i:], part) {
			return "", false
		}

		i += len(part)
	}

	rest := s[i:]
	if rest == "" {
		return "", false
	}

	for j := range len(rest) {
		if c := rest[j]; !lex.IsWord(c) && c != '.' && c != '/' && c != '-' {
			return "", false
		}
	}

	// The lazy capture leaves one trailing '/' to the optional one, unless
	// that would leave the capture empty.
	if len(rest) > 1 && rest[len(rest)-1] == '/' {
		rest = rest[:len(rest)-1]
	}

	return rest, true
}

// listingIncludePath reads p, an include's template, as
// `^([\w./-]*)#\s*([\w$]+)\.name\s*#$`: a directory and the name a listing's
// query row holds, `dbUpdates/#rsUpdates.name#`. It returns the directory as
// written and the query's name.
func listingIncludePath(p string) (prefix, query string, ok bool) {
	k := 0
	for k < len(p) && (lex.IsWord(p[k]) || p[k] == '.' || p[k] == '/' || p[k] == '-') {
		k++
	}

	if k >= len(p) || p[k] != '#' {
		return "", "", false
	}

	q := skipRegexpSpace(p, k+1)

	r := q
	for r < len(p) && (lex.IsWord(p[r]) || p[r] == '$') {
		r++
	}

	if r == q || !lex.HasPrefixFold(p[r:], ".name") {
		return "", "", false
	}

	e := skipRegexpSpace(p, r+len(".name"))
	if e != len(p)-1 || p[e] != '#' {
		return "", "", false
	}

	return p[:k], p[q:r], true
}

// ComputedPrefixTemplate reads an include whose directory is one computed span
// and whose file name is a literal template: `#backportdir#lucee.cfm` is a
// template called lucee.cfm in whatever directory the variable holds. It
// returns the file name. Masa includes its engine backports this way.
func ComputedPrefixTemplate(p string) (string, bool) {
	if len(p) < len("#x#a.cfm") || p[0] != '#' || strings.Count(p, "#") != 2 {
		return "", false
	}

	end := strings.LastIndexByte(p, '#')

	span, name := p[1:end], p[end+1:]
	if span == "" || strings.ContainsAny(span, " \t\"'()/\\") || name == "" || strings.ContainsAny(name, "/\\*") ||
		!strings.EqualFold(path.Ext(name), ".cfm") {
		return "", false
	}

	return name, true
}
