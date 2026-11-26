package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
)

// A caller file is read, and its lines found, once for the incoming calls to a
// method, not once for each call in it: with 8,000 calls in one file the answer
// took 871 ms, growing with the square of the file. The bound is generous; the
// answer takes about 20 ms.
func TestIncomingCallsOfAFileWithManyCallsAreLinear(t *testing.T) {
	dir := t.TempDir()
	svcSrc := "component {\n\tpublic void function foo() {}\n}\n"

	if err := os.WriteFile(filepath.Join(dir, "Svc.cfc"), []byte(svcSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	const calls = 8000

	page := "<cfscript>\nsvc = new Svc();\n" + strings.Repeat("svc.foo();\n", calls) + "</cfscript>\n"
	if err := os.WriteFile(filepath.Join(dir, "page.cfm"), []byte(page), 0o600); err != nil {
		t.Fatal(err)
	}

	s := newTestServer()
	s.References = true
	s.WorkspaceFolders = []string{dir}
	s.indexWorkspace()

	svcURI := cfpath.ToURI(filepath.Join(dir, "Svc.cfc"))
	openDoc(t, s, svcURI, svcSrc)

	items := prepareCallHierarchy(t, s, svcURI, protocol.Position{Line: 1, Character: 25})
	if len(items) != 1 {
		t.Fatalf("prepareCallHierarchy = %+v, want one item", items)
	}

	start := time.Now()
	incoming := incomingCalls(t, s, &items[0])

	if took := time.Since(start); took > 400*time.Millisecond {
		t.Errorf("incoming calls of %d calls in one file took %v, want well under a second", calls, took)
	}

	ranges := 0

	for _, c := range incoming {
		for _, r := range c.FromRanges {
			// "svc.foo();" on its own line: foo is at columns 4 to 7.
			if r.Start.Character != 4 || r.End.Character != 7 {
				t.Fatalf("a call's range is %+v, want columns 4-7", r)
			}

			ranges++
		}
	}

	if ranges != calls {
		t.Errorf("got %d call ranges, want %d", ranges, calls)
	}
}
