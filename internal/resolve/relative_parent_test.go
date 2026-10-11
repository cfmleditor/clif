package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestExplicitRelativeInheritance(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Base.cfc":      `component {function inherited(){}}`,
		"a/Base.cfc":    `component {function sibling(){}}`,
		"a/Child.cfc":   `component extends="../Base" {function run(){inherited();sibling();}}`,
		"a/Local.cfc":   `component extends="./base" {function run(){sibling();inherited();}}`,
		"b/Base.cfc":    `component {function borrowed(){}}`,
		"a/Missing.cfc": `component extends="../../missing/Base" {function run(){borrowed();}}`,
	})
	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}, Mappings: map[string]string{"base": filepath.Join(dir, "b")}}
	expectReasons(t, reasonsWith(t, r, dir, "a/Child.cfc"), map[string]string{"inherited": "", "sibling": "not found in extends chain"})
	expectReasons(t, reasonsWith(t, r, dir, "a/Local.cfc"), map[string]string{"sibling": "", "inherited": "not found in extends chain"})

	if got := r.ResolveFunc(filepath.Join(dir, "a", "Missing.cfc"), "borrowed", dir); got != nil {
		t.Fatal("borrowed missing parent", got)
	}

	for _, path := range []string{"../base", "../Base.cfc", `..\Base`} {
		if got := r.ComponentPath(path, filepath.Join(dir, "a")); got != filepath.Join(dir, "Base.cfc") {
			t.Fatalf("%s: %s", path, got)
		}
	}
}
