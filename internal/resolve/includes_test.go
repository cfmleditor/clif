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

// includeFixture is a component that mixes in two templates, extends a base,
// and a page that includes a helper through a mapping.
func includeFixture(t *testing.T) (dir string, r *Resolver) {
	t.Helper()

	dir = t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o750); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"Base.cfc": `<cfcomponent><cffunction name="baseFn"></cffunction></cfcomponent>`,
		"api.cfc": `<cfcomponent extends="Base">
	<cffunction name="getService"></cffunction>
	<cfinclude template="api-a.cfm">
	<cfinclude template="api-b.cfm">
</cfcomponent>`,
		"api-a.cfm":      `<cffunction name="fromA"></cffunction>`,
		"api-b.cfm":      `<cffunction name="fromB"></cffunction>`,
		"page.cfm":       `<cfinclude template="/app/lib/helper.cfm">`,
		"lib/helper.cfm": `<cffunction name="helperFn"></cffunction>`,
	}

	r = &Resolver{FS: vfs.OS{}, Index: index.New(), Mappings: map[string]string{"app": dir}}

	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}

		r.Index.IndexFile(cfpath.ToURI(p), body)
	}

	return dir, r
}

func bareCall(t *testing.T, r *Resolver, file, funcName string) (CallTarget, string) {
	t.Helper()

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	pr := parser.Parse(cfpath.ToURI(file), string(data))

	return r.ResolveCallTarget(&parser.CallSite{FuncName: funcName}, pr, filepath.Dir(file))
}

// TestBareCallResolvesThroughCfinclude covers a template mixed into a
// component: it runs in that component's variables scope, so it can call what
// the component declares, what the component extends, and what a sibling
// template declares; and a page can call what it includes. Before, all of
// these were "no qualifier, not in file".
func TestBareCallResolvesThroughCfinclude(t *testing.T) {
	dir, r := includeFixture(t)

	for _, tc := range []struct {
		file, fn, wantIn string
	}{
		{"api-a.cfm", "getService", "api.cfc"},     // the includer
		{"api-a.cfm", "fromB", "api-b.cfm"},        // a sibling mixin
		{"api-a.cfm", "baseFn", "Base.cfc"},        // the includer's extends chain
		{"api.cfc", "fromA", "api-a.cfm"},          // the includer calling its mixin
		{"page.cfm", "helperFn", "lib/helper.cfm"}, // through a mapping
	} {
		target, reason := bareCall(t, r, filepath.Join(dir, tc.file), tc.fn)
		if reason != "" {
			t.Errorf("%s: %s() unresolved: %s", tc.file, tc.fn, reason)

			continue
		}

		want := cfpath.ToURI(filepath.Join(dir, filepath.FromSlash(tc.wantIn)))
		if target.Kind != TargetInclude && target.Kind != TargetExtends || !cfpath.SamePath(string(target.URI), string(want)) {
			t.Errorf("%s: %s() landed on %s %s, want %s", tc.file, tc.fn, target.Kind, target.URI, want)
		}
	}

	if _, reason := bareCall(t, r, filepath.Join(dir, "api-a.cfm"), "nowhere"); reason == "" {
		t.Error("nowhere() resolved; a name no file in scope declares must stay unresolved")
	}

	// page.cfm is not in api.cfc's scope, so it cannot see api's functions.
	if _, reason := bareCall(t, r, filepath.Join(dir, "page.cfm"), "getService"); reason == "" {
		t.Error("page.cfm resolved getService() through a component it is not included by")
	}
}

// TestIncludeGraphFollowsReindex is the invalidation: the resolver keeps a
// reverse map of every include, and an edit that drops one has to reach it.
func TestIncludeGraphFollowsReindex(t *testing.T) {
	dir, r := includeFixture(t)
	mixin := filepath.Join(dir, "api-b.cfm")

	if _, reason := bareCall(t, r, mixin, "getService"); reason != "" {
		t.Fatalf("before the edit: %s", reason)
	}

	api := filepath.Join(dir, "api.cfc")
	edited := `<cfcomponent extends="Base">
	<cffunction name="getService"></cffunction>
	<cfinclude template="api-a.cfm">
</cfcomponent>`

	if err := os.WriteFile(api, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}

	r.Index.IndexFile(cfpath.ToURI(api), edited)

	if _, reason := bareCall(t, r, mixin, "getService"); reason == "" {
		t.Error("api-b.cfm still resolves through api.cfc after api.cfc stopped including it")
	}
}

func TestLuceeSourceMappingConnectsIncludedHelpers(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		"core/src/main/cfml/context/admin/services.update.functions.cfm":              `<cfscript>function stable(){}</cfscript>`,
		"core/src/main/cfml/context/admin/ext.functions.cfm":                          `<cfscript>function sortable(){}</cfscript>`,
		"test/tickets/Example.cfc":                                                    `component {function beforeAll(){include template="/admin/services.update.functions.cfm";include template="/admin/ext.functions.cfm";}function run(){stable();sortable();absent();}}`,
	})
	cfpath.InvalidateAppMappingsCache()
	t.Cleanup(cfpath.InvalidateAppMappingsCache)
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "test/tickets/Example.cfc"), map[string]string{"stable": "", "sortable": "", "absent": "no qualifier, not in file"})
}

// An include written `#application.baseUrl#/lib/helper.cfm` names the template
// from the application's root: the variable holds the path the site is
// installed under. Galaxie Blog writes every include so, and 67 bare calls to
// functions its page.cfm declares were "no qualifier, not in file".
func TestIncludeThroughARootVariableReachesTheTemplate(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o750); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"Application.cfc": `<cfcomponent></cfcomponent>`,
		"index.cfm":       `<cfinclude template="#application.baseUrl#/lib/helper.cfm"><cfinclude template="#application.baseUrl#/lib/view.cfm">`,
		"lib/helper.cfm":  `<cffunction name="helperFn"></cffunction>`,
		"lib/view.cfm":    `<cfset x = 1>`,
		"other.cfm":       `<cfinclude template="#call()#/lib/helper.cfm"><cfinclude template="#application.baseUrl#helper.cfm">`,
	}

	r := &Resolver{FS: vfs.OS{}, Index: index.New()}

	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}

		r.Index.IndexFile(cfpath.ToURI(p), body)
	}

	// view.cfm is included beside helper.cfm, so it sees what helper.cfm declares.
	if _, reason := bareCall(t, r, filepath.Join(dir, "lib", "view.cfm"), "helperFn"); reason != "" {
		t.Errorf("helperFn() from view.cfm: %s", reason)
	}

	// A call, or a variable not followed by a slash, is no root.
	for _, raw := range []string{"#call()#/lib/helper.cfm", "#application.baseUrl#helper.cfm"} {
		if got := r.IncludePath(raw, filepath.Join(dir, "other.cfm")); got != "" {
			t.Errorf("IncludePath(%q) = %q, want none", raw, got)
		}
	}

	if got := r.IncludePath("#application.baseUrl#/lib/helper.cfm", filepath.Join(dir, "index.cfm")); !cfpath.SamePath(got, filepath.Join(dir, "lib", "helper.cfm")) {
		t.Errorf("IncludePath through a root variable = %q", got)
	}
}

// An included template runs in its includer's variables scope, so
// `variables.helper` in it is what the includer set as `variables.helper`, as
// the unscoped `helper` already was. TestBox's HTML reporters set
// variables.helper and render every partial through an include: 108 findings.
func TestAVariablesScopedReceiverInAnIncludedTemplateIsTheIncluders(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "partials"), 0o750); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"Helper.cfc": `component { function safeId(){} }`,
		"Base.cfc": `component {
	function run() {
		variables.helper = new Helper();
		include "layout.cfm";
	}
	private string function renderPartial( required string name ) {
		include "partials/#arguments.name#.cfm";
	}
}`,
		"layout.cfm":     `<cfoutput>#variables.helper.safeId( 1 )# #variables.helper.nopeLayout()#</cfoutput>`,
		"partials/p.cfm": `<cfoutput>#variables.helper.nopePartial()#</cfoutput><cfset variables.own = 1>#variables.own.nothing()#`,
		"Alone.cfm":      `<cfoutput>#variables.helper.safeId( 1 )#</cfoutput>`,
	}

	r := &Resolver{FS: vfs.OS{}, Index: index.New()}

	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}

		r.Index.IndexFile(cfpath.ToURI(p), body)
	}

	got := reasonsWith(t, r, dir, "layout.cfm")

	if got["variables.helper.safeId"] != "" {
		t.Errorf("variables.helper.safeId in an included template: %q, want it typed by the includer", got["variables.helper.safeId"])
	}

	if got["variables.helper.nopeLayout"] == "" || got["variables.helper.nopeLayout"] == "variable 'variables.helper' has no component ref" {
		t.Errorf("variables.helper.nopeLayout: %q, want a method missing from Helper", got["variables.helper.nopeLayout"])
	}

	// A template nothing includes has no includer to ask.
	if alone := reasonsWith(t, r, dir, "Alone.cfm"); !strings.Contains(alone["variables.helper.safeId"], "no component ref") {
		t.Errorf("a template nothing includes was typed: %q", alone["variables.helper.safeId"])
	}
}

// A local that aliases a name the file holds as a Java class (a quoted
// import) is as dynamic as the name.
func TestAnAliasOfAnImportedJavaClassIsDynamic(t *testing.T) {
	dir := t.TempDir()
	body := `component {
	import "com.x.Server";
	function run() {
		application.srv = new Server();
		var s = application.srv;
		s.start();
	}
}`
	p := filepath.Join(dir, "T.cfc")

	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{FS: vfs.OS{}, Index: index.New()}
	r.Index.IndexFile(cfpath.ToURI(p), body)

	if got := reasonsWith(t, r, dir, "T.cfc"); got["s.start"] != "" {
		t.Errorf("s.start: %q, want it accepted", got["s.start"])
	}
}

// A local assigned a struct literal is a struct, whatever keys the literal
// has; a field set to one in another function may have grown a closure key.
func TestALocalAssignedAStructLiteralIsAStruct(t *testing.T) {
	dir := t.TempDir()
	body := `component {
	function init() {
		variables.settings = {};
	}
	function run() {
		var a = { "x": 1, "f": ( v ) => { return v; } };
		a.copy().append( { "k": 1 } );
		a.noSuchMemberOfAStruct();
		var m = {
			x : 1,
			y : [
				1, 2
			]
		};
		m.copy();
		var l = [
			1,
			2
		];
		l.len();
		variables.settings.provider();
	}
}`

	if err := os.WriteFile(filepath.Join(dir, "S.cfc"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{FS: vfs.OS{}, Index: index.New()}
	got := reasonsWith(t, r, dir, "S.cfc")

	if got["a.copy"] != "" || got["a.copy.append"] != "" {
		t.Errorf("struct members on a literal were reported: %v", got)
	}

	if got["m.copy"] != "" || got["l.len"] != "" {
		t.Errorf("a literal written over several lines was not read whole: %v", got)
	}

	if !strings.Contains(got["a.noSuchMemberOfAStruct"], "native.struct") {
		t.Errorf("a non-member on a literal struct: %q", got["a.noSuchMemberOfAStruct"])
	}

	if strings.Contains(got["variables.settings.provider"], "native.struct") {
		t.Errorf("a field set to {} in another function was checked as a plain struct: %q", got["variables.settings.provider"])
	}
}

// An element of a value that is dynamic is dynamic.
func TestAnElementOfADynamicValueIsDynamic(t *testing.T) {
	dir := t.TempDir()
	body := `component {
	function run() {
		var holder = createObject( "java", "x.Y" ).make();
		var list = holder.items();
		var first = list[ 1 ];
		first.anything();
	}
}`

	if err := os.WriteFile(filepath.Join(dir, "E.cfc"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{FS: vfs.OS{}, Index: index.New()}

	if got := reasonsWith(t, r, dir, "E.cfc"); got["first.anything"] != "" {
		t.Errorf("first.anything: %q, want it accepted", got["first.anything"])
	}
}

// A variable assigned from a call on a dynamic value is dynamic.
func TestAVariableAssignedFromACallOnADynamicValueIsDynamic(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"Application.cfc": `component {
	function onApplicationStart() {
		application.svc = createObject( "component", variables.root & ".Svc" );
	}
}`,
		"D.cfc": `component {
	function run() {
		var s = application.svc.setup();
		s.anything();
		var u = application.svc?.setup();
		u.anything();
		var g = isNull( application.svc ) ? "" : application.svc.thing();
		g.anything();
		var a = application.svc.items;
		a.anything();

	}
}`,
	}

	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	r := &Resolver{FS: vfs.OS{}, Index: index.New()}

	if got := reasonsWith(t, r, dir, "D.cfc"); got["s.anything"] != "" || got["u.anything"] != "" || got["g.anything"] != "" || got["a.anything"] != "" {
		t.Errorf("s.anything: %q, want it accepted", got["s.anything"])
	}
}

// An argument declared a native type holds one: a method that type's contract
// has is accepted, one it lacks is not.
func TestAnArgumentOfADeclaredNativeTypeAnswersItsContract(t *testing.T) {
	dir := t.TempDir()
	body := `component {
	function run( required query q, array a, string s ) {
		q.getMeta();
		q.getRecordCount();
		a.len();
		s.getBytes();
	}
}`

	if err := os.WriteFile(filepath.Join(dir, "N.cfc"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{FS: vfs.OS{}, Index: index.New()}
	got := reasonsWith(t, r, dir, "N.cfc")

	if got["q.getMeta"] != "" || got["q.getRecordCount"] != "" || got["a.len"] != "" {
		t.Errorf("methods of a declared native argument were reported: %v", got)
	}
}

// A struct's value and a query's column are of no shape the source gives.
func TestAMemberOfAStructOrAQueryIsDynamic(t *testing.T) {
	dir := t.TempDir()
	body := `component {
	function run() {
		var s = { x : 1 };
		s.x.anything();
		var q = queryNew( "a,b" );
		q.a.anything();
		q.a[ 1 ].anything();
		q.columnList.uCase();
		q.columnList.noSuchStringMember();
		var d = createObject( "java", "x.Y" ).make();
		d.prop.anything();
	}
}`

	if err := os.WriteFile(filepath.Join(dir, "M.cfc"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{FS: vfs.OS{}, Index: index.New()}
	got := reasonsWith(t, r, dir, "M.cfc")

	for _, k := range []string{"s.x.anything", "q.a.anything", "q.a.anything", "q.columnList.uCase", "d.prop.anything"} {
		if got[k] != "" {
			t.Errorf("%s: %q, want it accepted", k, got[k])
		}
	}

	if got["q.columnList.noSuchStringMember"] == "" {
		t.Error("columnList is a string: a method no string has was accepted")
	}
}

// An argument every caller passes a dynamic value is dynamic.
func TestAnArgumentEveryCallerPassesADynamicValueIsDynamic(t *testing.T) {
	dir := t.TempDir()
	body := `component {
	function a() {
		var x = createObject( "java", "x.Y" ).make();
		use( x );
		use( createObject( "java", "x.Y" ).other() );
	}
	function b() {
		use( someUnknownThing() );
	}
	private function use( o ) {
		o.anything();
	}
	private function onlyDynamic( p ) {
		p.anything();
	}
	function c() {
		var z = createObject( "java", "x.Y" ).make();
		onlyDynamic( z );
	}
}`

	if err := os.WriteFile(filepath.Join(dir, "G.cfc"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "G.cfc")

	if got["p.anything"] != "" {
		t.Errorf("p.anything: %q, want it accepted: its one caller passes a dynamic value", got["p.anything"])
	}

	if got["o.anything"] == "" {
		t.Errorf("o.anything was accepted though one caller passes a value nothing is known of")
	}
}

// A class in a framework's own namespace the bundled stubs lack is the stubs'
// gap, not a component that does not exist.
func TestAFrameworkClassTheStubsLackIsDynamic(t *testing.T) {
	dir := t.TempDir()
	body := `component {
	function run() {
		var a = new coldbox.system.orm.hibernate.util.NotInTheStubsXyz();
		a.anything();
		var b = new vendor.missing.Thing();
		b.anything();
	}
}`

	if err := os.WriteFile(filepath.Join(dir, "F.cfc"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	got := reasonsWith(t, &Resolver{}, dir, "F.cfc")

	if got["a.anything"] != "" {
		t.Errorf("a.anything: %q, want it accepted", got["a.anything"])
	}

	if !strings.Contains(got["b.anything"], "does not exist") {
		t.Errorf("b.anything: %q, want a component that does not exist", got["b.anything"])
	}
}

// A native type is claimed of a variable only where its assignment is the one
// that reaches the use: not when the assignment sits in a block that has
// closed before the use and the variable was assigned before it. And an index
// on the end of a chain reads an element, not the collection.
func TestAFlowAwareNativeClaim(t *testing.T) {
	dir := t.TempDir()
	body := `component {
	array function list() { return []; }
	function run() {
		var once = [];
		once.noSuchArrayMemberXyz();
		var out = "";
		if ( 1 ) {
			out = [];
		}
		out.noSuchArrayMemberXyz();
		var kept = "";
		if ( 1 ) {
			kept = [];
			kept.noSuchArrayMemberXyz();
		}
		var el = list()[ 1 ];
		el.anything();
	}
}`

	if err := os.WriteFile(filepath.Join(dir, "FL.cfc"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	got := reasonsWith(t, &Resolver{}, dir, "FL.cfc")

	if strings.Contains(got["out.noSuchArrayMemberXyz"], "native.array") {
		t.Errorf("out was claimed an array though the empty string may reach: %q", got["out.noSuchArrayMemberXyz"])
	}

	if !strings.Contains(got["kept.noSuchArrayMemberXyz"], "native.array") {
		t.Errorf("kept: %q, want the array its own block assigned it", got["kept.noSuchArrayMemberXyz"])
	}

	if got["el.anything"] != "" {
		t.Errorf("el.anything: %q, an element read is not the collection", got["el.anything"])
	}
}

// A variable assigned in a branch may hold what it held before: a use after
// the branch is checked against both, and one nothing conditional reaches is
// checked against the assignment that always runs.
func TestAUseAfterABranchHoldsEveryAssignmentThatReachesIt(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"A.cfc": `component { function onlyA(){} }`,
		"B.cfc": `component { function onlyB(){} }`,
		"Use.cfc": `component {
	function run( c ) {
		var x = new A();
		if ( c ) {
			x = new B();
		}
		x.onlyA();
		x.onlyB();
		x.neither();
		var y = new A();
		y = new B();
		y.onlyA();
	}
}`,
	})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")

	if got["x.onlyA"] != "" || got["x.onlyB"] != "" {
		t.Errorf("a method either alternative has was reported: onlyA %q, onlyB %q", got["x.onlyA"], got["x.onlyB"])
	}

	if !strings.Contains(got["x.neither"], "not found in") || !strings.Contains(got["x.neither"], "|") {
		t.Errorf("x.neither: %q, want a method neither has reported against both", got["x.neither"])
	}

	if !strings.Contains(got["y.onlyA"], "not found in B") {
		t.Errorf("y.onlyA: %q, want the assignment that always runs to decide", got["y.onlyA"])
	}
}
