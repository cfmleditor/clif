package resolve

import (
	"testing"
)

// The gateway family spells the whole custom-field recipe in tags:
// Gateway.cfc is cfcomponent with <cfset fields=array(...)> on each
// member, <cfreturn createObject("component","Field").init(...)> in the
// helpers, and <cfreturn variables.fields> in the getter. The element
// union is the source's again — no script-only reading.
func TestALoopOverATagFormFamilyTypesTheLoop(t *testing.T) {
	dir := t.TempDir()

	adminJava := "core/src/main/java/resource/context/admin"
	gateway := adminJava + "/gdriver"
	cfmlAdmin := "core/src/main/cfml/context/admin"

	writeFiles(t, dir, map[string]string{
		// The recognized-layout markers, as assumedLuceeAdmin checks them.
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		cfmlAdmin + "/services.update.functions.cfm":                                  `<cfadmin action="x">`,
		gateway + "/Field.cfc": `component {
	function getName() { return ""; }
}`,
		gateway + "/Group.cfc": `component {
	function getLevel() { return 2; }
}`,
		gateway + "/Gateway.cfc": `<cfcomponent>
<cfset fields=array()>
<cffunction name="field" returntype="component" access="private">
	<cfreturn createObject("component","Field").init(arguments.name)>
</cffunction>
<cffunction name="group" returntype="component" access="private">
	<cfreturn createObject("component","Group").init(arguments.name)>
</cffunction>
<cffunction name="getCustomFields" returntype="array">
	<cfreturn variables.fields>
</cffunction>
</cfcomponent>`,
		gateway + "/ActiveMQ.cfc": `<cfcomponent extends="Gateway">
<cfset fields=array(
	field("Provider URL","providerURL",".",true,"."),
	group("Context",".")
)>
<cffunction name="getId" returntype="string" access="public"><cfreturn "activemq"></cffunction>
</cfcomponent>`,
		cfmlAdmin + "/services.gateway.cfm": `<cfset driverNames=ComponentListPackageAsStruct("lucee.admin.gdriver",driverNames)>
<cfset drivers={}>
<cfloop collection="#driverNames#" index="n" item="fn">
	<cfif n EQ "Gateway" or n EQ "Field" or n EQ "Group">
		<cfcontinue>
	</cfif>
	<cfset drivers[n]=createObject('component',fn)>
</cfloop>
<cfset driver=drivers[n]>
<cfloop array="#driver.getCustomFields()#" index="f">
	<cfoutput>#f.getName()#</cfoutput>
	<cfoutput>#f.getLevel()#</cfoutput>
</cfloop>`,
	})

	got := reasonsInCFM(t, dir, cfmlAdmin+"/services.gateway.cfm")
	expectReasons(t, got, map[string]string{
		"driver.getCustomFields": "",
		"f.getName":              "",
		"f.getLevel":             "",
	})
}
