package resolve

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

// The files are read in parallel, and a tag's sites still come out in the
// order of the file list, which a report diffed against an earlier one needs.
func TestCustomTagSitesKeepTheFileOrder(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{"tags/widget.cfm": `<cfoutput>#attributes.svc.go()#</cfoutput>`}
	paths := []string{}

	// The first files are large and the rest small, so workers finish the
	// later ones first: a merge in the order they finish would not keep the
	// file order.
	filler := strings.Repeat("<p>text</p>\n", 60000)

	for i := range 40 {
		name := fmt.Sprintf("p%02d.cfm", i)
		files[name] = `<cf_widget svc="#x#">`

		if i < 8 {
			files[name] = filler + files[name]
		}

		paths = append(paths, filepath.Join(dir, name))
	}

	writeFiles(t, dir, files)

	r := &Resolver{FS: vfs.OS{}, Index: index.New(), InferArgsFiles: paths}

	sites, _ := r.customTagSites()
	if len(sites["widget"]) != len(paths) {
		t.Fatalf("found %d widget calls, want %d", len(sites["widget"]), len(paths))
	}

	for i, s := range sites["widget"] {
		if s.file != paths[i] {
			t.Fatalf("site %d is in %s, want %s: the merge did not keep the file order", i, s.file, paths[i])
		}
	}
}
