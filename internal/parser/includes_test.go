package parser

import (
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
