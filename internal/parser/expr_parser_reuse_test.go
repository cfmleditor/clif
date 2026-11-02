package parser

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.lsp.dev/uri"
)

// A tag file reads its expressions with one reused script parser. It must
// give the results fresh parsers gave, for every fixture in the repository
// and a page that mixes the expression shapes a tag holds.
func TestReusedExpressionParserGivesWhatFreshOnesGave(t *testing.T) {
	sources := map[string]string{
		"benchTag": benchTagCFC,
		"mixed": `<cfcomponent><cffunction name="a"><cfset var x = createObject("component","a.B").init()>
<cfset this.y = svc.load(1)><cfset z = x.go(svc.next()).done()><cfif svc.ok(1) and x.m()><cfoutput>#svc.name()# #x.n(1)#</cfoutput></cfif>
<cfset var q = new a.C(); q.d = e.f()><cfreturn x.g()></cffunction></cfcomponent>`,
	}

	err := filepath.WalkDir("../..", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if name := d.Name(); name == "target" || name == ".git" {
				return filepath.SkipDir
			}

			return nil
		}

		if ext := strings.ToLower(filepath.Ext(path)); ext == ".cfm" || ext == ".cfc" || ext == ".cfml" {
			if b, err := os.ReadFile(path); err == nil && len(b) <= 1<<20 {
				sources[path] = string(b)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) < 10 {
		t.Fatalf("found only %d sources", len(sources))
	}

	parse := func(name, src string) *ParseResult {
		return ParseWithOptions(uri.File("/t/"+filepath.Base(name)), src, &ParseOptions{ExtractCalls: true, ExtractLinks: true})
	}

	same := func(a, b *ParseResult) bool {
		return reflect.DeepEqual(a.Funcs, b.Funcs) && reflect.DeepEqual(a.ComponentRefs, b.ComponentRefs) &&
			reflect.DeepEqual(a.AllCalls(), b.AllCalls()) && reflect.DeepEqual(a.Scopes, b.Scopes) &&
			reflect.DeepEqual(a.Instantiations, b.Instantiations)
	}

	for name, src := range sources {
		freshExprParsers = true
		want, again := parse(name, src), parse(name, src)
		freshExprParsers = false

		// Two fresh parses agree, or the comparison below means nothing.
		if !same(want, again) {
			t.Errorf("%s: two parses with fresh sub-parsers differ", name)

			continue
		}

		if !same(want, parse(name, src)) {
			t.Errorf("%s: reused parser gave different results", name)
		}
	}
}
