package resolve

import (
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/vfs"
)

// TestAComputedPrefixIncludeDoesNotDeadlockTheResolver: `include "#dir#x.cfm"`
// has the resolver list the workspace's templates by file name. The listing
// read the file system, which takes the resolver's own lock, while that lock
// was already held for writing, so the first such include in a workspace
// stopped every request that reached it for good (found on Wheels' Test.cfc).
func TestAComputedPrefixIncludeDoesNotDeadlockTheResolver(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Rb.cfc": `component { function getKeyValue( k ){ return k; } }`,
		"Page.cfc": `component {
	function render(){
		var rb = new Rb();
		include "#backportdir#lucee.cfm";
	}
}`,
		"backport/lucee.cfm": `<cfoutput>#rb.getKeyValue( "x" )#</cfoutput>`,
	})

	r := &Resolver{FS: vfs.OS{}, WorkspaceFolders: []string{dir}}

	done := make(chan []string, 1)

	go func() { done <- r.templatesNamed("lucee.cfm") }()

	select {
	case got := <-done:
		if len(got) != 1 {
			t.Errorf("templatesNamed = %v, want the one lucee.cfm", got)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("listing the templates for a computed-prefix include never returned")
	}
}
