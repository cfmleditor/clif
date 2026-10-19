package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

	return reasonsInCFMWith(t, r, dir, page)
}

// reasonsInCFMWith is reasonsInCFM asked of one resolver, so a second ask
// sees what the first left in its caches.
func reasonsInCFMWith(t *testing.T, r *Resolver, dir, page string) map[string]string {
	t.Helper()

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

// mailFamilyFixture writes the mail-server family and a recipe page whose
// consumer part is tail, and answers the page's reasons.
func mailFamilyFixture(t *testing.T, tail string) map[string]string {
	t.Helper()

	dir := t.TempDir()
	mailservers := "core/src/main/java/resource/context/admin/mailservers"
	page := "core/src/main/cfml/context/admin/services.mail.form.cfm"

	writeFiles(t, dir, map[string]string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		"core/src/main/cfml/context/admin/services.update.functions.cfm":              `<cfadmin action="x">`,
		mailservers + "/GMail.cfc": `component {
	function getShortName() { return "Gmail"; }
	function getFields() { return []; }
}`,
		mailservers + "/Yahoo.cfc": `component {
	function getShortName() { return "Yahoo"; }
	function getFields() { return []; }
}`,
		page: `<cfscript>
variables.drivers={};
driverNames=ComponentListPackageAsStruct("lucee.admin.mailservers",driverNames);

loop collection=driverNames index="name" item="componentPath" {
	drivers[name]=createObject("component",componentPath);
}
</cfscript>
` + tail,
	})

	return reasonsInCFM(t, dir, page)
}

// Only a right-hand side that is the element read and nothing else holds a
// family member: a call on the element, an argument, a nested read or a
// builtin over it is something else, and was typed as the family.
func TestOnlyAWholeElementReadIsAFamilyMember(t *testing.T) {
	got := mailFamilyFixture(t, `<cfscript>
sn = drivers[form.type].getShortName();
a = sn.toUpperCase();
x = foo(drivers[k]);
b = x.toUpperCase();
y = drivers[k][j];
c = y.toUpperCase();
z = structCount(drivers[k].getFields());
d = z.toUpperCase();
</cfscript>`)

	for _, call := range []string{"sn.toUpperCase", "x.toUpperCase", "y.toUpperCase", "z.toUpperCase"} {
		if strings.Contains(got[call], "family") {
			t.Errorf("%s: %q, typed as the family", call, got[call])
		}
	}
}

// A script for-in over a struct binds its keys, which are strings: a
// registry's key is not one of the family.
func TestAForInOverARegistryBindsAKey(t *testing.T) {
	got := mailFamilyFixture(t, `<cfscript>
for (k in drivers) {
	a = k.toUpperCase();
}
</cfscript>`)

	if strings.Contains(got["k.toUpperCase"], "family") {
		t.Errorf("k.toUpperCase: %q, typed as the family", got["k.toUpperCase"])
	}
}

// A call made right on an element read is a call on the family, whatever
// its key spells: a dot in form.type read the receiver as a member.
func TestACallOnARegistryElementIsOnTheFamily(t *testing.T) {
	got := mailFamilyFixture(t, `<cfset n = drivers[form.type].getShortName()>
<cfset m = drivers[form.type].getNothing()>`)

	expectReasons(t, got, map[string]string{
		"drivers[form.type].getShortName": "",
		"drivers[form.type].getNothing":   "method 'getNothing' not found in the lucee.admin.mailservers family — missing in GMail, Yahoo",
	})
}

// An includer's recipe is read again once its text changes, and
// InvalidatePaths forgets it even when the change kept its stamp: the cache
// kept the first text's facts for the life of the resolver.
func TestAnEditedIncluderRecipeIsReadAgain(t *testing.T) {
	dir := t.TempDir()

	debug := "core/src/main/java/resource/context/admin/debug"
	cfmlAdmin := "core/src/main/cfml/context/admin"
	recipe := `<cfset driverNames=ComponentListPackageAsStruct("lucee.admin.debug",driverNames)>
<cfset drivers={}>
<cfloop collection="#driverNames#" index="n" item="fn">
	<cfset tmp=createObject('component',fn)>
	<cfset drivers[n]=tmp>
</cfloop>
<cfinclude template="debugging.templates.create.cfm"/>`

	writeFiles(t, dir, map[string]string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		cfmlAdmin + "/services.update.functions.cfm":                                  `<cfadmin action="x">`,
		debug + "/Classic.cfc":                 `component { function getLabel() { return ""; } }`,
		cfmlAdmin + "/debugging.templates.cfm": recipe,
		cfmlAdmin + "/debugging.templates.create.cfm": `<cfset driver=drivers[form.type]>
<cfset _l = driver.getNothing()>`,
	})

	page := cfmlAdmin + "/debugging.templates.create.cfm"
	includer := filepath.Join(dir, filepath.FromSlash(cfmlAdmin+"/debugging.templates.cfm"))
	family := "method 'getNothing' not found in the lucee.admin.debug family — missing in Classic"

	r := &Resolver{}
	r.FS, r.WorkspaceFolders = vfs.OS{}, []string{dir}

	expectReasons(t, reasonsInCFMWith(t, r, dir, page), map[string]string{"driver.getNothing": family})

	// The recipe gone, the text a different size: no family.
	writeFiles(t, dir, map[string]string{cfmlAdmin + "/debugging.templates.cfm": `<cfinclude template="debugging.templates.create.cfm"/>`})

	if got := reasonsInCFMWith(t, r, dir, page)["driver.getNothing"]; strings.Contains(got, "family") {
		t.Errorf("after the recipe was removed: %q", got)
	}

	// Back, then the same size and time again with another package: only
	// InvalidatePaths can tell.
	writeFiles(t, dir, map[string]string{cfmlAdmin + "/debugging.templates.cfm": recipe})
	expectReasons(t, reasonsInCFMWith(t, r, dir, page), map[string]string{"driver.getNothing": family})

	info, err := os.Stat(includer)
	if err != nil {
		t.Fatal(err)
	}

	writeFiles(t, dir, map[string]string{cfmlAdmin + "/debugging.templates.cfm": strings.Replace(recipe, "lucee.admin.debug", "lucee.admin.dbugg", 1)})

	if err := os.Chtimes(includer, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	r.InvalidatePaths()

	if got := reasonsInCFMWith(t, r, dir, page)["driver.getNothing"]; strings.Contains(got, "family") {
		t.Errorf("after InvalidatePaths: %q", got)
	}
}

// blockingFS holds the first read of one file until release is closed,
// telling started when it does.
type blockingFS struct {
	vfs.FS
	path    string
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func (b *blockingFS) ReadFile(path string) ([]byte, error) {
	if path == b.path {
		b.once.Do(func() {
			close(b.started)
			<-b.release
		})
	}

	return b.FS.ReadFile(path)
}

// One includer's read does not hold up another template's lookup: the cache
// took one resolver-wide lock across every includer's read and scan.
func TestAnIncluderReadDoesNotHoldUpAnother(t *testing.T) {
	dir := t.TempDir()
	cfmlAdmin := "core/src/main/cfml/context/admin"

	writeFiles(t, dir, map[string]string{
		cfmlAdmin + "/a.cfm":      `<cfinclude template="a.page.cfm"/>`,
		cfmlAdmin + "/a.page.cfm": `<cfset x = 1>`,
		cfmlAdmin + "/b.cfm":      `<cfinclude template="b.page.cfm"/>`,
		cfmlAdmin + "/b.page.cfm": `<cfset y = 1>`,
	})

	abs := func(name string) string { return filepath.Join(dir, filepath.FromSlash(cfmlAdmin+"/"+name)) }

	fsys := &blockingFS{FS: vfs.OS{}, path: abs("a.cfm"), started: make(chan struct{}), release: make(chan struct{})}
	defer close(fsys.release)

	r := &Resolver{}
	r.FS, r.WorkspaceFolders, r.Index = fsys, []string{dir}, index.New()

	for _, name := range []string{"a.cfm", "b.cfm"} {
		data, err := os.ReadFile(abs(name))
		if err != nil {
			t.Fatal(err)
		}

		r.Index.IndexParsed(cfpath.ToURI(abs(name)), string(data), parser.ParseWithOptions(cfpath.ToURI(abs(name)), string(data), &parser.ParseOptions{}))
	}

	go r.includeRegistries(abs("a.page.cfm"))

	<-fsys.started

	done := make(chan struct{})

	go func() {
		r.includeRegistries(abs("b.page.cfm"))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("b's lookup waited on a's read")
	}
}

// A backslash is an ordinary character in a CFML string, which escapes a
// quote only by doubling it: a string ending in one closes where it ends,
// and the listing loop's brace is still found.
func TestARegistryLoopBodyMayHoldABackslash(t *testing.T) {
	dir := t.TempDir()
	mailservers := "core/src/main/java/resource/context/admin/mailservers"
	page := "core/src/main/cfml/context/admin/services.mail.form.cfm"

	writeFiles(t, dir, map[string]string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		"core/src/main/cfml/context/admin/services.update.functions.cfm":              `<cfadmin action="x">`,
		mailservers + "/GMail.cfc": `component { function getShortName() { return "Gmail"; } }`,
		page: `<cfscript>
driverNames=ComponentListPackageAsStruct("lucee.admin.mailservers",driverNames);
loop collection=driverNames index="name" item="componentPath" {
	sep = "C:\";
	say = "a ""}"" b";
	drivers[name]=createObject("component",componentPath);
}
</cfscript>
<cfset driver = drivers[form.type]>
<cfset n = driver.getNothing()>`,
	})

	expectReasons(t, reasonsInCFM(t, dir, page), map[string]string{
		"driver.getNothing": "method 'getNothing' not found in the lucee.admin.mailservers family — missing in GMail",
	})
}

// Only a guard that keeps a name out of the registry excludes it: equal to
// it and skipping, or unequal to it and constructing. A construction made
// only when the key equals a name builds that member, which stays in.
func TestOnlyASkippingGuardExcludesAMember(t *testing.T) {
	cases := map[string]struct{ guard, missing string }{
		"equal and constructing":   {`if (name == 'GMail') drivers[name]=createObject("component",componentPath);`, "GMail, Yahoo"},
		"unequal and constructing": {`if (name != 'Yahoo') drivers[name]=createObject("component",componentPath);`, "GMail"},
		"unequal and skipping": {`if (name != 'Yahoo') continue;
	drivers[name]=createObject("component",componentPath);`, "GMail, Yahoo"},
	}

	for label, c := range cases {
		dir := t.TempDir()
		mailservers := "core/src/main/java/resource/context/admin/mailservers"
		page := "core/src/main/cfml/context/admin/services.mail.form.cfm"

		writeFiles(t, dir, map[string]string{
			"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
			"core/src/main/cfml/context/admin/services.update.functions.cfm":              `<cfadmin action="x">`,
			mailservers + "/GMail.cfc": `component {}`,
			mailservers + "/Yahoo.cfc": `component {}`,
			page: `<cfscript>
driverNames=ComponentListPackageAsStruct("lucee.admin.mailservers",driverNames);
loop collection=driverNames index="name" item="componentPath" {
	` + c.guard + `
}
</cfscript>
<cfset driver = drivers[form.type]>
<cfset n = driver.getNothing()>`,
		})

		got := reasonsInCFM(t, dir, page)["driver.getNothing"]
		if want := "method 'getNothing' not found in the lucee.admin.mailservers family — missing in " + c.missing; got != want {
			t.Errorf("%s: %q, want %q", label, got, want)
		}
	}
}

// A family is not one component, and its marker joins member paths with the
// `|` a union of components is spelled with: handed to ComponentPath, as
// go-to-type-definition hands ComponentOf's answer, it split into its bare
// paths and named whichever member came second. It is reported as unknown.
func TestAFamilyIsNoComponentOfAVariable(t *testing.T) {
	dir := t.TempDir()
	mailservers := "core/src/main/java/resource/context/admin/mailservers"
	page := "core/src/main/cfml/context/admin/services.mail.form.cfm"

	writeFiles(t, dir, map[string]string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		"core/src/main/cfml/context/admin/services.update.functions.cfm":              `<cfadmin action="x">`,
		mailservers + "/GMail.cfc": `component {}`,
		mailservers + "/Yahoo.cfc": `component {}`,
		page: `<cfscript>
driverNames=ComponentListPackageAsStruct("lucee.admin.mailservers",driverNames);
loop collection=driverNames index="name" item="componentPath" {
	drivers[name]=createObject("component",componentPath);
}
</cfscript>
<cfset driver = drivers[form.type]>
<cfset n = driver.getShortName()>`,
	})

	r := &Resolver{}
	r.FS, r.WorkspaceFolders, r.Index = vfs.OS{}, []string{dir}, index.New()

	file := filepath.Join(dir, filepath.FromSlash(page))

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	pr := parser.ParseWithOptions(cfpath.ToURI(file), string(data), &parser.ParseOptions{ExtractCalls: true})

	comp := r.ComponentOf("driver", 7, pr, filepath.Dir(file))
	if comp != "" {
		t.Errorf("ComponentOf(driver) = %q, which ComponentPath reads as %q", comp, r.ComponentPath(comp, filepath.Dir(file)))
	}
}
