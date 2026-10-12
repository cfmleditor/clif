package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestTopLevelClosureConstructorOwnership(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Product.cfc": `component {function ready(){}}`, "Other.cfc": `component {function other(){}}`})

	for _, tc := range []struct {
		name, body string
		want       []bool
	}{
		{"local", `dispatch(function(){var item=new Product();item.ready();});`, []bool{true}},
		{"siblings", `dispatch(function(){var item=new Product();item.ready();});dispatch(function(){var item=new Other();item.ready();});`, []bool{true, false}},
		{"nested", `dispatch(function(){var item=new Product();dispatch(function(){var item=new Other();item.ready();});item.ready();});`, []bool{false, true}},
		{"parameter", `item=new Product();dispatch(function(item){item.ready();});item.ready();`, []bool{false, true}},
		{"nested-parameter", `dispatch(function(){var item=new Product();dispatch(function(item){item.ready();});item.ready();});`, []bool{false, true}},
		{"escape", `dispatch(function(){var item=new Product();item.ready();});item.ready();`, []bool{true, false}},
		{"factory-local", `function make(){return new Product();}dispatch(function(){var item=make();item.ready();});item.ready();`, []bool{true, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, prefix := range []string{"", "\n/* shifted */\n"} {
				pr := parser.ParseWithOptions(cfpath.ToURI(filepath.Join(dir, "Use.cfm")), prefix+`<cfscript>`+tc.body+`</cfscript>`, &parser.ParseOptions{ExtractCalls: true})
				r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}

				if tc.name == "factory-local" {
					for _, ref := range pr.ComponentRefs {
						if ref.Variable == "item" {
							t.Fatal("callback local escaped to component refs", ref)
						}
					}
				}

				n := 0

				for _, call := range pr.AllCalls() {
					if call.FuncName != "ready" {
						continue
					}

					if n >= len(tc.want) {
						t.Fatal("extra call")
					}

					reason := r.CanResolveCall(&call, pr, dir)
					if got := reason == ""; got != tc.want[n] {
						t.Errorf("prefix %q call %+v accepted=%v want=%v reason=%s refs=%+v", prefix, call, got, tc.want[n], reason, pr.ComponentRefs)
					}

					n++
				}

				if n != len(tc.want) {
					t.Fatalf("extracted %d calls, want %d", n, len(tc.want))
				}
			}
		})
	}
}
