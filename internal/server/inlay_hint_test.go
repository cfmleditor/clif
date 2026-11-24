package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func inlayHints(t *testing.T, s *Server, docURI uri.URI, r protocol.Range) []protocol.InlayHint {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentInlayHint, protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Range:        r,
	})

	got, err := s.handleInlayHint(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}

	hints, _ := got.([]protocol.InlayHint)

	return hints
}

var wholeDocument = protocol.Range{End: protocol.Position{Line: 1000}}

func TestInlayHintShowsTheComponentsDotPath(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}
	s.Features.InlayHints = true

	user := uri.File(filepath.Join(dir, "models", "User.cfc"))
	openDoc(t, s, user, "/**\n * A user. {x}\n */\ncomponent extends=\"models.Base\" {\n\tfunction init() {}\n}\n")

	hints := inlayHints(t, s, user, wholeDocument)
	if len(hints) != 1 {
		t.Fatalf("%d hints, want 1: %+v", len(hints), hints)
	}

	h := hints[0]
	if label, _ := h.Label.(protocol.String); label != "Path: (models.User)" {
		t.Errorf("label %q, want %q", h.Label, "Path: (models.User)")
	}

	// Before the brace, after the attributes: line 3, past `component extends="models.Base"` (31 columns).
	if h.Position.Line != 3 || h.Position.Character != 31 {
		t.Errorf("at %+v, want line 3 column 31", h.Position)
	}

	// A range that does not hold the declaration holds no hint.
	if got := inlayHints(t, s, user, protocol.Range{End: protocol.Position{Line: 1}}); len(got) != 0 {
		t.Errorf("%d hints outside the range", len(got))
	}
}

func TestInlayHintNeedsTheFeatureAndAComponent(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	user := uri.File(filepath.Join(dir, "User.cfc"))
	openDoc(t, s, user, "component {\n}\n")

	if got := inlayHints(t, s, user, wholeDocument); len(got) != 0 {
		t.Errorf("%d hints with the feature off", len(got))
	}

	s.Features.InlayHints = true

	app := uri.File(filepath.Join(dir, "Application.cfc"))
	openDoc(t, s, app, "component {\n}\n")

	page := uri.File(filepath.Join(dir, "index.cfm"))
	openDoc(t, s, page, "<cfcomponent>\n</cfcomponent>\n")

	for _, u := range []uri.URI{app, page} {
		if got := inlayHints(t, s, u, wholeDocument); len(got) != 0 {
			t.Errorf("%s has %d hints, want none", u, len(got))
		}
	}

	if got := inlayHints(t, s, user, wholeDocument); len(got) != 1 {
		t.Errorf("%d hints for a component with the feature on", len(got))
	}
}

func TestInlayHintIsAdvertisedOnlyWhenOn(t *testing.T) {
	s := newTestServer()
	if p, _ := s.capabilities().InlayHintProvider.(protocol.Boolean); bool(p) {
		t.Error("inlay hints advertised with the feature off")
	}

	s.Features.InlayHints = true

	if p, _ := s.capabilities().InlayHintProvider.(protocol.Boolean); !bool(p) {
		t.Error("inlay hints not advertised with the feature on")
	}
}

func TestInlayHintOfATagComponentFollowsTheTag(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}
	s.Features.InlayHints = true

	user := uri.File(filepath.Join(dir, "models", "User.cfc"))
	openDoc(t, s, user, "<cfcomponent output=\"false\">\n</cfcomponent>\n")

	hints := inlayHints(t, s, user, wholeDocument)
	if len(hints) != 1 || hints[0].Position.Line != 0 || hints[0].Position.Character != uint32(len("<cfcomponent output=\"false\">")) {
		t.Fatalf("hints %+v, want one after the tag's >", hints)
	}

	closed := uri.File(filepath.Join(dir, "models", "Closed.cfc"))
	openDoc(t, s, closed, "<cfcomponent />\n")

	if got := inlayHints(t, s, closed, wholeDocument); len(got) != 0 {
		t.Errorf("a self-closed component got %d hints", len(got))
	}
}

func TestInlayHintOfABraceOnItsOwnLineSitsOnTheDeclarationLine(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}
	s.Features.InlayHints = true

	user := uri.File(filepath.Join(dir, "models", "User.cfc"))
	openDoc(t, s, user, "component extends=\"Base\"\n{\n}\n")

	hints := inlayHints(t, s, user, wholeDocument)
	if len(hints) != 1 || hints[0].Position.Line != 0 || hints[0].Position.Character != uint32(len("component extends=\"Base\"")) {
		t.Errorf("hints %+v, want one at the end of line 0", hints)
	}
}

func TestInlayHintComesBeforeATrailingComment(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}
	s.Features.InlayHints = true

	user := uri.File(filepath.Join(dir, "models", "User.cfc"))
	openDoc(t, s, user, "component extends=\"Base\" // legacy\n{\n}\n")

	hints := inlayHints(t, s, user, wholeDocument)
	if len(hints) != 1 || hints[0].Position.Character != uint32(len("component extends=\"Base\"")) {
		t.Errorf("hints %+v, want one before the comment", hints)
	}
}

func TestInlayHintOfAFileUnderNoRootIsSkipped(t *testing.T) {
	root := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{root}
	s.Features.InlayHints = true

	outside := uri.File(filepath.Join(t.TempDir(), "Stray.cfc"))
	openDoc(t, s, outside, "component {\n}\n")

	if got := inlayHints(t, s, outside, wholeDocument); len(got) != 0 {
		t.Errorf("a file outside every root got %d hints", len(got))
	}
}

func TestInlayHintShowsParameterNamesForCalls(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}
	s.Features.InlayHints = true

	page := uri.File(filepath.Join(dir, "index.cfm"))
	openDoc(t, s, page, `<cfscript>
function greet(required string name, numeric times = 1) {}
greet("Ada", 3);
greet(name = "Ada", times = 2);
greet(name: "Ada");
mid("hello", 1, 3);
if (true) {
	greet("Bob");
}
</cfscript>
<cfset x = greet("Cara", 2) />
`)

	hints := inlayHints(t, s, page, wholeDocument)

	var got []string

	for _, h := range hints {
		if h.Kind != protocol.InlayHintKindParameter {
			continue
		}

		label, _ := h.Label.(protocol.String)
		got = append(got, string(label))
	}

	want := []string{
		"name:", "times:", // greet("Ada", 3)
		"string:", "start:", "count:", // mid("hello", 1, 3)
		"name:",           // greet("Bob")
		"name:", "times:", // <cfset x = greet("Cara", 2) />
	}

	if len(got) != len(want) {
		t.Fatalf("parameter hints = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("hint[%d] = %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}

func TestInlayHintShowsParameterNamesForComponentAndConstructorCalls(t *testing.T) {
	dir := t.TempDir()
	userPath := filepath.Join(dir, "models", "User.cfc")
	userSrc := "component {\n\tfunction init(required string email, boolean active = true) {}\n\tfunction rename(required string fullName) {}\n}\n"

	if err := os.MkdirAll(filepath.Dir(userPath), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(userPath, []byte(userSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	s := newTestServer()
	s.WorkspaceFolders = []string{dir}
	s.Features.InlayHints = true

	userURI := uri.File(userPath)
	openDoc(t, s, userURI, userSrc)

	callerURI := uri.File(filepath.Join(dir, "service.cfc"))
	callerSrc := "component {\n\tfunction run() {\n\t\tvar u = new models.User(\"a@b.com\", false);\n\t\tu.rename(\"Ada\");\n\t}\n}\n"
	openDoc(t, s, callerURI, callerSrc)

	hints := inlayHints(t, s, callerURI, wholeDocument)

	var paramLabels []string

	for _, h := range hints {
		if h.Kind == protocol.InlayHintKindParameter {
			label, _ := h.Label.(protocol.String)
			paramLabels = append(paramLabels, string(label))
		}
	}

	want := []string{"email:", "active:", "fullName:"}
	if len(paramLabels) != len(want) {
		t.Fatalf("param hints = %v, want %v (all hints: %+v)", paramLabels, want, hints)
	}

	for i := range want {
		if paramLabels[i] != want[i] {
			t.Errorf("hint[%d] = %q, want %q", i, paramLabels[i], want[i])
		}
	}

	// Range filtering: requesting only line 3 returns only `fullName:`.
	line3 := inlayHints(t, s, callerURI, protocol.Range{
		Start: protocol.Position{Line: 3, Character: 0},
		End:   protocol.Position{Line: 3, Character: 40},
	})
	if len(line3) != 1 || line3[0].Label != protocol.String("fullName:") {
		t.Errorf("line 3 hints = %+v, want single fullName: hint", line3)
	}
}

func TestInlayHintSubSwitchesGateEachKindIndependently(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}
	s.Features.InlayHints = true

	u := uri.File(filepath.Join(dir, "models", "User.cfc"))
	openDoc(t, s, u, "component {\n\tfunction run() {\n\t\treturn mid(\"hello\", 1, 2);\n\t}\n}\n")

	// Both on (default when InlayHints is enabled): 1 component path + 3 param hints.
	both := inlayHints(t, s, u, wholeDocument)
	if len(both) != 4 {
		t.Fatalf("both enabled = %d hints (%+v), want 4", len(both), both)
	}

	// Parameter names off: only component path hint.
	s.Features.InlayHintsParameterNames = false
	pathOnly := inlayHints(t, s, u, wholeDocument)

	if len(pathOnly) != 1 || pathOnly[0].Label != protocol.String("Path: (models.User)") {
		t.Fatalf("path only = %+v, want single Path hint", pathOnly)
	}

	// Component path off, parameter names on: only the 3 param hints.
	s.Features.InlayHintsParameterNames = true
	s.Features.InlayHintsComponentPath = false
	paramsOnly := inlayHints(t, s, u, wholeDocument)

	if len(paramsOnly) != 3 {
		t.Fatalf("params only = %d hints (%+v), want 3", len(paramsOnly), paramsOnly)
	}
}
