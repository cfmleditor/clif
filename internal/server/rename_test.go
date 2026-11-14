package server

import (
	"context"
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

const renameSource = "component {\n\tfunction f( a ) {\n\t\tvar count = 0;\n\t\tcount = count + a;\n\t\treturn \"é #count#\";\n\t}\n}\n"

func renameServer(t *testing.T, src string) (*Server, uri.URI) {
	t.Helper()

	s := newTestServer()
	s.Features.Rename = true

	docURI := uri.File("/tmp/rename_test.cfc")
	openDoc(t, s, docURI, src)

	return s, docURI
}

func prepareRename(t *testing.T, s *Server, docURI uri.URI, line, char uint32) (any, error) {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentPrepareRename, protocol.PrepareRenameParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Position:     protocol.Position{Line: line, Character: char},
	})

	return s.handlePrepareRename(context.Background(), raw)
}

func renameTo(t *testing.T, s *Server, docURI uri.URI, line, char uint32, name string) (*protocol.WorkspaceEdit, error) {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentRename, protocol.RenameParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Position:     protocol.Position{Line: line, Character: char},
		NewName:      name,
	})

	got, err := s.handleRename(context.Background(), raw)
	edit, _ := got.(*protocol.WorkspaceEdit)

	return edit, err
}

func TestRenameAVariableEditsEveryUse(t *testing.T) {
	s, docURI := renameServer(t, renameSource)

	// On the declaration, `count` after `var `: line 2, columns 6 to 11.
	got, err := prepareRename(t, s, docURI, 2, 7)
	if err != nil {
		t.Fatal(err)
	}

	ph, ok := got.(protocol.PrepareRenamePlaceholder)
	if !ok || ph.Placeholder != "count" || ph.Range.Start.Line != 2 || ph.Range.Start.Character != 6 || ph.Range.End.Character != 11 {
		t.Fatalf("prepareRename answered %+v", got)
	}

	edit, err := renameTo(t, s, docURI, 3, 10, "total")
	if err != nil {
		t.Fatal(err)
	}

	edits := edit.Changes[docURI]
	if len(edits) != 4 {
		t.Fatalf("%d edits, want 4: %+v", len(edits), edits)
	}

	// Applied, the function reads as written with the new name, including the
	// one in the string's #...# after an é (columns are UTF-16).
	text := renameSource

	lines := strings.Split(text, "\n")

	for _, e := range slices.Backward(edits) {
		l := []rune(lines[e.Range.Start.Line])
		lines[e.Range.Start.Line] = string(l[:e.Range.Start.Character]) + e.NewText + string(l[e.Range.End.Character:])
	}

	want := "component {\n\tfunction f( a ) {\n\t\tvar total = 0;\n\t\ttotal = total + a;\n\t\treturn \"é #total#\";\n\t}\n}\n"
	if got := strings.Join(lines, "\n"); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenameRefusesWhatItCannotMakeSafe(t *testing.T) {
	s, docURI := renameServer(t, renameSource)

	// An argument: its callers name it.
	if _, err := prepareRename(t, s, docURI, 1, 14); err == nil || !strings.Contains(err.Error(), "argument") {
		t.Errorf("an argument: %v", err)
	}

	// Outside any function.
	if _, err := prepareRename(t, s, docURI, 0, 3); err == nil {
		t.Error("the component keyword can be renamed")
	}

	for name, why := range map[string]string{
		"a b": "not a valid", "": "not a valid", "1x": "not a valid", "return": "reserved", "Local": "reserved",
		"a": "already has", "A": "already has",
	} {
		if _, err := renameTo(t, s, docURI, 2, 8, name); err == nil || !strings.Contains(err.Error(), why) {
			t.Errorf("renaming to %q: %v, want an error mentioning %q", name, err, why)
		}
	}

	// A change of case alone is a rename.
	if _, err := renameTo(t, s, docURI, 2, 8, "COUNT"); err != nil {
		t.Errorf("a change of case: %v", err)
	}
}

func TestRenameNeedsTheFeature(t *testing.T) {
	s := newTestServer()

	if p, _ := s.capabilities().RenameProvider.(protocol.Boolean); bool(p) {
		t.Error("rename advertised with the feature off")
	}

	docURI := uri.File("/tmp/rename_off.cfc")
	openDoc(t, s, docURI, renameSource)

	if _, err := renameTo(t, s, docURI, 2, 8, "total"); err == nil {
		t.Error("a rename was done with the feature off")
	}

	s.Features.Rename = true

	opts, ok := s.capabilities().RenameProvider.(*protocol.RenameOptions)
	if !ok || opts.PrepareProvider == nil || !*opts.PrepareProvider {
		t.Errorf("rename is advertised as %#v with the feature on, want one that asks to be prepared", s.capabilities().RenameProvider)
	}
}
