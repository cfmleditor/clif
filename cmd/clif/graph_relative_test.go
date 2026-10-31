package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/codemap"
)

// TestGraphOfARelativePathKeepsItsCallEdges: with no config the paths as typed
// were the scan roots, so `graph .` indexed every file under a relative URI and
// each resolved call named a target that relPath turned into `../../x.cfc`. The
// edge pointed at no node and was dropped.
func TestGraphOfARelativePathKeepsItsCallEdges(t *testing.T) {
	root := t.TempDir()

	for name, content := range map[string]string{
		"shared/Svc.cfc": "component { function search() { return []; } }\n",
		"Caller.cfc":     "component { function go() { var s = new shared.Svc(); return s.search(); } }\n",
	} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	t.Chdir(root)

	m, err := buildGraph(&graphFlags{paths: []string{"."}, quiet: true})
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range m.Edges {
		if e.Kind == codemap.EdgeCalls && e.From == "Caller.cfc::go" && e.To == "shared/Svc.cfc::search" {
			return
		}
	}

	t.Fatalf("no call edge from go to search; edges: %+v", m.Edges)
}
