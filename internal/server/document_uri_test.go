package server

import (
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/config"
	"go.lsp.dev/uri"
)

// A command's URI argument is built by the client and may spell an open
// document differently from its didOpen; the command acts on the open
// document either way (intellilucee#13).
func TestACommandFindsAnOpenDocumentInAnySpelling(t *testing.T) {
	srv := newTestServer()
	opened := uri.URI("file:///work/my%20app/Page.cfm")
	srv.setDocument(opened, "<cfset a = 1>\n")

	for _, spelled := range []string{"file:///work/my%20app/Page.cfm", "file:///work/my app/Page.cfm", "file:///work/My%20App/page.cfm"} {
		if got := srv.documentURI(spelled); got != opened {
			t.Errorf("documentURI(%q) = %q, want the opened %q", spelled, got, opened)
		}
	}
}

// A document that is not open is said to be, rather than the command
// quietly doing nothing.
func TestFormatCommandReportsADocumentThatIsNotOpen(t *testing.T) {
	srv := newTestServer()
	srv.Formatting = config.DefaultResolvedFormatting()
	srv.Formatting.Enabled = true

	err := runFormatCommand(t, srv, uri.URI("file:///nowhere/Missing.cfm"))
	if err == nil || !strings.Contains(err.Error(), "not an open document") {
		t.Fatalf("err = %v, want one saying the document is not open", err)
	}
}
