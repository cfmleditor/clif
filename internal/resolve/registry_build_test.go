package resolve

import (
	"testing"
)

// A service getter that builds the array it returns — the body starts a
// local empty, appends values to it, and returns it (or a variables field
// assigned from it) — proves the appended values' common component as the
// collection's element:
//
//	<cffunction name="getBeans" returntype="array">
//		<cfset var built=arrayNew(1)>
//		<cfloop from="1" to="#rs.recordcount#" index="s">
//			<cfset bean=getManagerBean()>
//			<cfset arrayAppend(built, bean)>
//		</cfloop>
//		<cfreturn built>
//	</cffunction>
//
// The factory's own return — createObject("component", literal).init(...) —
// is the element, and the consumer's keyed read types against it: a method
// the element declares resolves, one it does not stays a finding.
func TestTheCollectionAFunctionBuildsProvesItsElement(t *testing.T) {
	dir := t.TempDir()

	writeFiles(t, dir, map[string]string{
		// The service: a factory method and two array builders. The cached
		// builder returns a variables field assigned from the built array —
		// the alias the real extendManager's getter spells. The mixed builder
		// appends a value nobody proves: an open world, so no answer.
		"pkg/service.cfc": `<cfcomponent>
	<cffunction name="getManagerBean" access="public" returntype="any">
		<cfset var managerBean=createObject("component","pkg.widget").init(variables.configBean)>
		<cfset managerBean.setSiteID(getSiteID())>
		<cfreturn managerBean>
	</cffunction>

	<cffunction name="getBeans" access="public" returntype="array">
		<cfset var built=arrayNew(1)>
		<cfset var bean="">
		<cfloop from="1" to="3" index="i">
			<cfset bean=getManagerBean()>
			<cfset arrayAppend(built, bean)>
		</cfloop>
		<cfreturn built>
	</cffunction>

	<cffunction name="getCachedBeans" access="public" returntype="array">
		<cfset var built=arrayNew(1)>
		<cfset arrayAppend(built, getManagerBean())>
		<cfset variables.cached=built>
		<cfreturn variables.cached>
	</cffunction>

	<cffunction name="getMixedBeans" access="public" returntype="array">
		<cfset var built=arrayNew(1)>
		<cfset arrayAppend(built, getManagerBean())>
		<cfset arrayAppend(built, somethingUnbuilt())>
		<cfreturn built>
	</cffunction>

	<cffunction name="getSiteID" access="public" returntype="string">
		<cfreturn "">
	</cffunction>
</cfcomponent>`,
		"pkg/widget.cfc": `<cfcomponent>
	<cffunction name="init" access="public" returntype="any">
		<cfreturn this>
	</cffunction>

	<cffunction name="getSiteID" access="public" returntype="string">
		<cfreturn "">
	</cffunction>

	<cffunction name="getLabel" access="public" returntype="string">
		<cfreturn "">
	</cffunction>
</cfcomponent>`,
		"pkg/page.cfm": `<cfset service = new pkg.service()>
<cfset sets = service.getBeans()>
<cfset cached = service.getCachedBeans()>
<cfset mixed = service.getMixedBeans()>
<cfset b = sets[1]>
<cfset x = b.getSiteID()>
<cfset y = cached[1].getLabel()>
<cfset z = b.getUndeclared()>`,
	})

	got := reasonsIn(t, dir, "pkg/page.cfm")
	expectReasons(t, got, map[string]string{
		"service.getBeans":       "",
		"service.getCachedBeans": "",
		// The mixed builder's own call is fine — the method is declared; it
		// is its element proof that the unprovable append withholds.
		"service.getMixedBeans": "",
		"b.getSiteID":           "",
		"b.getUndeclared":       "method 'getUndeclared' not found in pkg.widget",
		"cached[1].getLabel":    "",
	})
}
