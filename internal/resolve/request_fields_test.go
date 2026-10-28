package resolve

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestALazyRequestFieldIsWhatItsOnlyWriterStores: Mura's getCurrentUser()
// creates request.currentUser when it is absent and returns it, and nothing
// else writes that key, so it returns the facade it creates. A second writer
// anywhere, or a scan that does not know every file, withholds the answer.
func TestALazyRequestFieldIsWhatItsOnlyWriterStores(t *testing.T) {
	files := map[string]string{
		"Facade.cfc": `component { function init(){ return this; } }`,
		"Obj.cfc": `component {
	public function getCurrentUser() {
		if ( !structKeyExists(request,"currentUser") ) {
			request.currentUser=createObject("component","Facade").init();
		}
		return request.currentUser;
	}
}`,
	}

	returns := func(files map[string]string, batch bool) string {
		dir := t.TempDir()
		writeFiles(t, dir, files)

		r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
		if batch {
			r.InferArgsFiles = cfmlFilesIn(t, dir)
		}

		return r.ReturnComponentOf(r.ResolveFunc(filepath.Join(dir, "Obj.cfc"), "getCurrentUser", dir))
	}

	if got := returns(files, true); filepath.Base(got) != "Facade.cfc" {
		t.Errorf("only writer: %q, want Facade.cfc", got)
	}

	if got := returns(files, false); got != "" {
		t.Errorf("outside a batch scan: %q, want nothing", got)
	}

	files["page.cfm"] = `<cfset request.currentUser = "guest">`
	if got := returns(files, true); got != "" {
		t.Errorf("a second writer: %q, want nothing", got)
	}
}

// requestWriteReReference is the regular expression requestWriteCounts
// replaced, kept as the reference the scanner is held to.
func requestWriteReReference(name string) *regexp.Regexp {
	q := regexp.QuoteMeta(name)

	return regexp.MustCompile(`(?i)\brequest\s*\.\s*` + q + `\s*=[^=]` +
		`|\brequest\s*\[\s*["']` + q + `["']\s*\]\s*=[^=]` +
		`|structInsert\s*\(\s*request\s*,\s*["']` + q + `["']` +
		`|name\s*=\s*["']request\.` + q + `["']`)
}

// TestRequestWriteCountsMatchTheRegexItReplaced: one pass per write form over
// a file replaced a regex compiled and run once per request key; they must
// count the same writes.
func TestRequestWriteCountsMatchTheRegexItReplaced(t *testing.T) {
	content := `<cfset request.currentUser = createObject("component", "user")>
<cfset REQUEST . CurrentUser=x>
<cfif request.currentUser == y></cfif>
<cfset request.currentUserX = 1>
<cfset myrequest.currentUser = 2>
<cfset request.a.b = 3>
<cfset request[ "siteID" ] = 4>
<cfset request['siteid']=5>
<cfset request["siteid"] == 6>
<cfset structInsert( request, "flash", {} )>
<cfset StructInsert(request,'Flash',{})>
<cfparam name="request.layout" default="">
<cfparam NAME = 'request.Layout' default="">
<cfset x = request.currentUser>
`
	counts := requestWriteCounts(content)

	for _, name := range []string{"currentUser", "currentUserX", "a", "siteID", "flash", "layout", "missing"} {
		want := len(requestWriteReReference(name).FindAllStringIndex(content, -1))
		if got := counts[strings.ToLower(name)]; got != want {
			t.Errorf("request.%s: %d writes, the regex finds %d", name, got, want)
		}
	}
}

// TestALazyRequestFieldIsWhatItsApplicationStores: FW/1's Application.cfc
// creates request._framework_one in _get_framework_one() and returns it, and
// each of the corpus's five applications writes it in its own Application.cfc,
// one as its own subclass, so no file was the only writer and every call on
// what it returned was reported. A request runs its own application's
// Application.cfc, so another application's writes, in an Application.cfc
// neither extending the other, are not this request's. Every other writer must
// store the same component in statements its plan reads; a write inside a
// comment writes nothing.
func TestALazyRequestFieldIsWhatItsApplicationStores(t *testing.T) {
	app := func(construct string) string {
		return `component {
	function _get() {
		if ( !structKeyExists( request, "_fw" ) ) {
			// request._fw = new lib.Other();
			request._fw = ` + construct + `;
		}
		return request._fw;
	}
}`
	}

	files := map[string]string{
		"lib/Framework.cfc":   `component { function run() {} }`,
		"lib/Other.cfc":       `component {}`,
		"one/Application.cfc": app("new lib.Framework()"),
		"two/Application.cfc": app("new lib.Other()"),
		"one/Helper.cfc":      app("new lib.Framework( { trace = true } )"),
		"lib/MyApp.cfc":       `component { /* request._fw = new lib.Other(); */ }`,
	}

	returns := func(files map[string]string) string {
		dir := t.TempDir()
		writeFiles(t, dir, files)

		r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}, InferArgsFiles: cfmlFilesIn(t, dir)}

		return r.ReturnComponentOf(r.ResolveFunc(filepath.Join(dir, "one", "Application.cfc"), "_get", dir))
	}

	if got := returns(files); filepath.Base(got) != "Framework.cfc" {
		t.Errorf("another application's write beside one that agrees: %q, want Framework.cfc", got)
	}

	files["one/Helper.cfc"] = app("new lib.Other()")
	if got := returns(files); got != "" {
		t.Errorf("a writer in the application storing something else: %q, want nothing", got)
	}

	files["one/Helper.cfc"] = app("new lib.Framework()")
	files["two/Application.cfc"] = strings.Replace(app("new lib.Other()"), "component {", `component extends="one.Application" {`, 1)

	if got := returns(files); got != "" {
		t.Errorf("an application extending this one, storing something else: %q, want nothing", got)
	}

	files["two/Application.cfc"] = app("new lib.Other()")
	files["three/App.cfc"] = app("makeFramework()")

	if got := returns(files); got != "" {
		t.Errorf("a writer storing an untyped value: %q, want nothing", got)
	}

	delete(files, "three/App.cfc")

	files["page.cfm"] = `<cfset request._fw = new lib.Framework()>`
	if got := returns(files); got != "" {
		t.Errorf("a writer no plan reads: %q, want nothing", got)
	}
}
