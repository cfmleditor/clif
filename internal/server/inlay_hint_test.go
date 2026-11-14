package server

import (
	"context"
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
