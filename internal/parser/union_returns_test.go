package parser

import (
	"strings"
	"testing"
)

// TestABranchAssigningAValueNoComponentCanBeMakesTheReturnDynamic: a return
// reached by a branch that assigned a string, a number or an argument passed
// through may hold that value, so the function returns "$any" rather than the
// components the other branches name. A call left untyped by the parse is not
// such a value, since the resolver may type it, and an always-running default
// overwritten in both branches does not reach the return.
func TestABranchAssigningAValueNoComponentCanBeMakesTheReturnDynamic(t *testing.T) {
	cases := []struct {
		name, want, content string
	}{
		{"script string branch", "$any", `component {
function f( k ) {
	var r = "";
	if ( k == 1 ) {
		r = new models.A();
	} else {
		r = "x";
	}
	return r;
}
}`},
		{"script argument branch", "$any", `component {
function f( k, o ) {
	var r = "";
	if ( k == 1 ) {
		r = new models.A();
	} else {
		r = arguments.o;
	}
	return r;
}
}`},
		{"script default overwritten in both branches", "models.A|models.B", `component {
function f( k ) {
	var r = "";
	if ( k == 1 ) {
		r = new models.A();
	} else {
		r = new models.B();
	}
	return r;
}
}`},
		{"script untyped call in a branch", "models.A", `component {
function f( k ) {
	var r = new models.A();
	if ( k == 1 ) {
		r = make();
	}
	return r;
}
}`},
		{"tag argument branch", "$any", `<cfcomponent>
<cffunction name="f">
	<cfargument name="k">
	<cfargument name="o">
	<cfset var r = "">
	<cfif k EQ 1>
		<cfset r = createObject("component","models.A")>
	<cfelse>
		<cfset r = ARGUMENTS.o>
	</cfif>
	<cfreturn r>
</cffunction>
</cfcomponent>`},
		{"tag string branch", "$any", `<cfcomponent>
<cffunction name="f">
	<cfargument name="k">
	<cfset var r = "">
	<cfif k EQ 1>
		<cfset r = createObject("component","models.A")>
	<cfelse>
		<cfset r = "x">
	</cfif>
	<cfreturn r>
</cffunction>
</cfcomponent>`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pr := Parse(testURI, tc.content)

			for i := range pr.Funcs {
				if f := &pr.Funcs[i]; f.Name == "f" && f.ReturnComponent != tc.want {
					t.Errorf("f returns %q, want %q", f.ReturnComponent, tc.want)
				}
			}
		})
	}
}

// TestEveryReturnStatementCountsTowardTheReturn: a function returning a
// different component from each return statement returns either. Script
// syntax kept the last return's component and tag syntax the first's.
func TestEveryReturnStatementCountsTowardTheReturn(t *testing.T) {
	for name, content := range map[string]string{
		"script": `component {
function f( k ) {
	if ( k == 1 ) {
		return new models.A();
	}
	return new models.B();
}
}`,
		"tag": `<cfcomponent>
<cffunction name="f">
	<cfargument name="k">
	<cfif k EQ 1>
		<cfreturn createObject("component","models.A")>
	</cfif>
	<cfreturn createObject("component","models.B")>
</cffunction>
</cfcomponent>`,
	} {
		t.Run(name, func(t *testing.T) {
			pr := Parse(testURI, content)

			for i := range pr.Funcs {
				if f := &pr.Funcs[i]; f.Name == "f" && f.ReturnComponent != "models.A|models.B" {
					t.Errorf("f returns %q, want %q", f.ReturnComponent, "models.A|models.B")
				}
			}
		})
	}
}

func TestUnionComponents(t *testing.T) {
	for _, tc := range []struct{ a, b, want string }{
		{"models.B", "models.A", "models.A|models.B"},
		{"models.A", "MODELS.A", "models.A"},
		{"models.A|models.C", "models.B", "models.A|models.B|models.C"},
		{"models.A", "$any", "$any"},
		{"models.A", "", ""},
		{"models.A", NativeTypePrefix + "struct", ""},
		{"models.A", "$builtin.now", ""},
		{"models.A", "objs.#t#", "$any"},
	} {
		if got := unionComponents(tc.a, tc.b); got != tc.want {
			t.Errorf("unionComponents(%q, %q) = %q, want %q", tc.a, tc.b, got, tc.want)
		}
	}
}

// TestAContainerReplacedAroundAMemberWrite: replacing the container before a
// member's first write leaves the member typed, and replacing it after the
// last write ends the member's type there, rather than either untyping the
// member for the whole function. tassweb's itext.cfc writes `var temp = 0`,
// builds its helpers into temp, copies them out, and later reuses temp.
func TestAContainerReplacedAroundAMemberWrite(t *testing.T) {
	content := `<cfcomponent>
<cffunction name="f">
	<cfset var temp = 0>
	<cfset temp = StructNew()>
	<cfset temp.u = createObject("component","models.U")>
	<cfset temp.u.before()>
	<cfset temp = 1>
	<cfset temp.u.after()>
</cffunction>
</cfcomponent>`
	pr := ParseWithOptions(testURI, content, &ParseOptions{ExtractCalls: true})
	s := pr.Scopes[0]

	at := func(line uint32) string {
		var got *ComponentRef

		refs, _ := pr.cachedFuncRefs(s.Start, s.End)
		for i := range refs {
			if r := &refs[i]; r.Variable == "temp.u" && r.Line <= line && (got == nil || r.Line >= got.Line) {
				got = r
			}
		}

		if got == nil {
			return "none"
		}

		return got.Component
	}

	if got := at(5); got != "models.U" {
		t.Errorf("temp.u before the replacement = %q, want models.U", got)
	}

	if got := at(7); got != "" {
		t.Errorf("temp.u after the replacement = %q, want it untyped", got)
	}
}

// TestACopiedVariableHoldsWhatItsSourceHolds: `<cfset VARIABLES._u = temp.u>`
// gives _u what temp.u holds at that line, a member included, so a function
// assigning from a call on _u is typed by what that call returns.
func TestACopiedVariableHoldsWhatItsSourceHolds(t *testing.T) {
	content := `<cfcomponent>
<cffunction name="init">
	<cfset var temp = StructNew()>
	<cfset temp.u = createObject("component","models.U")>
	<cfset VARIABLES._u = temp.u>
	<cfset var local2 = createObject("component","models.V")>
	<cfset VARIABLES._v = local2>
	<cfreturn this>
</cffunction>
<cffunction name="make">
	<cfset var result = VARIABLES._u.build()>
	<cfreturn result>
</cffunction>
</cfcomponent>`
	pr := ParseWithOptions(testURI, content, &ParseOptions{
		ExtractCalls: true,
		FuncLookup: func(comp, method string) string {
			if comp == "models.U" && strings.EqualFold(method, "build") {
				return "models.Built"
			}

			return ""
		},
	})

	want := map[string]string{"_u": "models.U", "_v": "models.V"}
	for _, r := range pr.ComponentRefs {
		if w, ok := want[r.Variable]; ok {
			if r.Component != w {
				t.Errorf("%s = %q, want %q", r.Variable, r.Component, w)
			}

			delete(want, r.Variable)
		}
	}

	for name := range want {
		t.Errorf("no ref for %s", name)
	}

	for i := range pr.Funcs {
		if f := &pr.Funcs[i]; f.Name == "make" && f.ReturnComponent != "models.Built" {
			t.Errorf("make returns %q, want models.Built", f.ReturnComponent)
		}
	}
}

// TestACopyIsNotAMemberSourcesCall: a member assigned a plain variable in a
// tag file is typed by that variable, not read as a call on it with no name.
// A copy recorded as a pending call broke collectionValueSource, which reads
// a pending call with a receiver as a method call.
func TestACopyIsNotAMemberSourcesCall(t *testing.T) {
	content := `<cfcomponent>
<cffunction name="f">
	<cfset var h = createObject("component","models.T")>
	<cfset var c = StructNew()>
	<cfset c.h = h>
	<cfset c.h.go()>
</cffunction>
</cfcomponent>`
	pr := ParseWithOptions(testURI, content, &ParseOptions{ExtractCalls: true})
	s := pr.Scopes[0]

	refs, _ := pr.cachedFuncRefs(s.Start, s.End)
	for _, r := range refs {
		if r.Variable == "c.h" {
			if r.Component != "models.T" {
				t.Errorf("c.h = %q, want models.T", r.Component)
			}

			return
		}
	}

	t.Error("no ref for c.h")
}
