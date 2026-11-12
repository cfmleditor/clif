//go:build passcount

package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// The passes an editor action costs: tokens scanned over those of one plain
// scan of the file acted on. Measurement only, for a real workspace:
//
//	CLIF_PASSES_EDITOR=<workspace> CLIF_PASSES_FILE=<file> \
//	CLIF_PASSES_SECOND=<another file> CLIF_PASSES_HOVER=<line>:<col> \
//	CLIF_PASSES_IN=<line in a function> CLIF_PASSES_OUT=<line outside any> \
//	go test -tags passcount -run TestEditorActionPasses -v ./internal/server/
//
// Lines and columns are 0-based. Asynchronous work an action starts is given
// a moment to finish and is counted with it.
func TestEditorActionPasses(t *testing.T) {
	root := os.Getenv("CLIF_PASSES_EDITOR")
	if root == "" {
		t.Skip("CLIF_PASSES_EDITOR not set")
	}

	s := newTestServer()
	s.WorkspaceFolders = []string{root}
	s.configureSession(nil)
	s.indexWorkspace()

	ctx := context.Background()
	call := func(name string, plain int64, f func() error) {
		t.Helper()
		parser.ResetScanCounts()

		if err := f(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		time.Sleep(700 * time.Millisecond)

		tokens, parses := parser.ScanCounts()
		t.Logf("%-34s %6.2f passes, %d parses", name, float64(tokens)/float64(plain), parses)
	}

	file := filepath.Join(root, os.Getenv("CLIF_PASSES_FILE"))
	second := filepath.Join(root, os.Getenv("CLIF_PASSES_SECOND"))
	text, plain := readPlain(t, file)
	text2, plain2 := readPlain(t, second)
	u, u2 := uri.File(file), uri.File(second)

	open := func(u uri.URI, text string) func() error {
		return func() error {
			_, err := s.handleDidOpen(ctx, mustJSON(t, protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{URI: u, LanguageID: "cfml", Version: 1, Text: text},
			}))

			return err
		}
	}
	doc := protocol.TextDocumentIdentifier{URI: u}

	call("first open in a session", plain, open(u, text))
	call("a second component's open", plain2, open(u2, text2))
	call("reopening the first", plain, func() error {
		if _, err := s.handleDidClose(ctx, mustJSON(t, protocol.DidCloseTextDocumentParams{TextDocument: doc})); err != nil {
			return err
		}

		return open(u, text)()
	})
	call("folding", plain, func() error {
		_, err := s.handleFoldingRange(ctx, mustJSON(t, protocol.FoldingRangeParams{TextDocument: doc}))

		return err
	})
	call("document links", plain, func() error {
		_, err := s.handleDocumentLink(ctx, mustJSON(t, protocol.DocumentLinkParams{TextDocument: doc}))

		return err
	})

	var line, col uint32
	if _, err := fmtSscan(os.Getenv("CLIF_PASSES_HOVER"), &line, &col); err == nil {
		call("hover on a call", plain, func() error {
			_, err := s.handleHover(ctx, mustJSON(t, protocol.HoverParams{TextDocumentPositionParams: benchPos(u, line, col)}))

			return err
		})
	}

	version := int32(2)
	keystroke := func(env string) func() error {
		return func() error {
			var at uint32
			if _, err := fmtSscan(os.Getenv(env)+":0", &at, new(uint32)); err != nil {
				return err
			}

			s.mu.Lock()
			delete(s.changeWindowStart, u)
			s.mu.Unlock()

			version++
			_, err := s.handleDidChange(ctx, mustJSON(t, protocol.DidChangeTextDocumentParams{
				TextDocument: protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: doc, Version: version},
				ContentChanges: []protocol.TextDocumentContentChangeEvent{
					&protocol.TextDocumentContentChangePartial{
						Range: protocol.Range{Start: protocol.Position{Line: at}, End: protocol.Position{Line: at}},
						Text:  " ",
					},
				},
			}))

			return err
		}
	}

	call("a keystroke inside a function", plain, keystroke("CLIF_PASSES_IN"))
	call("a keystroke outside any function", plain, keystroke("CLIF_PASSES_OUT"))
}

func readPlain(t *testing.T, path string) (string, int64) {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	n := int64(1)

	sc := parser.NewScanner(string(b))
	for tok := sc.Next(); tok.Kind != parser.TokEOF; tok = sc.Next() {
		n++
	}

	return string(b), n
}

func fmtSscan(s string, line, col *uint32) (int, error) { return fmt.Sscanf(s, "%d:%d", line, col) }
