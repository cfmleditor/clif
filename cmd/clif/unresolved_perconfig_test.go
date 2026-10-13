package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/unresolved"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestUnresolvedReadsEachFileUnderItsOwnConfig: a workspace of two applications
// has a .clif.json each, and a scan rooted above both used to read all of it
// under the first. The second application's componentResolvers then never
// fired and its calls came out unresolved, with no error to say why.
// --one-config keeps the old reading.
func TestUnresolvedReadsEachFileUnderItsOwnConfig(t *testing.T) {
	root := t.TempDir()

	for name, content := range map[string]string{
		".clif.json":      `{"workspaceName":"mono"}`,
		"app2/.clif.json": `{"workspaceName":"app2","componentResolvers":[{"match":"svc","resolve":"app2.Svc","prefix":"svc","anchored":true}]}`,
		"app2/Svc.cfc":    `component { function run() {} }`,
		"app2/Page.cfc":   `component { function go() { svc.run(); } }`,
		"app1/Other.cfc":  `component { function x() {} }`,
	} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	scan := func(oneConfig bool) int {
		fl := &unresolvedFlags{oneConfig: oneConfig}
		args := []string{root}
		opt := unresolvedOptions(nil, args, fl)

		if !oneConfig {
			opt.ResolverFor = perFileResolver(vfs.OS{})
		}

		files := []string{filepath.Join(root, "app2", "Svc.cfc"), filepath.Join(root, "app2", "Page.cfc"), filepath.Join(root, "app1", "Other.cfc")}

		return len(unresolved.Scan(vfs.OS{}, files, nil, opt).Calls)
	}

	if got := scan(true); got != 1 {
		t.Fatalf("one config: %d unresolved, want 1 (the premise)", got)
	}

	if got := scan(false); got != 0 {
		t.Errorf("per-file config: %d unresolved, want 0", got)
	}
}
