package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestABranchUnionReturnHoldsWhateverTheCallTakes: Mura's
// feedBean.getIterator() gets one of two iterators from its bean factory on a
// test of its countOnly argument, and returns it:
//
//	if ( arguments.countOnly ) { var it = getBean( "beanIterator" ); }
//	else { var it = getBean( "contentIterator" ); }
//	return it;
//
// The parse types the return as both, through the bean rules. The resolver
// withheld it: the body branches on an argument, and with no call to decide
// the branch, reading the body again knew neither getBean(). The union names
// every branch reaching the one return, so it holds whichever the call takes,
// for a call passing only literals. A call passing a value it does not spell,
// and a body with a second return the union does not cover, still withhold
// it.
func TestABranchUnionReturnHoldsWhateverTheCallTakes(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"mura/cfobject.cfc":        `component { function getBean( name ) { return variables.factory.get( name ); } }`,
		"mura/BeanIterator.cfc":    `component { function hasNext() {} }`,
		"mura/ContentIterator.cfc": `component { function hasNext() {} }`,
		"mura/Feed.cfc": `component extends="mura.cfobject" {
	function getIterator( countOnly = false ) {
		if ( arguments.countOnly ) {
			var it = getBean( "beanIterator" );
		} else {
			var it = getBean( "contentIterator" );
		}
		return it;
	}
	function getEither( countOnly = false ) {
		if ( arguments.countOnly ) {
			return variables.factory.get( "other" );
		}
		if ( arguments.countOnly ) {
			var it = getBean( "beanIterator" );
		} else {
			var it = getBean( "contentIterator" );
		}
		return it;
	}
}`,
		"app/page.cfc": `component {
	function show( mura.Feed feed, flag ) {
		var iterator = feed.getIterator();
		iterator.hasNext();
		iterator.nowhere();
		var either = feed.getEither();
		either.nowhere();
		var counted = feed.getIterator( countOnly = true );
		counted.nowhere();
		var guessed = feed.getIterator( countOnly = arguments.flag );
		guessed.nowhere();
	}
}`,
	})

	r := &Resolver{FS: vfs.OS{}, WorkspaceFolders: []string{dir}, Index: index.New(), Resolvers: []parser.Resolver{
		{Match: `(?i)(?:^|\.)getBean\(\s*["']beanIterator["']\s*\)$`, Resolve: "mura.BeanIterator", Prefix: "getBean"},
		{Match: `(?i)(?:^|\.)getBean\(\s*["']contentIterator["']\s*\)$`, Resolve: "mura.ContentIterator", Prefix: "getBean"},
	}}

	for _, f := range []string{"mura/cfobject.cfc", "mura/BeanIterator.cfc", "mura/ContentIterator.cfc", "mura/Feed.cfc"} {
		file := filepath.Join(dir, filepath.FromSlash(f))

		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		r.Index.IndexParsed(cfpath.ToURI(file), string(data), r.ParseFile(file, string(data), IndexSpec()))
	}

	page := filepath.Join(dir, "app", "page.cfc")

	data, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}

	pr := r.ParseFile(page, string(data), BatchSpec())
	got := map[string]string{}

	calls := pr.AllCalls()
	for i := range calls {
		c := &calls[i]
		got[strings.TrimPrefix(c.Variable+"."+c.FuncName, ".")] = r.CanResolveCall(c, pr, filepath.Dir(page))
	}

	expectReasons(t, got, map[string]string{
		"iterator.hasNext": "",
		"iterator.nowhere": "method 'nowhere' not found in mura.BeanIterator|mura.ContentIterator",
		// getEither's other return is a call nothing types: dynamic, not the
		// union, which would report nowhere() missing from both iterators.
		"either.nowhere":  "",
		"counted.nowhere": "method 'nowhere' not found in mura.BeanIterator|mura.ContentIterator",
		// A value the call does not spell may be one evaluation would decide.
		"guessed.nowhere": "",
	})
}
