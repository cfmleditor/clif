package codemap_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/codemap"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
)

// A call on a receiver nothing typed, by the name of a member function no
// component declares, is a built-in, and is decided once every file has been
// scanned (resolve.ClassifyOptions.DeferCounts). The map counts it, and the
// cache stores the file with it counted: a file stored before its pending
// calls were settled would serve a later build a graph missing them.
func TestAPendingCallIsCountedAndStoredSettled(t *testing.T) {
	root := t.TempDir()
	page := filepath.Join(root, "page.cfm")

	if err := os.WriteFile(page, []byte("<cfscript>x.trim();</cfscript>\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cache := &jsonCache{m: map[string][]byte{}}
	build := func() *codemap.Map {
		fsys := vfs.OS{}

		return codemap.Build(&codemap.Options{
			Root: root, Files: []string{page}, FS: fsys, Workers: 1, Cache: cache,
			Resolver: &resolve.Resolver{FS: fsys, Index: index.New(), WorkspaceFolders: []string{root}},
		})
	}

	first := build()
	if first.Stats.CallSites != 1 || first.Stats.Builtin != 1 || first.Stats.Unresolved != 0 {
		t.Fatalf("stats = %+v, want one call site, set aside as a built-in", first.Stats)
	}

	if len(cache.m) != 1 {
		t.Fatalf("%d files stored, want 1", len(cache.m))
	}

	for _, raw := range cache.m {
		var g codemap.FileGraph
		if err := json.Unmarshal(raw, &g); err != nil {
			t.Fatal(err)
		}

		if g.Stats.Builtin != 1 {
			t.Errorf("the cached graph has %d built-ins, want 1: it was stored before its pending call was settled", g.Stats.Builtin)
		}
	}

	if again := build(); again.Stats.Builtin != 1 || again.Stats.Cached != 1 {
		t.Errorf("a build from the cache has stats %+v, want the built-in counted and the file cached", again.Stats)
	}
}

// jsonCache stores a file's graph as the store does: serialised when saved, so
// a later change to the graph in memory does not reach it.
type jsonCache struct {
	m map[string][]byte
}

func (c *jsonCache) Load(fingerprint, hash string) (*codemap.FileGraph, bool) {
	raw, ok := c.m[fingerprint+"\x00"+hash]
	if !ok {
		return nil, false
	}

	var g codemap.FileGraph
	if err := json.Unmarshal(raw, &g); err != nil {
		return nil, false
	}

	return &g, true
}

func (c *jsonCache) Save(fingerprint, hash string, g *codemap.FileGraph) error {
	raw, err := json.Marshal(g)
	if err != nil {
		return err
	}

	c.m[fingerprint+"\x00"+hash] = raw

	return nil
}
