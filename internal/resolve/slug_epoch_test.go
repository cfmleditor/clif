package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

// pausingFS reads a file and then waits, once, before handing it over: the
// moment between a lookup's read and its store, when an invalidation can run.
type pausingFS struct {
	vfs.FS
	name    string
	read    chan struct{} // closed once the paused read has its data
	release chan struct{} // closed to let it return
	paused  bool
}

func (f *pausingFS) ReadFile(path string) ([]byte, error) {
	data, err := f.FS.ReadFile(path)

	if !f.paused && strings.EqualFold(filepath.Base(path), f.name) {
		f.paused = true

		close(f.read)
		<-f.release
	}

	return data, err
}

// TestASlugReadBeforeAnInvalidationIsNotKeptAfterIt: a lookup reads box.json
// (or module.json), the file changes and the path caches are dropped, and the
// lookup then stores what it read. Kept on the owner, which every caller's
// view shares, that stale slug outlived the invalidation that was meant to
// drop it. It is stored only if no invalidation ran while it was read.
func TestASlugReadBeforeAnInvalidationIsNotKeptAfterIt(t *testing.T) {
	for _, c := range []struct {
		name, file, before, after, want string
		ask                             func(r *Resolver, dir string) string
	}{
		{"box.json", "box.json", `{"slug":"pone"}`, `{"slug":"ptwo"}`, "ptwo", (*Resolver).boxSlug},
		{"module.json", "module.json", `{"name":"mone","main":"Module.cfc"}`, `{"name":"mtwo","main":"Module.cfc"}`, "mtwo", (*Resolver).lucliModuleName},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, c.file)

			if err := os.WriteFile(file, []byte(c.before), 0o600); err != nil {
				t.Fatal(err)
			}

			fsys := &pausingFS{FS: vfs.OS{}, name: c.file, read: make(chan struct{}), release: make(chan struct{})}
			r := &Resolver{FS: fsys, Index: index.New(), WorkspaceFolders: []string{dir}}

			done := make(chan string, 1)

			go func() { done <- c.ask(r, dir) }()

			<-fsys.read

			if err := os.WriteFile(file, []byte(c.after), 0o600); err != nil {
				t.Fatal(err)
			}

			r.InvalidatePaths()
			close(fsys.release)
			<-done

			if got := c.ask(r, dir); got != c.want {
				t.Errorf("after the invalidation the cache answers %q, want %q", got, c.want)
			}
		})
	}
}
