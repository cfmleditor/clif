package resolve

import (
	"math/rand"
	"slices"
	"strings"
	"testing"
)

// The scanners in text_rules.go stand in for regular expressions, and the only
// thing that makes that safe is that they answer the same on every input. The
// expressions are kept (the *Regex functions) and are the reference here.
//
// The text is random but not uniform: it is drawn from the tokens the
// expressions care about — the keywords in mixed case, every kind of
// whitespace including \v, which RE2's \s leaves out, quotes, runs of '=',
// '>' and the tag openers, and the names themselves — so that a fraction of
// the cases come near a match, and the rest sit on the edges of one.
var scannerFragments = []string{
	"function", "Function", "FUNCTION", "functions", "xfunction", "_function", "var", "VAR", "variable", "for", "FOR",
	"in", "IN", "int", "item", "index", "name", "Name", "names", "returnVariable", "returnvariable", "result", "local.",
	"LOCAL.", "variables.", "Variables.", "arguments.", "this.", "<cffunction", "<CFFUNCTION", "<cffunctionx", "</cffunction>",
	"</CFFUNCTION>", "<cfargument a=\"b\">", "<cfset", "init", "Init", "INIT", "x", "X", "arguments.x", "getFoo", "a",
	"attributes.file", "(", ")", "{", "}", "[", "]", "=", "==", "===", "+=", "-=", "*=", "/=", "&=", "!=", "<=", "++", "--", "+", "-",
	";", ",", ".", "$", "_", "\"", "'", ">", "<", "#", " ", "  ", "\t", "\n", "\r", "\f", "\v", " ", "é", "\xff", "1", "7",
	"=\"", "='", "\"x\"", "'x'", "\"init\"", "'init'", "= ", " =", " = ", "x = 1", "x == 1", "x++", "var x", "var x;",
	"for (var x in", "for (x in", "item=\"x\"", "name = 'init'", "name=\"init\"", "function init(", "function init (",
	"function  x(", "<cffunction name=\"init\">", "<cffunction hint=\"h\" name='init' output=\"false\">",
}

var scannerNames = []string{"init", "x", "arguments.x", "getFoo", "a", "name", "in", "var", "function", "attributes.file", "X", "Init", "_x", "x1"}

// namedFragments are the shapes each rule looks for, written around name, so
// that a good share of the cases are one edit from a match.
var namedFragments = []string{
	"function %s(", "function %s (", "function  %s(", "FUNCTION\t%s\n(", "<cffunction name=\"%s\">", "<cffunction hint=\"h\" name='%s'>",
	"<CFFUNCTION NAME = \"%s\" output=\"false\">", "{ body( %s ) }", "{ if (a) { %s = 1; } }", "</cffunction>", "var %s", "var %s;", "for (%s in a)",
	"for (var %s in a)", "%s = 1", "%s += 1", "%s++", "%s == 1", "local.%s = 1", "variables.%s = 1", "item=\"%s\"", "returnVariable = '%s'",
	"%sSuffix", "prefix%s", ".%s", "$%s", "\"%s\"", "'%s'",
}

func randomText(r *rand.Rand, name string) string {
	var b strings.Builder

	n := 1 + r.Intn(24)

	for range n {
		if r.Intn(3) == 0 {
			b.WriteString(strings.ReplaceAll(namedFragments[r.Intn(len(namedFragments))], "%s", name))
		} else {
			b.WriteString(scannerFragments[r.Intn(len(scannerFragments))])
		}

		// Mostly adjacent, sometimes separated, so that boundaries are tested
		// both ways.
		if r.Intn(3) == 0 {
			b.WriteByte(" \t\n"[r.Intn(3)])
		}
	}

	return b.String()
}

func TestTheTextScannersAreTheExpressionsTheyReplace(t *testing.T) {
	r := rand.New(rand.NewSource(20261106))

	const cases = 60000

	var writes, assigns, bodies int

	for range cases {
		name := scannerNames[r.Intn(len(scannerNames))]
		text := randomText(r, name)

		if want, got := writesFunctionRegex(text, name), writesFunction(text, name); want != got {
			t.Fatalf("writesFunction(%q, %q) = %v, the expression says %v", text, name, got, want)
		} else if want {
			writes++
		}

		if want, got := assignsNameRegex(text, name), assignsName(text, name); want != got {
			t.Fatalf("assignsName(%q, %q) = %v, the expression says %v", text, name, got, want)
		} else if want {
			assigns++
		}

		if want, got := functionBodyNamedRegex(text, name), functionBodyNamed(text, name); want != got {
			t.Fatalf("functionBodyNamed(%q, %q) = %q, the expression says %q", text, name, got, want)
		} else if want != "" {
			bodies++
		}
	}

	// A comparison that never sees a match proves nothing. Each rule must have
	// matched in a real share of the cases, and missed in most.
	for name, n := range map[string]int{"writesFunction": writes, "assignsName": assigns, "functionBodyNamed": bodies} {
		if n < cases/200 || n > cases*9/10 {
			t.Errorf("%s answered yes in %d of %d cases: the generator no longer exercises it both ways", name, n, cases)
		}
	}
}

// The shapes each expression is written for, one by one, so that a failure of
// the random test above has a name and a reader has the rule in front of them.
func TestTheTextScannersOnTheirOwnShapes(t *testing.T) {
	type c struct {
		text, name string
		want       bool
	}

	t.Run("writesFunction", func(t *testing.T) {
		for _, tt := range []c{
			{"function init() {}", "init", true},
			{"FUNCTION  Init  (", "init", true},
			{"public function init(", "init", true},
			{"function\n\tinit\n(", "init", true},
			{"function initFoo(", "init", false},
			{"function init", "init", false},
			{"xfunction init(", "init", false},
			{"_function init(", "init", false},
			{"function\vinit(", "init", false},
			{"function init\v(", "init", false},
			{"function init\f(", "init", true},
			{"<cffunction name=\"init\">", "init", true},
			{"<CFFUNCTION hint=\"h\" NAME='init' output=\"false\">", "init", true},
			{"<cffunction name = \"init\" >", "init", true},
			{"<cffunction\nname=\"init\">", "init", true},
			{"<cffunction name=\"initFoo\">", "init", false},
			{"<cffunction name=init>", "init", false},
			{"<cffunction>name=\"init\"", "init", false},
			{"<cffunction names=\"init\">", "init", false},
			{"<cffunction xname=\"init\">", "init", true}, // no \b before name here, unlike functionBodyNamed
			{"<cffunction", "init", false},
			{"name=\"init\"", "init", false},
			{"<cffunctionname=\"init\">", "init", false},
		} {
			if got := writesFunction(tt.text, tt.name); got != tt.want || got != writesFunctionRegex(tt.text, tt.name) {
				t.Errorf("writesFunction(%q, %q) = %v, want %v (expression %v)", tt.text, tt.name, got, tt.want, writesFunctionRegex(tt.text, tt.name))
			}
		}
	})

	t.Run("assignsName", func(t *testing.T) {
		for _, tt := range []c{
			{"x = 1", "x", true},
			{"x=1", "x", true},
			{"x += 1", "x", true},
			{"x&=a", "x", true},
			{"x++", "x", true},
			{"x --", "x", true},
			{"x == 1", "x", false},
			{"x\v= 1", "x", false},
			{"x\f= 1", "x", true},
			{"var\vx", "x", false},
			{"for\v(x in a)", "x", false},
			{"x === 1", "x", false},
			{"x != 1", "x", false},
			{"x <= 1", "x", false},
			{"x =", "x", false},
			{"x +=", "x", false},
			{"x +==1", "x", false},
			{"a.x = 1", "x", false},
			{"$x = 1", "x", false},
			{"1x = 1", "x", false},
			{"_x = 1", "x", false},
			{"(x = 1", "x", true},
			{"local.x = 1", "x", true},
			{"VARIABLES.x = 1", "x", true},
			{"alocal.x = 1", "x", false},
			{"xlocal.x = 1", "x", false},
			{"this.x = 1", "x", false},
			{"var x", "x", true},
			{"VAR\tx;", "x", true},
			{"var xy", "x", false},
			{"var x_", "x", false},
			{"avar x", "x", false},
			{"for (x in a)", "x", true},
			{"for(var x in a)", "x", true},
			{"FOR ( VAR\nx\nIN a", "x", true},
			{"for (var x of a)", "x", true}, // `var x` is a declaration whatever follows
			{"for (x int a)", "x", false},
			{"for (x in", "x", true},
			{"xfor (x in a)", "x", false},
			{"for (xx in a)", "x", false},
			{"for (var in a)", "x", false},
			{"item=\"x\"", "x", true},
			{"returnVariable = 'x'", "x", true},
			{"<cfloop index=\"x\" from=\"1\">", "x", true},
			{"variable=\"x\"", "x", true},
			{"avariable=\"x\"", "x", false},
			{"names=\"x\"", "x", false},
			{"item=\"xy\"", "x", false},
			{"item=\"x", "x", false},
			{"result==\"x\"", "x", false},
			{"x = 1", "arguments.x", false},
			{"arguments.x = 1", "arguments.x", true},
			{"a.arguments.x = 1", "arguments.x", false},
		} {
			if got := assignsName(tt.text, tt.name); got != tt.want || got != assignsNameRegex(tt.text, tt.name) {
				t.Errorf("assignsName(%q, %q) = %v, want %v (expression %v)", tt.text, tt.name, got, tt.want, assignsNameRegex(tt.text, tt.name))
			}
		}
	})

	t.Run("functionBodyNamed", func(t *testing.T) {
		for _, tt := range []struct{ text, name, want string }{
			{"function init() { return this; }", "init", "return this;"},
			{"function init() { if (a) { b(); } }", "init", "if (a) { b(); }"},
			{"function init() { a } function init() { b }", "init", ""},
			{"function init", "init", ""},
			{"function init(", "init", ""},
			{"function init() no braces", "init", ""},
			{"function init() { unclosed", "init", ""},
			{"<cffunction name=\"init\"><cfargument name=\"a\">body()</cffunction>", "init", "body()"},
			{"<cffunction name=\"init\">one</cffunction><cffunction name=\"init\">two</cffunction>", "init", ""},
			{"<cffunction name=\"init\">one</cffunction><cffunction name=\"other\">two</cffunction>", "init", "one"},
			{"<cffunction name=\"init\">no close", "init", ""},
			{"<cffunction xname=\"init\">a</cffunction>", "init", ""},
			{"<cffunction x name=\"init\">a</cffunction>", "init", "a"},
			{"<cffunctionx name=\"init\">a</cffunction>", "init", ""},
			{"<cffunction hint=\"a>b\" name=\"init\">a</cffunction>", "init", ""},
			{"<cffunction name=\"init\">a</CFFUNCTION>", "init", "a"},
			{"<cffunction name=\"init\">a</cffunction> function init() { b }", "init", "a"},
		} {
			if got := functionBodyNamed(tt.text, tt.name); got != tt.want || got != functionBodyNamedRegex(tt.text, tt.name) {
				t.Errorf("functionBodyNamed(%q, %q) = %q, want %q (expression %q)", tt.text, tt.name, got, tt.want, functionBodyNamedRegex(tt.text, tt.name))
			}
		}
	})

	t.Run("a name that is not plain takes the expression", func(t *testing.T) {
		for _, name := range []string{"", "a b", "a>b", "é", "a\tb"} {
			if plainName(name) {
				t.Errorf("plainName(%q) = true", name)
			}
		}
	})
}

// A literal tested before an expression is a promise: every match of the
// expression holds it. If one did not, a file the expression would have matched
// is skipped, quietly. The text here is built from whole matches of each
// expression, the same cut short, in the other case and run together, so that
// matches occur and the promise is tried where it is most likely to break.
func TestEveryPrefilterLiteralIsHeldByEveryMatchOfItsExpression(t *testing.T) {
	type pair struct {
		name    string
		re      func(string) bool
		literal string
	}

	pairs := []pair{
		{"superInitEntityRe", superInitEntityRe.MatchString, "super.init("},
		{"superInitArgRe", superInitArgRe.MatchString, "super.init("},
		{"binderMapRe map", binderMapRe.MatchString, "map("},
		{"binderMapRe to", binderMapRe.MatchString, "to("},
		{"includeUDFRe", includeUDFRe.MatchString, "includeUDF("},
		{"moduleHelperRe", moduleHelperRe.MatchString, "applicationHelper"},
		{"appHelperRe", appHelperRe.MatchString, "applicationHelper"},
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

// quotedNameStarts and renderStarts are held to their expressions the same way,
// on text drawn from what each looks for: the name or the call, beside quotes,
// brackets and the words that almost make one.
func TestTheNameAndRenderScannersAreTheExpressionsTheyReplace(t *testing.T) {
	r := rand.New(rand.NewSource(11))

	renderPieces := []string{
		"view", "View", "VIEW", "setView", "SETVIEW", "renderView", "RenderView", "set", "render", "xview", "preview", "setrenderView", "renderset",
		"(", " (", "\n(", "\t(", "\v(", ")", "\"", "'", "/", ".", "_", "$", "1", " ", "\n", "event.", "return ", "views", "viewer(",
	}
	namePieces := []string{"x", "X", "name", "Name", "xname", "name1", "_name", "$name", "name_", "\"", "'", "\"\"", " ", "\n", "=", "(", ")", ".", "-", "é"}

	var renderHits, nameHits int

	for range 50000 {
		var b strings.Builder

		for range 1 + r.Intn(10) {
			b.WriteString(renderPieces[r.Intn(len(renderPieces))])
		}

		text := b.String()

		var want []int
		for _, m := range renderStartRe.FindAllStringIndex(text, -1) {
			want = append(want, m[0])
		}

		got := renderStarts(text)
		if len(want) != len(got) || (len(want) > 0 && !slices.Equal(want, got)) {
			t.Fatalf("renderStarts(%q) = %v, the expression says %v", text, got, want)
		}

		renderHits += len(want)

		name := scannerNames[r.Intn(len(scannerNames))]

		b.Reset()

		for range 1 + r.Intn(12) {
			if r.Intn(2) == 0 {
				b.WriteString(name)
			} else {
				b.WriteString(namePieces[r.Intn(len(namePieces))])
			}
		}

		text = b.String()

		if want, got := quotedNameStartsRegex(text, name), quotedNameStarts(text, name); !slices.Equal(want, got) {
			t.Fatalf("quotedNameStarts(%q, %q) = %v, the expression says %v", text, name, got, want)
		} else {
			nameHits += len(want)
		}
	}

	if renderHits < 2000 || nameHits < 2000 {
		t.Errorf("the generator found %d render starts and %d quoted names: it does not exercise them", renderHits, nameHits)
	}
}
