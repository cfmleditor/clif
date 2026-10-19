package resolve

import (
	"strings"
	"testing"
)

// The logging admin's registry is filled through a factory, and the pages
// that consume it never name the container they read:
//
//	loop collection="#names#" index="n" item="fn">
//		<cfif n NEQ "Appender" and n NEQ "Field" and n NEQ "Group">
//			<cfset arrayAppend(arr,createObject("component",fn))>
//		</cfif>
//	</cfloop>
//	<cfset appenders=doSortedStruct(arr)>
//
// doSortedStruct is the same template's own function, whose body walks the
// array the appends shipped into and stores each element in the struct it
// returns; appenders is that struct. The factory fill is still the recipe
// the other templates' collection loops follow: a package's components,
// less the names the guard compares, held in a struct, so the calls on an
// element are the family again — if the factory's body proves the shape.
//
// The consumers then read the container indirectly:
//
//	<cfloop list="appender,layout" item="_name">
//		<cfset drivers=variables[_name&"s"]>
//		<cfloop collection="#drivers#" index="driverClass" item="driver">
//
// _name is the list-loop item bound to the literals appender and layout,
// so each _name's turn names one of the template's own registry
// containers, and driver is a member of whichever turn's family. The two
// turns' families the receiver cannot tell apart statically — the
// receiver is a union of the families, and a call is answered when every
// one of them answers it, since each holds the receiver on its own turn.
//
// The two families' helpers are kept apart by their directories: an
// appender's fields and a layout's fields are different components spelled
// the same, and the loop over the getter holds both, as lists of files.
func TestAFactoryFilledRegistryTypesItsIndirectPickup(t *testing.T) {
	dir := t.TempDir()
	page := loggingAdminFiles(t, dir, `<cfloop list="appender,layout" item="_name">
	<cfset drivers=variables[_name&"s"]>
	<cfloop collection="#drivers#" index="driverClass" item="driver">
		<cfset _n=driver.getCustomFields()>
		<cfoutput>#driver.getClass()#</cfoutput>
		<cfloop array="#driver.getCustomFields()#" index="field">
			<cfoutput>#field.getName()#</cfoutput>
			<cfoutput>#field.getLevel()#</cfoutput>
			<cfoutput>#field.getUndeclared()#</cfoutput>
		</cfloop>
	</cfloop>
</cfloop>`)

	got := reasonsInCFM(t, dir, page)
	expectReasons(t, got, map[string]string{
		// getCustomFields is a base's own: both families declare it.
		"driver.getCustomFields": "",
		// The field loop's element is both families' helper sets: a method
		// any helper of either family declares resolves; one no helper
		// anywhere declares stays reported.
		"field.getName":  "",
		"field.getLevel": "",
	})

	if reason := got["driver.getClass"]; !strings.Contains(reason, "not found in the lucee.admin.logging.appender family") {
		t.Errorf("driver.getClass: %q, want the first family's missing-method reason", reason)
	}

	if reason := got["field.getUndeclared"]; !strings.Contains(reason, "not found in ") {
		t.Errorf("field.getUndeclared: %q, want the union's missing-method reason", reason)
	}

	if !strings.Contains(got["field.getUndeclared"], "Field|Group") {
		t.Errorf("field.getUndeclared: %q, want the helper components named", got["field.getUndeclared"])
	}
}

// The pickup is only for a bound list item: a `variables[v&"s"]` read whose
// v no list loop binds names no container this template proves, and the
// receiver reads as it did — no component ref.
func TestAnUnboundNamePickupStaysUnresolved(t *testing.T) {
	dir := t.TempDir()

	adminJava := "core/src/main/java/resource/context/admin"
	mailservers := adminJava + "/mailservers"
	cfmlAdmin := "core/src/main/cfml/context/admin"

	writeFiles(t, dir, map[string]string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		cfmlAdmin + "/services.update.functions.cfm":                                  `<cfadmin action="x">`,
		mailservers + "/MailServer.cfc": `component {
	function getShortName() { return ""; }
}`,
		mailservers + "/GMail.cfc": `component {
	function getShortName() { return "gmail"; }
}`,
		// A recipe this template never reaches: the read's concatenation is
		// not of a list-loop item, so the registry backs no receiver.
		cfmlAdmin + "/services.other.form.cfm": `<cfscript>
registrations={};
registrations=ComponentListPackageAsStruct("lucee.admin.mailservers",registrations);
registries={};
loop collection=registrations index="n" item="fn" {
	registries[n]=createObject("component",fn);
}
function fill(reg) {
	sct={};
	for (el in arguments.reg) {
		sct[el.getShortName()]=el;
	}
	return sct;
}
</cfscript>
<cfset registries=fill(registries)>
<cfloop list="mailservers" item="_kind">
	<cfset drivers=variables[_kind&"s"]>
	<cfloop collection="#drivers#" index="driverClass" item="driver">
		<cfoutput>#driver.getShortName()#</cfoutput>
	</cfloop>
</cfloop>`,
	})

	got := reasonsInCFM(t, dir, cfmlAdmin+"/services.other.form.cfm")
	expectReasons(t, got, map[string]string{
		"driver.getShortName": "variable 'driver' has no component ref",
	})
}

// A loop whose collection is read off its own variable — the item the loop
// binds is the receiver of the collection's call — gives no element: the
// collection is a chain off the loop variable, whose element is what the
// collection holds, and following that went round for ever. It overflowed
// the stack, which no recover catches; the answer is no type.
func TestALoopOverItsOwnVariableDoesNotRecurse(t *testing.T) {
	dir := t.TempDir()

	writeFiles(t, dir, map[string]string{
		"tags.cfm":     `<cfloop array="#x.getItems()#" item="x"><cfoutput>#x.getName()#</cfoutput></cfloop>`,
		"filtered.cfm": `<cfloop array="#x.getItems().filter(f)#" item="x"><cfoutput>#x.getName()#</cfoutput></cfloop>`,
		"Script.cfc": `component {
	function f() {
		for ( row in row.getRows() ) {
			row.getName();
		}
	}
}`,
	})

	for _, page := range []string{"tags.cfm", "filtered.cfm"} {
		got := reasonsInCFM(t, dir, page)
		if reason := got["x.getName"]; reason == "" {
			t.Errorf("%s: x.getName accepted, want it untyped", page)
		}
	}

	got := reasonsIn(t, dir, "Script.cfc")
	if reason := got["row.getName"]; reason == "" {
		t.Errorf("row.getName accepted, want it untyped")
	}
}

// loggingAdminFiles is the logging admin's two families and the template
// whose factory fills their registries, with the consuming page create in
// server.logging.create.cfm; it answers that page's path.
func loggingAdminFiles(t *testing.T, dir, create string) string {
	t.Helper()

	adminJava := "core/src/main/java/resource/context/admin"
	logging := adminJava + "/logging"
	appender := logging + "/appender"
	layout := logging + "/layout"
	cfmlAdmin := "core/src/main/cfml/context/admin"

	appenderBase := `component {
	fields=array();
	public string function getAppenderOnly() { return ""; }
	private component function field() { return createObject("component","Field").init(); }
	private component function group() { return createObject("component","Group").init(); }
	public array function getCustomFields() { return fields; }
}`
	layoutBase := `component {
	fields=array();
	private component function field() { return createObject("component","Field").init(); }
	private component function group() { return createObject("component","Group").init(); }
	public array function getCustomFields() { return fields; }
}`

	writeFiles(t, dir, map[string]string{
		// The recognized-layout markers, as assumedLuceeAdmin checks them.
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		cfmlAdmin + "/services.update.functions.cfm":                                  `<cfadmin action="x">`,

		// The two families' bases are skipped by the guards' compares, and
		// the helpers are the only components the getters' literals name.
		appender + "/Appender.cfc": appenderBase,
		appender + "/Field.cfc": `component {
	function getName() { return ""; }
}`,
		appender + "/Group.cfc": `component {
	function getLevel() { return 2; }
}`,
		// Each member's own file proves a non-empty literal set.
		appender + "/Console.cfc": `component extends="Appender" {
	fields=array(field("Mode","mode"),group("Times",".",3));
	public string function getId() { return "console"; }
}`,
		appender + "/Datasource.cfc": `component extends="Appender" {
	fields=array(field("DSN","dsn"),group("X","x",2));
	public string function getId() { return "datasource"; }
}`,

		layout + "/Layout.cfc": layoutBase,
		layout + "/Field.cfc": `component {
	function getName() { return ""; }
}`,
		layout + "/Group.cfc": `component {
	function getLevel() { return 2; }
}`,
		layout + "/Classic.cfc": `component extends="Layout" {
	fields=array(field("Pattern","pattern"),group("G","g"));
	public string function getId() { return "classic"; }
}`,
		layout + "/HTML.cfc": `component extends="Layout" {
	fields=array(field("Title","title"),group("H","h"));
	public string function getId() { return "html"; }
}`,

		// The factory that builds the registries, and the consumers', in the
		// spans the real admin's template spells: a listing loop whose
		// constructions a factory function turns into the struct the page
		// reads back under the item-plus-concatenation name. Both recipes
		// reuse one listing struct's and one array's name, the way the real
		// template does — each listing's own loop and its own factory call
		// are the pair that names its container.
		cfmlAdmin + "/server.logging.cfm": `<cfscript>
	function doSortedStruct(arr) {
		var sct=structNew('ordered');
		loop array=arguments.arr item="local.el" {
			sct[el.getClass()]=el;
		}
		return sct;
	}
</cfscript>
<cfset arr=[]>
<cfset names=structnew("linked")>
<cfset names=ComponentListPackageAsStruct("lucee.admin.logging.appender",names)>
<cfloop collection="#names#" index="n" item="fn">
	<cfif n NEQ "Appender" and n NEQ "Field" and n NEQ "Group">
		<cfset arrayAppend(arr,createObject("component",fn))>
	</cfif>
</cfloop>
<cfset appenders=doSortedStruct(arr)>

<cfset names=ComponentListPackageAsStruct("logging.layout",names)>
<cfloop collection="#names#" index="n" item="fn">
	<cfif n NEQ "Layout" and n NEQ "Field" and n NEQ "Group">
		<cfset arrayAppend(arr,createObject("component",fn))>
	</cfif>
</cfloop>
<cfset layouts=doSortedStruct(arr)>
<cfcase value="create"><cfinclude template="server.logging.create.cfm"/></cfcase>`,

		cfmlAdmin + "/server.logging.create.cfm": create,
	})

	return cfmlAdmin + "/server.logging.create.cfm"
}

// Both turns of `<cfloop list="appender,layout">` run, so driver holds an
// appender on one and a layout on the other: a method only one family
// declares fails on the other family's turn, and is a finding.
func TestAMethodEveryIndirectFamilyLacksIsReported(t *testing.T) {
	dir := t.TempDir()
	page := loggingAdminFiles(t, dir, `<cfloop list="appender,layout" item="_name">
	<cfset drivers=variables[_name&"s"]>
	<cfloop collection="#drivers#" index="driverClass" item="driver">
		<cfoutput>#driver.getAppenderOnly()#</cfoutput>
		<cfoutput>#driver.getCustomFields()#</cfoutput>
	</cfloop>
</cfloop>`)

	got := reasonsInCFM(t, dir, page)
	expectReasons(t, got, map[string]string{
		"driver.getCustomFields": "",
	})

	if reason := got["driver.getAppenderOnly"]; !strings.Contains(reason, "not found in the logging.layout family") {
		t.Errorf("driver.getAppenderOnly: %q, want the layout family's missing-method reason", reason)
	}
}

// A turn whose container is not one of the template's registries proves
// nothing about the receiver on that turn, so the pickup proves nothing:
// foos is an ordinary struct, and driver is not an appender when _n is foo.
func TestAnIndirectPickupWithAnUnprovenTurnDeclines(t *testing.T) {
	dir := t.TempDir()
	page := loggingAdminFiles(t, dir, `<cfset foos={}>
<cfloop list="appender,foo" item="_n">
	<cfset drivers=variables[_n&"s"]>
	<cfloop collection="#drivers#" index="driverClass" item="driver">
		<cfoutput>#driver.getAppenderOnly()#</cfoutput>
	</cfloop>
</cfloop>`)

	got := reasonsInCFM(t, dir, page)
	expectReasons(t, got, map[string]string{
		"driver.getAppenderOnly": "variable 'driver' has no component ref",
	})
}
