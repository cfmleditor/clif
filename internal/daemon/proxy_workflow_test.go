//go:build !wasm

package daemon

import (
	"context"
	"net"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/index"
	cflog "github.com/cfmleditor/clif/internal/log"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/server"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func TestProxyClientSeesSharedEditsAndSurvivesDisconnect(t *testing.T) {
	sock := shortSock(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)

	tracker := NewConnTracker()
	served := make(chan error, 1)

	go func() {
		served <- Serve(ctx, sock, cflog.NewLogger(false), index.New(), tracker, &server.Settings{})
	}()

	t.Cleanup(func() {
		cancel()

		select {
		case err := <-served:
			if err != nil {
				t.Errorf("daemon stopped with an error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("daemon did not stop after cancellation")
		}
	})

	waitForSocket(t, sock)
	conn, direct := dialRPC(t, ctx, sock)
	t.Cleanup(func() { _ = conn.Close() })

	// The second editor uses the same bidirectional bridge as the CLI.
	// A pipe supplies its two stdio streams while the daemon uses a Unix socket.
	input, client := net.Pipe()

	t.Cleanup(func() {
		_ = client.Close()
		_ = input.Close()
	})

	proxied := make(chan error, 1)
	go func() { proxied <- proxy(sock, input, input) }()

	rpc := jsonrpc2.NewConn(jsonrpc2.NewStream(client))
	rpc.Go(ctx, func(context.Context, *jsonrpc2.Request) (any, error) { return nil, nil })

	call := func(c jsonrpc2.Conn, method string, params, result any) {
		t.Helper()

		if _, err := c.Call(ctx, method, params, result); err != nil {
			t.Fatalf("%s: %v", method, err)
		}
	}
	notify := func(c jsonrpc2.Conn, method string, params any) {
		t.Helper()

		if err := c.Notify(ctx, method, params); err != nil {
			t.Fatalf("%s: %v", method, err)
		}
	}

	for _, c := range []jsonrpc2.Conn{direct, rpc} {
		call(c, protocol.MethodInitialize, protocol.InitializeParams{}, nil)
	}

	doc := protocol.TextDocumentIdentifier{URI: cfpath.ToURI(filepath.Join(t.TempDir(), "Service.cfc"))}
	notify(direct, protocol.MethodTextDocumentDidOpen, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI: doc.URI, LanguageID: "cfml", Version: 1,
			Text: "component { function oldSharedName() {} }",
		},
	})

	// A request on the first connection confirms that the earlier notification
	// was processed before we query the shared index through a different client.
	var symbols []protocol.DocumentSymbol
	call(direct, protocol.MethodTextDocumentDocumentSymbol, protocol.DocumentSymbolParams{TextDocument: doc}, &symbols)

	if len(symbols) != 1 || len(symbols[0].Children) != 1 || symbols[0].Children[0].Name != "oldSharedName" {
		t.Fatalf("opening client symbols = %+v", symbols)
	}

	hasSymbol := func(name string) bool {
		t.Helper()

		var result []protocol.SymbolInformation
		call(rpc, protocol.MethodWorkspaceSymbol, protocol.WorkspaceSymbolParams{Query: name}, &result)

		return slices.ContainsFunc(result, func(s protocol.SymbolInformation) bool {
			return s.Name == name && s.Location.URI == doc.URI
		})
	}
	if !hasSymbol("oldSharedName") {
		t.Fatal("proxy client could not find the symbol from the opening client")
	}

	symbols = nil
	call(rpc, protocol.MethodTextDocumentDocumentSymbol, protocol.DocumentSymbolParams{TextDocument: doc}, &symbols)

	if len(symbols) != 0 {
		t.Fatalf("proxy client received document symbols for another client's open buffer: %+v", symbols)
	}

	notify(direct, protocol.MethodTextDocumentDidChange, protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: doc, Version: 2},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			&protocol.TextDocumentContentChangeWholeDocument{Text: "component { function newSharedName() {} }"},
		},
	})
	call(direct, protocol.MethodTextDocumentDocumentSymbol, protocol.DocumentSymbolParams{TextDocument: doc}, &symbols)

	if len(symbols) != 1 || len(symbols[0].Children) != 1 || symbols[0].Children[0].Name != "newSharedName" {
		t.Fatalf("editing client symbols = %+v", symbols)
	}

	if hasSymbol("oldSharedName") || !hasSymbol("newSharedName") {
		t.Fatal("proxy client did not see the shared index replace the old declaration")
	}

	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	// Wait for the daemon to process the disconnect before probing its
	// other session. EOF can reach the socket after Close returns.
	poll := time.NewTicker(5 * time.Millisecond)
	defer poll.Stop()

	for tracker.Count() != 1 {
		select {
		case <-poll.C:
		case <-ctx.Done():
			t.Fatalf("daemon kept %d clients after the direct connection closed", tracker.Count())
		}
	}

	if !hasSymbol("newSharedName") {
		t.Fatal("proxy client lost the index when the editing client disconnected")
	}

	if err := client.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-proxied:
		if err != nil {
			t.Fatalf("proxy closed with an error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("proxy did not exit after editor disconnect")
	}

	select {
	case <-tracker.Done():
	case <-ctx.Done():
		t.Fatal("daemon retained the closed proxy session")
	}
}
