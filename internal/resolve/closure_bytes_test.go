package resolve

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestClosureStoresUseByteIdentity(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A.cfc": `component {function onlyA(){}}`, "B.cfc": `component {function onlyB(){}}`})

	body := `each(xs,function(){var x=new A();x.onlyA();each(xs,function(){var x=new B();x.onlyB();x.onlyA();});x.onlyA();});each(xs,function(){var x=new B();x.onlyB();x.onlyA();});`

	sources := []string{`component {function run(){` + body + `}}`, `<cfcomponent><cffunction name="run"><cfscript>` + body + `</cfscript></cffunction></cfcomponent>`}
	for _, src := range sources {
		for _, prefix := range []string{"", "\n/* shifted */\n"} {
			text := prefix + src
			r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
			pr := parser.ParseWithOptions(cfpath.ToURI(filepath.Join(dir, "Page.cfc")), text, &parser.ParseOptions{ExtractCalls: true})
			count := 0

			for _, call := range pr.AllCalls() {
				if call.FuncName != "onlyA" && call.FuncName != "onlyB" {
					continue
				}

				count++
				// Calls immediately following onlyB belong to its B store and cannot see A.
				before := text[:call.StartByte]
				wrong := strings.HasSuffix(before, `x.onlyB();x.`)

				reason := r.CanResolveCall(&call, pr, dir)
				if (reason == "") == wrong {
					t.Errorf("%+v: %q wrong=%v", call, reason, wrong)
				}
			}

			if count != 6 {
				t.Fatalf("checked %d", count)
			}
		}
	}
}
