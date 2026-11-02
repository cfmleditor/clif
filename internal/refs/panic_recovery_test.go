package refs

import (
	"os"
	"path/filepath"
	"testing"

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

// A panic on one file ends neither the search nor the process it runs in
// (textDocument/references searches inside the language server).
func TestAReferenceSearchSurvivesAPanicOnOneFile(t *testing.T) {
	dir := t.TempDir()
	good, bad := filepath.Join(dir, "good.cfm"), filepath.Join(dir, "bad.cfm")

	for _, f := range []string{good, bad} {
		if err := os.WriteFile(f, []byte(`<cfset svc.GetReport()>`), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	fsys := panicReadFS{FS: vfs.OS{}, bad: "bad.cfm"}
	entries := findInFiles(fsys, []string{good, bad}, &Options{FuncName: "GetReport", Resolver: testResolver(dir)})

	if len(entries) != 1 || entries[0].File != good {
		t.Fatalf("entries = %v, want the one in good.cfm", entries)
	}
}
