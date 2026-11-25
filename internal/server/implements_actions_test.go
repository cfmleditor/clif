package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func refactorActionsAt(t *testing.T, s *Server, docURI uri.URI, line uint32) []protocol.CodeAction {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Range: protocol.Range{
			Start: protocol.Position{Line: line, Character: 0},
			End:   protocol.Position{Line: line, Character: 0},
		},
		Context: protocol.CodeActionContext{
			Only: []protocol.CodeActionKind{protocol.CodeActionKindRefactor},
		},
	})

	got, err := s.handleCodeAction(context.Background(), raw)
	if err != nil {
		t.Fatalf("handleCodeAction error: %v", err)
	}

	actions, _ := got.([]protocol.CodeAction)

	return actions
}

func TestImplementsActionsScriptAndTransitiveParent(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	u := writeCFCs(t, s, dir, map[string]string{
		"IBase.cfc":  "interface {\n\tpublic numeric function area();\n}\n",
		"ISub.cfc":   "interface extends=\"IBase\" {\n\tpublic numeric function perimeter(required numeric scale);\n}\n",
		"Square.cfc": "component implements=\"ISub\" {\n}\n",
	})

	actions := refactorActionsAt(t, s, u["Square.cfc"], 0)
	if len(actions) != 3 {
		t.Fatalf("got %d actions, want 3 (bulk + perimeter + area): %+v", len(actions), actions)
	}

	if actions[0].Title != "Implement all missing interface methods" {
		t.Errorf("actions[0].Title = %q, want bulk title", actions[0].Title)
	}

	edits := actions[0].Edit.Changes[u["Square.cfc"]]
	if len(edits) != 1 {
		t.Fatalf("bulk edit count = %d, want 1", len(edits))
	}

	text := edits[0].NewText
	if !strings.Contains(text, "public numeric function perimeter( required numeric scale ) {") {
		t.Errorf("bulk stub missing perimeter signature:\n%s", text)
	}

	if !strings.Contains(text, "public numeric function area() {") {
		t.Errorf("bulk stub missing inherited IBase.area signature:\n%s", text)
	}
}

func TestImplementsActionsTagCFCAndInheritedExclusion(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	files := map[string]string{
		"IShape.cfc": "interface {\n\tpublic numeric function area();\n\tpublic string function label(required string prefix);\n}\n",
		"Parent.cfc": "component {\n\tpublic numeric function area() { return 1; }\n}\n",
		"Child.cfc":  "<cfcomponent extends=\"Parent\" implements=\"IShape\">\n</cfcomponent>\n",
	}

	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	s.indexWorkspace()

	childURI := cfpath.ToURI(filepath.Join(dir, "Child.cfc"))
	openDoc(t, s, childURI, files["Child.cfc"])

	// `area` is inherited from Parent, so only `label` should be offered (single action, no bulk).
	actions := refactorActionsAt(t, s, childURI, 0)
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1 (label only): %+v", len(actions), actions)
	}

	if actions[0].Title != "Implement label() from IShape" {
		t.Errorf("title = %q, want Implement label() from IShape", actions[0].Title)
	}

	text := actions[0].Edit.Changes[childURI][0].NewText
	if !strings.Contains(text, `<cffunction name="label" access="public" returntype="string">`) ||
		!strings.Contains(text, `<cfargument name="prefix" type="string" required="true">`) {
		t.Errorf("tag stub mismatch:\n%s", text)
	}
}

func TestImplementsActionsOnlyOnHeaderOrCloser(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	u := writeCFCs(t, s, dir, map[string]string{
		"IShape.cfc": "interface {\n\tpublic numeric function area();\n}\n",
		"Square.cfc": "component implements=\"IShape\" {\n\tpublic void function helper() {}\n}\n",
	})

	// Line 1 is inside the body, not the header or closing brace.
	if actions := refactorActionsAt(t, s, u["Square.cfc"], 1); len(actions) != 0 {
		t.Errorf("expected no actions on body line, got %+v", actions)
	}

	// Line 2 is the closing brace.
	if actions := refactorActionsAt(t, s, u["Square.cfc"], 2); len(actions) != 1 {
		t.Errorf("expected 1 action on closing brace, got %+v", actions)
	}
}
