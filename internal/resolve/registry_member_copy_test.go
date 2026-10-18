package resolve

import (
	"strings"
	"testing"
)

// A family member's own getter that copies its own fields proves the same
// element set the copy's source holds:
//
//	<cfset fields=array(field(...),group(...))>
//	<cffunction name="getCustomFields" returntype="array">
//		<cfif !isNull(form._name)>
//			<cfset var fields=duplicate(variables.fields)>
//			<cfloop array="#fields#" index="local.i" item="local.field">
//				<cfset field.setDefaultValue(...)>
//			</cfloop>
//		</cfif>
//		<cfreturn fields>
//	</cffunction>
//
// The copy is a duplicate of the member's own fields — a deep copy keeps a
// collection's elements — so the getter's loop item reads as the member's
// own literal's components, and the family's element proof holds over the
// copy as it does over a plain literal: the consumer's loop over the getter
// keeps the union of the families' helpers.
func TestAMemberGettersOwnCopyOfItsFieldsProvesTheElements(t *testing.T) {
	dir := t.TempDir()

	appender := "core/src/main/java/resource/context/admin/logging/appender"
	cfmlAdmin := "core/src/main/cfml/context/admin"

	writeFiles(t, dir, map[string]string{
		// The recognized-layout markers, as assumedLuceeAdmin checks them.
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		cfmlAdmin + "/services.update.functions.cfm":                                  `<cfadmin action="x">`,

		appender + "/Appender.cfc": `component {
	fields=array();
	private component function field() { return createObject("component","Field").init(); }
	private component function group() { return createObject("component","Group").init(); }
	public array function getCustomFields() { return fields; }
}`,
		appender + "/Field.cfc": `component {
	function getName() { return ""; }
	function getDefaultValue() { return ""; }
	function setDefaultValue(v) { return ""; }
}`,
		appender + "/Group.cfc": `component {
	function getLevel() { return 2; }
}`,

		// One member's getter is the base's plain return; the other's own
		// overridden getter copies the fields it walks, the way the real
		// DatasourceAppender spells it.
		appender + "/Console.cfc": `component extends="Appender" {
	fields=array(field("Mode","mode"),group("Times",".",3));
	public string function getId() { return "console"; }
}`,
		appender + "/Datasource.cfc": `<cfcomponent extends="Appender">
	<cfset fields=array(field("DSN","dsn","",true,"."),field("Table","table","LOGS",true,"."))>
	<cffunction name="getId" returntype="string" access="public"><cfreturn "datasource"></cffunction>

	<cffunction name="getCustomFields" returntype="array" output="false">
		<cfif !isNull(form._name)>
			<cfset var fields=duplicate(variables.fields)>
			<cfloop array="#fields#" index="local.i" item="local.field">
				<cfif field.getName() EQ "Path">
					<cfset local.dv=field.getDefaultValue()>
					<cfset field.setDefaultValue(dv&"x")>
				</cfif>
			</cfloop>
		</cfif>
		<cfreturn fields>
	</cffunction>
</cfcomponent>`,

		// The recipe and the consumer in one template, with the guard's skip
		// of the base and the two helpers.
		cfmlAdmin + "/debugging.logging.cfm": `<cfset driverNames=ComponentListPackageAsStruct("lucee.admin.logging.appender",driverNames)>
<cfset drivers={}>
<cfloop collection="#driverNames#" index="n" item="fn">
	<cfif n EQ "Appender" or n EQ "Field" or n EQ "Group">
		<cfcontinue>
	</cfif>
	<cfset tmp=createObject('component',fn)>
	<cfset drivers[trim(tmp.getId())]=tmp>
</cfloop>
<cfset driver=drivers[trim(form.type)]>
<cfloop array="#driver.getCustomFields()#" index="field">
	<cfoutput>#field.getName()#</cfoutput>
	<cfoutput>#field.getLevel()#</cfoutput>
	<cfoutput>#field.getUndeclared()#</cfoutput>
</cfloop>`,
	})

	got := reasonsInCFM(t, dir, cfmlAdmin+"/debugging.logging.cfm")
	expectReasons(t, got, map[string]string{
		"driver.getCustomFields": "",
		"field.getName":          "",
		"field.getLevel":         "",
	})

	if reason := got["field.getUndeclared"]; !strings.Contains(reason, "not found in ") {
		t.Errorf("field.getUndeclared: %q, want the union's missing-method reason", reason)
	}
}

// The member's own getter's loop is typed the same way: its item reads as
// the member's own fields' components, so the copy's own page holds.
func TestTheMemberGettersOwnLoopOverItsCopyTypesItsItem(t *testing.T) {
	dir := t.TempDir()

	appender := "core/src/main/java/resource/context/admin/logging/appender"

	writeFiles(t, dir, map[string]string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		"core/src/main/cfml/context/admin/services.update.functions.cfm":              `<cfadmin action="x">`,

		appender + "/Appender.cfc": `component {
	fields=array();
	private component function field() { return createObject("component","Field").init(); }
	private component function group() { return createObject("component","Group").init(); }
	public array function getCustomFields() { return fields; }
}`,
		appender + "/Field.cfc": `component {
	function getName() { return ""; }
	function getDefaultValue() { return ""; }
	function setDefaultValue(v) { return ""; }
}`,
		appender + "/Group.cfc": `component {
	function getLevel() { return 2; }
}`,
		appender + "/Datasource.cfc": `<cfcomponent extends="Appender">
	<cfset fields=array(field("DSN","dsn","",true,"."),field("Table","table","LOGS",true,"."))>
	<cffunction name="getId" returntype="string" access="public"><cfreturn "datasource"></cffunction>

	<cffunction name="getCustomFields" returntype="array" output="false">
		<cfif !isNull(form._name)>
			<cfset var fields=duplicate(variables.fields)>
			<cfloop array="#fields#" index="local.i" item="local.field">
				<cfif field.getName() EQ "Path">
					<cfset local.dv=field.getDefaultValue()>
					<cfset field.setDefaultValue(dv&"x")>
				</cfif>
			</cfloop>
		</cfif>
		<cfreturn fields>
	</cffunction>
</cfcomponent>`,
	})

	got := reasonsIn(t, dir, appender+"/Datasource.cfc")
	expectReasons(t, got, map[string]string{
		"field.getName":         "",
		"field.getDefaultValue": "",
		"field.setDefaultValue": "",
	})
}

// A copy whose source the member's own file does not prove — a getter that
// copies an unassigned variable, or assigns the copy something else — leaves
// the element untyped as it was.
func TestAnUnprovablyCopiedGetterStaysUntyped(t *testing.T) {
	dir := t.TempDir()

	appender := "core/src/main/java/resource/context/admin/logging/appender"
	cfmlAdmin := "core/src/main/cfml/context/admin"

	writeFiles(t, dir, map[string]string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		cfmlAdmin + "/services.update.functions.cfm":                                  `<cfadmin action="x">`,

		appender + "/Appender.cfc": `component {
	fields=array();
	private component function field() { return createObject("component","Field").init(); }
	public array function getCustomFields() { return fields; }
}`,
		appender + "/Field.cfc": `component {
	function getName() { return ""; }
}`,
		// The getter copies a variable the member's own file never assigns.
		appender + "/Other.cfc": `<cfcomponent extends="Appender">
	<cffunction name="getId" returntype="string" access="public"><cfreturn "other"></cffunction>

	<cffunction name="getCustomFields" returntype="array" output="false">
		<cfset var fields=duplicate(variables.unassigned)>
		<cfreturn fields>
	</cffunction>
</cfcomponent>`,
		cfmlAdmin + "/debugging.other.cfm": `<cfset driverNames=ComponentListPackageAsStruct("lucee.admin.logging.appender",driverNames)>
<cfset drivers={}>
<cfloop collection="#driverNames#" index="n" item="fn">
	<cfif n EQ "Appender" or n EQ "Field">
		<cfcontinue>
	</cfif>
	<cfset drivers[n]=createObject('component',fn)>
</cfloop>
<cfset driver=drivers[n]>
<cfloop array="#driver.getCustomFields()#" index="field">
	<cfoutput>#field.getName()#</cfoutput>
</cfloop>`,
	})

	got := reasonsInCFM(t, dir, cfmlAdmin+"/debugging.other.cfm")
	expectReasons(t, got, map[string]string{
		"field.getName": "variable 'field' has no component ref",
	})
}
