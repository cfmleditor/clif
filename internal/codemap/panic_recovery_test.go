package codemap_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/codemap"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
)

type panicReadFS struct {
	vfs.FS
	bad string
}

func (p panicReadFS) ReadFile(path string) ([]byte, error) {
	if filepath.Base(path) == p.bad {
		panic("a bad file")
	}

	return p.FS.ReadFile(path)
}

func (p panicReadFS) ReadFileString(path string) (string, error) {
	data, err := p.ReadFile(path)

	return string(data), err
}

// A panic on one file ends neither the build nor the process it runs in (the
// editor's generateCodeMap builds inside the language server); the file is
// counted as failed and the rest of the map is built.
func TestAMapBuildSurvivesAPanicOnOneFile(t *testing.T) {
	root := t.TempDir()
	good, bad := filepath.Join(root, "Good.cfc"), filepath.Join(root, "Bad.cfc")

	for _, f := range []string{good, bad} {
		if err := os.WriteFile(f, []byte("component {\n\tfunction run() {}\n}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	fsys := panicReadFS{FS: vfs.OS{}, bad: "Bad.cfc"}
	m := codemap.Build(&codemap.Options{
		Root: root, Files: []string{good, bad}, FS: fsys, Workers: 2,
		Resolver: &resolve.Resolver{FS: fsys, Index: index.New(), WorkspaceFolders: []string{root}},
	})

	if m.Stats.Failed != 1 {
		t.Fatalf("failed = %d, want 1", m.Stats.Failed)
	}

	found := false

	for _, n := range m.Nodes {
		if n.ID == "Good.cfc::run" {
			found = true
		}
	}

	if !found {
		t.Fatal("the file that scanned is missing from the map")
	}
}
