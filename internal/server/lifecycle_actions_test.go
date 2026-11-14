package server

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// applyActionEdit applies the single edit of an action to text.
func applyActionEdit(t *testing.T, text string, a *protocol.CodeAction) string {
	t.Helper()

	for _, edits := range a.Edit.Changes {
		if len(edits) != 1 {
			t.Fatalf("%d edits, want 1", len(edits))
		}

		e := edits[0]
		off := func(p protocol.Position) int { return positionOffset(text, p) }

		return text[:off(e.Range.Start)] + e.NewText + text[off(e.Range.End):]
	}

	t.Fatal("the action has no edit")

	return ""
}

func positionOffset(text string, p protocol.Position) int {
	off := 0
	for range p.Line {
		off += strings.IndexByte(text[off:], '\n') + 1
	}

	return off + int(p.Character)
}

func lifecycleCodeActions(t *testing.T, s *Server, docURI uri.URI) []protocol.CodeAction {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
	})

	got, err := s.handleCodeAction(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}

	actions, _ := got.([]protocol.CodeAction)

	return actions
}

func actionTitled(actions []protocol.CodeAction, title string) (*protocol.CodeAction, bool) {
	for i := range actions {
		if actions[i].Title == title {
			return &actions[i], true
		}
	}

	return nil, false
}

func TestLifecycleActionsAddTheEventsAnApplicationLacks(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	app := uri.File(filepath.Join(dir, "Application.cfc"))
	src := "component {\n\tthis.name = \"x\";\n\n\tpublic boolean function onApplicationStart() {\n\t\treturn true;\n\t}\n\n\tpublic void function ONREQUESTEND() {}\n}\n"
	openDoc(t, s, app, src)

	actions := lifecycleCodeActions(t, s, app)

	// What it declares, in any case, is not offered again.
	for _, declared := range []string{"onApplicationStart", "onRequestEnd"} {
		if _, ok := actionTitled(actions, "Add "+declared+"() to Application.cfc"); ok {
			t.Errorf("%s is offered though the component declares it", declared)
		}
	}

	one, ok := actionTitled(actions, "Add onSessionStart() to Application.cfc")
	if !ok {
		t.Fatal("onSessionStart is not offered")
	}

	got := applyActionEdit(t, src, one)

	want := "component {\n\tthis.name = \"x\";\n\n\tpublic boolean function onApplicationStart() {\n\t\treturn true;\n\t}\n\n\tpublic void function ONREQUESTEND() {}\n\n\tpublic void function onSessionStart() {\n\t\treturn;\n\t}\n}\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}

	all, ok := actionTitled(actions, "Add all missing application events to Application.cfc")
	if !ok {
		t.Fatal("the add-all action is missing")
	}

	filled := applyActionEdit(t, src, all)
	for _, st := range lifecycleStubs {
		if !strings.Contains(strings.ToLower(filled), "function "+strings.ToLower(st.name)+"(") {
			t.Errorf("%s is not in the filled component", st.name)
		}
	}

	if strings.Count(strings.ToLower(filled), "function onapplicationstart(") != 1 {
		t.Error("onApplicationStart was added again")
	}

	// And the result is a component that declares every event.
	declared := declaredFunctionNames(filled, nil)
	for _, st := range lifecycleStubs {
		if !declared[strings.ToLower(st.name)] {
			t.Errorf("the filled component does not parse as declaring %s", st.name)
		}
	}
}

func TestLifecycleActionsInATagComponent(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	app := uri.File(filepath.Join(dir, "application.cfc"))
	src := "<cfcomponent>\n\t<cfset this.name = \"x\">\n</cfcomponent>\n"
	openDoc(t, s, app, src)

	one, ok := actionTitled(lifecycleCodeActions(t, s, app), "Add onSessionStart() to Application.cfc")
	if !ok {
		t.Fatal("onSessionStart is not offered in a tag component")
	}

	want := "<cfcomponent>\n\t<cfset this.name = \"x\">\n\n\t<cffunction name=\"onSessionStart\" access=\"public\" returntype=\"void\">\n\t</cffunction>\n</cfcomponent>\n"
	if got := applyActionEdit(t, src, one); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestLifecycleActionsAreForAnApplicationOnly(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	other := uri.File(filepath.Join(dir, "Service.cfc"))
	openDoc(t, s, other, "component {\n}\n")

	for _, a := range lifecycleCodeActions(t, s, other) {
		if strings.HasPrefix(a.Title, "Add ") && strings.HasSuffix(a.Title, "to Application.cfc") {
			t.Errorf("%q offered outside an Application.cfc", a.Title)
		}
	}

	// And one that declares all of them has nothing to add.
	app := uri.File(filepath.Join(dir, "Application.cfc"))

	var full strings.Builder

	full.WriteString("component {\n")

	for _, st := range lifecycleStubs {
		full.WriteString(st.script + "\n")
	}

	full.WriteString("}\n")
	openDoc(t, s, app, full.String())

	for _, a := range lifecycleCodeActions(t, s, app) {
		if strings.HasSuffix(a.Title, "to Application.cfc") {
			t.Errorf("%q offered when every event is declared", a.Title)
		}
	}
}
