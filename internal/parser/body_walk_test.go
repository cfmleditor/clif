package parser

import (
	"fmt"
	"slices"
	"sort"
	"testing"

	"go.lsp.dev/uri"
)

// bodyAnswers is what the lazy readers say of every function of pr, as text.
func bodyAnswers(pr *ParseResult) []string {
	var out []string

	for _, s := range pr.Scopes {
		vars := slices.Clone(pr.FuncVars(s.Start, s.End))
		sort.Strings(vars)

		refs, links := pr.FuncRefs(s.Start, s.End)

		var refText []string
		for _, r := range refs {
			refText = append(refText, fmt.Sprintf("%s@%d=%s", r.Variable, r.Line, r.Component))
		}

		for _, l := range links {
			refText = append(refText, fmt.Sprintf("link %s@%d", l.Path, l.Line))
		}

		for _, c := range pr.FuncCalls(s.Start, s.End) {
			refText = append(refText, fmt.Sprintf("call %s.%s@%d", c.Variable, c.FuncName, c.Line))
		}

		sort.Strings(refText)
		out = append(out, fmt.Sprintf("%s %d-%d vars %v %v", s.Name, s.Start, s.End, vars, refText))
	}

	return out
}

// A function body read on its own, after an edit, is read as the parse reads
// it: the same regions, the same output context, the same comments, the same
// doc comment, the same islands. Each case is a place where a walk of the
// body's text alone, which the lazy readers used to run, reads it otherwise.
func TestABodyReadAloneIsReadAsTheParseReadsIt(t *testing.T) {
	for _, c := range []struct{ name, uri, src string }{
		{"a doc comment types an argument", "file:///t.cfc", `component {
	/**
	 * @svc.doc_generic models.Service
	 */
	function run( required svc ) {
		var x = svc.make();
	}
}`},
		{"a function inside <cfoutput> reads its text as evaluated", "file:///t.cfm", `<cfoutput>
<cffunction name="report" output="false">
	<cfargument name="stats">
	<cfsavecontent variable="local.out">#chr(13)# #describe( arguments.stats )#</cfsavecontent>
	<cfreturn local.out>
</cffunction>
</cfoutput>`},
		{"a function after a comment that never closes is comment", "file:///t.cfc", `<cfcomponent>
<!------->
<cffunction name="lost">
	<cfscript>
		var y = structKeyExists( a, "b" );
	</cfscript>
</cffunction>
</cfcomponent>`},
		{"a <cfscript> island in a tag function", "file:///t.cfc", `<cfcomponent>
<cffunction name="load">
	<cfargument name="id">
	<cfset var a = new app.One()>
	<cfscript>
		var b = a.next();
		c = createObject( "component", "app.Two" );
	</cfscript>
	<cfset var d = b.last()>
</cffunction>
</cfcomponent>`},
		{"an import names a later new", "file:///t.cfc", `import models.User;
component {
	function make() {
		var u = new User();
		return u.save();
	}
}`},
		{"a string holds an expression that holds a string", "file:///t.cfc", `component {
	function f() {
		var s = "#g("local.a=1")#";
		return s;
	}
}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			parsed := bodyAnswers(ParseWithOptions(uri.URI(c.uri), c.src, &ParseOptions{ExtractCalls: true}))

			pr := ParseWithOptions(uri.URI(c.uri), c.src, &ParseOptions{ExtractCalls: true})
			for _, s := range pr.Scopes {
				pr.InvalidateFunc(s.Start, s.End)
			}

			if alone := bodyAnswers(pr); !slices.Equal(alone, parsed) {
				t.Errorf("read alone:\n%v\nthe parse:\n%v", alone, parsed)
			}

			// Without call extraction FuncCalls walks the body for them.
			uncalled := ParseWithOptions(uri.URI(c.uri), c.src, nil)
			for i, s := range uncalled.Scopes {
				var want []string

				for _, call := range pr.FuncCalls(s.Start, s.End) {
					want = append(want, fmt.Sprintf("%s.%s@%d", call.Variable, call.FuncName, call.Line))
				}

				var got []string

				for _, call := range uncalled.FuncCalls(s.Start, s.End) {
					got = append(got, fmt.Sprintf("%s.%s@%d", call.Variable, call.FuncName, call.Line))
				}

				sort.Strings(want)
				sort.Strings(got)

				if !slices.Equal(got, want) {
					t.Errorf("function %d: calls without extraction %v, with %v", i, got, want)
				}
			}
		})
	}
}

// A function's variables are what the parse's walk declared on its lines:
// a tag function's arguments, which the script parser the lazy read ran over
// a tag body never saw, and not a name written inside a string's
// interpolation, which that read, scanning strings quote to quote, took for
// a declaration.
func TestFuncVarsAreTheWalksDeclarations(t *testing.T) {
	for _, c := range []struct {
		name, src string
		want      []string
	}{
		{"tag arguments", `<cfcomponent>
<cffunction name="save">
	<cfargument name="name">
	<cfargument name="age">
	<cfset var saved = true>
</cffunction>
</cfcomponent>`, []string{"age", "name", "saved"}},
		{"a string's interpolation", `component {
	function f() {
		var s = "#g("local.a=1")#";
	}
}`, []string{"s"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			pr := ParseWithOptions("file:///t.cfc", c.src, nil)
			s := pr.Scopes[0]

			for _, when := range []string{"parsed", "after an edit"} {
				if when == "after an edit" {
					pr.InvalidateFunc(s.Start, s.End)
				}

				got := slices.Clone(pr.FuncVars(s.Start, s.End))
				sort.Strings(got)

				if !slices.Equal(got, c.want) {
					t.Errorf("%s: FuncVars = %v, want %v", when, got, c.want)
				}
			}
		})
	}
}

// An edit keeps what the parse filed for the functions it did not touch, and
// a reparse keeps what its own walk filed: a reparse cleared the calls it had
// just recorded, and an edit that added a line left every function after it
// under a key no question asked for.
func TestEditsKeepTheParsesCalls(t *testing.T) {
	src := `component {
	function a() {
		x.one();
	}
	function b() {
		y.two();
	}
}`
	calls := func(pr *ParseResult, name string) []string {
		s := scopeNamed(pr, name)

		var out []string
		for _, c := range pr.FuncCalls(s.Start, s.End) {
			out = append(out, c.Variable+"."+c.FuncName)
		}

		return out
	}

	pr := ParseWithOptions("file:///t.cfc", src, &ParseOptions{ExtractCalls: true})

	// A line added inside a(): b() moves down a line.
	pr.ApplyEdit(2, 11, 2, 11, "\n\t\tz.three();")

	if got := calls(pr, "a"); !slices.Equal(got, []string{"x.one", "z.three"}) {
		t.Errorf("a() after an edit inside it: %v", got)
	}

	if got := calls(pr, "b"); !slices.Equal(got, []string{"y.two"}) {
		t.Errorf("b() after an edit above it: %v", got)
	}

	// An edit outside every function reparses the file.
	pr.ApplyEdit(0, 11, 0, 11, " ")

	if got := calls(pr, "b"); !slices.Equal(got, []string{"y.two"}) {
		t.Errorf("b() after a reparse: %v", got)
	}
}

// What a function body assigns to the variables scope is read again after an
// edit changes the body: it was read once, on the first question, and kept
// for the life of the parse.
func TestHasScopedAssignmentSeesAnEdit(t *testing.T) {
	src := `component {
	function a() {
		x = 1;
	}
}`
	pr := ParseWithOptions("file:///t.cfc", src, nil)

	if pr.HasScopedAssignment(ScopeVariables, "helper") {
		t.Fatal("helper is assigned before the edit")
	}

	pr.ApplyEdit(2, 8, 2, 8, " variables.helper = 2;")

	if !pr.HasScopedAssignment(ScopeVariables, "helper") {
		t.Error("an edit assigned variables.helper, and HasScopedAssignment does not see it")
	}
}
