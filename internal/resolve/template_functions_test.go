package resolve

import (
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/testutil"
	"github.com/cfmleditor/clif/internal/vfs"
)

// The names a template declares are the function spans' as they are the scan
// parse's: templateDeclares reads the spans and no longer parses each
// template in full, so over every template the fixtures hold the two name
// sets must be one. A component is left out: its parse adds the accessors its
// properties generate, which are in no span, and templateDeclares reads only
// .cfm and .cfml files.
func TestFunctionSpansNameWhatTheScanParseDeclares(t *testing.T) {
	r := NewResolver(vfs.OS{}, index.New(), &Settings{})
	files, withFuncs := 0, 0

	same := func(path, text string) {
		byParse := map[string]bool{}
		for _, f := range r.ParseFile(path, text, BatchSpec()).Funcs {
			byParse[strings.ToLower(f.Name)] = true
		}

		bySpans := map[string]bool{}
		for _, s := range r.functionSpansIn(path, text) {
			bySpans[strings.ToLower(s.Name)] = true
		}

		if !maps.Equal(byParse, bySpans) {
			t.Errorf("%s: the scan parse declares %v, the spans %v", path, byParse, bySpans)
		}

		files++

		if len(byParse) > 0 {
			withFuncs++
		}
	}

	same("a.cfm", "<cfscript>\nfunction parseXML( x ) { return x; }\nfunction other() {}\n</cfscript><p>x</p>")
	same("b.cfm", "<cffunction name=\"helper\"><cfreturn 1></cffunction><cffunction name=\"two\"></cffunction>")
	same("c.cfm", "<cfoutput>no functions here</cfoutput>")
	same("d.cfml", "<cfscript>f = function() { return 1; }; function g() {}</cfscript>")

	err := filepath.WalkDir(testutil.Testdata(), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		switch strings.ToLower(filepath.Ext(path)) {
		case ".cfm", ".cfml":
		default:
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		same(path, string(b))

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if files < 10 || withFuncs < 3 {
		t.Errorf("%d files, %d declaring functions: the fixtures do not reach the comparison", files, withFuncs)
	}
}
