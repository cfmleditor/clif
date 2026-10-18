package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

// The Lucee admin templates fill a struct from a package's components and
// every call on an element of it is a call on one of that family:
//
//	driverNames = ComponentListPackageAsStruct("lucee.admin.mailservers", driverNames);
//	loop ... drivers[name] = createObject("component", componentPath);
//	driver = drivers[trim(form.type)];
//	driver.getShortName();
//
// The family's real source is the package's directory, which the checkout's
// admin holds under core/src/main/java/resource, so each call is checked
// against component it may be. The declarations the source names as the
// abstract base and the field/group helpers stay out: the templates'
// guards skip them, and they do not declare the drivers' contract.
//
// Two shapes are the corpus's: the same template holds both the recipe and
// the consumer (services.mail.form.cfm), and tag templates write
// drivers[n]=tmp with tmp=createObject in a cfloop collection.
func TestARegistryStructElementIsAFamilyComponent(t *testing.T) {
	dir := t.TempDir()

	mailservers := "core/src/main/java/resource/context/admin/mailservers"

	writeFiles(t, dir, map[string]string{
		// The recognized-layout markers, as assumedLuceeAdmin checks them.
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {
	function setUp() {}
}`,
		"core/src/main/cfml/context/admin/services.update.functions.cfm": `<cfadmin action="x">`,
		// One family member the recipe's own guard keeps out: the base, which
		// declares a contract method of its own to show a missing-on-everyone
		// call against.
		mailservers + "/MailServer.cfc": `component {
	function getShortName() { return ""; }
	function intercedeNothing() { return ""; }
}`,
		mailservers + "/GMail.cfc": `component {
	function getShortName() { return "Gmail"; }
	function getHost() { return "smtp.gmail.com"; }
}`,
		mailservers + "/Yahoo.cfc": `component {
	function getShortName() { return "Yahoo"; }
	function getHost() { return "smtp.mail.yahoo.com"; }
}`,
		// The registry and the consumer in one template, script form, with the
		// guard skipping the base's name.
		"core/src/main/cfml/context/admin/services.mail.form.cfm": `<cfscript>
variables.drivers={};
driverNames=ComponentListPackageAsStruct("lucee.admin.mailservers",driverNames);

loop collection=driverNames index="name" item="componentPath" {
	if(name == 'MailServer') continue;
	drivers[name]=createObject("component",componentPath);
}
</cfscript>
<cfset driver = drivers[trim(form.type)]>
<cfset _n = driver.getShortName()>
<cfoutput>#driver.getDisappearAbbreviation()#</cfoutput>`,
	})

	got := reasonsInCFM(t, dir, "core/src/main/cfml/context/admin/services.mail.form.cfm")
	expectReasons(t, got, map[string]string{
		"driver.getShortName": "",
		// getDisappearAbbreviation is declared by no member, and the guard's
		// skip of MailServer keeps the interface out of the family's contract.
		"driver.getDisappearAbbreviation": "method 'getDisappearAbbreviation' not found in the lucee.admin.mailservers family — missing in GMail, Yahoo",
	})
	_ = got
}

// The same recipe written as tag templates: the driver base and the
// constructor loop of debugging.templates.cfm, whose consumer page
// debugging.templates.create.cfm is included by it.
func TestAnIncludedRegistryFamilyTypesTheConsumer(t *testing.T) {
	dir := t.TempDir()

	debug := "core/src/main/java/resource/context/admin/debug"
	cfmlAdmin := "core/src/main/cfml/context/admin"

	writeFiles(t, dir, map[string]string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		cfmlAdmin + "/services.update.functions.cfm":                                  `<cfadmin action="x">`,
		debug + "/Classic.cfc": `component extends="Debug" {
	function getLabel() { return ""; }
}`,
		debug + "/Modern.cfc": `component extends="Debug" {
	function getLabel() { return ""; }
	function onlyModernDeclares() { return ""; }
}`,
		debug + "/Debug.cfc": `component {
	public array function getCustomFields() { return []; }
	function getLabel() { return ""; }
}`,
		debug + "/Field.cfc": `component {
	function getName() { return ""; }
}`,
		// The registry template, tag form, skipping its helper names.
		cfmlAdmin + "/debugging.templates.cfm": `<cfset driverNames=ComponentListPackageAsStruct("lucee.admin.debug",driverNames)>
<cfset drivers={}>
<cfloop collection="#driverNames#" index="n" item="fn">
	<cfif n EQ "Debug" or n EQ "Field">
		<cfcontinue>
	</cfif>
	<cfset tmp=createObject('component',fn)>
	<cfset drivers[trim(tmp.getId())]=tmp>
</cfloop>
<cfcase value="create"><cfinclude template="debugging.templates.create.cfm"/></cfcase>`,
		cfmlAdmin + "/debugging.templates.create.cfm": `<cfset driver=drivers[trim(form.type)]>
<cfset _l = driver.getLabel()>
<cfoutput>#driver.onlyModernDeclares()#</cfoutput>`,
	})

	got := reasonsInCFM(t, dir, cfmlAdmin+"/debugging.templates.create.cfm")
	expectReasons(t, got, map[string]string{
		"driver.getLabel":           "",
		"driver.onlyModernDeclares": "method 'onlyModernDeclares' not found in the lucee.admin.debug family — missing in Classic",
	})
}

// A method no member declares is still answered when the template tests its
// existence before it calls it — structKeyExists(driver,'getBundleName') or
// !isNull(driver.getBundleName) guard the optional driver contract methods —
// and a negated structKeyExists is not such a guard.
func TestAGuardedOptionalFamilyMethodIsDynamic(t *testing.T) {
	dir := t.TempDir()

	cdriver := "core/src/main/java/resource/context/admin/cdriver"

	writeFiles(t, dir, map[string]string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		"core/src/main/cfml/context/admin/services.update.functions.cfm":              `<cfadmin action="x">`,
		cdriver + "/RamCache.cfc": `component {
	function getLabel() { return ""; }
}`,
		"core/src/main/cfml/context/admin/services.cache.create.cfm": `<cfscript>
listNames=ComponentListPackageAsStruct("lucee.admin.cdriver",listNames);
drivers={};
loop struct=listNames index="n" item="fn" {
	drivers[n]=createObject("component",fn);
}
</cfscript>
<cfset driver=drivers[connection.class]>
<cfif !isNull(driver.getBundleName)>
	<cfoutput>#driver.getBundleName()#</cfoutput>
</cfif>
<cfoutput>#structKeyExists(driver,'getMaven') ? driver.getMaven() : ''#</cfoutput>`,
	})

	got := reasonsInCFM(t, dir, "core/src/main/cfml/context/admin/services.cache.create.cfm")
	expectReasons(t, got, map[string]string{
		"driver.getBundleName": "",
		"driver.getMaven":      "",
	})
}

// A receiver the family cannot type keeps the acceptance it had: a struct
// element whose struct holds no registered recipe reports the variable as
// it did, no component ref.
func TestAnUnregisteredStructElementStaysUnresolved(t *testing.T) {
	dir := t.TempDir()

	writeFiles(t, dir, map[string]string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		"core/src/main/cfml/context/admin/services.update.functions.cfm":              `<cfadmin action="x">`,
		"core/src/main/cfml/context/admin/services.other.form.cfm": `<cfscript>
data={};
x = data[form.key];
y = x.getShortName();
</cfscript>`,
	})

	got := reasonsInCFM(t, dir, "core/src/main/cfml/context/admin/services.other.form.cfm")
	expectReasons(t, got, map[string]string{
		"x.getShortName": "variable 'x' has no component ref",
	})
}

// reasonsInCFM is reasonsIn for a template page: it indexes the files the
// dir holds (.cfc components and other .cfm templates, whose includes' the
// page's variables flow through), then parses page and asks every call in it.
func reasonsInCFM(t *testing.T, dir, page string) map[string]string {
	t.Helper()

	r := &Resolver{}
	r.FS, r.WorkspaceFolders = vfs.OS{}, []string{dir}

	if r.Index == nil {
		r.Index = index.New()
	}

	err := filepath.WalkDir(dir, func(p string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if ext := filepath.Ext(p); ext != ".cfc" && ext != ".cfm" {
			return nil
		}

		// The page itself is parsed below; only its includers need indexing.
		if p == filepath.Join(dir, filepath.FromSlash(page)) {
			return nil
		}

		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}

		r.Index.IndexParsed(cfpath.ToURI(p), string(data), parser.ParseWithOptions(cfpath.ToURI(p), string(data), &parser.ParseOptions{}))

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(dir, filepath.FromSlash(page))

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	// ExtractCalls and FuncLookup as the unresolved scan passes them.
	parseOptions := parser.ParseOptions{ExtractCalls: true, FuncLookup: r.FuncLookup(filepath.Dir(file))}

	pr := parser.ParseWithOptions(cfpath.ToURI(file), string(data), &parseOptions)

	got := map[string]string{}

	for i := range pr.AllCalls() {
		c := &pr.AllCalls()[i]
		key := strings.Join(append(append([]string{c.Variable}, hopNames(c.Chain)...), c.FuncName), ".")
		key = strings.TrimPrefix(key, ".")

		got[key] = r.CanResolveCall(c, pr, filepath.Dir(file))
	}

	return got
}
