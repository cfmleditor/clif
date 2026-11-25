package server

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func selectionRanges(t *testing.T, s *Server, docURI uri.URI, positions ...protocol.Position) []protocol.SelectionRange {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentSelectionRange, protocol.SelectionRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Positions:    positions,
	})

	got, err := s.handleSelectionRange(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}

	ranges, _ := got.([]protocol.SelectionRange)

	return ranges
}

func flattenSelectionTexts(content string, sr protocol.SelectionRange) []string {
	lines := parser.NewLineTable(content)

	var out []string

	for cur := &sr; cur != nil; cur = cur.Parent {
		start := positionToByteOffset(content, lines, cur.Range.Start)
		end := positionToByteOffset(content, lines, cur.Range.End)
		out = append(out, content[start:end])
	}

	return out
}

func TestSelectionRangeIsAdvertised(t *testing.T) {
	s := newTestServer()
	if p, _ := s.capabilities().SelectionRangeProvider.(protocol.Boolean); !bool(p) {
		t.Error("selectionRangeProvider is not advertised")
	}
}

func TestSelectionRangeExpandsAcrossCFScriptConstructs(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	docURI := uri.File(filepath.Join(dir, "User.cfc"))
	src := "component {\n\tfunction greet(name) {\n\t\treturn trim(\"hello\");\n\t}\n}\n"
	openDoc(t, s, docURI, src)

	// Cursor inside "hello" on line 2 (0-based), col 16 ('e').
	got := selectionRanges(t, s, docURI, protocol.Position{Line: 2, Character: 16})
	if len(got) != 1 {
		t.Fatalf("got %d selection ranges, want 1", len(got))
	}

	texts := flattenSelectionTexts(src, got[0])
	// Should expand strictly through: word -> string -> ("hello") -> function body -> component body -> document.
	wantContains := []string{
		"hello",
		`"hello"`,
		`("hello")`,
		src,
	}

	for _, w := range wantContains {
		if !slices.Contains(texts, w) {
			t.Errorf("selection chain %q missing %q", texts, w)
		}
	}
}

func TestSelectionRangeExpandsAcrossCFMLTagsAndInterpolation(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	docURI := uri.File(filepath.Join(dir, "page.cfm"))
	src := "<cfoutput>\n\t<div>#user.getName()#</div>\n</cfoutput>\n"
	openDoc(t, s, docURI, src)

	// Cursor on `getName` on line 1, col 13.
	got := selectionRanges(t, s, docURI, protocol.Position{Line: 1, Character: 13})
	if len(got) != 1 {
		t.Fatalf("got %d selection ranges, want 1", len(got))
	}

	texts := flattenSelectionTexts(src, got[0])
	wantContains := []string{
		"getName",
		"user.getName()",
		"#user.getName()#",
		"<div>#user.getName()#</div>",
		"<cfoutput>\n\t<div>#user.getName()#</div>\n</cfoutput>",
	}

	for _, w := range wantContains {
		if !slices.Contains(texts, w) {
			t.Errorf("selection chain %q missing %q", texts, w)
		}
	}
}

func TestSelectionRangePathologicalAndBoundaryInputs(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()

	// Empty document & out-of-range positions must never panic.
	emptyURI := uri.File(filepath.Join(dir, "empty.cfm"))
	openDoc(t, s, emptyURI, "")

	if got := selectionRanges(t, s, emptyURI, protocol.Position{Line: 99, Character: 99}); len(got) != 1 {
		t.Fatalf("empty doc returned %d ranges, want 1", len(got))
	}

	// Thousands of unmatched closing tags and brackets must stay O(N).
	mismatched := strings.Repeat("<div>", 1500) + strings.Repeat("</span>", 1500) +
		"<cfscript>" + strings.Repeat("(", 1500) + strings.Repeat("]", 1500) + "</cfscript>"
	badURI := uri.File(filepath.Join(dir, "bad.cfm"))
	openDoc(t, s, badURI, mismatched)

	if got := selectionRanges(t, s, badURI, protocol.Position{Line: 0, Character: 10}); len(got) != 1 {
		t.Fatalf("mismatched doc returned %d ranges, want 1", len(got))
	}
}

// A range's columns count UTF-16 units, on a line with multi-byte characters
// (the emoji is two) and on one without, in the same document.
func TestSelectionRangeColumnsCountUTF16(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	prefix := `x = "é😀"; f(`
	src := "component {\nfunction f(a) {}\nfunction run() {\n" + prefix + "bar);\ny = f(baz);\n}\n}\n"
	u := uri.File(filepath.Join(dir, "Cols.cfc"))
	openDoc(t, s, u, src)

	wide := uint32(len(utf16.Encode([]rune(prefix))))
	plain := uint32(len(utf16.Encode([]rune("y = f("))))

	for _, c := range []struct {
		name string
		pos  protocol.Position
		want protocol.Range
	}{
		{"after multi-byte characters", protocol.Position{Line: 3, Character: wide + 1}, protocol.Range{
			Start: protocol.Position{Line: 3, Character: wide}, End: protocol.Position{Line: 3, Character: wide + 3},
		}},
		{"on a plain line", protocol.Position{Line: 4, Character: plain + 1}, protocol.Range{
			Start: protocol.Position{Line: 4, Character: plain}, End: protocol.Position{Line: 4, Character: plain + 3},
		}},
	} {
		got := selectionRanges(t, s, u, c.pos)
		if len(got) != 1 || got[0].Range != c.want {
			t.Errorf("%s: innermost range = %+v, want %+v", c.name, got, c.want)
		}
	}
}

// One long line of nested calls: a chain of ranges as long as the nesting, each
// converted from a byte offset to a column. The column of each used to be
// measured from the line's start, so the cost grew with the square of the
// nesting: 16,000 levels took 1.35 s and take milliseconds now.
func TestSelectionRangeOnALongLineIsLinear(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	const n = 16000

	src := "component {\nfunction f(a) {}\nfunction run() {\nx = " + strings.Repeat("f(", n) + "1" + strings.Repeat(")", n) + ";\n}\n}\n"
	u := uri.File(filepath.Join(dir, "Deep.cfc"))
	openDoc(t, s, u, src)

	start := time.Now()
	got := selectionRanges(t, s, u, protocol.Position{Line: 3, Character: uint32(4 + 2*n)})

	if took := time.Since(start); took > 500*time.Millisecond {
		t.Errorf("a %d-deep chain took %v, want well under a second", n, took)
	}

	depth := 0
	for r := got[0].Parent; r != nil; r = r.Parent {
		depth++
	}

	if depth < n {
		t.Errorf("chain is %d ranges deep, want at least one per call (%d)", depth, n)
	}
}
