package server

import (
	"context"
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestWantsKind(t *testing.T) {
	refactor := protocol.CodeActionKindRefactor

	for _, c := range []struct {
		only []protocol.CodeActionKind
		kind protocol.CodeActionKind
		want bool
	}{
		{nil, refactor, true},
		{[]protocol.CodeActionKind{protocol.CodeActionKindQuickFix}, refactor, false},
		{[]protocol.CodeActionKind{protocol.CodeActionKindQuickFix, protocol.CodeActionKindRefactor}, refactor, true},
		{[]protocol.CodeActionKind{"source"}, refactor, false},
		// A parent kind asks for its children, never a child for its parent.
		{[]protocol.CodeActionKind{refactor}, protocol.CodeActionKindRefactorRewrite, true},
		{[]protocol.CodeActionKind{protocol.CodeActionKindRefactorRewrite}, refactor, false},
		{[]protocol.CodeActionKind{protocol.CodeActionKindRefactorExtract}, protocol.CodeActionKindRefactorRewrite, false},
		// A prefix of the name that is not a whole segment is no parent.
		{[]protocol.CodeActionKind{"refac"}, refactor, false},
		{[]protocol.CodeActionKind{"refactor.re"}, protocol.CodeActionKindRefactorRewrite, false},
		// The empty kind is every kind.
		{[]protocol.CodeActionKind{""}, refactor, true},
	} {
		if got := wantsKind(c.only, c.kind); got != c.want {
			t.Errorf("only %v, kind %q: got %v, want %v", c.only, c.kind, got, c.want)
		}
	}
}

// The edits a code action carries are refactorings, handed only to a client
// that asks for them: the light bulb's request for quick fixes alone, or for a
// narrower refactoring than these, gets none.
func TestCodeActionsHandRefactoringsOnlyToAClientThatAsks(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	app := uri.File(filepath.Join(dir, "Application.cfc"))
	openDoc(t, s, app, "component {\n\tthis.name = \"x\";\n}\n")

	ask := func(only ...protocol.CodeActionKind) (refactors, commands int) {
		t.Helper()

		raw := makeCall(t, protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: app},
			Context:      protocol.CodeActionContext{Only: only},
		})

		got, err := s.handleCodeAction(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}

		actions, _ := got.([]protocol.CodeAction)

		for _, a := range actions {
			switch {
			case a.Edit != nil:
				if a.Kind == nil || *a.Kind != protocol.CodeActionKindRefactor {
					t.Errorf("%q carries an edit and is not a refactor", a.Title)
				}

				refactors++
			case a.Command.Command != "":
				commands++
			}
		}

		return refactors, commands
	}

	if r, c := ask(); r == 0 || c == 0 {
		t.Errorf("an unlimited request: %d refactors, %d commands; want both", r, c)
	}

	if r, _ := ask(protocol.CodeActionKindRefactor); r == 0 {
		t.Error("a request for refactorings got none")
	}

	for _, only := range [][]protocol.CodeActionKind{
		{protocol.CodeActionKindQuickFix},
		{protocol.CodeActionKindRefactorExtract},
		{protocol.CodeActionKindSourceOrganizeImports},
	} {
		if r, _ := ask(only...); r != 0 {
			t.Errorf("a request for %v got %d refactorings", only, r)
		}
	}
}
