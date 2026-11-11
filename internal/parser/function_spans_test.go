package parser

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.lsp.dev/uri"
)

// A span reads a function where the parse reads it: its name, access,
// parameters, literal defaults and body, by offset in the file.
func TestFunctionSpansReadEachKindOfFunction(t *testing.T) {
	src := `component {
	/** doc */
	public string function first( required string a, b = "x""y", c = 3, d = f() ) access="remote" {
		var s = "{#listLast( p, "{" )#";
		function inner() { return 1; }
		return s;
	}

	private any function abstractOne();

	this.helper = function( z = 'q' ) { return z; };
}
`
	spans := ParseWithOptions("file:///t/C.cfc", src, &ParseOptions{}).FunctionSpans()

	byName := map[string]FunctionSpan{}
	for _, s := range spans {
		byName[s.Name] = s
	}

	first := byName["first"]
	if first.Kind != FunctionDeclared || first.Access != "remote" || first.Line != 2 {
		t.Fatalf("first = %+v", first)
	}

	if got := src[first.Start:first.Params]; !strings.HasPrefix(got, "public string function first(") {
		t.Errorf("first starts %q", got)
	}

	if got := first.ParamsText(src); got != ` required string a, b = "x""y", c = 3, d = f() ` {
		t.Errorf("first params %q", got)
	}

	if want := map[string]string{"b": `x"y`}; !reflect.DeepEqual(first.Defaults, want) {
		t.Errorf("first defaults %v, want %v", first.Defaults, want)
	}

	if body := first.BodyText(src); !strings.HasPrefix(strings.TrimSpace(body), "var s") || !strings.HasSuffix(strings.TrimSpace(body), "return s;") {
		t.Errorf("first body %q", body)
	}

	if src[first.End-1] != '}' {
		t.Errorf("first ends at %q", src[first.End-1])
	}

	if inner := byName["inner"]; inner.Kind != FunctionNested || strings.TrimSpace(inner.BodyText(src)) != "return 1;" {
		t.Errorf("inner = %+v", inner)
	}

	if a := byName["abstractOne"]; a.Kind != FunctionDeclared || a.Access != "private" || a.Body != -1 {
		t.Errorf("abstractOne = %+v", a)
	}

	if h := byName["helper"]; h.Kind != FunctionValue || h.Defaults["z"] != "q" || strings.TrimSpace(h.BodyText(src)) != "return z;" {
		t.Errorf("helper = %+v", h)
	}

	tag := `<cfcomponent>
<!--- <cffunction name="commented"></cffunction> --->
<cffunction name="load" access="package" hint="a > b">
	<cfargument name="id" default="1">
	<cfreturn 1>
</cffunction>
</cfcomponent>`

	spans = Parse("file:///t/T.cfc", tag).FunctionSpans()
	if len(spans) != 1 {
		t.Fatalf("tag spans = %+v", spans)
	}

	if s := spans[0]; s.Kind != FunctionTag || s.Name != "load" || s.Access != "package" || !strings.HasPrefix(strings.TrimSpace(s.BodyText(tag)), "<cfargument") ||
		!strings.HasSuffix(tag[:s.End], "</cffunction>") || s.Params != -1 {
		t.Errorf("tag span = %+v", s)
	}
}

// The functions-only reading is the parse's walk with nothing settled, so
// it finds what a full parse finds, at the same offsets; and a parse edited
// since it ran reads its spans again rather than handing back stale offsets.
func TestFunctionSpansOfAgreesWithTheParse(t *testing.T) {
	n := 0

	err := filepath.WalkDir("../../testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".cfc") && !strings.HasSuffix(path, ".cfm") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(data)
		file := "file:///x/" + filepath.ToSlash(path)
		full := Parse(uri.URI(file), content).FunctionSpans()

		if light := FunctionSpansOf(file, content); !reflect.DeepEqual(full, light) {
			t.Errorf("%s: FunctionSpansOf = %+v\nthe parse's = %+v", path, light, full)
		}

		n++

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if n == 0 {
		t.Fatal("found no fixtures under ../../testdata")
	}

	src := "component {\n\tfunction a() {\n\t\treturn 1;\n\t}\n\tfunction b() { return 2; }\n}\n"
	pr := Parse("file:///t/E.cfc", src)
	pr.ApplyEdit(2, 2, 2, 2, "x = 1; ")

	b, ok := FunctionSpanNamed(pr.FunctionSpans(), "b")
	if !ok || strings.TrimSpace(b.BodyText(pr.Content)) != "return 2;" {
		t.Errorf("after an edit, b = %+v in %q", b, pr.Content)
	}
}

// The headers the resolver's own finders read and the parse did not: each
// lost its function, or its access, until the parse learned it. The finders
// that read the parse's functions now depend on it.
func TestTheParseReadsEveryFunctionHeader(t *testing.T) {
	src := `component {
	private string[] function strings() { return []; }
	package final whatever function modified() { return 1; }
	final package function finalFirst() { return 1; }
	static numeric function staticTyped() { return 1; }
	component function makes() { return this; }
	public function function returnsFunction() { return () => 1; }
	function secured() secured:api access:"remote" { return 1; }
	public static function staticPublic() { return 1; }
}
`
	pr := Parse("file:///t/H.cfc", src)
	spans := pr.FunctionSpans()

	for _, tt := range []struct{ name, access, returns string }{
		{"strings", "private", "array"},
		{"modified", "package", "whatever"},
		{"finalFirst", "package", ""},
		{"staticTyped", "", "numeric"},
		{"makes", "", "component"},
		{"returnsFunction", "public", "function"},
		{"secured", "remote", ""},
		{"staticPublic", "public", ""},
	} {
		s, ok := FunctionSpanNamed(spans, tt.name)
		if !ok {
			t.Errorf("%s: no span in %+v", tt.name, spans)

			continue
		}

		if s.Access != tt.access || s.Kind != FunctionDeclared || strings.TrimSpace(s.BodyText(src)) == "" {
			t.Errorf("%s = %+v, want access %q", tt.name, s, tt.access)
		}

		if !strings.HasSuffix(strings.TrimSpace(src[s.Start:s.Params]), tt.name+"(") {
			t.Errorf("%s starts %q", tt.name, src[s.Start:s.Params])
		}

		found := false

		for i := range pr.Funcs {
			if f := &pr.Funcs[i]; f.Name == tt.name {
				found = true

				if f.ReturnType != tt.returns {
					t.Errorf("%s returns %q, want %q", tt.name, f.ReturnType, tt.returns)
				}
			}
		}

		if !found {
			t.Errorf("%s: not in Funcs", tt.name)
		}
	}

	// `static numeric function` starts at its modifier.
	if s, _ := FunctionSpanNamed(spans, "staticTyped"); !strings.HasPrefix(src[s.Start:], "static numeric") {
		t.Errorf("staticTyped starts %q", src[s.Start:s.Params])
	}

	// A tag function written as text in a script block is no function, and
	// a self-closing one has no body and swallows no function after it.
	tag := `<cfscript>assert( "a <cffunction> in a string" );</cfscript>
<cffunction name="declared" />
<cffunction name="greet"><cfreturn 1></cffunction>`

	tagSpans := Parse("file:///t/page.cfm", tag).FunctionSpans()
	if len(tagSpans) != 2 || tagSpans[0].Name != "declared" || tagSpans[0].Body != -1 || tagSpans[1].Name != "greet" ||
		strings.TrimSpace(tagSpans[1].BodyText(tag)) != "<cfreturn 1>" {
		t.Errorf("tag spans = %+v", tagSpans)
	}
}
