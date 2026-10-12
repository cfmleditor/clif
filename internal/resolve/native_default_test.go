package resolve

import (
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// A native default (`StructNew()`, `{}`) that the variable is later replaced
// from is set aside when the method called is not one the native type has:
// tassweb sets `VARIABLES._parent = StructNew()` in every component's
// pseudo-constructor and init() replaces it with an untyped argument, so the
// default was the nearest assignment and 7,000 calls on what init() stored
// were checked against a struct. The componentResolver for `_parent`, the
// next step, answers instead, and a later assignment in the same function is
// still found.
func TestANativeDefaultDoesNotOutrankWhatReplacesIt(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Kernel.cfc": `component { function getService(name){} }`,
		"User.cfc":   `component { function login(){} }`,
		"Persist.cfc": `<cfcomponent>
<cfset VARIABLES._parent = StructNew() />
<cffunction name="init">
	<cfargument name="parent" type="struct" required="true" />
	<cfset VARIABLES._parent = ARGUMENTS.parent />
	<cfreturn this />
</cffunction>
<cffunction name="load">
	<cfset var svc = _parent.getService("x") />
</cffunction>
<cffunction name="check">
	<cfargument name="coreuser" required="false" />
	<cfset var result = StructNew() />
	<cfif structKeyExists(ARGUMENTS, "coreuser")>
		<cfset result = ARGUMENTS.coreuser />
		<cfset result.login() />
	<cfelse>
		<cfset result = createObject("component", "User") />
	</cfif>
</cffunction>
</cfcomponent>`,
	})

	r := &Resolver{Resolvers: []parser.Resolver{{Match: `^_parent(?:\(\))?$`, Resolve: "Kernel", Prefix: "_parent", Anchored: true}}}
	// As the unresolved scan parses it: without the builtin return lookup,
	// StructNew() types nothing and there is no default to set aside.
	got := reasonsWithOptions(t, r, dir, "Persist.cfc", func(file string, o *parser.ParseOptions) {
		o.BuiltinReturnLookup = r.BuiltinReturnLookupFor(file)
	})

	expectReasons(t, got, map[string]string{
		"_parent.getService": "",
		"result.login":       "",
	})

	for call, reason := range got {
		if strings.Contains(reason, "native.struct") {
			t.Errorf("%s was checked against the StructNew() default: %s", call, reason)
		}
	}
}
