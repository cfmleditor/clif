package server

import (
	"context"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// A document whose lines end with a bare carriage return, as old Mac files
// do, is counted in lines as the editor counts them: the protocol ends a line
// at \r. Before, the whole file was line 0. Formatting it writes \r back, not
// the line feeds the analysis holds it with.
func TestABareCarriageReturnEndsALine(t *testing.T) {
	s := newTestServer()
	docURI := uri.File("/ws/old.cfc")
	text := strings.Join([]string{
		`<cfcomponent>`,
		`<cffunction name="first">`,
		`<cfreturn 1>`,
		`</cffunction>`,
		`<cffunction name="second">`,
		`<cfreturn 2>`,
		`</cffunction>`,
		`</cfcomponent>`,
	}, "\r") + "\r"

	open := mustJSON(t, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, LanguageID: "cfml", Version: 1, Text: text},
	})
	if _, err := s.handleDidOpen(context.Background(), open); err != nil {
		t.Fatal(err)
	}

	res, err := s.handleDocumentSymbol(context.Background(), mustJSON(t, protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
	}))
	if err != nil {
		t.Fatal(err)
	}

	lines := map[string]uint32{}

	var walk func([]protocol.DocumentSymbol)

	walk = func(syms []protocol.DocumentSymbol) {
		for _, sym := range syms {
			lines[sym.Name] = sym.Range.Start.Line
			walk(sym.Children)
		}
	}

	if syms, ok := res.([]protocol.DocumentSymbol); ok {
		walk(syms)
	} else {
		t.Fatalf("symbols: %#v", res)
	}

	if lines["first"] != 1 || lines["second"] != 4 {
		t.Fatalf("functions on lines %v, want first on 1 and second on 4", lines)
	}

	edits := s.withDocumentLineBreaks(docURI, []protocol.TextEdit{{NewText: "a\nb\n"}})
	if edits[0].NewText != "a\rb\r" {
		t.Fatalf("an edit to the document writes %q, want its own line breaks", edits[0].NewText)
	}
}
