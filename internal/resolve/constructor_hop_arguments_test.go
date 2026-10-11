package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestConstructorChainPreservesProducerArguments(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Product.cfc": `component {function ready(){}}`,
		"Factory.cfc": `component {function choose(required any value){return arguments.value;}}`,
	})

	src := `<cfscript>
var good=new Factory().choose(new Product());
good.ready();
good.absent();
</cfscript>`
	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	pr := parser.ParseWithOptions(cfpath.ToURI(filepath.Join(dir, "Use.cfm")), src, &parser.ParseOptions{ExtractCalls: true, FuncLookup: r.FuncLookup(dir)})
	checked := 0

	for _, call := range pr.AllCalls() {
		if call.Variable != "good" {
			continue
		}

		checked++

		reason := r.CanResolveCall(&call, pr, dir)
		if (reason == "") != (call.FuncName == "ready") {
			t.Errorf("%+v: %s", call, reason)
		}
	}

	if checked != 2 {
		t.Fatalf("checked %d", checked)
	}
}
