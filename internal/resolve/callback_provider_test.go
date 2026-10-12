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

func TestProviderCallbackUsesInvocationArguments(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Service.cfc": `component {function go(){}}`,
		"Other.cfc":   `component {function other(){}}`,
		"Provider.cfc": `component {
 function direct(callback){arguments.callback(new Service());return new Other();}
 function alias(callback){var cb=arguments.callback;var value=new Service();cb(value);return new Other();}
 function supplied(value,callback){arguments.callback(arguments.value);return new Other();}
 function conflicting(callback){arguments.callback(new Service());arguments.callback(new Other());return new Service();}
 function unknown(callback){arguments.callback(runtimeValue);return new Service();}
 function unused(callback){return new Service();}
 function overwritten(callback){arguments.callback=unknown;callback(new Service());}
 function returnedEscape(callback){arguments.callback(new Service());return external(arguments.callback);}
 function escaped(callback){variables.saved=arguments.callback;arguments.callback(new Service());}
 }`,
	})

	src := `component {function run(){var p=new Provider();
 p.direct(function(x){x.go();});
 p.alias(function(x){x.go();});
 p.supplied(new Service(),function(x){x.go();});
 var value=new Service();
 p.supplied(value,function(x){x.go();});
 p.conflicting(function(x){x.go();});
 p.unknown(function(x){x.go();});
 p.unused(function(x){x.go();});
 p.overwritten(function(x){x.go();});
 p.returnedEscape(function(x){x.go();});
 p.escaped(function(x){x.go();});
 }}`
	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	pr := parser.ParseWithOptions(cfpath.ToURI(filepath.Join(dir, "Page.cfc")), src, &parser.ParseOptions{ExtractCalls: true})
	count := 0

	for _, call := range pr.AllCalls() {
		if call.FuncName != "go" {
			continue
		}

		count++
		line := strings.Split(src, "\n")[call.Line]
		want := strings.Contains(line, "p.direct") || strings.Contains(line, "p.alias") || strings.Contains(line, "p.supplied")

		reason := r.CanResolveCall(&call, pr, dir)
		if (reason == "") != want {
			t.Errorf("%s: %q want resolved %v", line, reason, want)
		}
	}

	if count != 10 {
		t.Fatalf("checked %d calls", count)
	}
}
