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

// A batch scan's FuncLookup cache is the owner's, shared by every
// caller-context view, and each view resolves with its own application's
// mappings. A library with no Application.cfc does not have a test suite's
// `/root` mapping, so the answer it gives from the suite's directory must not
// be the one the suite's own view is then served: which asked first was the
// scan's goroutine order, and spreadsheet-cfml's chainable calls came and went
// with it.
func TestFuncLookupAnswersDoNotLeakBetweenApplications(t *testing.T) {
	dir := t.TempDir()
	lib := filepath.Join(dir, "lib")
	app := filepath.Join(dir, "test")
	specs := filepath.Join(app, "specs")

	files := map[string]string{
		filepath.Join(lib, "Lib.cfc"):         `component accessors="true" { property name="x"; }`,
		filepath.Join(app, "Application.cfc"): `component { this.mappings[ "/root" ] = getDirectoryFromPath( getCurrentTemplatePath() ) & "../lib/"; }`,
		filepath.Join(specs, "spec.cfm"):      `<cfset s = 1>`,
	}

	for path, src := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	want := filepath.Join(lib, "Lib.cfc")

	for _, libraryFirst := range []bool{false, true} {
		idx := index.New()
		idx.IndexFile(cfpath.ToURI(want), files[want])

		r := &Resolver{FS: vfs.OS{}, Index: idx, SnapshotScan: true, WorkspaceFolders: []string{dir}}

		if libraryFirst {
			r.forCaller(lib).FuncLookup(specs)("root.Lib", "setX")
		}

		if got := r.FuncLookup(specs)("root.Lib", "setX"); !cfpath.SamePath(got, want) {
			t.Errorf("library view asked first = %v: setX() from the suite returns %q, want %q", libraryFirst, got, want)
		}
	}
}
