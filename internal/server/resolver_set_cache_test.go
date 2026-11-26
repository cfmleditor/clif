package server

import (
	"io/fs"
	"sync/atomic"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

type readCountingFS struct {
	vfs.FS
	reads atomic.Int64
}

func (c *readCountingFS) ReadFile(path string) ([]byte, error) {
	c.reads.Add(1)

	return c.FS.ReadFile(path)
}

func (c *readCountingFS) ReadDir(path string) ([]fs.DirEntry, error) {
	c.reads.Add(1)

	return c.FS.ReadDir(path)
}

// TestAWorkspaceWithNoResolverRulesIsDiscoveredOnce: when nothing names a
// framework and the source implies none, the resolver set is empty, and that
// answer was never kept. Every cfResolverSet call, one per call site of an
// inlay hint, hover or completion, read the project's files again to find
// there were none (63% of a 579 KB file's inlay hints).
func TestAWorkspaceWithNoResolverRulesIsDiscoveredOnce(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, root+"/Application.cfc", "component {}")
	writeCFC(t, root+"/Plain.cfc", "component { function run() {} }")

	counting := &readCountingFS{FS: vfs.OS{}}
	s := newTestServer()
	s.FS = counting
	s.WorkspaceFolders = []string{root}

	s.cfResolverSet()

	first := counting.reads.Load()
	if first == 0 {
		t.Skip("discovery read nothing here, so there is nothing to count")
	}

	for range 50 {
		s.cfResolverSet()
	}

	if got := counting.reads.Load(); got != first {
		t.Errorf("50 more resolver-set lookups read the file system %d more times; want none", got-first)
	}

	s.invalidateResolver()
	s.cfResolverSet()

	if got := counting.reads.Load(); got == first {
		t.Error("an invalidation did not make the next lookup discover again")
	}
}
