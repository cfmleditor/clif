package parser

import (
	"slices"
	"testing"
)

// TestInstantiationsAreRecordedWhereverWritten: the code map's instantiates
// edge came from component-level refs alone, so a local `var s = new Svc()`,
// an inline `new Svc().handle()` and one in an argument list drew none (#308).
// Every construction is recorded, in either syntax, with the line it is on,
// and two on one line are two.
func TestInstantiationsAreRecordedWhereverWritten(t *testing.T) {
	cases := map[string]struct {
		src   string
		lines []uint32
	}{
		"script": {
			src: "component {\nfunction f() {\n" +
				"createObject(\"component\", \"shared.Svc\").handle();\n" + // 2
				"g(new shared.Svc());\n" + // 3
				"g(createObject(\"component\", \"shared.Svc\"));\n" + // 4
				"return createObject(\"component\", \"shared.Svc\");\n" + // 5
				"}\n" +
				"function h() { new shared.Svc(); var x = [new shared.Svc()]; var y = { a: new shared.Svc() }; z = 1 + new shared.Svc().n(); }\n" + // 7, four times
				"function i() { var s = new shared.Svc(); variables.t = new shared.Svc(); writeOutput(\"new shared.Svc()\"); }\n" + // 8, twice; not the string
				"function j() { s = \"#new shared.Svc()#\" & \"#new shared.Svc()#\"; }\n" + // 9, twice: each span counts from zero
				"}\n",
			lines: []uint32{2, 3, 4, 5, 7, 7, 7, 7, 8, 8, 9, 9},
		},
		"tag": {
			src: "<cffunction name=\"tf\">\n" +
				"<cfset var s = new shared.Svc()>\n" + // 1
				"<cfset createObject(\"component\", \"shared.Svc\").handle()>\n" + // 2
				"<cfreturn new shared.Svc()>\n" + // 3
				"</cffunction>\n" +
				"<cfset application.x = new shared.Svc()>\n" + // 5
				"<cfset a = new shared.Svc()><cfset b = new shared.Svc()>\n" + // 6, twice
				"<cfoutput>#new shared.Svc().a()# #new shared.Svc().b()#</cfoutput>\n" + // 7, twice
				"<cfset a.x = new shared.Svc()><cfset b.y = new shared.Svc()>\n", // 8, twice: member sets are wrapped
			lines: []uint32{1, 2, 3, 5, 6, 6, 7, 7, 8, 8},
		},
	}

	for name, c := range cases {
		pr := ParseWithOptions("file:///x/A.cfc", c.src, &ParseOptions{ExtractCalls: true})

		var lines []uint32

		for _, in := range pr.Instantiations {
			if in.Component != "shared.Svc" {
				t.Errorf("%s: instantiation of %q on line %d", name, in.Component, in.Line)
			}

			lines = append(lines, in.Line)
		}

		slices.Sort(lines)

		if !slices.Equal(lines, c.lines) {
			t.Errorf("%s: instantiations on lines %v, want %v", name, lines, c.lines)
		}
	}
}

// TestAParseWithoutCallsRecordsNoInstantiations: the record is the code map's,
// and is gated as a call is.
func TestAParseWithoutCallsRecordsNoInstantiations(t *testing.T) {
	pr := Parse("file:///x/A.cfc", "component { function f() { var s = new shared.Svc(); } }")
	if len(pr.Instantiations) != 0 {
		t.Errorf("recorded %+v", pr.Instantiations)
	}
}

// TestAConditionalReturnReturnsItsBranches: `return flag ? new A() : new B()`
// was read as returning flag, so the function declared no component.
func TestAConditionalReturnReturnsItsBranches(t *testing.T) {
	src := `component {
function both( flag ) { return flag ? new models.A() : createObject( "component", "models.B" ); }
function other( flag ) { return flag ? new models.A() : getThing(); }
}`
	pr := ParseWithOptions("file:///x/F.cfc", src, &ParseOptions{ExtractCalls: true})

	want := map[string]string{"both": "models.A|models.B", "other": ""}
	for _, f := range pr.Funcs {
		if got := f.ReturnComponent; got != want[f.Name] {
			t.Errorf("%s returns %q, want %q", f.Name, got, want[f.Name])
		}
	}

	if len(pr.Instantiations) != 3 {
		t.Errorf("instantiations %+v, want three", pr.Instantiations)
	}
}

// TestAConditionalReturnTypesItsOwnFunction: a named function nested in the
// body is filed after the one being parsed, and took its return type.
func TestAConditionalReturnTypesItsOwnFunction(t *testing.T) {
	src := `component {
function outer( flag ) { function helper() {} return flag ? new models.A() : new models.B(); }
}`
	pr := ParseWithOptions("file:///x/F.cfc", src, &ParseOptions{ExtractCalls: true})

	want := map[string]string{"outer": "models.A|models.B", "helper": ""}
	checkReturns(t, pr, want)
}

// TestEveryReturnArmTypesItsOwnFunction: each way a script return is typed,
// and a call's caller, goes to the function whose body holds it, not to a
// function nested in that body before it.
func TestEveryReturnArmTypesItsOwnFunction(t *testing.T) {
	src := `component {
function viaVar() { function h1() {} var x = new models.A(); return x; }
function viaNew() { function h2() {} return new models.A(); }
function viaCall() { function h3() {} svc.save(); return createObject( "component", "models.A" ); }
}`
	pr := ParseWithOptions("file:///x/F.cfc", src, &ParseOptions{ExtractCalls: true})

	checkReturns(t, pr, map[string]string{
		"viaVar": "models.A", "viaNew": "models.A", "viaCall": "models.A", "h1": "", "h2": "", "h3": "",
	})

	for _, c := range pr.AllCalls() {
		if c.FuncName == "save" && c.Caller != "viaCall" {
			t.Errorf("save() has caller %q, want viaCall", c.Caller)
		}
	}
}

// TestATagConditionalReturnReturnsItsBranches: `<cfreturn>` reads a returned
// conditional as the script return does.
func TestATagConditionalReturnReturnsItsBranches(t *testing.T) {
	src := `<cfcomponent>
<cffunction name="both"><cfargument name="flag"><cfreturn arguments.flag ? new models.A() : createObject( "component", "models.B" )></cffunction>
<cffunction name="other"><cfargument name="flag"><cfreturn arguments.flag ? new models.A() : getThing()></cffunction>
</cfcomponent>`
	pr := ParseWithOptions("file:///x/F.cfc", src, &ParseOptions{ExtractCalls: true})

	checkReturns(t, pr, map[string]string{"both": "models.A|models.B", "other": ""})
}

// TestAParenthesisedConditionalReturnIsRead: `return ( c ? a : b )` is the
// conditional the parentheses hold, in either syntax; a tag return of a
// prefixed construction names the path, not the prefix.
func TestAParenthesisedConditionalReturnIsRead(t *testing.T) {
	script := `component {
function s( flag ) { return ( flag ? new models.A() : new models.B() ); }
}`
	checkReturns(t, ParseWithOptions("file:///x/S.cfc", script, &ParseOptions{ExtractCalls: true}),
		map[string]string{"s": "models.A|models.B"})

	tag := `<cfcomponent>
<cffunction name="t"><cfargument name="flag"><cfreturn ( arguments.flag ? new models.A() : new models.B() )></cffunction>
<cffunction name="p"><cfreturn new cfml:models.A()></cffunction>
</cfcomponent>`
	checkReturns(t, ParseWithOptions("file:///x/T.cfc", tag, &ParseOptions{ExtractCalls: true}),
		map[string]string{"t": "models.A|models.B", "p": "models.A"})
}

// TestConstructionInReadsWhatNewNames: the branch reader agrees with
// readNewComponentPath on each form it reads, and declines what it cannot.
func TestConstructionInReadsWhatNewNames(t *testing.T) {
	resolve := func(expr string) string { return "resolved:" + expr }
	imports := func(name string) string {
		if name == "User" {
			return "models.User"
		}

		return name
	}

	for text, want := range map[string]string{
		`new models.A()`:                          "models.A",
		`new "models.A"( 1 )`:                     "models.A",
		`new cfml:models.A()`:                     "models.A",
		`new java:java.io.File( "x" )`:            `resolved:createObject("java","java.io.File")`,
		`new User()`:                              "models.User",
		`new component { function f() {} }`:       "",
		`new models.A().init()`:                   "",
		`createObject( "component", "models.B" )`: "models.B",
		`createObject( "java", "java.io.File" )`:  `resolved:createObject("java","java.io.File")`,
		`createObject( "component", name )`:       "",
		`getThing()`:                              "",
	} {
		if got := constructionIn(text, resolve, imports); got != want {
			t.Errorf("%s: %q, want %q", text, got, want)
		}
	}
}

func checkReturns(t *testing.T, pr *ParseResult, want map[string]string) {
	t.Helper()

	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		if got := f.ReturnComponent; got != want[f.Name] {
			t.Errorf("%s returns %q, want %q", f.Name, got, want[f.Name])
		}
	}
}

// TestAConditionalReturnRecordsNoCallOfItsOwn: reading a branch must not
// record the calls a computed createObject path makes; the statement's own
// walk records each once, on its line.
func TestAConditionalReturnRecordsNoCallOfItsOwn(t *testing.T) {
	src := `component {
function f( flag ) {
	return flag ? createObject( "component", "pkg." & name() ) : new models.B();
}
}`
	pr := ParseWithOptions("file:///x/F.cfc", src, &ParseOptions{ExtractCalls: true})

	var lines []uint32

	for _, c := range pr.AllCalls() {
		if c.FuncName == "name" {
			lines = append(lines, c.Line)
		}
	}

	if !slices.Equal(lines, []uint32{2}) {
		t.Errorf("name() recorded on lines %v, want [2]", lines)
	}
}
