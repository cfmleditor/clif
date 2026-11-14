package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/resolve"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// What an editing session costs in time and memory, on a real workspace.
// Measurement only:
//
//	CLIF_EDITOR_SESSION=<workspace> CLIF_EDITOR_FILE=<file under it> \
//	CLIF_EDITOR_IN=<line inside a function> CLIF_EDITOR_OUT=<line outside any> \
//	CLIF_EDITOR_HOVER=<line>:<col> [CLIF_EDITOR_EDITS=<n, default 400>] \
//	go test -run TestEditorSession -v ./internal/server/
//
// Lines and columns are 0-based. Each edit is followed by a hover, which waits
// on the document's lock and so on any reparse the edit deferred. The heap is
// read after a collection, so a figure that climbs with the edit count is a
// parse holding on to what an edit should have replaced: the reparse that
// listed every property once more per keystroke ran to 72MB after 2,400.
func TestEditorSession(t *testing.T) {
	root := os.Getenv("CLIF_EDITOR_SESSION")
	if root == "" {
		t.Skip("CLIF_EDITOR_SESSION not set")
	}

	ctx := context.Background()

	// As the server limits its caches when it runs as a daemon.
	resolve.LimitTextCaches(16 << 20)

	s := newTestServer()
	s.WorkspaceFolders = []string{root}
	s.configureSession(nil)
	s.indexWorkspace()

	file := filepath.Join(root, os.Getenv("CLIF_EDITOR_FILE"))

	text, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	u := uri.File(file)
	doc := protocol.TextDocumentIdentifier{URI: u}

	open := func() {
		_, err := s.handleDidOpen(ctx, mustJSON(t, protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{URI: u, LanguageID: "cfml", Version: 1, Text: string(text)},
		}))
		if err != nil {
			t.Fatal(err)
		}
	}

	open()
	time.Sleep(time.Second)

	var hoverLine, hoverCol, inLine, outLine uint32

	_, _ = fmt.Sscanf(os.Getenv("CLIF_EDITOR_HOVER"), "%d:%d", &hoverLine, &hoverCol)
	_, _ = fmt.Sscanf(os.Getenv("CLIF_EDITOR_IN"), "%d", &inLine)
	_, _ = fmt.Sscanf(os.Getenv("CLIF_EDITOR_OUT"), "%d", &outLine)

	edits := 400
	_, _ = fmt.Sscanf(os.Getenv("CLIF_EDITOR_EDITS"), "%d", &edits)

	version := int32(1)

	// change applies one edit as a keystroke is: no burst in progress.
	change := func(from, to protocol.Position, with string) time.Duration {
		s.mu.Lock()
		delete(s.changeWindowStart, u)
		s.mu.Unlock()

		version++
		start := time.Now()

		_, err := s.handleDidChange(ctx, mustJSON(t, protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: doc, Version: version},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{
				&protocol.TextDocumentContentChangePartial{Range: protocol.Range{Start: from, End: to}, Text: with},
			},
		}))
		if err != nil {
			t.Fatal(err)
		}

		return time.Since(start)
	}

	hover := func() time.Duration {
		start := time.Now()

		_, err := s.handleHover(ctx, mustJSON(t, protocol.HoverParams{TextDocumentPositionParams: benchPos(u, hoverLine, hoverCol)}))
		if err != nil {
			t.Fatal(err)
		}

		return time.Since(start)
	}

	quantile := func(d []time.Duration, q float64) time.Duration {
		d = slices.Clone(d)
		slices.Sort(d)

		return d[int(float64(len(d)-1)*q)].Round(time.Microsecond)
	}

	memory := func() string {
		runtime.GC()
		runtime.GC()

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		return fmt.Sprintf("heap %.0fMB", float64(m.HeapInuse)/(1<<20))
	}

	t.Logf("after open: %s", memory())

	for _, c := range []struct {
		name string
		line uint32
	}{{"keystroke inside a function", inLine}, {"keystroke outside any function", outLine}} {
		var changed, hovered []time.Duration

		for i := range edits {
			at := protocol.Position{Line: c.line}
			end := at

			with := " "
			if i%2 == 1 { // the space just typed, taken out again
				with, end.Character = "", 1
			}

			changed = append(changed, change(at, end, with))

			time.Sleep(15 * time.Millisecond)

			hovered = append(hovered, hover())
		}

		t.Logf("%-32s didChange p50 %v p95 %v | hover after p50 %v p95 %v | %s", c.name,
			quantile(changed, .5), quantile(changed, .95), quantile(hovered, .5), quantile(hovered, .95), memory())
	}

	// A line added and taken out again inside a function moves every line
	// after it; closing and reopening ends the document's parse.
	for range edits / 2 {
		at := protocol.Position{Line: inLine}
		change(at, at, "// x\n")
		change(at, protocol.Position{Line: inLine + 1}, "")
	}

	t.Logf("%d line add/remove pairs in a function: %s", edits/2, memory())

	for range edits / 4 {
		if _, err := s.handleDidClose(ctx, mustJSON(t, protocol.DidCloseTextDocumentParams{TextDocument: doc})); err != nil {
			t.Fatal(err)
		}

		open()
	}

	time.Sleep(time.Second)
	t.Logf("%d close/open cycles: %s", edits/4, memory())
}
