package resolve

import "testing"

// TestAConditionalAssignmentHoldsItsBranches: `x = c ? a : b` and `x = a ?: b`
// left x untyped whatever the branches held (#308). Each branch is typed as an
// assignment of its own and x is their union; a branch holding an untyped
// argument types nothing.
func TestAConditionalAssignmentHoldsItsBranches(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"shared/Svc.cfc":  `component { function search(){} function onlySvc(){} }`,
		"shared/Fake.cfc": `component { function search(){} }`,
		"Client.cfc": `component {
function typedArg( shared.Svc injected ) {
	variables.c1 = isObject( arguments.injected ) ? arguments.injected : new shared.Svc();
	variables.c1.search();
}
function untypedArg( any injected ) {
	var c2 = arguments.injected ?: new shared.Svc();
	c2.search();
}
function twoComponents( flag ) {
	var c3 = flag ? new shared.Svc() : ( new shared.Fake() );
	c3.search();
	c3.neither();
}
function nested( a, b ) {
	var c4 = a ? new shared.Svc() : b ? createObject( "component", "shared.Svc" ) : new shared.Svc();
	c4.search();
}
}`,
		"Tags.cfc": `<cfcomponent>
<cffunction name="f">
	<cfset var c5 = true ? new shared.Svc() : new shared.Svc()>
	<cfset c5.search()>
</cffunction>
</cfcomponent>`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Client.cfc"), map[string]string{
		"variables.c1.search": "",
		"c2.search":           "variable 'c2' has no component ref",
		"c3.search":           "",
		"c3.neither":          "method 'neither' not found in shared.Fake|shared.Svc",
		"c4.search":           "",
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Tags.cfc"), map[string]string{
		"c5.search": "",
	})
}

func TestTernaryBranches(t *testing.T) {
	for rhs, want := range map[string][]string{
		"c ? a : b":              {"a", "b"},
		"a ?: b":                 {"a", "b"},
		"f( x ? 1 : 2 ) ? a : b": {"a", "b"},
		"c ? ( d ? a : b ) : e":  {"a", "b", "e"},
		"c ? { k: 1 } : b":       {"{ k: 1 }", "b"},
		"svc.get( c ? 1 : 2 )":   nil,
		"new X()":                nil,
		"c ? a":                  nil,
	} {
		got, ok := ternaryBranches(rhs)
		if ok != (want != nil) || len(got) != len(want) {
			t.Errorf("%q: got %q (%v), want %q", rhs, got, ok, want)

			continue
		}

		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%q: got %q, want %q", rhs, got, want)
			}
		}
	}
}
