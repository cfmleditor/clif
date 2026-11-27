package server

import (
	"io/fs"
	"slices"
	"sync/atomic"
	"testing"

	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
	"go.lsp.dev/protocol"
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
		t.Fatal("discovery read nothing, so there is nothing to count")
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

// TestAFileDiscoveryReadInvalidatesTheKeptAnswer: the empty answer is kept
// until a file discovery read changes, and the project's markers are not all
// CFML. A plugin's config.xml, created after a session found no framework,
// must drop it as an edit to a .cfc does; the watcher also has to be asked for
// it (it is neither a CFML file nor a mapping JSON).
func TestAFileDiscoveryReadInvalidatesTheKeptAnswer(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, root+"/Plain.cfc", "component { function run() {} }")

	s := newTestServer()
	s.FS = vfs.OS{}
	s.WorkspaceFolders = []string{root}
	s.Features.WatchedFiles = true

	s.cfResolverSet()

	if !s.resolversBuilt {
		t.Fatal("the resolver set was not built, so there is no answer to keep")
	}

	marker := root + "/plugin/config.xml"
	if !s.getResolver().DiscoveryAffected(marker) {
		t.Fatalf("discovery did not record %s as a file it read", marker)
	}

	s.applyWatchedFileChanges([]protocol.FileEvent{{URI: cfpath.ToURI(marker), Type: protocol.FileChangeTypeCreated}})

	if s.resolversBuilt {
		t.Error("a file discovery read was created and the kept answer stayed")
	}

	if !slices.Contains(watchedGlobs, "**/plugin/config.xml") {
		t.Error("the client is not asked to watch plugin/config.xml")
	}
}

// TestSavingADiscoveryFileInTheEditorInvalidatesTheKeptAnswer is the editor's
// twin of TestAFileDiscoveryReadInvalidatesTheKeptAnswer: a save reaches
// invalidateSourceFile, which asked about CFML and mapping JSON alone, so a
// plugin's config.xml or a JSON marker saved in an editor that does not watch
// files left the empty answer standing.
func TestSavingADiscoveryFileInTheEditorInvalidatesTheKeptAnswer(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, root+"/Plain.cfc", "component { function run() {} }")

	s := newTestServer()
	s.FS = vfs.OS{}
	s.WorkspaceFolders = []string{root}

	s.cfResolverSet()

	if !s.resolversBuilt {
		t.Fatal("the resolver set was not built, so there is no answer to keep")
	}

	s.invalidateSourceFile(root + "/plugin/config.xml")

	if s.resolversBuilt {
		t.Error("saving a file discovery read left the kept answer in place")
	}
}

// TestAnUnrelatedJSONFileDoesNotDropTheResolver: the project's markers are
// exact files, but discovery lists directories, and everything under a listed
// directory counts for DiscoveryAffected. A JSON file nobody reads (an npm
// install, a build's output) must not drop the resolver for that.
func TestAnUnrelatedJSONFileDoesNotDropTheResolver(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, root+"/Plain.cfc", "component { function run() {} }")

	s := newTestServer()
	s.FS = vfs.OS{}
	s.WorkspaceFolders = []string{root}
	s.Features.WatchedFiles = true

	s.cfResolverSet()

	// A project whose root discovery listed, as a ColdBox application's is.
	_, _ = resolve.TrackDiscoveryFS(s.FS, s.discoveryDependencies).ReadDir(root)

	unrelated := root + "/node_modules/pkg/package.json"
	if !s.getResolver().DiscoveryAffected(unrelated) {
		t.Fatal("the broad check does not cover the file here, so this test shows nothing")
	}

	s.applyWatchedFileChanges([]protocol.FileEvent{{URI: cfpath.ToURI(unrelated), Type: protocol.FileChangeTypeChanged}})
	s.invalidateSourceFile(unrelated)

	if !s.resolversBuilt {
		t.Error("a JSON file no discovery read dropped the resolver")
	}
}

// TestSavingAnUnrelatedFileBuildsNoResolver: a save of a file that is no
// source asks whether discovery read it, and with no resolver built there is
// nothing kept to invalidate, so the answer is no without building one (and
// reading the project to do it).
func TestSavingAnUnrelatedFileBuildsNoResolver(t *testing.T) {
	root := t.TempDir()

	s := newTestServer()
	s.FS = vfs.OS{}
	s.WorkspaceFolders = []string{root}

	s.invalidateSourceFile(root + "/README.md")

	if s.resolver != nil || s.resolversBuilt {
		t.Error("saving a README built the resolver")
	}
}
