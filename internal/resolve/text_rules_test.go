package resolve

import (
	"math/rand"
	"slices"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
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

	var assigns int

	for range cases {
		name := scannerNames[r.Intn(len(scannerNames))]
		text := randomText(r, name)

		if want, got := assignsNameRegex(text, name), assignsName(text, name); want != got {
			t.Fatalf("assignsName(%q, %q) = %v, the expression says %v", text, name, got, want)
		} else if want {
			assigns++
		}
	}

	// A comparison that never sees a match proves nothing. Each rule must have
	// matched in a real share of the cases, and missed in most. (writesFunction
	// and functionBodyNamed were held to their expressions here too; they read
	// the parse's functions now, which is not what any expression says: see
	// TestTheFunctionReadersReadTheParsesFunctions.)
	if assigns < cases/200 || assigns > cases*9/10 {
		t.Errorf("assignsName answered yes in %d of %d cases: the generator no longer exercises it both ways", assigns, cases)
	}
}

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

// The shapes each expression is written for, one by one, so that a failure of
// the random test above has a name and a reader has the rule in front of them.
func TestTheTextScannersOnTheirOwnShapes(t *testing.T) {
	type c struct {
		text, name string
		want       bool
	}

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

func pickText(r *rand.Rand, pieces []string, most int) string {
	var b strings.Builder

	for range 1 + r.Intn(most) {
		b.WriteString(pieces[r.Intn(len(pieces))])
	}

	return b.String()
}

// The expressions that begin with a fixed word are run at the places that word
// is. Each is compared with the expression it stands for on text built from the
// expression's own pieces — its word in either case and beside what almost makes
// a match — and the comparison is of the submatch indices, so a capture that
// moved is a failure too.
func TestTheAnchoredExpressionsMatchWhatTheyReplace(t *testing.T) {
	r := rand.New(rand.NewSource(13))

	render := []string{
		"view", "View", "setView", "renderView", "RENDERVIEW", "xview", "preview", "view (", "view(", "\n", " ", "\t", "\v", "(", ")", ",", "\"", "'", "/", ".",
		"home/index", "x.y", "a-b", "view=", "view = ", "module=\"m\"", "module = 'x'", "#variables.v#", "#variables.", "#", "/sub/page", "data", "=", ";", "{", "}",
		`view("home/index")`, `renderView( view = "x/y", module="m" )`, `setView('/a.b/c-d')`, `View ( "p" , x )`, `view("#variables.v#/sub")`, `renderView( view="#variables.handler#/x/" )`, `view("a")view("b")`, `xview("q")`,
		"function", "FUNCTION", "functions", "xfunction", "function ", "function  run(", "function\tinit (", "function (", "function run", "function _x1(",
	}
	intercept := []string{
		"announce", "ANNOUNCE", "announceInterception", "AnnounceInterception", "processState", "processstate", "announces", "xannounce", "(", " (", "\n(", "\v(", ")", ",",
		"\"", "'", " ", "\n", "x", "state", "data", "{", "}", "=",
	}
	fw1 := []string{
		"view", "View", "xview", "views", "(", " (", ")", ",", "\"", "'", ":", "path", "path=", "path = ", "mod:", "a-b:", "x/y", "z", "/", " ", "\n", "\t", "_", ".",
		`view("a/b")`, `view( path = 'mod:x/y' , z)`, `view("m-n:p/q",`, `View ( "p" )`, `xview("q")`, `view("a b")`, `view(path="x")`,
	}
	prc := []string{
		"prc", "PRC", "xprc", "prcs", ".", " . ", "response", "Response", "[", "]", "\"", "'", "=", "==", " ", "\n", "x", "(", ")", ".response", "[\"response\"]", "['response']", "prc.response =",
	}
	cet := []string{
		"customErrorTemplate", "CUSTOMERRORTEMPLATE", "customErrorTemplates", "xcustomErrorTemplate", ":", "=", " : ", " = ", "\"", "'", "/error.cfm", "x", "#", "#x#", "\n", ",", "{", "}", ";", "a b",
	}

	var hits [5]int

	for range 40000 {
		text := pickText(r, render, 14)

		if want, got := renderCallRe.FindAllStringSubmatchIndex(text, -1), renderCallMatches(text, renderStarts(text)); !slices.EqualFunc(want, got, slices.Equal) {
			t.Fatalf("renderCallMatches(%q) = %v, the expression says %v", text, got, want)
		} else {
			hits[0] += len(want)
		}

		if want, got := computedRenderRe.FindAllStringSubmatchIndex(text, -1), computedRenderMatches(text, renderStarts(text)); !slices.EqualFunc(want, got, slices.Equal) {
			t.Fatalf("computedRenderMatches(%q) = %v, the expression says %v", text, got, want)
		} else {
			hits[1] += len(want)
		}

		text = pickText(r, intercept, 12)

		var wantStarts []int
		for _, m := range interceptStartRe.FindAllStringIndex(text, -1) {
			wantStarts = append(wantStarts, m[0])
		}

		if got := interceptStarts(text); !slices.Equal(wantStarts, got) {
			t.Fatalf("interceptStarts(%q) = %v, the expression says %v", text, got, wantStarts)
		} else {
			hits[2] += len(wantStarts)
		}

		text = pickText(r, fw1, 14)

		if want, got := fw1ViewCallRe.FindAllStringSubmatchIndex(text, -1), fw1ViewCallMatches(text); !slices.EqualFunc(want, got, slices.Equal) {
			t.Fatalf("fw1ViewCallMatches(%q) = %v, the expression says %v", text, got, want)
		} else {
			hits[3] += len(want)
		}

		text = pickText(r, prc, 10)

		if want, got := prcResponseWriteRe.MatchString(text), writesPrcResponse(text); want != got {
			t.Fatalf("writesPrcResponse(%q) = %v, the expression says %v", text, got, want)
		} else if want {
			hits[4]++
		}

		text = pickText(r, cet, 12)

		var wantValues []string
		for _, m := range customErrorTemplateRe.FindAllStringSubmatch(text, -1) {
			wantValues = append(wantValues, m[1])
		}

		if got := customErrorTemplateValues(text); !slices.Equal(wantValues, got) {
			t.Fatalf("customErrorTemplateValues(%q) = %q, the expression says %q", text, got, wantValues)
		}
	}

	for i, n := range hits {
		if n < 300 {
			t.Errorf("expression %d matched %d times: the generator does not exercise it", i, n)
		}
	}
}

// initStoredArg's per-line test, and the scanner for it, on the lines of an
// init() as they are written.
func TestTheLineWriteScannerIsTheExpressionItReplaces(t *testing.T) {
	r := rand.New(rand.NewSource(17))

	pieces := []string{
		"variables.", "VARIABLES.", "arguments.", "this.", "local.", "set", "SET", "x", "X", "xx", "$", "_", ".", "=", "==", "===", "!=", "<=", " ", "\t", "\v", "(", ")",
		";", "1", "\"", "'", ",", "name", "getX(", "setX(", "setx (", "SetX\t(", "x =", "x=", "x==", "x = arguments.p", "variables.x = arguments.p;",
	}

	var hits int

	for range 60000 {
		name := scannerNames[r.Intn(len(scannerNames))]
		line := pickText(r, pieces, 10)

		if r.Intn(3) == 0 {
			line += strings.Repeat(" ", r.Intn(2)) + name + pickText(r, pieces, 4)
		}

		if want, got := lineWritesNameRegex(line, name), lineWritesName(line, name); want != got {
			t.Fatalf("lineWritesName(%q, %q) = %v, the expression says %v", line, name, got, want)
		} else if want {
			hits++
		}
	}

	if hits < 300 {
		t.Errorf("the expression matched %d lines: the generator does not exercise it", hits)
	}
}

// lastMemberAssignmentAt's search, on text built from the expression's pieces:
// the prefixes it allows, whitespace that crosses lines, and what ends a line.
func TestTheMemberAssignmentScannerIsTheExpressionItReplaces(t *testing.T) {
	r := rand.New(rand.NewSource(23))

	pieces := []string{
		"prc.x", "PRC.X", "prc.xy", "xprc.x", "arguments.", "ARGUMENTS.", "<cfset", "<CFSET", "<cfset ", "<cfset\n", "<cfsetprc.x", " ", "  ", "\t", "\n", "\n\n", "\r\n", "\v", "\f",
		"=", " = ", "==", "x", "1", "new A()", "f(", ")", ";", "/>", ">", "\"", "# ", "foo.bar", "variables.", "rc.x", "= \n", "\n= ",
	}
	assign := []string{
		"prc.x = new A();", "<cfset prc.x = f(1)>", "  arguments.prc.x = 2;\n", "\tprc.x=x /> ", "<cfset  arguments.prc.x = y>\n", "prc.x ==1", "prc.x =\n3", "prc.x\n= 4;",
	}

	var hits int

	for range 40000 {
		var parts []string

		for range 1 + r.Intn(10) {
			if r.Intn(4) == 0 {
				parts = append(parts, assign[r.Intn(len(assign))])
			} else {
				parts = append(parts, pieces[r.Intn(len(pieces))])
			}
		}

		text := strings.Join(parts, "")
		want := memberAssignRe("prc", "x").FindAllStringSubmatchIndex(text, -1)

		if got := memberAssignMatches(text, "prc", "x"); !slices.EqualFunc(want, got, slices.Equal) {
			t.Fatalf("memberAssignMatches(%q) = %v, the expression says %v", text, got, want)
		}

		hits += len(want)
	}

	if hits < 2000 {
		t.Errorf("the expression matched %d times: the generator does not exercise it", hits)
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
