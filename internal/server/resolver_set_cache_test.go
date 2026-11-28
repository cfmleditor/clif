package server

import (
	"io/fs"
	"os"
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

	// Asked of a resolver of its own, so the session holds none and the event
	// meets the kept answer alone: the branch discoveryResolver answers with the
	// one a kept set is answered by.
	marker := root + "/plugin/config.xml"
	if !(&resolve.Resolver{Discovery: s.discoveryDependencies}).DiscoveryReadsFile(marker) {
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

// TestAWatchedUnrelatedFileBuildsNoResolver is TestSavingAnUnrelatedFileBuilds
// NoResolver for the watcher's door: a batch for a file that is no source
// asked whether discovery read it by building the resolver.
func TestAWatchedUnrelatedFileBuildsNoResolver(t *testing.T) {
	root := t.TempDir()

	s := newTestServer()
	s.FS = vfs.OS{}
	s.WorkspaceFolders = []string{root}
	s.Features.WatchedFiles = true

	s.applyWatchedFileChanges([]protocol.FileEvent{
		{URI: cfpath.ToURI(root + "/README.md"), Type: protocol.FileChangeTypeChanged},
		{URI: cfpath.ToURI(root + "/node_modules/pkg/package.json"), Type: protocol.FileChangeTypeChanged},
	})

	if s.resolver != nil || s.resolversBuilt {
		t.Error("a watched batch of unrelated files built the resolver")
	}
}

// TestAKeptEmptyResolverSetIsAnsweredByOneResolver: a session that kept an
// empty resolver set holds no resolver of its own, and an unrelated save asks
// the one it is answered by. That builds it once; the saves after it must not
// read the project again.
func TestAKeptEmptyResolverSetIsAnsweredByOneResolver(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, root+"/Plain.cfc", "component { function run() {} }")

	counting := &readCountingFS{FS: vfs.OS{}}
	s := newTestServer()
	s.FS = counting
	s.WorkspaceFolders = []string{root}

	s.cfResolverSet()

	if s.resolver != nil || !s.resolversBuilt {
		t.Fatalf("want a kept empty set and no resolver, got resolver %v and built %v", s.resolver != nil, s.resolversBuilt)
	}

	s.invalidateSourceFile(root + "/README.md")

	if s.resolver == nil {
		t.Fatal("the save did not use a resolver to answer")
	}

	first := counting.reads.Load()

	for range 20 {
		s.invalidateSourceFile(root + "/README.md")
	}

	if got := counting.reads.Load(); got != first {
		t.Errorf("20 more unrelated saves read the file system %d more times; want none", got-first)
	}

	if !s.resolversBuilt {
		t.Error("an unrelated save dropped the kept resolver set")
	}
}

// TestAChangedMarkerClearsWhatTheMappingWalkKeptWithNoResolver: what cfpath
// read from a marker (a Mura plugin's config.xml) is kept process-wide, and
// outlives the resolver, which invalidateResolver drops. With no resolver
// there is nothing to ask whether discovery read the file, and the change must
// still clear what the walk kept, at the editor's save and at the watcher.
func TestAChangedMarkerClearsWhatTheMappingWalkKeptWithNoResolver(t *testing.T) {
	plugin := func(pkg string) string { return "<plugin><package>" + pkg + "</package></plugin>" }

	for name, change := range map[string]func(s *Server, file string){
		"save": func(s *Server, file string) { s.invalidateSourceFile(file) },
		"watch": func(s *Server, file string) {
			s.Features.WatchedFiles = true
			s.applyWatchedFileChanges([]protocol.FileEvent{{URI: cfpath.ToURI(file), Type: protocol.FileChangeTypeChanged}})
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			file := root + "/plugin/config.xml"
			writeCFC(t, root+"/Plain.cfc", "component {}")

			if err := os.MkdirAll(root+"/plugin", 0o755); err != nil {
				t.Fatal(err)
			}

			if err := os.WriteFile(file, []byte(plugin("pone")), 0o600); err != nil {
				t.Fatal(err)
			}

			cfpath.InvalidateAppMappingsCache()
			t.Cleanup(cfpath.InvalidateAppMappingsCache)

			if _, ok := cfpath.LoadAppMappings(root)["pone"]; !ok {
				t.Fatal("the mapping walk did not read the plugin's package, so there is no kept answer to go stale")
			}

			if !cfpath.WasProbed(file) {
				t.Fatal("the mapping walk did not record the marker as probed")
			}

			if err := os.WriteFile(file, []byte(plugin("ptwo")), 0o600); err != nil {
				t.Fatal(err)
			}

			s := newTestServer()
			s.FS = vfs.OS{}
			s.WorkspaceFolders = []string{root}
			change(s, file)

			if s.resolver != nil {
				t.Error("the change built a resolver to answer")
			}

			got := cfpath.LoadAppMappings(root)
			if _, stale := got["pone"]; stale {
				t.Errorf("the mapping walk still holds the plugin's old package: %v", got)
			}

			if _, ok := got["ptwo"]; !ok {
				t.Errorf("the mapping walk does not hold the plugin's new package: %v", got)
			}
		})
	}
}

// TestAChangedBoxSlugIsNoticed: a component named by a box.json slug resolves
// through the resolver's slug cache, and box.json is neither CFML nor a
// mapping, so nothing invalidated the cache when it changed: the old slug went
// on resolving and the new one did not, at the editor's save and the watcher.
func TestAChangedBoxSlugIsNoticed(t *testing.T) {
	for name, change := range map[string]func(s *Server, file string){
		"save": func(s *Server, file string) { s.invalidateSourceFile(file) },
		"watch": func(s *Server, file string) {
			s.Features.WatchedFiles = true
			s.applyWatchedFileChanges([]protocol.FileEvent{{URI: cfpath.ToURI(file), Type: protocol.FileChangeTypeChanged}})
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			lib := root + "/lib"
			box := lib + "/box.json"

			if err := os.MkdirAll(lib, 0o755); err != nil {
				t.Fatal(err)
			}

			writeCFC(t, lib+"/Foo.cfc", "component { function run() {} }")

			if err := os.WriteFile(box, []byte(`{"slug":"pone"}`), 0o600); err != nil {
				t.Fatal(err)
			}

			s := newTestServer()
			s.FS = vfs.OS{}
			s.WorkspaceFolders = []string{root}

			// ResolveFunc, as go-to-definition asks: it resolves on the view of
			// the caller's application (forCaller), not on the resolver itself,
			// and a slug kept on the view alone was not one an invalidation saw.
			r := s.getResolver()
			if got := r.ResolveFunc("pone.Foo", "run", lib); got == nil {
				t.Fatal("the slug does not name the component here, so this test shows nothing")
			}

			if err := os.WriteFile(box, []byte(`{"slug":"ptwo"}`), 0o600); err != nil {
				t.Fatal(err)
			}

			change(s, box)

			r = s.getResolver()
			if got := r.ResolveFunc("pone.Foo", "run", lib); got != nil {
				t.Errorf("the old slug still resolves, to %s", got.URI)
			}

			if got := r.ResolveFunc("ptwo.Foo", "run", lib); got == nil {
				t.Error("the new slug does not resolve")
			}
		})
	}
}

// TestAWatchedRemovalBuildsNoResolver: a batch that only removed a file has
// nothing to drop from a resolver that is not built, and dropped it by
// building one.
func TestAWatchedRemovalBuildsNoResolver(t *testing.T) {
	root := t.TempDir()
	gone := root + "/Gone.cfc"
	writeCFC(t, gone, "component { function run() {} }")

	s := newTestServer()
	s.FS = vfs.OS{}
	s.WorkspaceFolders = []string{root}
	s.Features.WatchedFiles = true
	s.index.IndexFile(cfpath.ToURI(gone), "component { function run() {} }")

	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}

	s.applyWatchedFileChanges([]protocol.FileEvent{{URI: cfpath.ToURI(gone), Type: protocol.FileChangeTypeDeleted}})

	if s.index.HasFile(cfpath.ToURI(gone)) {
		t.Fatal("the removal was not applied, so the batch did not reach the end")
	}

	if s.peekResolver() != nil {
		t.Error("a batch that only removed a file built the resolver")
	}
}

// TestAChangedMarkerIsNoticedInASessionWithAConfigFile: a session with a
// config file never runs the discovery that would record a Mura plugin's
// config.xml, so the resolver is not dropped when it changes; but its path
// caches hold resolutions made from the mappings the marker gave, and those
// must go with what the mapping walk kept.
func TestAChangedMarkerIsNoticedInASessionWithAConfigFile(t *testing.T) {
	plugin := func(pkg string) string { return "<plugin><package>" + pkg + "</package></plugin>" }

	root := t.TempDir()
	file := root + "/plugin/config.xml"
	writeCFC(t, root+"/Foo.cfc", "component {}")

	if err := os.MkdirAll(root+"/plugin", 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(file, []byte(plugin("pone")), 0o600); err != nil {
		t.Fatal(err)
	}

	cfpath.InvalidateAppMappingsCache()
	t.Cleanup(cfpath.InvalidateAppMappingsCache)

	s := newTestServer()
	s.FS = vfs.OS{}
	s.WorkspaceFolders = []string{root}
	s.ConfigPath = root + "/.clif.json"

	r := s.getResolver()
	if got := r.ComponentPath("pone.Foo", root); got == "" {
		t.Fatal("the plugin's package does not name the component here, so this test shows nothing")
	}

	if r.DiscoveryReadsFile(file) {
		t.Fatal("discovery recorded the marker, so the resolver is dropped for it and this test shows nothing")
	}

	if err := os.WriteFile(file, []byte(plugin("ptwo")), 0o600); err != nil {
		t.Fatal(err)
	}

	s.invalidateSourceFile(file)

	r = s.getResolver()
	if got := r.ComponentPath("pone.Foo", root); got != "" {
		t.Errorf("the plugin's old package still resolves, to %s", got)
	}

	if got := r.ComponentPath("ptwo.Foo", root); got == "" {
		t.Error("the plugin's new package does not resolve")
	}
}
