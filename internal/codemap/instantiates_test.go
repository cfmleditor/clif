package codemap_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/cfmleditor/clif/internal/codemap"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestInstantiatesEdgesComeFromConstructions: the edge came from the file's
// component-level refs alone, so a local or inline `new` drew none (#308).
// The refs still draw theirs, and a construction a ref already drew is
// counted once.
func TestInstantiatesEdgesComeFromConstructions(t *testing.T) {
	root := t.TempDir()

	var files []string

	for rel, src := range map[string]string{
		"shared/Svc.cfc":     "component { function handle() {} }\n",
		"shared/Factory.cfc": "component { shared.Svc function make() { return new shared.Svc(); } }\n",
		"Caller.cfc": "component {\n" +
			"function local() { var s = new shared.Svc(); s.handle(); }\n" +
			"function inline() { new shared.Svc().handle(); }\n" +
			"function typed() { variables.f = new shared.Factory(); variables.made = variables.f.make(); }\n" +
			"}\n",
	} {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, p)
	}

	fsys := vfs.OS{}
	m := codemap.Build(&codemap.Options{
		Root: root, Files: files, FS: fsys, Workers: 1,
		Resolver: &resolve.Resolver{FS: fsys, Index: index.New(), WorkspaceFolders: []string{root}},
	})

	var got []string

	for _, e := range m.Edges {
		if e.Kind == codemap.EdgeInstantiates && e.From != "shared/Factory.cfc::make" {
			got = append(got, fmt.Sprintf("%s -> %s x%d", e.From, e.To, e.Count))
		}
	}

	slices.Sort(got)

	want := []string{
		"Caller.cfc::inline -> shared/Svc.cfc x1",
		"Caller.cfc::local -> shared/Svc.cfc x1",
		"Caller.cfc::typed -> shared/Factory.cfc x1",
		"Caller.cfc::typed -> shared/Svc.cfc x1",
	}
	if !slices.Equal(got, want) {
		t.Errorf("instantiates edges:\n got %q\nwant %q", got, want)
	}
}
