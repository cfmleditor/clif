package resolve

import (
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestAFileThatIsNoSourceUnderABeanRootIsNotRead: DiscoveryReadsFile is the
// exact check, and what bean discovery reads under a managed root is .cfc, so
// a JSON file or a README there is read by nothing. (DiscoveryAffected, the
// broad check for sources, covers everything under the root.)
func TestAFileThatIsNoSourceUnderABeanRootIsNotRead(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"beans/Svc.cfc": `component {}`, "beans/data.json": `{}`})

	r := &Resolver{
		FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir},
		BeanPaths: map[string]string{"app": dir + "/beans"},
	}

	if !r.DiscoveryAffected(dir + "/beans/data.json") {
		t.Fatal("the bean root does not cover the file here, so this test shows nothing")
	}

	if r.DiscoveryReadsFile(dir + "/beans/data.json") {
		t.Error("a JSON file under a bean root counts as read by discovery")
	}
}
