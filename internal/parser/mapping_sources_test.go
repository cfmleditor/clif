package parser

import (
	"os"
	"path/filepath"
	"slices"
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
