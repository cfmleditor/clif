package server

import (
	"context"
	"testing"
	"time"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func partial(startLine, endLine uint32, text string) protocol.TextDocumentContentChangeEvent {
	return &protocol.TextDocumentContentChangePartial{
		Range: protocol.Range{
			Start: protocol.Position{Line: startLine},
			End:   protocol.Position{Line: endLine},
		},
		Text: text,
	}
}

// The index shifts the lines after an edit by the lines the changes added
// less the lines they replaced; a whole-document replacement has no range to
// measure and counts for nothing.
func TestLineDeltaIsLinesAddedLessLinesReplaced(t *testing.T) {
	for _, c := range []struct {
		name    string
		changes []protocol.TextDocumentContentChangeEvent
		want    int
	}{
		{"none", nil, 0},
		{"a word inside a line", []protocol.TextDocumentContentChangeEvent{partial(3, 3, "x")}, 0},
		{"a line break typed", []protocol.TextDocumentContentChangeEvent{partial(3, 3, "\n")}, 1},
		{"three lines pasted", []protocol.TextDocumentContentChangeEvent{partial(3, 3, "a\nb\nc\n")}, 3},
		{"two lines deleted", []protocol.TextDocumentContentChangeEvent{partial(3, 5, "")}, -2},
		{"two lines replaced by four", []protocol.TextDocumentContentChangeEvent{partial(3, 5, "a\nb\nc\nd\ne")}, 2},
		{"several changes add up", []protocol.TextDocumentContentChangeEvent{partial(1, 1, "\n"), partial(4, 6, ""), partial(9, 9, "\n\n")}, 1 - 2 + 2},
		{"a whole document counts nothing", []protocol.TextDocumentContentChangeEvent{&protocol.TextDocumentContentChangeWholeDocument{Text: "a\nb\nc"}}, 0},
	} {
		if got := lineDelta(c.changes); got != c.want {
			t.Errorf("%s: lineDelta = %d, want %d", c.name, got, c.want)
		}
	}
}

// A burst is a run of changes less than the window apart; the count restarts
// when the window is over.
func TestTheChangeCountRestartsAfterTheWindow(t *testing.T) {
	s := newTestServer()
	doc := uri.URI("file:///t.cfc")
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	for i, want := range []int{1, 2, 3} {
		if got := s.noteChange(doc, t0.Add(time.Duration(i)*50*time.Millisecond)); got != want {
			t.Errorf("change %d inside the window counted %d, want %d", i+1, got, want)
		}
	}

	// 200ms after the window opened is still inside it; past it, the count restarts.
	if got := s.noteChange(doc, t0.Add(200*time.Millisecond)); got != 4 {
		t.Errorf("a change at the window's edge counted %d, want 4", got)
	}

	if got := s.noteChange(doc, t0.Add(201*time.Millisecond)); got != 1 {
		t.Errorf("a change past the window counted %d, want 1", got)
	}

	if got := s.noteChange("file:///other.cfc", t0.Add(201*time.Millisecond)); got != 1 {
		t.Errorf("another document counted %d, want its own 1", got)
	}
}

func callDidChange(t *testing.T, srv *Server, docURI uri.URI, changes []protocol.TextDocumentContentChangeEvent) error {
	t.Helper()

	_, err := srv.handleDidChange(context.Background(), makeCall(t, protocol.MethodTextDocumentDidChange, protocol.DidChangeTextDocumentParams{
		TextDocument:   protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI}},
		ContentChanges: changes,
	}))

	return err
}

// A notification that cannot be read, names no change, is for a file that is
// not CFML or for a document the server never opened, leaves everything as it
// was.
func TestDidChangeIgnoresWhatItCannotApply(t *testing.T) {
	srv := newTestServer()
	doc := uri.URI("file:///ws/A.cfc")
	openDoc(t, srv, doc, "component {\n\tfunction a() {}\n}\n")

	if _, err := srv.handleDidChange(context.Background(), []byte("{not json")); err == nil {
		t.Error("an unreadable notification was accepted")
	}

	if err := callDidChange(t, srv, doc, nil); err != nil {
		t.Errorf("a notification with no changes failed: %v", err)
	}

	txt := uri.URI("file:///ws/notes.txt")
	if err := callDidChange(t, srv, txt, []protocol.TextDocumentContentChangeEvent{partial(0, 0, "x")}); err != nil {
		t.Errorf("a change to a file that is not CFML failed: %v", err)
	}

	if _, ok := srv.getDocument(txt); ok {
		t.Error("a file that is not CFML was stored")
	}

	never := uri.URI("file:///ws/Never.cfc")
	if err := callDidChange(t, srv, never, []protocol.TextDocumentContentChangeEvent{partial(0, 0, "x")}); err != nil {
		t.Errorf("a change to a document that was never opened failed: %v", err)
	}

	if _, ok := srv.getDocument(never); ok {
		t.Error("a document that was never opened was stored")
	}

	if got, _ := srv.getDocument(doc); got != "component {\n\tfunction a() {}\n}\n" {
		t.Errorf("the open document changed: %q", got)
	}
}

// A whole-document replacement of an open file replaces its text and its
// signatures: the old function is gone from the symbols and the index, the new
// one is in both.
func TestDidChangeWholeDocumentReplacesTheParse(t *testing.T) {
	srv := newTestServer()
	doc := uri.URI("file:///ws/A.cfc")
	openDoc(t, srv, doc, "component {\n\tfunction a() {}\n}\n")

	next := "component {\n\n\tfunction b() {}\n}\n"
	if err := callDidChange(t, srv, doc, []protocol.TextDocumentContentChangeEvent{&protocol.TextDocumentContentChangeWholeDocument{Text: next}}); err != nil {
		t.Fatal(err)
	}

	if got, _ := srv.getDocument(doc); got != next {
		t.Errorf("text = %q, want %q", got, next)
	}

	if names := symbolNames(t, srv, doc); len(names) != 1 || names[0] != "b@2" {
		t.Errorf("symbols = %v, want b on line 2", names)
	}

	if len(srv.index.Lookup("a")) != 0 || len(srv.index.Lookup("b")) != 1 {
		t.Errorf("index holds a:%d b:%d, want a gone and b indexed", len(srv.index.Lookup("a")), len(srv.index.Lookup("b")))
	}
}
