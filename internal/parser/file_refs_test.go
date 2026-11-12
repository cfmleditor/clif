package parser

import (
	"slices"
	"testing"
)

// A closing tag left without its '>' ends where the next tag opens, and a
// `</` with no name is a stray '<': the reading of paths stepped over both to
// the next '>', and lost the tag after them. Custom tags were read the right
// way already; one walk reads both now.
func TestAClosingTagDoesNotSwallowTheTagAfterIt(t *testing.T) {
	src := "<cfoutput>\n<p>x</p\n<cfinclude template=\"a.cfm\">\n</<cfinclude template=\"b.cfm\">\n</cfoutput>\n"

	if got, want := ExtractIncludes(src), []string{"a.cfm", "b.cfm"}; !slices.Equal(got, want) {
		t.Errorf("includes = %v, want %v", got, want)
	}

	var paths []string
	for _, l := range ExtractLinks(src) {
		paths = append(paths, l.Path)
	}

	if want := []string{"a.cfm", "b.cfm"}; !slices.Equal(paths, want) {
		t.Errorf("links = %v, want %v", paths, want)
	}
}

// A parse records includes as its walk goes, wherever they stand: inside a
// function body as outside, in script and in tags, and an include read
// through the walk's lookahead is recorded once.
func TestAParseRecordsItsIncludes(t *testing.T) {
	src := `component {
	include "top.cfm";
	function f() {
		if ( x ) include "inner.cfm";
		cfinclude( template = "call.cfm" );
		arr.include( "member.cfm" );
		// include "comment.cfm";
		var s = 'include "string.cfm"';
	}
	include template="lucee.cfm";
}
`
	pr := ParseWithOptions("file:///t/a.cfc", src, nil)

	var got []string
	for _, s := range pr.IncludeSites() {
		got = append(got, s.Path)
	}

	if want := []string{"top.cfm", "inner.cfm", "call.cfm", "lucee.cfm"}; !slices.Equal(got, want) {
		t.Errorf("IncludeSites = %v, want %v", got, want)
	}

	if !slices.Equal(got, slices.Collect(func(yield func(string) bool) {
		for _, s := range IncludeSites(src) {
			if !yield(s.Path) {
				return
			}
		}
	})) {
		t.Errorf("a parse's includes differ from the text reading's")
	}
}

// A parse that records links answers links, custom tags and includes from
// its record; an edit inside a function changes the content, and the next
// question reads the edited text.
func TestAParsesLinksFollowAnEdit(t *testing.T) {
	src := "component {\n\tfunction f() {\n\t\tcf_box( title = \"x\" );\n\t}\n}\n"
	pr := ParseWithOptions("file:///t/a.cfc", src, &ParseOptions{ExtractLinks: true})

	if calls := pr.CustomTagCalls(nil); len(calls) != 1 || calls[0].Name != "box" {
		t.Fatalf("CustomTagCalls = %+v, want one call of box", calls)
	}

	// Inside f's body: an in-function edit.
	if kind := pr.ApplyEdit(2, 0, 2, 0, "\t\tinclude \"added.cfm\";\n"); kind != EditInFunc {
		t.Fatalf("edit kind = %v, want EditInFunc", kind)
	}

	if got := pr.Includes(); !slices.Equal(got, []string{"added.cfm"}) {
		t.Errorf("Includes after the edit = %v, want [added.cfm]", got)
	}

	links := pr.DocumentLinks()
	if len(links) != 1 || links[0].Path != "added.cfm" || links[0].Line != 2 {
		t.Errorf("DocumentLinks after the edit = %+v, want added.cfm on line 2", links)
	}

	if calls := pr.CustomTagCalls(nil); len(calls) != 1 || calls[0].Line != 3 {
		t.Errorf("CustomTagCalls after the edit = %+v, want box on line 3", calls)
	}
}

// The tags a configured prefix makes custom tags come in the order the walk
// met them, among the other custom tag calls.
func TestPrefixedTagsKeepTheirPlace(t *testing.T) {
	src := "<cf_a>\n<cfx>\n<cfmodule template=\"m.cfm\">\n<cfy></cfy>\n"

	var got []string
	for _, c := range CustomTagCalls(src, []string{"cf"}) {
		got = append(got, c.Name)
	}

	if want := []string{"a", "x", "m.cfm", "y", "y"}; !slices.Equal(got, want) {
		t.Errorf("CustomTagCalls = %v, want %v", got, want)
	}
}
