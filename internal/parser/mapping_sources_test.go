package parser

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The Application.cfc walk follows the includes IncludeSites reads, so an
// include in a comment, a string or a page's prose runs nothing, nor does one in
// a method a constructor only declares, and a real one runs where it stands.
// The walk's tokens already stepped over comments and strings; the prose and
// the HTML comment were read as includes before it used IncludeSites.
func TestApplicationSourcesFollowOnlyLiveIncludes(t *testing.T) {
	root := filepath.FromSlash("/app")
	files := map[string]string{
		"Application.cfc": `component {
	// include "lineComment.cfm";
	/* include "blockComment.cfm"; */
	this.note = "include 'inString.cfm'";
	function onApplicationStart() { include "inMethod.cfm"; }
	include "script.cfm";
	this.mappings["/after"] = expandPath("./after");
}`,
		"script.cfm": `<cfset this.mappings["/script"] = expandPath("./script")>
<p>Remember to include "prose.cfm" before you deploy.</p>
<!-- include "htmlComment.cfm" -->
<cfset this.note = 'Don''t include "doubled.cfm"'>
<!--- <cfinclude template="tagComment.cfm"> --->
<cffunction name="f"><cfinclude template="inTagMethod.cfm"></cffunction>
<cfinclude template="tag.cfm">`,
		"tag.cfm": `<cfset this.mappings["/tag"] = expandPath("./tag")>`,
	}

	for _, dead := range []string{"lineComment", "blockComment", "inString", "prose", "tagComment", "htmlComment", "doubled", "inMethod", "inTagMethod"} {
		files[dead+".cfm"] = `<cfset this.mappings["/` + dead + `"] = expandPath("./` + dead + `")>`
	}

	var read []string

	got := ApplicationSources(filepath.Join(root, "Application.cfc"), nil, func(file string) ([]byte, error) {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return nil, err
		}

		content, ok := files[filepath.ToSlash(rel)]
		if !ok {
			return nil, os.ErrNotExist
		}

		read = append(read, filepath.ToSlash(rel))

		return []byte(content), nil
	}, nil).Mappings

	if want := []string{"Application.cfc", "script.cfm", "tag.cfm"}; !slices.Equal(read, want) {
		t.Errorf("read %v, want %v", read, want)
	}

	for _, live := range []string{"script", "tag", "after"} {
		if got[live] != filepath.Join(root, live) {
			t.Errorf("mapping %q = %q, want %q", live, got[live], filepath.Join(root, live))
		}
	}

	for _, dead := range []string{"lineComment", "blockComment", "inString", "prose", "tagComment", "htmlComment", "doubled", "inMethod", "inTagMethod"} {
		if _, ok := got[dead]; ok {
			t.Errorf("mapping %q came from an include that does not run", dead)
		}
	}
}

// The walk skips the methods the parse finds, whole, and nothing else. A
// method's braces are the parse's, so one in a string or a comment neither
// ends the body early nor runs it on; a declaration with no body ends at its
// own `;` or `/>`; and the word "function" in a page's prose declares
// nothing. The walk's own skipper read a body-less declaration on to the next
// method's closing brace, and prose on to the end of the page.
func TestApplicationSourcesSkipOnlyTheParsesMethods(t *testing.T) {
	root := filepath.FromSlash("/app")
	files := map[string]string{
		"Application.cfc": `component {
	this.mappings["/first"] = expandPath("./first");
	function onError( exception, event );
	this.mappings["/afterDeclaration"] = expandPath("./afterDeclaration");
	function onApplicationStart() {
		var note = "} in a string"; // } in a comment
		/* } */ var s = "#f( "}" )#";
		this.mappings["/inMethod"] = expandPath("./inMethod");
	}
	this.note = "function g() {";
	include "page.cfm";
	this.mappings["/last"] = expandPath("./last");
}`,
		"page.cfm": `<p>This function sets the mappings.</p>
<cfset this.mappings["/prose"] = expandPath("./prose")>
<cffunction name="declared" />
<cfset this.mappings["/afterTag"] = expandPath("./afterTag")>
<cffunction name="h"><cfset x = "<!--- } --->"><cfset this.mappings["/inTagMethod"] = expandPath("./inTagMethod")></cffunction>
<cfset this.mappings["/afterTagMethod"] = expandPath("./afterTagMethod")>`,
	}

	got := ApplicationSources(filepath.Join(root, "Application.cfc"), nil, func(file string) ([]byte, error) {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return nil, err
		}

		content, ok := files[filepath.ToSlash(rel)]
		if !ok {
			return nil, os.ErrNotExist
		}

		return []byte(content), nil
	}, nil).Mappings

	for _, live := range []string{"first", "afterDeclaration", "prose", "afterTag", "afterTagMethod", "last"} {
		if got[strings.ToLower(live)] != filepath.Join(root, live) {
			t.Errorf("mapping %q = %q, want %q", live, got[strings.ToLower(live)], filepath.Join(root, live))
		}
	}

	for _, dead := range []string{"inMethod", "inTagMethod"} {
		if _, ok := got[strings.ToLower(dead)]; ok {
			t.Errorf("mapping %q came from a method's body", dead)
		}
	}
}

// A mappings struct, a custom tag path array and a path function's arguments
// are split where the script scanner reads a comma, bracket or brace: none in
// a comment, a string or a quoted argument inside a string's #...# counts.
func TestApplicationSourcesSplitExpressionsAtCode(t *testing.T) {
	root := filepath.FromSlash("/app")
	content := `component {
	this.mappings = { "/note": "#lCase( "}" )#", "/a": expandPath( "./a" ) /* }, "/x": "y" */, "/b": expandPath( "./b" ) };
	this.customTagPaths = [ expandPath( "./tags" ) /* , "x" */, expandPath( "./more" ) ];
	this.mappings["/c"] = left( expandPath( "./c" ) /* , 1 */, len( expandPath( "./c" ) ) );
}`

	got := ApplicationSources(filepath.Join(root, "Application.cfc"), nil, func(string) ([]byte, error) {
		return []byte(content), nil
	}, nil)

	for _, name := range []string{"a", "b", "c"} {
		if got.Mappings[name] != filepath.Join(root, name) {
			t.Errorf("mapping %q = %q, want %q", name, got.Mappings[name], filepath.Join(root, name))
		}
	}

	if want := []string{filepath.Join(root, "tags"), filepath.Join(root, "more")}; !slices.Equal(got.CustomTagPaths, want) {
		t.Errorf("custom tag paths %q, want %q", got.CustomTagPaths, want)
	}
}

// The mapping walk keeps one reading of each text, and a text read for
// another kind of file is a reading of its own.
func TestMappingFactsKeepOneReadingPerText(t *testing.T) {
	var facts MappingFacts

	text := "component { function onRequest() {} }"
	first := facts.fact("/a/Application.cfc", text)

	if facts.fact("/b/Application.cfc", string([]byte(text))) != first {
		t.Fatal("the same text in another file was read again")
	}

	if facts.fact("/a/Application.cfm", text) == first {
		t.Error("a .cfm reading shares a .cfc one, though the extension decides how a page is read")
	}

	if spans := first.functionSpans("/a/Application.cfc"); len(spans) != 1 || spans[0].Name != "onRequest" {
		t.Errorf("spans %+v, want onRequest", spans)
	}

	facts.Clear()

	if facts.fact("/a/Application.cfc", text) == first {
		t.Error("Clear kept the reading")
	}
}
