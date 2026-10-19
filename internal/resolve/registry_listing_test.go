package resolve

import (
	"runtime"
	"strings"
	"testing"
)

// The datasource admin's create page picks its driver straight from the
// listing struct the recipe fills, and reads its fields' elements under a
// struct key:
//
//	<cfset dbdriver = form.type>
//	<cfset driver=createObject("component", drivernames[ dbdriver ])>
//	<cfset driver.init(datasource)>
//	<cfset fields=driver.getFields()>
//	<cfloop collection="#fields#" item="idx">
//		<cfset field = fields[idx]>
//		<cfset type  = field.getType()>
//
// The recipes that walk the listing create exactly this construction per
// key, guard what they keep, and store it in their containers — so the
// construction is a member of the family the recipes' guards serve: the
// fact's members, with the listing's base and helpers kept out. The family
// here is Other.cfc alone (the guard keeps the base Driver out), whose
// getFields returns the fields its own init assigned a literal of — the
// member-copy rule — whose items' field() returns types/Field.cfc. The
// element read under the key is the collection's element contract, so the
// page's field calls read against types/Field.cfc's own declarations.
func TestADriverFromTheListingAndItsKeyedElementsTypeThePage(t *testing.T) {
	dir := t.TempDir()

	dbdriver := "core/src/main/java/resource/context/admin/dbdriver"
	cfmlAdmin := "core/src/main/cfml/context/admin"

	writeFiles(t, dir, map[string]string{
		// The recognized-layout markers, as assumedLuceeAdmin checks them.
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		cfmlAdmin + "/services.update.functions.cfm":                                  `<cfadmin action="x">`,

		// The base the guard keeps out of the registry, and the one member
		// the registry serves, whose fields the member-copy rule proves.
		dbdriver + "/Driver.cfc": `<cfcomponent output="no">
	<cffunction name="field" returntype="component" access="private">
		<cfreturn createObject("component","types.Field").init(arguments.displayName)>
	</cffunction>
	<cffunction name="getType" returntype="numeric"><cfreturn 1></cffunction>
</cfcomponent>`,
		dbdriver + "/Other.cfc": `<cfcomponent extends="types.Driver" implements="types.IDatasource">
	<cffunction name="init" returntype="void">
		<cfargument name="data" required="yes" type="struct">
		<cfscript>
		fields=array(
			field("Class","class",data.classname,true,"."),
			field("Connection String","dsn",data.dsn,true,".")
		);
		</cfscript>
	</cffunction>
	<cffunction name="getFields" returntype="array">
		<cfreturn fields>
	</cffunction>
	<cffunction name="getName" returntype="string"><cfreturn "Other"></cffunction>
	<cffunction name="getDescription" returntype="string"><cfreturn "."></cffunction>
	<cffunction name="onBeforeUpdate" returntype="void"><cfreturn></cffunction>
</cfcomponent>`,

		dbdriver + "/types/Driver.cfc": `<cfcomponent output="no">
	<cffunction name="field" returntype="component" access="private">
		<cfreturn createObject("component","Field").init(arguments.displayName,arguments.name)>
	</cffunction>
	<cffunction name="getType" returntype="numeric"><cfreturn 1></cffunction>
</cfcomponent>`,
		dbdriver + "/types/Field.cfc": `<cfcomponent output="no">
	<cffunction name="init" returntype="types.Field">
		<cfargument name="displayName" required="true" type="string">
		<cfargument name="name" required="true" type="string">
	</cffunction>
	<cffunction name="getType" returntype="numeric"><cfreturn 1></cffunction>
	<cffunction name="getName" returntype="string"><cfreturn ""></cffunction>
</cfcomponent>`,

		// The recipe the includer spells, with the isInstanceOf split the
		// real one keeps: the listing's constructions guarded into two
		// containers.
		cfmlAdmin + "/services.datasource.cfm": `<cfscript>
driverNames=ComponentListPackageAsStruct("lucee.admin.dbdriver",driverNames);
variables.drivers={};
variables.selectors={};
loop collection=driverNames index="n" item="fn" {
	if(n!="Driver" && n!="IDriver") {
		obj = createObject("component",fn);
		if(isInstanceOf( obj, "types.IDriverSelector" ))
			variables.selectors[n] = obj;
		else if(isInstanceOf( obj, "types.IDatasource" ))
			variables.drivers[n] = obj;
	}
}
</cfscript>
<cfcase value="create"><cfinclude template="services.datasource.create.cfm"/></cfcase>`,

		cfmlAdmin + "/services.datasource.create.cfm": `<cfif StructKeyExists(form,"run") and form.run EQ "create2">
	<cfset dbdriver = form.type>
	<cfset driver=createObject("component", drivernames[ dbdriver ])>
	<cfset driver.init(datasource)>
	<cfset fields=driver.getFields()>
</cfif>
<cfloop collection="#fields#" item="idx">
	<cfset field = fields[idx]>
	<cfset t=field.getType()>
	<cfset n=field.getName()>
	<cfset x=field.getUndeclared()>
</cfloop>`,
	})

	got := reasonsInCFM(t, dir, cfmlAdmin+"/services.datasource.create.cfm")
	expectReasons(t, got, map[string]string{
		"driver.init":      "",
		"driver.getFields": "",
		"field.getType":    "",
		"field.getName":    "",
	})

	if reason := got["field.getUndeclared"]; !strings.Contains(reason, "not found in ") {
		t.Errorf("field.getUndeclared: %q, want the missing-method reason", reason)
	}
}

// A receiver the parse guessed dynamic asks whether a registry family
// overrides the guess, and only a file a recipe reaches can hold one: a
// file and includers with no registry skip the assignment walk. The walk
// reads cached assignment facts, so this pins the cost per call staying
// constant rather than catching a quadratic; the gate saves about a sixth
// of the allocation here.
func TestADynamicGuessIsNotWalkedWithoutARegistry(t *testing.T) {
	allocs := func(n int) uint64 {
		dir := t.TempDir()

		var body strings.Builder
		for range n {
			body.WriteString("\t\tsvc.go();\n")
		}

		writeFiles(t, dir, map[string]string{
			"Svc.cfc": `component {
	function init( name ) {
		variables.svc = createObject( "component", "plugins.#name#.Service" );
	}
	function run() {
` + body.String() + `	}
}`,
		})

		var before, after runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&before)

		got := reasonsIn(t, dir, "Svc.cfc")

		runtime.ReadMemStats(&after)

		if r := got["svc.go"]; r != "" {
			t.Fatalf("svc.go: %q, want the dynamic guess to accept it", r)
		}

		return after.TotalAlloc - before.TotalAlloc
	}

	small, large := allocs(100), allocs(800)
	if ratio := float64(large) / float64(small); ratio > 20 {
		t.Errorf("allocation grew %.1fx for 8x the calls on a dynamic receiver (%d to %d bytes)", ratio, small, large)
	}
}
