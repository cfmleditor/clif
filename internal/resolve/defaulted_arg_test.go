package resolve

import (
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
