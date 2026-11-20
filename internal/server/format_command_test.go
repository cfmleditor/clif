package server

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type formatEditConn struct {
	fakeConn
	edit *protocol.ApplyWorkspaceEditParams
}

func (c *formatEditConn) Call(_ context.Context, method string, params, _ any) (jsonrpc2.ID, error) {
	if method == protocol.MethodWorkspaceApplyEdit {
		c.edit, _ = params.(*protocol.ApplyWorkspaceEditParams)
	}

	return jsonrpc2.ID{}, nil
}

func TestFormatCommandPreservesLineEndingsAndDocumentVersion(t *testing.T) {
	for _, versioned := range []bool{false, true} {
		name := "legacy"
		if versioned {
			name = "versioned"
		}

		t.Run(name, func(t *testing.T) {
			conn := &formatEditConn{}
			s := newTestServer()
			s.conn = conn
			s.Formatting.Enabled = true
			s.clientDocChanges = versioned

			file := uri.File(filepath.Join(t.TempDir(), "Format.cfc"))
			openDoc(t, s, file, "<cfcomponent>\r<cfset x=1>\r</cfcomponent>\r")
			s.setDocVersion(file, 42)

			raw := makeCall(t, protocol.MethodTextDocumentFormatting, protocol.DocumentFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: file},
				Options:      protocol.FormattingOptions{InsertSpaces: true},
			})

			formatted, err := s.handleFormatting(t.Context(), raw)
			if err != nil {
				t.Fatal(err)
			}

			want, ok := formatted.([]protocol.TextEdit)
			if !ok || len(want) == 0 || !strings.Contains(want[0].NewText, "\r") {
				t.Fatalf("fixture must produce formatting edits with bare CR line endings, got %+v", formatted)
			}

			cmd := makeCall(t, protocol.MethodWorkspaceExecuteCommand, protocol.ExecuteCommandParams{
				Command:   "clif.format",
				Arguments: lspAnyArgs(string(file)),
			})
			if _, err := s.handleExecuteCommand(t.Context(), cmd); err != nil {
				t.Fatal(err)
			}

			if conn.edit == nil {
				t.Fatal("missing workspace/applyEdit request")
			}

			got := conn.edit.Edit.Changes[file]
			if versioned {
				changes := conn.edit.Edit.DocumentChanges
				if len(changes) != 1 {
					t.Fatalf("want one versioned document edit, got %+v", conn.edit.Edit)
				}

				edit, ok := changes[0].(*protocol.TextDocumentEdit)
				if !ok || edit.TextDocument.URI != file || edit.TextDocument.Version == nil || *edit.TextDocument.Version != 42 {
					t.Fatalf("want edit for version 42 of %s, got %+v", file, changes[0])
				}

				for _, entry := range edit.Edits {
					textEdit, ok := entry.(*protocol.TextEdit)
					if !ok {
						t.Fatalf("want text edit, got %T", entry)
					}

					got = append(got, *textEdit)
				}
			}

			if !slices.Equal(got, want) {
				t.Errorf("clif.format edits = %+v, want the same edits as textDocument/formatting = %+v", got, want)
			}
		})
	}
}
