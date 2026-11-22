package server

import (
	"context"
	"testing"

	"github.com/cfmleditor/clif/internal/config"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func runFormatCommand(t *testing.T, srv *Server, docURI uri.URI) error {
	t.Helper()

	req := makeCall(t, protocol.MethodWorkspaceExecuteCommand, protocol.ExecuteCommandParams{
		Command:   "clif.format",
		Arguments: lspAnyArgs(string(docURI)),
	})

	_, err := srv.handleExecuteCommand(context.Background(), req)

	return err
}

// TestFormatCommandGatesOnEnabled pins the gate clif.format was missing.
// It hands s.Formatting straight to the formatter, and an unconfigured one is
// the zero value — every flag false, including WhitespaceOnly, the guard that
// stops the formatter writing back a file whose non-whitespace content it
// changed. textDocument/formatting has always returned early here; this
// command did not, which is what made a zero-valued s.Formatting dangerous
// rather than merely inert.
//
// The content below is one the guard rejects, so reaching the formatter at all
// surfaces as an error: with the gate, the command declines and there is none.
func TestFormatCommandGatesOnEnabled(t *testing.T) {
	srv := newTestServer()
	docURI := uri.URI("file:///t.cfm")
	srv.setDocument(docURI, "<CFOUTPUT>   <CFSET a=1>   </CFOUTPUT>\n")

	srv.Formatting = config.ResolvedFormatting{WhitespaceOnly: true}

	if err := runFormatCommand(t, srv, docURI); err != nil {
		t.Errorf("the command formatted with formatting disabled: %v", err)
	}
}

// TestFormatCommandStillRunsWhenEnabled is the other side of the gate.
func TestFormatCommandStillRunsWhenEnabled(t *testing.T) {
	srv := newTestServer()
	docURI := uri.URI("file:///t.cfm")
	srv.setDocument(docURI, "<cfoutput>\n<cfset a = 1 />\n</cfoutput>\n")

	srv.Formatting = config.DefaultResolvedFormatting()
	srv.Formatting.Enabled = true

	if err := runFormatCommand(t, srv, docURI); err != nil {
		t.Errorf("clif.format with formatting enabled: %v", err)
	}
}

func TestFormattingWithSelfCloseTagsDisabled(t *testing.T) {
	srv := newTestServer()
	docURI := uri.URI("file:///t.cfm")
	srv.setDocument(docURI, "<CFSET color=\"purple\">\n<cfinclude template=\"header.cfm\">\n")

	srv.Formatting = config.DefaultResolvedFormatting()
	srv.Formatting.Enabled = true
	srv.Formatting.SelfCloseTags = false

	req := makeCall(t, protocol.MethodTextDocumentFormatting, protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Options:      protocol.FormattingOptions{TabSize: 4, InsertSpaces: true},
	})

	res, err := srv.handleFormatting(context.Background(), req)
	if err != nil {
		t.Fatalf("handleFormatting: %v", err)
	}

	edits, _ := res.([]protocol.TextEdit)
	if len(edits) == 0 {
		t.Fatal("expected formatting edits when SelfCloseTags is false")
	}

	got := applyEdits(t, srv.documents[docURI], edits)
	want := "<cfset color = \"purple\">\n\n<cfinclude template=\"header.cfm\">\n"

	if got != want {
		t.Errorf("formatted output:\ngot:  %q\nwant: %q", got, want)
	}
}
