package resolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

func funcCacheFixture(t *testing.T) (string, *index.Index) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "Base.cfc")
	src := `component { function save(){} }`

	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	idx := index.New()
	idx.IndexFile(cfpath.ToURI(path), src)

	return path, idx
}

// In the editor a lookup holds for one index generation: the edit that
// removes a method is an index write, and the next lookup must see it.
func TestALookupIsAskedAgainAfterAnIndexWrite(t *testing.T) {
	path, idx := funcCacheFixture(t)
	r := &Resolver{FS: vfs.OS{}, Index: idx, IndexTracksFiles: true, WorkspaceFolders: []string{filepath.Dir(path)}}

	if r.LookupFuncWithExtends(path, "SAVE") == nil || r.LookupFuncWithExtends(path, "save") == nil {
		t.Fatal("save() not found before the edit")
	}

	idx.IndexFile(cfpath.ToURI(path), `component { function load(){} }`)

	if d := r.LookupFuncWithExtends(path, "save"); d != nil {
		t.Errorf("save() still found after the index dropped it (line %d)", d.Line)
	}
}

// A batch scan's files do not change while it runs, so its answers are kept
// across the index writes its lazy indexing makes; this pins that the cache
// is in use there.
func TestABatchScanKeepsItsLookups(t *testing.T) {
	path, idx := funcCacheFixture(t)
	r := &Resolver{FS: vfs.OS{}, Index: idx, SnapshotScan: true, WorkspaceFolders: []string{filepath.Dir(path)}}

	if r.LookupFuncWithExtends(path, "save") == nil {
		t.Fatal("save() not found")
	}

	idx.IndexFile(cfpath.ToURI(path), `component { function load(){} }`)

	if r.LookupFuncWithExtends(path, "save") == nil {
		t.Error("a batch scan asked again after an index write")
	}
}
