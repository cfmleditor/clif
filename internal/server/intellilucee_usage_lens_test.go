package server

import (
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

// Adapted from IntelliLucee's function usage markers at
// 83e5e802d98d06845b220dfdd72db23da47c8be7. In LSP the editor can render a CodeLens.
func TestIntelliLuceeFunctionUsageCodeLens(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, filepath.Join(root, ".clif.json"), `{"references":{"enabled":true}}`)
	client := startWorkflow(t, root, false)
	doc := client.open(filepath.Join(root, "Tasks.cfc"), `component {
  function helper() {}
  function once() {}
  function run() {
    helper();
    helper();
    once();
    var text = "helper()";
    // helper();
  }
}`)

	var lenses []protocol.CodeLens
	client.call(protocol.MethodTextDocumentCodeLens, protocol.CodeLensParams{TextDocument: doc}, &lenses)

	got := make(map[uint32][]string)

	for _, lens := range lenses {
		if lens.Command.Title == "" {
			client.call(protocol.MethodCodeLensResolve, lens, &lens)
		}

		if lens.Command.Title != "" {
			line := lens.Range.Start.Line
			got[line] = append(got[line], strings.ToLower(lens.Command.Title))
		}
	}

	for line, want := range map[uint32]string{1: "2", 2: "1"} {
		titles := got[line]
		if len(titles) != 1 || !strings.HasPrefix(titles[0], want+" ") {
			t.Errorf("lenses at declaration line %d = %v, want one lens showing %s usages", line, titles, want)
		}
	}

	for _, line := range []uint32{4, 5, 6, 7, 8} {
		if len(got[line]) != 0 {
			t.Errorf("unexpected usage lens on line %d: %v", line, got[line])
		}
	}
}
