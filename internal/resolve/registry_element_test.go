package resolve

import (
	"testing"
)

// The registry consumers loop over the array a family member's getter
// returns — the admin's custom-field set — and their page calls the
// accessor of each element:
//
//	<cfloop array="#driver.getCustomFields()#" index="field">
//		<cfoutput>#field.getName()#</cfoutput>
//
// debug/Debug.cfc declares field()/group() returning
// createObject("component","Field"/"Group").init(...), and
// getCustomFields() returning `fields`; every member's own file assigns
// that variable an array literal of those calls (the array( ... ) script
// form). The element is the union of the components the literals name, and
// elements are read under the union's own rule: a method any of them
// declares resolves.
func TestALoopOverAFamilyArrayGetterReturnsTheLiteralsElements(t *testing.T) {
	dir := t.TempDir()

	adminJava := "core/src/main/java/resource/context/admin"
	debug := adminJava + "/debug"
	cfmlAdmin := "core/src/main/cfml/context/admin"

	writeFiles(t, dir, map[string]string{
		// The recognized-layout markers, as assumedLuceeAdmin checks them.
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		cfmlAdmin + "/services.update.functions.cfm":                                  `<cfadmin action="x">`,
		debug + "/Debug.cfc": `component {
	fields=array();
	private component function field() { return createObject("component","Field").init(); }
	private component function group() { return createObject("component","Group").init(); }
	public array function getCustomFields() { return fields; }
}`,
		debug + "/Field.cfc": `component {
	function getName() { return ""; }
	function getType() { return ""; }
}`,
		debug + "/Group.cfc": `component {
	function getLevel() { return 2; }
}`,
		// One member's fields array literal names both helpers.
		debug + "/Classic.cfc": `component extends="Debug" {
	fields=array(group("Times",".",3),field("Id","id","",false,"."));
	public string function getId() { return "classic"; }
}`,
		debug + "/Simple.cfc": `component extends="Debug" {
	fields=array(field("Id","id","",false,"."),field("Type","type",""));
	public string function getId() { return "simple"; }
}`,
		// The recipe and the consumer in one template, with the guard table's
		// skip of the base and the two helpers.
		cfmlAdmin + "/debugging.templates.cfm": `<cfset driverNames=ComponentListPackageAsStruct("lucee.admin.debug",driverNames)>
<cfset drivers={}>
<cfloop collection="#driverNames#" index="n" item="fn">
	<cfif n EQ "Debug" or n EQ "Field" or n EQ "Group">
		<cfcontinue>
	</cfif>
	<cfset tmp=createObject('component',fn)>
	<cfset drivers[trim(tmp.getId())]=tmp>
</cfloop>
<cfset driver=drivers[trim(form.type)]>
<cfset _l = driver.getCustomFields().len()>
<cfloop array="#driver.getCustomFields()#" index="field">
	<cfoutput>#field.getName()#</cfoutput>
	<cfoutput>#field.getType()#</cfoutput>
	<cfoutput>#field.getLevel()#</cfoutput>
	<cfoutput>#field.getLevelPlus()#</cfoutput>
</cfloop>`,
	})

	got := reasonsInCFM(t, dir, cfmlAdmin+"/debugging.templates.cfm")
	expectReasons(t, got, map[string]string{
		"driver.getCustomFields": "",
		// A chain through the family resolves: the array itself is dynamic.
		"driver.getCustomFields.len": "",
		// Union of the literals' items: Field and Group. Both spell getName
		// and getType (Field), getLevel (Group).
		"field.getName":      "",
		"field.getType":      "",
		"field.getLevel":     "",
		"field.getLevelPlus": "method 'getLevelPlus' not found in Field|Group",
	})
}

func TestALoopOverAnIncludedFamilyListLiteralTypesTheLoop(t *testing.T) {
	dir := t.TempDir()

	adminJava := "core/src/main/java/resource/context/admin"
	ai := adminJava + "/aidriver"
	cfmlAdmin := "core/src/main/cfml/context/admin"

	writeFiles(t, dir, map[string]string{
		// The recognized-layout markers, as assumedLuceeAdmin checks them.
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		cfmlAdmin + "/services.update.functions.cfm":                                  `<cfadmin action="x">`,
		// The aidriver family spells its literal list form over a variables
		// prefix, with named arguments in its items, and its base declares
		// nothing — the constructions alone name the components.
		ai + "/AI.cfc": `component {
	variables.fields=[];
	private function field(string name) { return createObject("component","Field").init(arguments.name); }
	private function group() { return createObject("component","Group").init(); }
	public function getCustomFields() { return variables.fields; }
}`,
		ai + "/Field.cfc": `component {
	function getName() { return ""; }
}`,
		ai + "/Group.cfc": `component {
	function getLevel() { return 2; }
}`,
		ai + "/Claude.cfc": `component extends="AI" {
	variables.fields = [
		group("Endpoint", "."),
		field(displayName = "URL", name = "url", defaultValue = "."),
		field(displayName = "Key", name = "key", required = true)
	];
	public string function getId() { return "claude"; }
}`,
		cfmlAdmin + "/services.ai.cfm": `<cfset driverNames=ComponentListPackageAsStruct("lucee.admin.aidriver",driverNames)>
<cfset drivers={}>
<cfloop collection="#driverNames#" index="n" item="fn">
	<cfif n EQ "AI" or n EQ "Field" or n EQ "Group">
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

	got := reasonsInCFM(t, dir, cfmlAdmin+"/services.ai.cfm")
	expectReasons(t, got, map[string]string{
		"driver.getCustomFields": "",
		"f.getName":              "",
		"f.getLevel":             "",
	})
}
