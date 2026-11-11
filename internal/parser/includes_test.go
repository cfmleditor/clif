package parser

import (
	"maps"
	"slices"
	"testing"
)

func TestExtractIncludes(t *testing.T) {
	content := `<cfcomponent>
	<cfinclude template="api-shared.cfm" />
	<cfinclude template = '/tassweb/packages/tass/core/error.cfm'>
	<cfinclude template="#dynamicDir#/page.cfm">
	<cfinclude template="#dynamicPage#.cfm">
	<cfinclude template="API-SHARED.cfm">
	<cfmodule template="module.cfm">
	<a href="page.cfm">x</a>
	<!--- <cfinclude template="commented.cfm"> <!--- nested ---> still comment --->
	<cfscript>
		include "script-form.cfm";
		include template="script-named.cfm";
		cfinclude(template="script-call.cfm");
		arr.include("member.cfm");
		include "not-cfml.js";
	</cfscript>
</cfcomponent>`

	want := []string{
		"api-shared.cfm",
		"/tassweb/packages/tass/core/error.cfm",
		"./*.cfm",
		"script-form.cfm",
		"script-named.cfm",
		"script-call.cfm",
	}

	if got := ExtractIncludes(content); !slices.Equal(got, want) {
		t.Errorf("ExtractIncludes:\n got  %q\n want %q", got, want)
	}
}

func TestExtractIncludesAfterAComment(t *testing.T) {
	content := "<!--- a\nb ---><cfinclude template=\"after.cfm\">"
	if got := ExtractIncludes(content); !slices.Equal(got, []string{"after.cfm"}) {
		t.Errorf("got %q", got)
	}
}

// TestAComputedIncludeOfAListedNameOrATagAttribute: a suite that lists its
// spec directory and includes each name, `specs/#file#`, includes every
// template there, and only beside a non-recursive listing of that directory
// for templates; a custom tag's `#attributes.file#` is reported in one
// spelling for the resolver to read from the tag's callers.
func TestAComputedIncludeOfAListedNameOrATagAttribute(t *testing.T) {
	const listing = `var specs = DirectoryList( ExpandPath( "specs" ), false, "name", "*.cfm" );`

	for _, tc := range []struct {
		name, source string
		want         []string
	}{
		{"listed", listing + ` include "specs/#file#";`, []string{"specs/*.cfm"}},
		{"listed with a slash", `DirectoryList( "./specs/", false, "name", "*.cfm" ); include "./specs/#file#";`, []string{"./specs/*.cfm"}},
		{"not listed", `include "specs/#file#";`, nil},
		{"another directory listed", listing + ` include "other/#file#";`, nil},
		{"recursive", `DirectoryList( ExpandPath( "specs" ), true, "name", "*.cfm" ); include "specs/#file#";`, nil},
		{"any file listed", `DirectoryList( ExpandPath( "specs" ), false, "name" ); include "specs/#file#";`, nil},
		{"no directory", listing + ` include "#file#";`, nil},
		{"tag attribute", `<cfinclude template="#attributes.File#">`, []string{"#attributes.file#"}},
		{"tag attribute path", `<cfinclude template="#attributes.a.b#">`, nil},
		{"tag attribute with a directory", `<cfinclude template="x/#attributes.file#">`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractIncludes(tc.source); !slices.Equal(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// A path that begins with one plain variable and a slash is read, as the path
// from the application's root; a call, a variable with no slash after it and a
// second span are not.
func TestAnIncludeThroughARootVariableIsRead(t *testing.T) {
	src := `<cfinclude template="#application.baseUrl#/lib/a.cfm">` + "\n" +
		`<cfinclude template="#getBase()#/lib/b.cfm">` + "\n" +
		`<cfinclude template="#application.baseUrl#lib/c.cfm">` + "\n" +
		`<cfinclude template="#application.baseUrl#/lib/#page#.cfm">` + "\n" +
		`<cfinclude template="/lib/d.cfm">` + "\n"

	got := ExtractIncludes(src)
	if len(got) != 2 || got[0] != "#application.baseUrl#/lib/a.cfm" || got[1] != "/lib/d.cfm" {
		t.Errorf("includes = %q", got)
	}
}

// An include is read where the file's syntax makes it one: not in a script
// comment, a string or a page's prose, which the scan for the keyword read as
// live. One written in a script component, as a tag or a statement, is.
func TestAnIncludeIsReadOnlyWhereItIsAStatement(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         []string
	}{
		{"line comment", "<cfscript>\n// include \"a.cfm\";\n</cfscript>", nil},
		{"block comment", "<cfscript>\n/* include \"b.cfm\"; */\n</cfscript>", nil},
		{"string", "<cfscript>\nx = 'please include \"c.cfm\" here';\n</cfscript>", nil},
		{"prose", `<p>Remember to include "d.cfm" in your page</p>`, nil},
		{"in a cfset", `<cfset include = "e.cfm">`, nil},
		{"script component", "component {\nfunction f() {\n// include \"f.cfm\";\ninclude \"g.cfm\";\n}\n}", []string{"g.cfm"}},
		{"tag in a script component", "component {\nfunction f() {\n<cfinclude template=\"h.cfm\">\n}\n}", []string{"h.cfm"}},
		{"tagless template", "include \"i.cfm\";", []string{"i.cfm"}},
		{"a doubled quote", `<cfinclude template="j""k.cfm">`, nil},
		{"a '>' in another value", `<cfinclude x="a>b" template="l.cfm">`, []string{"l.cfm"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractIncludes(tc.source); !slices.Equal(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// Every include ExtractLinks links is one ExtractIncludes reads: the two are
// one walk, the links the literal paths of every kind.
func TestEveryLinkedIncludeIsAnInclude(t *testing.T) {
	src := "<cfinclude template=\"a.cfm\">\n<cfmodule template=\"m.cfm\">\n<a href=\"p.cfm\">x</a>\n" +
		"<cfscript>\ninclude \"b.cfm\";\ncfinclude(template=\"c.cfm\");\n// include \"no.cfm\";\n</cfscript>"

	var linked []string

	for _, l := range ExtractLinks(src) {
		linked = append(linked, l.Path)
	}

	if want := []string{"a.cfm", "m.cfm", "p.cfm", "b.cfm", "c.cfm"}; !slices.Equal(linked, want) {
		t.Fatalf("links %q, want %q", linked, want)
	}

	if got, want := ExtractIncludes(src), []string{"a.cfm", "b.cfm", "c.cfm"}; !slices.Equal(got, want) {
		t.Fatalf("includes %q, want %q", got, want)
	}
}

// A function that includes the template its argument names makes each
// unqualified call of it in the same file an include: Wheels' Global.cfc loads
// the routes through $include(template = "/config/routes.cfm"), which runs
// them in Global's own variables scope. A call through another object, a
// computed argument and an unrelated function are not.
func TestACallOfAnIncludeWrapperIsAnInclude(t *testing.T) {
	src := `component {
	public void function $include(required string template) {
		include "#LCase(arguments.template)#";
	}
	public void function loadRoutes() {
		$include(template = "/config/routes.cfm");
		$include("/config/positional.cfm");
		application.wo.$include(template = "/config/other.cfm");
		$include(template = path);
		other(template = "/config/no.cfm");
	}
}`

	if got, want := ExtractIncludes(src), []string{"/config/routes.cfm", "/config/positional.cfm"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// IncludeWrappers names each function that includes an argument, with the
// argument and its place: the fact both the include scan and the resolver's
// Wheels wrapper proof read.
func TestIncludeWrappersNameTheArgumentAndItsPlace(t *testing.T) {
	src := `component {
	function $include(required string template) { include "#LCase(arguments.template)#"; }
	function render(string layout = "", required string tpl) { include "#arguments.tpl#"; }
	function other(required string x) { include "fixed.cfm"; }
	function computed(required string x) { include "#arguments.x & '.cfm'#"; }
}`

	got := IncludeWrappers(src)

	want := map[string]IncludeWrapper{
		"$include": {Param: "template", Position: 0},
		"render":   {Param: "tpl", Position: 1},
	}
	if !maps.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
