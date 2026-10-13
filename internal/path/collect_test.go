package path

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

func writeTree(t *testing.T, root string, files ...string) {
	t.Helper()

	for _, f := range files {
		p := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func names(root string, files []string) []string {
	out := make([]string, 0, len(files))

	for _, f := range files {
		rel, _ := filepath.Rel(root, f)
		out = append(out, filepath.ToSlash(rel))
	}

	sort.Strings(out)

	return out
}

// TestCollectCFMLSkipsWhatNoScanShouldEnter: a git worktree inside a project is
// a complete second copy of the codebase, and every dot-directory and vendored
// folder is likewise not the project. The unresolved scan, the code map, the
// editor and find-references each carried this rule, one of them without it.
func TestCollectCFMLSkipsWhatNoScanShouldEnter(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root,
		"a.cfc", "page.cfm", "inc.cfs", "notes.txt", "style.css",
		"models/b.cfc",
		".claude/worktrees/copy/a.cfc", ".git/hooks/x.cfm",
		"node_modules/pkg/c.cfc", "vendor/lib/d.cfc", "target/e.cfc", "bower_components/f.cfc",
	)

	got := names(root, CollectCFML(vfs.OS{}, []string{root}))
	want := []string{"a.cfc", "inc.cfs", "models/b.cfc", "page.cfm"}

	if len(got) != len(want) {
		t.Fatalf("collected %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("collected %v, want %v", got, want)
		}
	}
}

// TestCollectCFMLEntersARootThatIsADotDirectory: the rule applies below the
// root, so pointing a scan at a vendored package or a dot-directory works.
func TestCollectCFMLEntersARootThatIsADotDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vendor", "pkg")
	writeTree(t, root, "a.cfc", "inner/b.cfc")

	if got := CollectCFML(vfs.OS{}, []string{root}); len(got) != 2 {
		t.Errorf("collected %v from an explicitly named vendor root, want both files", got)
	}

	dot := filepath.Join(t.TempDir(), ".hidden")
	writeTree(t, dot, "a.cfc")

	if got := CollectCFML(vfs.OS{}, []string{dot}); len(got) != 1 {
		t.Errorf("collected %v from a dot-directory root, want its file", got)
	}
}

// TestCollectCFMLSkipsAnUnstattableRoot covers a search root that is not there.
// filepath.Walk reports it with a nil FileInfo, and dereferencing that took
// down the whole search instead of skipping the one root — so a stale
// workspacePaths entry, or a folder removed since the editor opened it, turned
// every Find All References into an internal error.
func TestCollectCFMLSkipsAnUnstattableRoot(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, "a.cfc")

	got := CollectCFML(vfs.OS{}, []string{filepath.Join(dir, "does-not-exist"), dir})

	if len(got) != 1 || filepath.Base(got[0]) != "a.cfc" {
		t.Errorf("CollectCFML = %v, want just a.cfc from the root that exists", got)
	}
}
