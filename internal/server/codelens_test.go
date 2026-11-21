package server

import (
	"context"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type countingReadFS struct {
	vfs.FS

	reads atomic.Int64
}

func (c *countingReadFS) ReadFile(path string) ([]byte, error) {
	c.reads.Add(1)

	return c.FS.ReadFile(path)
}

func TestCodeLensSharesOneWorkspaceScanAndCountsCrossFileCalls(t *testing.T) {
	root := t.TempDir()
	servicePath := filepath.Join(root, "Service.cfc")
	callerPath := filepath.Join(root, "Caller.cfc")

	serviceSrc := strings.Join([]string{
		"component accessors=\"true\" {",
		"  property string label;",
		"  function alpha() {}",
		"  function beta() {}",
		"  function gamma() {}",
		"  function run() {",
		"    alpha();",
		"    beta();",
		"  }",
		"}",
	}, "\n")
	callerSrc := strings.Join([]string{
		"component {",
		"  function go() {",
		"    var svc = new Service();",
		"    svc.alpha();",
		"    svc.alpha();",
		"  }",
		"}",
	}, "\n")

	writeCFC(t, servicePath, serviceSrc)
	writeCFC(t, callerPath, callerSrc)

	srv := newTestServer()
	srv.References = true
	srv.WorkspaceFolders = []string{root}
	counter := &countingReadFS{FS: srv.FS}
	srv.FS = counter

	docURI := uri.File(servicePath)
	openReq := makeCall(t, protocol.MethodTextDocumentDidOpen, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, Text: serviceSrc},
	})

	if _, err := srv.handleDidOpen(context.Background(), openReq); err != nil {
		t.Fatal(err)
	}

	counter.reads.Store(0)

	lensReq := makeCall(t, protocol.MethodTextDocumentCodeLens, protocol.CodeLensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
	})

	res, err := srv.handleCodeLens(context.Background(), lensReq)
	if err != nil {
		t.Fatalf("handleCodeLens: %v", err)
	}

	lenses, ok := res.([]protocol.CodeLens)
	if !ok {
		t.Fatalf("handleCodeLens returned %T, want []protocol.CodeLens", res)
	}

	// Caller.cfc is read once for all 4 declared functions, not once per function.
	if reads := counter.reads.Load(); reads > 2 {
		t.Errorf("ReadFile called %d times across 4 functions, want shared workspace scan (<= 2)", reads)
	}

	got := map[uint32]string{}
	for _, l := range lenses {
		got[l.Range.Start.Line] = l.Command.Title
	}

	// alpha: 1 same-file + 2 cross-file = 3 usages; beta: 1 same-file = 1 usage;
	// gamma/run/synthetic accessors have 0 usages and get no lens.
	if got[2] != "3 usages" {
		t.Errorf("alpha lens = %q, want %q (all=%v)", got[2], "3 usages", got)
	}

	if got[3] != "1 usage" {
		t.Errorf("beta lens = %q, want %q (all=%v)", got[3], "1 usage", got)
	}

	if len(got) != 2 {
		t.Errorf("got %d lenses (%v), want 2", len(got), got)
	}
}

func TestCodeLensDisabledWithoutReferencesFlag(t *testing.T) {
	root := t.TempDir()
	client := startWorkflow(t, root, false)

	if client.capabilities.CodeLensProvider != nil {
		t.Fatalf("CodeLensProvider = %#v, want nil when references.enabled is false", client.capabilities.CodeLensProvider)
	}

	doc := client.open(filepath.Join(root, "Tasks.cfc"), "component {\n  function a() {}\n  function b() { a(); }\n}")

	var lenses []protocol.CodeLens
	client.call(protocol.MethodTextDocumentCodeLens, protocol.CodeLensParams{TextDocument: doc}, &lenses)

	if len(lenses) != 0 {
		t.Errorf("lenses = %+v, want empty when references disabled", lenses)
	}
}
