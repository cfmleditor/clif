package parser

import (
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The expressions the include scanners replaced, kept as the reference the
// scanners must agree with.
var (
	refIncludeTagAt    = regexp.MustCompile(`^(?i)<cfinclude\b[^>]*?\btemplate\s*=\s*["']([^"'#]+|(?:[^"'#*\\/][^"'#*\\]*/)?#[^"'#/]+#\.cfm|[^"'#*\\/][^"'#*\\]*/#[^"'#/]+#|#attributes\.\w+#|#[\w.]+#/[^"'#*\\]+)["']`)
	refIncludeScriptAt = regexp.MustCompile(`^(?i)(?:cf)?include\s*\(?\s*(?:template\s*=\s*)?["']([^"'#]+|(?:[^"'#*\\/][^"'#*\\]*/)?#[^"'#/]+#\.cfm|[^"'#*\\/][^"'#*\\]*/#[^"'#/]+#|#attributes\.\w+#|#[\w.]+#/[^"'#*\\]+)["']`)
	// refRootForm is the shape RootVariablePath narrows by the variable's name.
	refRootForm        = regexp.MustCompile(`^#[\w.]+#/[^#*\\]+$`)
	refListingDir      = regexp.MustCompile(`(?i)^#\s*getDirectoryFromPath\s*\(\s*getCurrentTemplatePath\s*\(\s*\)\s*\)\s*#([\w./-]+?)/?$`)
	refListingIncluded = regexp.MustCompile(`(?i)^([\w./-]*)#\s*([\w$]+)\.name\s*#$`)
)

// siteAtRegexp reads the include at i, an "include", with the expressions.
func siteAtRegexp(content string, i int) (IncludeSite, bool) {
	start, re := includeFormAtRegexp(content, i)
	if re == nil {
		return IncludeSite{}, false
	}

	m := re.FindStringSubmatchIndex(content[start:min(len(content), start+includeWindow)])
	if m == nil {
		return IncludeSite{}, false
	}

	p := content[start+m[2] : start+m[3]]
	if _, rooted := RootVariablePath(strings.TrimSpace(p)); !rooted && refRootForm.MatchString(strings.TrimSpace(p)) {
		return IncludeSite{}, false
	}

	return includeSite(content, p, start)
}

// siteAtScanner reads the include at i with the scanners.
func siteAtScanner(content string, i int) (IncludeSite, bool) {
	start, form := includeFormAt(content, i)
	if form == nil {
		return IncludeSite{}, false
	}

	window := content[start:min(len(content), start+includeWindow)]

	from, to, ok := form(window)
	if !ok {
		return IncludeSite{}, false
	}

	return includeSite(content, window[from:to], start)
}

func includeFormAtRegexp(content string, i int) (int, *regexp.Regexp) {
	start := i
	if i >= 2 && strings.EqualFold(content[i-2:i], "cf") {
		start = i - 2
	}

	if start >= 1 && content[start-1] == '<' {
		if start == i {
			return 0, nil // "<include" is not a CFML tag
		}

		return start - 1, refIncludeTagAt
	}

	if start >= 1 {
		if c := content[start-1]; c == '.' || c == '_' || isAlnum(c) {
			return 0, nil
		}
	}

	return start, refIncludeScriptAt
}

// scannersAgree reports the first "include" in content the scanners and the
// expressions read differently.
func scannersAgree(t *testing.T, name, content string) {
	t.Helper()

	for i := indexFold(content, "include"); i >= 0; i = indexFoldFrom(content, "include", i+len("include")) {
		got, gotOK := siteAtScanner(content, i)
		want, wantOK := siteAtRegexp(content, i)

		if got != want || gotOK != wantOK {
			t.Fatalf("%s: %q at %d: scanners %v %v, expressions %v %v", name, content, i, got, gotOK, want, wantOK)
		}
	}
}

var includeSamples = map[string]string{
	"tag":                      `<cfinclude template="a.cfm">`,
	"tag upper":                `<CFINCLUDE TEMPLATE = 'b.cfml' >`,
	"tag later attr":           `<cfinclude x="1" template="c.cfm">`,
	"tag dynamic":              `<cfinclude template="#x#.cfm">`,
	"tag dynamic then static":  `<cfinclude template="#x#" template="d.cfm">`,
	"tag computed name":        `<cfinclude template="pages/#x.y#.cfm">`,
	"tag computed name upper":  `<cfinclude template="#x#.CFM">`,
	"tag computed directory":   `<cfinclude template="#x#/a.cfm">`,
	"tag root variable":        `<cfinclude template="#application.baseUrl#/lib/a.cfm">`,
	"script root variable":     `include "#variables.rootPath#/lib/a.cfm"; include "#dynamicDir#/b.cfm";`,
	"root variable and a span": `<cfinclude template="#application.baseUrl#/lib/#page#.cfm">`,
	"tag computed mapping":     `<cfinclude template="/m/#x#.cfm">`,
	"tag computed part":        `<cfinclude template="page_#x#.cfm">`,
	"tag computed two":         `<cfinclude template="#x##y#.cfm">`,
	"tag computed backslash":   `<cfinclude template="a\#x#.cfm">`,
	"tag computed both":        `<cfinclude template="#d#/#x#.cfm">`,
	"tag backslash directory":  `<cfinclude template="a\b/#x#.cfm">`,
	"tag glob directory":       `<cfinclude template="*/#x#.cfm">`,
	"script computed name":     `include "../v/#local.view#.cfm";`,
	"script computed cfml":     `include "#view#.cfml";`,
	"script listed name":       `include "specs/#file#";`,
	"script listed and listed": `x = directoryList( expandPath( "specs" ), false, "name", "*.cfm" ); include "specs/#file#";`,
	"script listed recursive":  `x = directoryList( expandPath( "specs" ), true, "name", "*.cfm" ); include "specs/#file#";`,
	"tag attribute":            `<cfinclude template="#attributes.file#">`,
	"tag attribute dotted":     `<cfinclude template="#attributes.a.b#">`,
	"tag attribute upper":      `<CFINCLUDE TEMPLATE="#Attributes.File#">`,
	"tag no boundary":          `<cfincludex template="e.cfm">`,
	"tag attr word":            `<cfinclude mytemplate="f.cfm">`,
	"tag after end":            `<cfinclude a="1"> template="g.cfm"`,
	"script":                   `include "h.cfm";`,
	"script paren":             `cfinclude( template = "i.cfm" );`,
	"script template no eq":    `include template "j.cfm";`,
	"script mixed quotes":      `include "k.cfm';`,
	"member":                   `arr.include("l.cfm")`,
	"ident tail":               `myinclude "m.cfm"`,
	"comment":                  `<!--- <cfinclude template="n.cfm"> --->`,
	"url":                      `<cfinclude template="http://x/o.cfm">`,
	"not cfml":                 `<cfinclude template="p.js">`,
	"spaces in path":           `<cfinclude template=" q.cfm ">`,
	"open overlapping close":   `<!--- a <!---> <cfinclude template="r.cfm"> ---> <cfinclude template="s.cfm">`,
	"nested comments":          `<!--- <!--- x ---> <cfinclude template="t.cfm"> ---> <cfinclude template="u.cfm">`,
	"listing": `<cfdirectory action="list" directory="#getDirectoryFromPath(getCurrentTemplatePath())#dbUpdates" name="rsUpdates" filter="*.cfm">` +
		`<cfloop query="rsUpdates"><cfinclude template="dbUpdates/#rsUpdates.name#"></cfloop>`,
	"listing spaced": `<CFDIRECTORY ACTION='list' DIRECTORY='# getDirectoryFromPath ( getCurrentTemplatePath ( ) ) #up/' NAME='q' FILTER='*.cfm'>` +
		`<cfinclude template='up/# q.NAME #'>`,
	"listing slash only": `<cfdirectory action="list" directory="#getDirectoryFromPath(getCurrentTemplatePath())#/" name="q" filter="*.cfm"><cfinclude template="/#q.name#">`,
	"listing recursive":  `<cfdirectory action="list" recurse="yes" directory="#getDirectoryFromPath(getCurrentTemplatePath())#a" name="q" filter="*.cfm"><cfinclude template="a/#q.name#">`,
	"listing nested open": `<cfdirectory <cfdirectory action="list" directory="#getDirectoryFromPath(getCurrentTemplatePath())#a" name="q" filter="*.cfm">` +
		`<cfinclude template="a/#q.name#">`,
	"listing unclosed": `<cfdirectory action="list" directory="#getDirectoryFromPath(getCurrentTemplatePath())#a" name="q" filter="*.cfm"`,
	"listing commented": `<!--- <cfdirectory action="list" directory="#getDirectoryFromPath(getCurrentTemplatePath())#a" name="q" filter="*.cfm"> --->` +
		`<cfinclude template="a/#q.name#">`,
}

// The include forms are read at a found keyword by the Application.cfc walk;
// what IncludeSites finds is TestExtractIncludes' and its neighbours'.
func TestIncludeScannersMatchTheirExpressions(t *testing.T) {
	for name, src := range includeSamples {
		scannersAgree(t, name, src)
	}

	// Splice fragments of the samples together at random, so the scanners
	// meet every form cut short, run together and nested in the others.
	var pieces []string

	for _, src := range includeSamples {
		for i := 0; i < len(src); i += 3 {
			pieces = append(pieces, src[i:min(len(src), i+1+i%11)])
		}
	}

	pieces = append(pieces, " ", "\t", "\n", "\v", "=", "'", `"`, "#", ">", "<", "(", ")", ".", "/", "-", "$", "_", "<!---", "--->",
		"include", "cfinclude", "<cfinclude", "<!--->", "<!--", "-->", "template", "a.cfm", "attributes", "x")
	slices.Sort(pieces)

	rng := rand.New(rand.NewPCG(3, 4))

	for range 100000 {
		var b strings.Builder
		for range rng.IntN(24) {
			b.WriteString(pieces[rng.IntN(len(pieces))])
		}

		scannersAgree(t, "spliced", b.String())
	}
}

func TestListingScannersMatchTheirExpressions(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 6))
	pieces := []string{"#", " ", "\t", "getDirectoryFromPath", "GETCURRENTTEMPLATEPATH", "getCurrentTemplatePath", "(", ")", "/", "a", ".", "-", "_", "x y", "#", "", "1"}
	includePieces := []string{"#", " ", "\t", "q", "$", ".name", ".NAME", ".nam", "/", "up", ".", "-", "_", "'", "x"}

	for range 100000 {
		var dir, inc strings.Builder
		for range rng.IntN(12) {
			dir.WriteString(pieces[rng.IntN(len(pieces))])
		}

		for range rng.IntN(10) {
			inc.WriteString(includePieces[rng.IntN(len(includePieces))])
		}

		gotDir, ok := listingDir(dir.String())

		want := refListingDir.FindStringSubmatch(dir.String())
		if ok != (want != nil) || ok && gotDir != want[1] {
			t.Fatalf("listingDir(%q) = %q, %v; expression %v", dir.String(), gotDir, ok, want)
		}

		prefix, query, ok := listingIncludePath(inc.String())

		want = refListingIncluded.FindStringSubmatch(inc.String())
		if ok != (want != nil) || ok && (prefix != want[1] || query != want[2]) {
			t.Fatalf("listingIncludePath(%q) = %q, %q, %v; expression %v", inc.String(), prefix, query, ok, want)
		}
	}
}
