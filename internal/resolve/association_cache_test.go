package resolve

import (
	"path/filepath"
	"testing"
)

func TestAssociationLookupReusesSource(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Post.cfc":  "component {\nfunction config(){ belongsTo('author', modelName='User'); belongsTo('author', modelName='Other'); }\n}",
		"Empty.cfc": "component {}",
	})

	cfs := newCountingFS()
	r := &Resolver{FS: cfs}
	path := filepath.Join(dir, "Post.cfc")
	empty := filepath.Join(dir, "Empty.cfc")

	for range 100 {
		def := r.associationFunc(path, "AUTHOR")
		if def == nil || def.Name != "AUTHOR" || def.ReturnComponent != "User" || def.Line != 1 {
			t.Fatalf("association = %+v", def)
		}

		def.ReturnComponent = "Mutated"

		if r.associationFunc(path, "missing") != nil || r.associationFunc(empty, "missing") != nil {
			t.Fatal("invented association")
		}
	}

	for _, file := range []string{path, empty} {
		if cfs.count(file) != 1 {
			t.Errorf("%s read %d times; want once", file, cfs.count(file))
		}
	}
	// Different-sized edits must invalidate the shared lexical cache.
	writeFiles(t, dir, map[string]string{"Post.cfc": "component { function config(){ belongsTo('editor', modelName='Editor'); } }"})

	if r.associationFunc(path, "author") != nil {
		t.Fatal("stale association survived edit")
	}

	if def := r.associationFunc(path, "editor"); def == nil || def.ReturnComponent != "Editor" {
		t.Fatalf("edited association = %+v", def)
	}
}
