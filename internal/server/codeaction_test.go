package server

import (
	"context"
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestCodeActionOnlyFiltersReturnedActions(t *testing.T) {
	s := newTestServer()
	app := uri.File(filepath.Join(t.TempDir(), "Application.cfc"))
	openDoc(t, s, app, "component {\n}\n")

	for _, kind := range []protocol.CodeActionKind{protocol.CodeActionKindQuickFix, protocol.CodeActionKindRefactor} {
		t.Run(string(kind), func(t *testing.T) {
			raw := makeCall(t, protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: app},
				Context:      protocol.CodeActionContext{Only: []protocol.CodeActionKind{kind}},
			})

			got, err := s.handleCodeAction(context.Background(), raw)
			if err != nil {
				t.Fatal(err)
			}

			actions, _ := got.([]protocol.CodeAction)
			if kind == protocol.CodeActionKindRefactor {
				if _, ok := actionTitled(actions, "Add onSessionStart() to Application.cfc"); !ok {
					t.Fatal("missing lifecycle refactoring")
				}
			}

			for i := range actions {
				action := &actions[i]
				if action.Kind == nil || *action.Kind != kind {
					t.Errorf("request for %s returned %q with kind %v", kind, action.Title, action.Kind)
				}
			}
		})
	}
}

func TestWantsKind(t *testing.T) {
	refactor := protocol.CodeActionKindRefactor

	for _, c := range []struct {
		only []protocol.CodeActionKind
		want bool
	}{
		{nil, true},
		{[]protocol.CodeActionKind{protocol.CodeActionKindQuickFix}, false},
		{[]protocol.CodeActionKind{protocol.CodeActionKindQuickFix, protocol.CodeActionKindRefactor}, true},
		{[]protocol.CodeActionKind{"source"}, false},
	} {
		if got := wantsKind(c.only, refactor); got != c.want {
			t.Errorf("only %v: got %v, want %v", c.only, got, c.want)
		}
	}
}
