package path

import (
	"errors"
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

// TestCollectCFCKeepsComponentsOnly: the editor's index holds components alone,
// by the same skip rule as every other scan.
func TestCollectCFCKeepsComponentsOnly(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, "a.cfc", "page.cfm", "m/b.CFC", ".git/c.cfc", "node_modules/d.cfc")

	got := names(root, CollectCFC(vfs.OS{}, []string{root}))
	want := []string{"a.cfc", "m/b.CFC"}

	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v, want %v", got, want)
	}
}

// Inject walk failures so these checks run on Windows and privileged accounts
// too, where chmod cannot reliably make a directory unreadable.
type failingCollectFS struct {
	vfs.OS
	denied, failedRoot string
}

func (f failingCollectFS) Walk(root string, fn filepath.WalkFunc) error {
	if root == f.failedRoot {
		return &os.PathError{Op: "walk", Path: root, Err: os.ErrNotExist}
	}

	return f.OS.Walk(root, func(path string, info os.FileInfo, err error) error {
		if path != f.denied {
			return fn(path, info, err)
		}

		if err := fn(path, info, &os.PathError{Op: "readdir", Path: path, Err: os.ErrPermission}); err != nil {
			return err
		}

		return filepath.SkipDir
	})
}

func TestCollectWithErrorsKeepsReadableFiles(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, "a-denied/hidden.cfm", "z-readable.cfc")
	failing := failingCollectFS{denied: filepath.Join(root, "a-denied"), failedRoot: filepath.Join(root, "missing")}

	files, err := CollectWithErrors(failing, []string{failing.failedRoot, root}, IsCFMLFile)
	if !errors.Is(err, os.ErrNotExist) || !errors.Is(err, os.ErrPermission) {
		t.Errorf("collection error = %v, want both root and entry failures", err)
	}

	if got := names(root, files); len(got) != 1 || got[0] != "z-readable.cfc" {
		t.Errorf("files = %v, want the readable file after the failed entry", got)
	}
}

func TestCollectWithErrorsExcludesIgnoredDirectories(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, "vendor/hidden.cfm", "a.cfc")
	failing := failingCollectFS{denied: filepath.Join(root, "vendor")}

	files, err := CollectWithErrors(failing, []string{root}, IsCFMLFile)
	if err != nil || len(files) != 1 || filepath.Base(files[0]) != "a.cfc" {
		t.Errorf("files = %v, error = %v; want only a.cfc without errors", files, err)
	}

	// An explicitly requested root is inspected even when its name is one
	// normally excluded from a workspace scan.
	_, err = CollectWithErrors(failing, []string{failing.denied}, IsCFMLFile)
	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("explicit unreadable root: error = %v, want permission failure", err)
	}
}
