package resolve

import (
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestDiscoveryReadsFileIsExact: DiscoveryReadsFile is the check for a file
// that is no source, and counts only a file discovery read or probed. A JSON
// file or a README in a directory discovery listed (a bean root, the
// workspace root) is read by nothing, though DiscoveryAffected, the broad check
// for sources, covers everything under it.
func TestDiscoveryReadsFileIsExact(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"beans/Svc.cfc": `component {}`, "beans/data.json": `{}`, "plugin/config.xml": `<plugin/>`})

	deps := &DiscoveryDependencies{}
	r := &Resolver{
		FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}, Discovery: deps,
		BeanPaths: map[string]string{"app": dir + "/beans"},
	}

	// What discovery does: lists the bean root and the project root, and
	// probes the marker it looks for.
	tracked := TrackDiscoveryFS(r.FS, deps)
	_, _ = tracked.ReadDir(dir + "/beans")
	_, _ = tracked.ReadDir(dir)
	_, _ = tracked.ReadFile(dir + "/plugin/config.xml")

	data := dir + "/beans/data.json"
	if !r.DiscoveryAffected(data) {
		t.Fatal("the listed bean root does not cover the file here, so this test shows nothing")
	}

	if r.DiscoveryReadsFile(data) {
		t.Error("a JSON file in a listed directory counts as read by discovery")
	}

	if r.DiscoveryReadsFile(dir + "/README.md") {
		t.Error("a README in the listed project root counts as read by discovery")
	}

	if !r.DiscoveryReadsFile(dir + "/plugin/config.xml") {
		t.Error("a marker discovery read does not count")
	}

	// A probe that found nothing is recorded, so creating the file is noticed.
	_, _ = tracked.ReadFile(dir + "/box.json")

	if !r.DiscoveryReadsFile(dir + "/box.json") {
		t.Error("a marker discovery probed and did not find does not count")
	}
}
