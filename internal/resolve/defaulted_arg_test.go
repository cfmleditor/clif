package resolve

import (
	"runtime"
	"strings"
	"testing"
)

// A defaulted argument: the body guards its absence and constructs what it
// holds then, so the argument is that component when the caller passes none:
//
//	<cfargument name="keyFactory" type="any">
//	<cfif not structKeyExists(arguments, "keyFactory")>
//		<cfset arguments.keyFactory=createObject("component","mura.publisherKeys").init(...)>
//	</cfif>
//	<cfset arguments.keyFactory.setMode("publish")>
//
// The calls on the argument read against the constructed component's own
// contracts; a method it does not declare stays a finding.
func TestADefaultedArgumentTypesToItsConstruction(t *testing.T) {
	dir := t.TempDir()

	writeFiles(t, dir, map[string]string{
		"mura/publisherKeys.cfc": `<cfcomponent>
	<cffunction name="init" access="public" returntype="any">
		<cfreturn this>
	</cffunction>

	<cffunction name="setMode" access="public" returntype="any">
		<cfreturn this>
	</cffunction>

	<cffunction name="get" access="public" returntype="string">
		<cfreturn "">
	</cffunction>
</cfcomponent>`,
		"mura/publisher.cfc": `<cfcomponent>
	<cffunction name="publish" access="public" returntype="any">
		<cfargument name="keyMode" type="string" default="copy" required="true">
		<cfargument name="keyFactory" type="any">

		<cfif not structKeyExists(arguments, "keyFactory")>
			<cfset arguments.keyFactory=createObject("component","mura.publisherKeys").init(arguments.keyMode)>
		</cfif>

		<cfset arguments.keyFactory.setMode("publish")>

		<cfset var keys=arguments.keyFactory>
		<cfset var value=keys.get("contentID")>
		<cfset arguments.keyFactory.rekey()>

		<cfreturn value>
	</cffunction>
</cfcomponent>`,
		"mura/page.cfm": `<cfset publisher = new mura.publisher()>
<cfset publisher.publish()>
<cfset keys = new mura.publisherKeys()>
<cfset ok = keys.getUndeclared()>`,
	})

	got := reasonsIn(t, dir, "mura/publisher.cfc")
	expectReasons(t, got, map[string]string{
		// The defaulted argument's calls read against publisherKeys: one
		// declared, one not.
		"arguments.keyFactory.setMode": "",
		"arguments.keyFactory.rekey":   "method 'rekey' not found in mura.publisherKeys",
		// The plain struct .get() member stays dynamic.
		"keys.get": "",
	})
}

// A guarded function's arguments are found once, not per call on them: the
// body was tokenised again for every call site on a guarded argument the
// guard leaves untyped (here re-set from a call), so a function making n
// such calls cost n times its own length. Allocation at eight times
// the calls must stay near eight times, not sixty-four.
func TestADefaultedArgumentIsFoundOncePerFunction(t *testing.T) {
	allocs := func(n int) uint64 {
		dir := t.TempDir()

		var body strings.Builder
		for range n {
			body.WriteString("\t\t<cfset arguments.opts.go(\"x\")>\n")
		}

		writeFiles(t, dir, map[string]string{
			"mura/publisherKeys.cfc": `<cfcomponent><cffunction name="setMode"><cfreturn this></cffunction></cfcomponent>`,
			"mura/publisher.cfc": `<cfcomponent>
	<cffunction name="publish">
		<cfargument name="opts" type="any">
		<cfif not structKeyExists(arguments, "opts")>
			<cfset arguments.opts=getDefaults()>
		</cfif>
` + body.String() + `	</cffunction>
</cfcomponent>`,
		})

		var before, after runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&before)
		reasonsIn(t, dir, "mura/publisher.cfc")
		runtime.ReadMemStats(&after)

		return after.TotalAlloc - before.TotalAlloc
	}

	small, large := allocs(100), allocs(800)
	if ratio := float64(large) / float64(small); ratio > 20 {
		t.Errorf("allocation grew %.1fx for 8x the calls on a defaulted argument (%d to %d bytes)", ratio, small, large)
	}
}

// A loop over an argument does not give its variable the argument's own
// type: the loop variable holds an element. With the argument defaulted to
// a UserList, `u` was checked against UserList itself, so `u.count()` was
// accepted and `u.nope()` reported as missing from it.
func TestALoopOverAnArgumentIsNotTheArgumentItself(t *testing.T) {
	dir := t.TempDir()

	writeFiles(t, dir, map[string]string{
		"UserList.cfc": `component { function count() {} }`,
		"Svc.cfc": `component {
	function run( items ) {
		if ( !structKeyExists( arguments, "items" ) ) {
			arguments.items = new UserList();
		}
		for ( var u in arguments.items ) {
			u.nope();
			u.count();
		}
	}
}`,
	})

	got := reasonsIn(t, dir, "Svc.cfc")
	for _, call := range []string{"u.nope", "u.count"} {
		if r := got[call]; strings.Contains(r, "UserList") {
			t.Errorf("%s: %q; the loop variable is an element, not the UserList", call, r)
		}
	}
}
