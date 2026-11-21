package server

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

func TestWorkflowWorkspaceSymbolResolve(t *testing.T) {
	root := t.TempDir()
	src := "component {\n\tpublic string\n\tfunction renderItem() {\n\t\treturn \"ok\";\n\t}\n}\n"

	t.Run("client advertising location.range resolveSupport receives URI-only symbols and resolves exact ranges", func(t *testing.T) {
		client := startWorkflowWithCapabilities(t, root, map[string]any{
			"workspace": map[string]any{
				"symbol": map[string]any{
					"resolveSupport": map[string]any{
						"properties": []string{"location.range"},
					},
				},
			},
		})

		opts, ok := client.capabilities.WorkspaceSymbolProvider.(*protocol.WorkspaceSymbolOptions)
		if !ok || opts.ResolveProvider == nil || !*opts.ResolveProvider {
			t.Fatalf("WorkspaceSymbolProvider = %#v, want ResolveProvider=true", client.capabilities.WorkspaceSymbolProvider)
		}

		doc := client.open(filepath.Join(root, "Widget.cfc"), src)

		var raw jsontext.Value
		client.call(protocol.MethodWorkspaceSymbol, protocol.WorkspaceSymbolParams{Query: "renderItem"}, &raw)

		if strings.Contains(string(raw), `"range"`) {
			t.Fatalf("deferred workspace/symbol payload should omit range, got %s", string(raw))
		}

		var symbols []protocol.WorkspaceSymbol
		if err := json.Unmarshal(raw, &symbols); err != nil {
			t.Fatalf("unmarshal WorkspaceSymbol: %v", err)
		}

		if len(symbols) != 1 || symbols[0].Name != "renderItem" {
			t.Fatalf("symbols = %+v, want [renderItem]", symbols)
		}

		var resolved protocol.WorkspaceSymbol
		client.call(protocol.MethodWorkspaceSymbolResolve, symbols[0], &resolved)

		loc, ok := resolved.Location.(*protocol.Location)
		if !ok {
			t.Fatalf("resolved.Location = %#v, want *protocol.Location", resolved.Location)
		}

		want := protocol.Location{
			URI:   doc.URI,
			Range: conformanceDeclarationRange(t, src, `function |renderItem|()`),
		}
		if *loc != want {
			t.Errorf("resolved location = %+v, want %+v", *loc, want)
		}

		// Resolving a symbol that is no longer in the index falls back cleanly
		// to a zero-range Location for the same URI instead of failing.
		stale := symbols[0]
		stale.Name = "removedMethod"

		var fallback protocol.WorkspaceSymbol
		client.call(protocol.MethodWorkspaceSymbolResolve, stale, &fallback)

		fallbackLoc, ok := fallback.Location.(*protocol.Location)
		if !ok {
			t.Fatalf("fallback.Location = %#v, want *protocol.Location", fallback.Location)
		}

		if *fallbackLoc != (protocol.Location{URI: doc.URI}) {
			t.Errorf("fallback location = %+v, want zero range for %s", *fallbackLoc, doc.URI)
		}
	})

	t.Run("client without resolveSupport keeps full SymbolInformation", func(t *testing.T) {
		client := startWorkflow(t, root, false)

		if enabled, ok := client.capabilities.WorkspaceSymbolProvider.(protocol.Boolean); !ok || !bool(enabled) {
			t.Fatalf("WorkspaceSymbolProvider = %#v, want Boolean(true)", client.capabilities.WorkspaceSymbolProvider)
		}

		doc := client.open(filepath.Join(root, "FullWidget.cfc"), src)

		var symbols []protocol.SymbolInformation
		client.call(protocol.MethodWorkspaceSymbol, protocol.WorkspaceSymbolParams{Query: "renderItem"}, &symbols)

		if len(symbols) != 1 || symbols[0].Location.URI != doc.URI {
			t.Fatalf("symbols = %+v, want 1 SymbolInformation for %s", symbols, doc.URI)
		}

		wantRange := conformanceDeclarationRange(t, src, `function |renderItem|()`)
		if symbols[0].Location.Range != wantRange {
			t.Errorf("range = %+v, want %+v", symbols[0].Location.Range, wantRange)
		}
	})
}
