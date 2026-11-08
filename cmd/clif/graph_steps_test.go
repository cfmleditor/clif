package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/codemap"
	"github.com/cfmleditor/clif/internal/codemap/store"
)

// graphWorkspace writes a two-component workspace and returns its root.
func graphWorkspace(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	for name, content := range map[string]string{
		"Svc.cfc":    "component { function search() { return []; } }\n",
		"Caller.cfc": "component { function go() { var s = new Svc(); return s.search(); } }\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

// A path may name a file: the map is then rooted at its directory but every
// file under it is still scanned.
func TestAGraphOfAFileIsRootedAtItsDirectory(t *testing.T) {
	root := graphWorkspace(t)

	m, err := buildGraph(&graphFlags{paths: []string{filepath.Join(root, "Caller.cfc")}, quiet: true})
	if err != nil {
		t.Fatal(err)
	}

	if m.Root != root {
		t.Errorf("root = %q, want the file's directory %q", m.Root, root)
	}
}

// A path that holds no CFML is an error that names where it looked.
func TestAGraphOfNoCFMLIsAnError(t *testing.T) {
	_, err := buildGraph(&graphFlags{paths: []string{t.TempDir()}, quiet: true})
	if err == nil {
		t.Fatal("a directory with no CFML built a map")
	}
}

// --db saves the whole map before any view narrows it, and a second run with
// the same files reads them from the cache it saved.
func TestAGraphSavedToADatabaseIsWholeAndCachesTheNextRun(t *testing.T) {
	root := graphWorkspace(t)
	dbPath := filepath.Join(t.TempDir(), "map.db")

	first, err := buildGraph(&graphFlags{paths: []string{root}, quiet: true, db: dbPath})
	if err != nil {
		t.Fatal(err)
	}

	db, err := store.OpenReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	if _, nodeErr := db.Node("Caller.cfc::go"); nodeErr != nil {
		t.Errorf("the saved map has no go function: %v", nodeErr)
	}

	if _, nodeErr := db.Node("Svc.cfc::search"); nodeErr != nil {
		t.Errorf("the saved map has no search function: %v", nodeErr)
	}

	callees, err := db.Callees("Caller.cfc::go", 10)
	_ = db.Close()

	if err != nil || len(callees) == 0 {
		t.Errorf("the saved map has no call from go (%d callees, %v)", len(callees), err)
	}

	// The second run is the same map, whether or not it used the cache.
	second, err := buildGraph(&graphFlags{paths: []string{root}, quiet: true, db: dbPath})
	if err != nil {
		t.Fatal(err)
	}

	if len(second.Edges) != len(first.Edges) || len(second.Nodes) != len(first.Nodes) {
		t.Errorf("a cached run built %d nodes and %d edges, the first %d and %d", len(second.Nodes), len(second.Edges), len(first.Nodes), len(first.Edges))
	}

	var calls int

	for _, e := range second.Edges {
		if e.Kind == codemap.EdgeCalls {
			calls++
		}
	}

	if calls == 0 {
		t.Error("the cached run lost its call edge")
	}
}

// The config's entry and utility globs are merged in front of the command
// line's, on the caller's flags.
func TestTheConfigsGlobsAreMergedIntoTheFlags(t *testing.T) {
	root := graphWorkspace(t)

	cfg := `{"workspaceName":"g","codemap":{"entry":["Caller.cfc"],"utility":["Svc.cfc"],"hideUtility":true}}`
	if err := os.WriteFile(filepath.Join(root, ".clif.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	f := &graphFlags{paths: []string{root}, quiet: true, entryGlobs: []string{"extra"}}
	if _, err := buildGraph(f); err != nil {
		t.Fatal(err)
	}

	if len(f.entryGlobs) != 2 || f.entryGlobs[0] != "Caller.cfc" || f.entryGlobs[1] != "extra" {
		t.Errorf("entry globs = %v, want the config's first then the flag's", f.entryGlobs)
	}

	if len(f.utilGlobs) != 1 || f.utilGlobs[0] != "Svc.cfc" || !f.hideUtility {
		t.Errorf("utility = %v hide=%v, want the config's", f.utilGlobs, f.hideUtility)
	}
}
