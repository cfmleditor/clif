package resolve

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

// The resolvers ShareCallerIndex gives one index to ask it concurrently, each
// from its own goroutine; the index has a lock of its own for that, where
// each resolver's own lock guarded the shared map. Run with -race.
func TestResolversSharingACallerIndexAskItConcurrently(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{}
	paths := []string{}

	for i := range 20 {
		name := fmt.Sprintf("p%02d.cfm", 19-i)
		files[name] = `<cfset svc.save( x )>`
		paths = append(paths, filepath.Join(dir, name))
	}

	writeFiles(t, dir, files)

	rs := []*Resolver{}
	for range 4 {
		rs = append(rs, &Resolver{FS: vfs.OS{}, Index: index.New(), InferArgsFiles: paths})
	}

	restore := ShareCallerIndex(rs)
	defer restore()

	var wg sync.WaitGroup

	for _, r := range rs {
		wg.Go(func() {
			for range 50 {
				if got := r.callerFiles("save"); len(got) != len(paths) {
					t.Errorf("callers of save = %d, want %d", len(got), len(paths))

					return
				}
			}
		})
	}

	wg.Wait()
}
