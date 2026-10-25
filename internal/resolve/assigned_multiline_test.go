package resolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// TestAChainOverSeveralLinesIsReadWhole: a chain written a call to a line,
//
//	<cfset iterator = $.getBean( "feed" )
//		.loadBy( feedid = id )
//		.getIterator()>
//
// is what its last call returns. Each line's parentheses balance, so the
// assignment was read to the end of its first, and the variable was typed as
// the bean that line names; on MasaCMS a call on it was then checked against
// the feed bean, whose onMissingMethod accepts anything. A chain that breaks
// on a later line (configure() returns nothing known) leaves the variable
// untyped rather than the bean.
func TestAChainOverSeveralLinesIsReadWhole(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"mura/Feed.cfc": `component {
	function load() { return this; }
	function configure( settings ) { variables.settings = settings; }
	function getIterator() { return new mura.Iterator(); }
	function onMissingMethod( name, args ) {}
}`,
		"mura/Iterator.cfc": `component { function hasNext() {} }`,
		"page.cfm": `<cfset iterator = feed
	.load()
	.getIterator()>
<cfset configured = feed
	.configure( 1 )
	.getIterator()>
<cfset more = iterator.hasNext() & configured.hasNext()>
<cfset feed = new mura.Feed()>`,
	})

	// The reasons themselves cannot show it: this parse types the broken
	// chain $any, where the scan's leaves it empty, so the read is asked.
	reasonsWith(t, &Resolver{}, dir, "mura/Iterator.cfc")

	r := &Resolver{}
	reasonsWith(t, r, dir, "page.cfm")

	file := filepath.Join(dir, "page.cfm")

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	pr := parser.ParseWithOptions(cfpath.ToURI(file), string(data), &parser.ParseOptions{ExtractCalls: true})

	if rhs, _, ok := r.localAssignmentAt(pr, "iterator", 0, 6); !ok || rhs != "feed.load().getIterator()" {
		t.Errorf("iterator's assignment: %q, %v; want the whole chain", rhs, ok)
	}

	if rhs, _, ok := r.localAssignmentAt(pr, "configured", 0, 6); !ok || rhs != "feed.configure( 1 ).getIterator()" {
		t.Errorf("configured's assignment: %q, %v; want the whole chain", rhs, ok)
	}

	if got := r.assignedFromCall("configured", 6, "", pr, dir, nil, lookupCtx{}); got != "" {
		t.Errorf("configured is %q, want nothing: the chain breaks at configure()", got)
	}
}
