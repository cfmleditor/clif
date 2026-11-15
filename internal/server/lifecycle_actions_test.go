package server

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"

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
		Range:        protocol.Range{End: protocol.Position{Line: 1000}},
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
		// The events whose stub changes what a request does are left out.
		if changesRequests[strings.ToLower(st.name)] {
			continue
		}

		if !strings.Contains(strings.ToLower(filled), "function "+strings.ToLower(st.name)+"(") {
			t.Errorf("%s is not in the filled component", st.name)
		}
	}

	if strings.Count(strings.ToLower(filled), "function onapplicationstart(") != 1 {
		t.Error("onApplicationStart was added again")
	}

	// And the result is a component that declares every event the action adds.
	declared := declaredFunctionNames(filled)

	for _, st := range lifecycleStubs {
		if changesRequests[strings.ToLower(st.name)] {
			continue
		}

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

func TestLifecycleStubsFollowACRLFFile(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	app := uri.File(filepath.Join(dir, "Application.cfc"))
	src := "component {\r\n\tthis.name = \"x\";\r\n\r\n\tpublic void function ONREQUESTEND() {}\r\n}\r\n"
	openDoc(t, s, app, src)

	one, ok := actionTitled(lifecycleCodeActions(t, s, app), "Add onSessionStart() to Application.cfc")
	if !ok {
		t.Fatal("onSessionStart is not offered")
	}

	got := applyActionEdit(t, src, one)
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Errorf("a bare LF in the result: %q", got)
	}

	if strings.Contains(got, "\r\n\r\n\r\n") {
		t.Errorf("an extra blank line: %q", got)
	}
}

func TestLifecycleStubsUseTheFilesIndentation(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	app := uri.File(filepath.Join(dir, "Application.cfc"))
	src := "component {\n    this.name = \"x\";\n\n    public void function ONREQUESTEND() {}\n}\n"
	openDoc(t, s, app, src)

	one, ok := actionTitled(lifecycleCodeActions(t, s, app), "Add onSessionStart() to Application.cfc")
	if !ok {
		t.Fatal("onSessionStart is not offered")
	}

	got := applyActionEdit(t, src, one)
	if strings.Contains(got, "\t") {
		t.Errorf("a tab in a file indented with spaces:\n%s", got)
	}

	if !strings.Contains(got, "\n    public void function onSessionStart() {\n        return;\n    }\n") {
		t.Errorf("the stub is not indented as the file is:\n%s", got)
	}
}

func TestLifecycleStubsIgnoreADocCommentsIndentation(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	app := uri.File(filepath.Join(dir, "Application.cfc"))
	src := "/**\n * The app.\n */\ncomponent {\n\tthis.name = \"a\";\n}\n"
	openDoc(t, s, app, src)

	one, ok := actionTitled(lifecycleCodeActions(t, s, app), "Add onSessionStart() to Application.cfc")
	if !ok {
		t.Fatal("onSessionStart is not offered")
	}

	if got := applyActionEdit(t, src, one); !strings.Contains(got, "\n\tpublic void function onSessionStart() {\n\t\treturn;\n\t}\n") {
		t.Errorf("the stub is not tab-indented:\n%s", got)
	}
}

func TestLifecycleStubsBlankLineHandling(t *testing.T) {
	for name, c := range map[string]struct{ src, want string }{
		"an indented blank line before the closer": {
			"component {\n\tthis.name = \"a\";\n\t\n}\n",
			"component {\n\tthis.name = \"a\";\n\t\n\tpublic void function onSessionStart() {\n\t\treturn;\n\t}\n}\n",
		},
		"a closer on the line of code": {
			"component {\n\tthis.name = \"a\"; }\n",
			"component {\n\tthis.name = \"a\"; \n\n\tpublic void function onSessionStart() {\n\t\treturn;\n\t}\n}\n",
		},
	} {
		dir := t.TempDir()
		s := newTestServer()
		s.WorkspaceFolders = []string{dir}

		app := uri.File(filepath.Join(dir, "Application.cfc"))
		openDoc(t, s, app, c.src)

		one, ok := actionTitled(lifecycleCodeActions(t, s, app), "Add onSessionStart() to Application.cfc")
		if !ok {
			t.Fatalf("%s: onSessionStart is not offered", name)
		}

		if got := applyActionEdit(t, c.src, one); got != c.want {
			t.Errorf("%s:\ngot  %q\nwant %q", name, got, c.want)
		}
	}
}

func TestACFCommentHoldingTheClosingTagIsNotTheEnd(t *testing.T) {
	src := "<cfcomponent>\n</cfcomponent>\n<!--- </cfcomponent> --->\n"
	if end, tags, ok := parser.ComponentBodyEnd(src); !ok || !tags || end != strings.Index(src, "</cfcomponent>") {
		t.Errorf("end %d tags %v ok %v", end, tags, ok)
	}
}

func TestAnEventSetAsAPropertyIsDeclared(t *testing.T) {
	declared := declaredFunctionNames("component {\n\tthis.onError = function( e ) {};\n\tthis.onErrorLog = 1;\n}\n")
	if !declared["onerror"] {
		t.Error("this.onError = function is not counted as declared")
	}

	if got := declaredFunctionNames("component {\n\t// this.onSessionStart = function() {}\n\tx = \"this.onError = 1\";\n\ty = this.onAbort == 1;\n}\n"); got["onsessionstart"] || got["onerror"] || got["onabort"] {
		t.Errorf("a comment, a string and a comparison declared events: %v", got)
	}
}

func TestLifecycleStubsFollowTheIndentationOfTheFunctions(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	app := uri.File(filepath.Join(dir, "Application.cfc"))
	src := "<cfcomponent>\n<cffunction name=\"onApplicationStart\">\n\t<cfargument name=\"x\">\n</cffunction>\n</cfcomponent>\n"
	openDoc(t, s, app, src)

	one, ok := actionTitled(lifecycleCodeActions(t, s, app), "Add onSessionStart() to Application.cfc")
	if !ok {
		t.Fatal("onSessionStart is not offered")
	}

	got := applyActionEdit(t, src, one)
	if !strings.Contains(got, "\n<cffunction name=\"onSessionStart\"") || !strings.Contains(got, "\n\t<cfargument") {
		t.Errorf("the stub is not flush left like its siblings, with nested lines tab-indented:\n%s", got)
	}
}

// The events are offered where the component is declared or closed.
func TestLifecycleActionsAreOfferedOnTheDeclarationAndTheEnd(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	app := uri.File(filepath.Join(dir, "Application.cfc"))
	openDoc(t, s, app, "component {\n\tthis.name = \"a\";\n\n\tthis.x = 1;\n}\n")

	at := func(line uint32) int {
		raw := makeCall(t, protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: app},
			Range:        protocol.Range{Start: protocol.Position{Line: line}, End: protocol.Position{Line: line}},
		})

		got, err := s.handleCodeAction(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}

		actions, _ := got.([]protocol.CodeAction)
		n := 0

		for i := range actions {
			if strings.HasPrefix(actions[i].Title, "Add ") {
				n++
			}
		}

		return n
	}

	if at(0) == 0 || at(4) == 0 {
		t.Error("the events are not offered on the declaration and the closing brace")
	}

	if got := at(1); got != 0 {
		t.Errorf("%d events offered on a plain line", got)
	}
}

func TestTheBulkActionLeavesOutEventsThatChangeRequests(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	app := uri.File(filepath.Join(dir, "Application.cfc"))
	src := "component {\n\tthis.name = \"a\";\n}\n"
	openDoc(t, s, app, src)

	all, ok := actionTitled(lifecycleCodeActions(t, s, app), "Add all missing application events to Application.cfc")
	if !ok {
		t.Fatal("the bulk action is missing")
	}

	got := applyActionEdit(t, src, all)
	for _, risky := range []string{"onError", "onCFCRequest", "onRequest(", "onMissingTemplate", "onAbort"} {
		if strings.Contains(got, "function "+risky) {
			t.Errorf("%s is in the bulk action", risky)
		}
	}

	if !strings.Contains(got, "function onSessionStart") {
		t.Error("an inert event is missing from the bulk action")
	}

	// Each is still offered on its own.
	if _, ok := actionTitled(lifecycleCodeActions(t, s, app), "Add onError() to Application.cfc"); !ok {
		t.Error("onError is not offered on its own")
	}
}

func TestStubsAreWrittenWithBareCRLineBreaksWhenTheFileHasThem(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	app := uri.File(filepath.Join(dir, "Application.cfc"))
	raw := "component {\r\tthis.name = \"a\";\r}\r"
	open := makeCall(t, protocol.MethodTextDocumentDidOpen, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: app, LanguageID: "cfml", Version: 1, Text: raw},
	})

	if _, err := s.handleDidOpen(context.Background(), open); err != nil {
		t.Fatal(err)
	}

	one, ok := actionTitled(lifecycleCodeActions(t, s, app), "Add onSessionStart() to Application.cfc")
	if !ok {
		t.Fatal("onSessionStart is not offered")
	}

	for _, edits := range one.Edit.Changes {
		if strings.Contains(edits[0].NewText, "\n") {
			t.Errorf("a line feed in an edit to a CR-only file: %q", edits[0].NewText)
		}
	}
}

func TestAMemberNamedLikeAnEventDeclaresNothing(t *testing.T) {
	got := declaredFunctionNames("component {\n\tconfig.this.onError = handler;\n\tx.function onSessionStart( a ) {}\n}\n")
	if got["onerror"] || got["onsessionstart"] {
		t.Errorf("a member declared an event: %v", got)
	}
}
