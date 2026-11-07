package parser

import (
	"slices"
	"strings"
	"testing"
)

// TestADirectoryListingIncludeIsAGlob: Mura's configBean lists dbUpdates/*.cfm
// beside itself and includes each one it finds. Only that literal shape is a
// static answer.
func TestADirectoryListingIncludeIsAGlob(t *testing.T) {
	const listing = `<cfdirectory action="list" directory="#getDirectoryFromPath(getCurrentTemplatePath())#dbUpdates" name="rsUpdates" filter="*.cfm" sort="name asc">`

	const include = `<cfloop query="rsUpdates"><cfinclude template="dbUpdates/#rsUpdates.name#"></cfloop>`

	for _, tc := range []struct {
		name, source string
		want         []string
	}{
		{"listed and included", listing + include, []string{"dbUpdates/*.cfm"}},
		{"recursive listing", `<cfdirectory action="list" recurse="true" directory="#getDirectoryFromPath(getCurrentTemplatePath())#dbUpdates" name="rsUpdates" filter="*.cfm">` + include, nil},
		{"computed directory", `<cfdirectory action="list" directory="#variables.root#dbUpdates" name="rsUpdates" filter="*.cfm">` + include, nil},
		{"other filter", `<cfdirectory action="list" directory="#getDirectoryFromPath(getCurrentTemplatePath())#dbUpdates" name="rsUpdates" filter="*.txt">` + include, nil},
		{"include from another directory", listing + `<cfinclude template="other/#rsUpdates.name#">`, nil},
		{"include of another query", listing + `<cfinclude template="dbUpdates/#rsOther.name#">`, nil},
		{"listing in a comment", `<!--- ` + listing + ` --->` + include, nil},
		{"parent directory", `<cfdirectory action="list" directory="#getDirectoryFromPath(getCurrentTemplatePath())#../x" name="rsUpdates" filter="*.cfm"><cfinclude template="../x/#rsUpdates.name#">`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractIncludes(tc.source); !slices.Equal(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestIncludeSitesSayWhereEachIncludeIs: one site per statement, repeats
// included, each at the offset its statement starts, the directory listing's
// glob at its <cfinclude>. ExtractIncludes is the same paths without repeats.
func TestIncludeSitesSayWhereEachIncludeIs(t *testing.T) {
	src := `<cfinclude template="a.cfm">x<cfscript>include "a.cfm";</cfscript>` +
		`<cfdirectory action="list" directory="#getDirectoryFromPath(getCurrentTemplatePath())#up" name="q" filter="*.cfm">` +
		`<cfinclude template="up/#q.name#">`

	got := IncludeSites(src)

	want := []IncludeSite{
		{Path: "a.cfm", Offset: 0},
		{Path: "a.cfm", Offset: strings.Index(src, `include "a.cfm"`)},
		{Path: "up/*.cfm", Offset: strings.Index(src, `<cfinclude template="up/`)},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	if paths := ExtractIncludes(src); !slices.Equal(paths, []string{"a.cfm", "up/*.cfm"}) {
		t.Fatalf("ExtractIncludes: %q", paths)
	}
}

// A <cfdirectory> listing of *.cfm beside an include of `dir/#file#` makes the
// include every template in dir, when the file spells that directory.
func TestACfdirectoryListingMakesAComputedIncludeAGlob(t *testing.T) {
	src := `<cfcomponent>
<cfset variables.updates = variables.root & "installer/updates/">
<cffunction name="run">
<cfdirectory action="list" directory="#variables.updates#" name="scripts" filter="*.cfm">
<cfinclude template="../../installer/updates/#migration.file#">
</cffunction>
</cfcomponent>`

	if got := ExtractIncludes(src); !slices.Equal(got, []string{"../../installer/updates/*.cfm"}) {
		t.Errorf("got %v", got)
	}

	if got := ExtractIncludes(strings.Replace(src, `filter="*.cfm"`, `filter="*.txt"`, 1)); len(got) != 0 {
		t.Errorf("a listing of other files made a glob: %v", got)
	}
}
