package resolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestFieldTextProofIsMadeOncePerRoot: fieldTextProved walks every method of
// the file, and every guarded field a call reads asks it. Its answer is the
// file's and the root's alone, so a second ask is served from the source.
func TestFieldTextProofIsMadeOncePerRoot(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Svc.cfc")
	src := `component {
	function init() { variables.helper = new Helper(); variables.other = 1; return this; }
	function run() { return variables.helper.go(); }
}`

	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{FS: vfs.OS{}, WorkspaceFolders: []string{dir}, Index: index.New()}
	source := r.sourceInfo(file)

	if source.derived == nil {
		t.Skip("the source keeps no derived facts here")
	}

	fd := &parser.FunctionDef{URI: cfpath.ToURI(file)}
	e := &producerEvaluation{resolver: r, fd: fd}
	methods := sourcePlans(source).all()

	for _, path := range []string{"variables.helper", "variables.other", "variables.absent"} {
		want := e.fieldTextProvedUncached(&source, methods, variablesRoot(path))
		if got := e.fieldTextProved(&source, methods, path); got != want {
			t.Errorf("%s: first ask = %v, uncached = %v", path, got, want)
		}

		if _, ok := source.derived.fieldProofs.Load(variablesRoot(path)); !ok {
			t.Errorf("%s: the answer was not kept", path)
		}

		if got := e.fieldTextProved(&source, methods, path); got != want {
			t.Errorf("%s: second ask = %v, uncached = %v", path, got, want)
		}
	}
}
