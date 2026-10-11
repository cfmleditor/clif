package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

func TestORMUtilityFactoryUsesSharedInterface(t *testing.T) {
	dir := t.TempDir()
	source := `component {function run(){
 var factory=new cborm.models.util.ORMUtilFactory();
 var utility=factory.getORMUtil();
 utility.getDefaultDatasource();
 utility.engineOnlyMissing();
 factory.engineOnlyMissing();
 }}`
	writeFiles(t, dir, map[string]string{"Use.cfc": source})

	r := &Resolver{}
	reasonsWith(t, r, dir, "Use.cfc")
	pr := parser.ParseWithOptions(cfpath.ToURI(filepath.Join(dir, "Use.cfc")), source, &parser.ParseOptions{ExtractCalls: true, FuncLookup: r.FuncLookup(dir)})
	checked := 0

	for _, call := range pr.AllCalls() {
		if call.Variable != "utility" && call.Variable != "factory" {
			continue
		}

		checked++

		target, reason := r.ResolveCallTarget(&call, pr, dir)
		switch {
		case call.Variable == "factory" && call.FuncName == "engineOnlyMissing":
			if reason == "" {
				t.Fatal("factory accepted missing method")
			}
		case call.FuncName == "engineOnlyMissing":
			// Existing interface semantics permit extensions dynamically, but do not
			// claim a shared API declaration or select one concrete engine branch.
			if reason != "" || target.Kind != TargetDynamic || target.URI != "" {
				t.Fatalf("interface extension target=%+v reason=%q", target, reason)
			}
		default:
			if reason != "" || target.Kind != TargetComponent || target.URI == "" {
				t.Fatalf("shared API target=%+v reason=%q", target, reason)
			}
		}
	}

	if checked != 4 {
		t.Fatalf("checked %d calls", checked)
	}
}
