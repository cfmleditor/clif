package resolve

import (
	"math/rand"
	"slices"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// hasWrittenFunc and functionBodyNamed read the functions the parse found
// (parser.FunctionSpans), so they answer as the parse reads the file: a
// function in a comment is none, a brace in a string does not end a body, a
// '>' in a quoted attribute does not end a tag, and a function generated for
// a property is not written.
func TestTheFunctionReadersReadTheParsesFunctions(t *testing.T) {
	t.Run("hasWrittenFunc", func(t *testing.T) {
		for _, tt := range []struct {
			text, name string
			want       bool
		}{
			{"component { function init() {} }", "init", true},
			{"component { public any function Init() {} }", "init", true},
			{"component { function initFoo() {} }", "init", false},
			{"component {\n// function init() {}\n}", "init", false},
			{"component { x = \"function init() {}\"; }", "init", false},
			{"component { this.init = function() {}; }", "init", true},
			{"component accessors=true { property name=\"foo\"; }", "getFoo", false},
			{"component accessors=true { property name=\"foo\"; function getFoo() {} }", "getFoo", true},
			{"<cfcomponent><cffunction name=\"init\"></cffunction></cfcomponent>", "init", true},
			{"<cfcomponent><cffunction hint=\"a > b\" name=\"init\"></cffunction></cfcomponent>", "init", true},
			{"<cfcomponent><cffunction name=\"initFoo\"></cffunction></cfcomponent>", "init", false},
			{"<cfcomponent><!--- <cffunction name=\"init\"></cffunction> ---></cfcomponent>", "init", false},
		} {
			pr := parser.Parse("file:///t/C.cfc", tt.text)
			if got := hasWrittenFunc(pr, tt.name); got != tt.want {
				t.Errorf("hasWrittenFunc(%q, %q) = %v, want %v", tt.text, tt.name, got, tt.want)
			}
		}
	})

	t.Run("functionBodyNamed", func(t *testing.T) {
		for _, tt := range []struct{ text, name, want string }{
			{"function init() { return this; }", "init", "return this;"},
			{"function init() { if (a) { b(); } }", "init", "if (a) { b(); }"},
			{"function init() { x = \"}\"; return x; }", "init", "x = \"}\"; return x;"},
			{"/* function init() { a } */ function init() { b }", "init", "b"},
			{"function init() { a } function init() { b }", "init", ""},
			{"function init", "init", ""},
			{"function init(", "init", ""},
			{"function init() no braces", "init", ""},
			{"function init() { unclosed", "init", ""},
			{"function init();", "init", ""},
			{"<cffunction name=\"init\"><cfargument name=\"a\">body()</cffunction>", "init", "body()"},
			{"<cffunction name=\"init\">one</cffunction><cffunction name=\"init\">two</cffunction>", "init", ""},
			{"<cffunction name=\"init\">one</cffunction><cffunction name=\"other\">two</cffunction>", "init", "one"},
			{"<cffunction name=\"init\">no close", "init", ""},
			{"<cffunction xname=\"init\">a</cffunction>", "init", ""},
			{"<cffunction x name=\"init\">a</cffunction>", "init", "a"},
			{"<cffunction hint=\"a>b\" name=\"init\">a</cffunction>", "init", "a"},
			{"<cffunction name=\"init\">a</CFFUNCTION>", "init", "a"},
			{"<!--- <cffunction name=\"init\">x</cffunction> ---><cffunction name=\"init\">a</cffunction>", "init", "a"},
			{"<cffunction name=\"init\">a</cffunction> function init() { b }", "init", "a"},
		} {
			if got := functionBodyNamed(tt.text, parser.FunctionSpansOf("", tt.text), tt.name); got != tt.want {
				t.Errorf("functionBodyNamed(%q, %q) = %q, want %q", tt.text, tt.name, got, tt.want)
			}
		}
	})
}

// A literal tested before a reader is a promise: everything the reader finds
// holds it. If one did not, a file the reader would have read is skipped,
// quietly. The text here is built from whole calls each reader reads, the
// same cut short, in the other case and run together, so that calls occur and
// the promise is tried where it is most likely to break.
func TestEveryPrefilterLiteralIsHeldByEveryCallItsReaderReads(t *testing.T) {
	type pair struct {
		name    string
		re      func(string) bool
		literal string
	}

	entity := func(s string) bool {
		e, _ := superInitEntity(s)

		return e != ""
	}
	argument := func(s string) bool {
		_, a := superInitEntity(s)

		return a != ""
	}
	maps := func(s string) bool { return len(binderMaps(s)) > 0 }

	pairs := []pair{
		{"superInitEntity super", entity, "super"},
		{"superInitEntity init", entity, "init"},
		{"superInitEntity argument", argument, "super"},
		{"binderMaps map", maps, "map"},
		{"binderMaps to", maps, "to"},
		{"includedUDFPaths", func(s string) bool { return len(includedUDFPaths(s)) > 0 }, "includeUDF"},
		{"helperSetting module", func(s string) bool { return len(helperSetting(s, true)) > 0 }, "applicationHelper"},
		{"helperSetting application", func(s string) bool { return len(helperSetting(s, false)) > 0 }, "applicationHelper"},
	}

	pieces := []string{
		"super.init( entityName = \"cbContent\" )", "SUPER.INIT(\"x\")", "super.init( arguments.entity, a )", "super.init(entity)", "super.init (x)",
		"super . init(", "super.init", "super.initx(", "supper.init(",
		"binder.map( \"a\" ).to( \"b.C\" )", "MAP(\"a\").TO(\"b\")", "map( 'a' )\n.\nto( 'b' )", "map(\"a\")", ".to(\"b\")", "map (\"a\").to(\"b\")",
		"includeUDF( \"#moduleMapping#/helpers.cfm\" )", "INCLUDEUDF(udflibrary=\"#moduleMapping#/x\")", "includeUDF", "include UDF(",
		"this.applicationHelper = [ \"a.cfm\" ]", "THIS.APPLICATIONHELPER=[\"a\"]", "applicationHelper : \"x\"", "applicationHelper = 'x'", "applicationHelpers=", "application Helper =",
		"{", "}", "(", ")", ",", ".", "\"", "'", "=", ":", "[", "]", "#", "\n", " ", "x", "entityName", "arguments.", "return ", "function ",
	}

	r := rand.New(rand.NewSource(7))
	matched := map[string]int{}

	for range 40000 {
		var b strings.Builder

		for range 1 + r.Intn(6) {
			b.WriteString(pieces[r.Intn(len(pieces))])

			if r.Intn(3) == 0 {
				b.WriteByte(" \n"[r.Intn(2)])
			}
		}

		text := b.String()

		for _, p := range pairs {
			if p.re(text) {
				matched[p.name]++

				if !mentions(text, p.literal) {
					t.Fatalf("%s matches %q, which does not hold the literal %q its prefilter asks for", p.name, text, p.literal)
				}
			}
		}
	}

	for _, p := range pairs {
		if matched[p.name] < 100 {
			t.Errorf("%s matched in only %d texts: the generator does not exercise it", p.name, matched[p.name])
		}
	}
}

// allTokens remembers its answer, so a text asked for twice is tokenized once.
// What it hands out is shared: its capacity is its length, so an append copies
// rather than writing into the memory, and the callers that edit tokens copy first.
func TestAllTokensIsRememberedAndShared(t *testing.T) {
	text := "function f() {\n\tx = 1;\n\n\ty = 2;\n\treturn x + y;\n}\n// a comment long enough to pass the minimum\n"

	first := allTokens(text)
	if len(first) == 0 || &first[0] != &allTokens(text)[0] {
		t.Fatal("the same text was tokenized twice")
	}

	if cap(first) != len(first) {
		t.Errorf("cap %d beyond len %d: an append would write into the memory", cap(first), len(first))
	}

	before := slices.Clone(first)
	_ = significantTokens(text)

	if !slices.Equal(first, before) {
		t.Error("significantTokens edited the remembered tokens")
	}
}

// splitStatements cuts at semicolons and block braces, keeps a struct literal's
// braces inside its statement, and returns windows onto the tokens it was given.
func TestSplitStatementsReturnsWindowsOntoItsInput(t *testing.T) {
	body := allTokens("a = 1; if (x) { b = 2; } c = { k: 1, j: { z: 2 } }; return c")

	var got []string

	for _, st := range splitStatements(body) {
		var words []string
		for _, tk := range st {
			words = append(words, tk.Value)
		}

		got = append(got, strings.Join(words, " "))

		if cap(st) != len(st) {
			t.Errorf("statement %q has capacity %d beyond its length %d", got[len(got)-1], cap(st), len(st))
		}
	}

	want := []string{"a = 1", "if ( x )", "b = 2", "c = { k : 1 , j : { z : 2 } }", "return c"}
	if !slices.Equal(got, want) {
		t.Errorf("statements = %q, want %q", got, want)
	}
}
