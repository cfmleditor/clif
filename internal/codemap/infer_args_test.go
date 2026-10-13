package codemap_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/codemap"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/unresolved"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestTheMapTypesAnArgumentAsUnresolvedDoes: `use( thing )` has an untyped
// argument, and its one caller passes `new models.Item()`. `unresolved` types
// the argument from that caller (Options.InferArgs, its default), so
// `thing.ping()` is accepted; the map ran the same resolver without
// InferArgsFiles, so the call was an unresolved edge to an external node and
// Svc::use had no edge into Item::ping. A call the scan accepts must be an
// edge on the map.
func TestTheMapTypesAnArgumentAsUnresolvedDoes(t *testing.T) {
	dir := t.TempDir()

	var files []string

	for name, src := range map[string]string{
		"models/Item.cfc": `component { function ping(){} }`,
		"svc/Svc.cfc": `component {
function use( required thing ){ arguments.thing.ping(); }
function run(){ use( new models.Item() ); }
}`,
	} {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, p)
	}

	rep := unresolved.Scan(vfs.OS{}, files, nil, &unresolved.Options{WorkspaceFolders: []string{dir}, InferArgs: true})
	if len(rep.Calls) != 0 {
		t.Fatalf("the fixture is meant to resolve under unresolved: %+v", rep.Calls)
	}

	fsys := vfs.OS{}
	m := codemap.Build(&codemap.Options{
		Root:              dir,
		Files:             files,
		FS:                fsys,
		Resolver:          &resolve.Resolver{FS: fsys, Index: index.New(), WorkspaceFolders: []string{dir}},
		Workers:           2,
		IncludeUnresolved: true,
	})

	if m.Stats.Unresolved != 0 {
		t.Errorf("unresolved reports 0 findings, the map %d", m.Stats.Unresolved)
	}

	found := false

	for _, e := range m.Edges {
		if e.From == "svc/Svc.cfc::use" && e.To == "models/Item.cfc::ping" {
			found = true
		}
	}

	if !found {
		t.Errorf("no edge svc/Svc.cfc::use -> models/Item.cfc::ping; edges: %+v", m.Edges)
	}
}
