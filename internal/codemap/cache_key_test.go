package codemap_test

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/cfmleditor/clif/internal/codemap"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
)

type memCache struct {
	mu sync.Mutex
	m  map[string]*codemap.FileGraph
}

func (c *memCache) Load(fingerprint, hash string) (*codemap.FileGraph, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	g, ok := c.m[fingerprint+"\x00"+hash]
	if !ok {
		return nil, false
	}

	cp := *g

	return &cp, true
}

func (c *memCache) Save(fingerprint, hash string, g *codemap.FileGraph) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.m[fingerprint+"\x00"+hash] = g

	return nil
}

// TestIdenticalFilesAtTwoPathsAreBothInACachedMap: a cached build served one
// byte-identical file's graph for the other, so the second dropped out of the map
// with its edges (#306).
func TestIdenticalFilesAtTwoPathsAreBothInACachedMap(t *testing.T) {
	root := t.TempDir()

	var files []string

	for _, rel := range []string{"appA/auth/callback.cfm", "appB/auth/callback.cfm"} {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte("<cfscript>writeOutput(now());</cfscript>\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, p)
	}

	cache := &memCache{m: map[string]*codemap.FileGraph{}}

	for run := range 2 {
		fsys := vfs.OS{}
		m := codemap.Build(&codemap.Options{
			Root: root, Files: files, FS: fsys, Workers: 1, Cache: cache,
			Resolver: &resolve.Resolver{FS: fsys, Index: index.New(), WorkspaceFolders: []string{root}},
		})

		var got []string

		for _, n := range m.Nodes {
			if n.Kind == codemap.KindFile {
				got = append(got, n.ID)
			}
		}

		slices.Sort(got)

		if want := []string{"appA/auth/callback.cfm", "appB/auth/callback.cfm"}; !slices.Equal(got, want) {
			t.Fatalf("run %d: file nodes %v, want %v", run, got, want)
		}
	}
}
