package parser

import (
	"math/rand"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/conv"
)

// LineTable replaced six line tables, each written for its own package. The
// definitions below are those copies as they were, so every answer the table
// gives is held to the one it replaced over generated texts, including the
// edges a fixture rarely spells: the empty text, a text with no trailing
// newline, a leading newline and runs of consecutive newlines.

// refLineStartTable is the ParseResult's lineStartTable (result.go).
func refLineStartTable(content string) []int {
	starts := make([]int, 1, strings.Count(content, "\n")+1)

	for i := 0; ; {
		idx := strings.IndexByte(content[i:], '\n')
		if idx < 0 {
			return starts
		}

		i += idx + 1
		starts = append(starts, i)
	}
}

// refOffsetsFromTable is offsetsFromTable (result.go), behind lineOffsets.
func refOffsetsFromTable(starts []int, size, startLine, endLine int) (start, end int) {
	if startLine < 0 {
		startLine = 0
	}

	if startLine >= len(starts) {
		return -1, -1
	}

	start = starts[startLine]

	switch {
	case endLine < startLine:
		end = start
	case endLine+1 < len(starts):
		end = starts[endLine+1]
	default:
		end = size
	}

	return start, end
}

// refParseResultLineAt is (*ParseResult).lineAt (result.go), and the same
// search collectWrites' functionAt made over the int32 table.
func refParseResultLineAt(starts []int, off int) int {
	return sort.Search(len(starts), func(i int) bool { return starts[i] > off }) - 1
}

// refBuildLineIdx is buildLineIdx (cfparser.go).
func refBuildLineIdx(src string) []int32 {
	idx := make([]int32, 1, strings.Count(src, "\n")+1)

	for off := 0; off < len(src); {
		i := strings.IndexByte(src[off:], '\n')
		if i < 0 {
			break
		}

		off += i + 1

		idx = append(idx, conv.Int32(off))
	}

	return idx
}

// refLineAtOffset is lineAtOffset (cfparser.go), and (*tagParser).lineAt
// (tag_parser.go), which was the same search written out again.
func refLineAtOffset(idx []int32, offset int) int {
	lo, hi := 0, len(idx)
	for lo < hi {
		mid := (lo + hi) / 2
		if int(idx[mid]) <= offset {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	return lo - 1
}

// refLineStartsOf and refLineOffset are lineStartsOf and lineOffset
// (internal/resolve/guarded_call.go).
func refLineStartsOf(content string) []int {
	starts := []int{0}

	for off := 0; ; {
		i := strings.IndexByte(content[off:], '\n')
		if i < 0 {
			return starts
		}

		off += i + 1
		starts = append(starts, off)
	}
}

func refLineOffset(starts []int, n int) (int, bool) {
	if n < 0 || n >= len(starts) {
		return 0, false
	}

	return starts[n], true
}

// refNewLineIndex and refLineOf are newLineIndex and lineIndex.lineOf
// (internal/resolve/loop_element.go): the offsets of the newlines rather than
// of the line starts.
func refNewLineIndex(s string) []int {
	var idx []int

	for at := 0; ; {
		i := strings.IndexByte(s[at:], '\n')
		if i < 0 {
			return idx
		}

		idx = append(idx, at+i)
		at += i + 1
	}
}

func refLineOf(l []int, offset int) int {
	return sort.SearchInts(l, offset)
}

func lineTableTexts() []string {
	texts := []string{"", "a", "\n", "\n\n", "\n\n\n", "a\n", "\na", "a\nb", "a\nb\n", "a\n\n\nb", "no newline", "x = 1;\ny = f(\n  2 );\nz = y\n"}

	rng := rand.New(rand.NewSource(7))
	alphabet := []byte("ab\n\n \t")

	for range 1500 {
		var b strings.Builder
		for range rng.Intn(60) {
			b.WriteByte(alphabet[rng.Intn(len(alphabet))])
		}

		texts = append(texts, b.String())
	}

	return texts
}

func TestLineTableMatchesTheTablesItReplaced(t *testing.T) {
	for _, text := range lineTableTexts() {
		table := NewLineTable(text)
		starts := refLineStartTable(text)
		idx := refBuildLineIdx(text)
		resolveStarts := refLineStartsOf(text)
		breaks := refNewLineIndex(text)

		if table.Lines() != len(starts) || len(idx) != len(starts) || len(resolveStarts) != len(starts) {
			t.Fatalf("%q: %d lines, want %d (lineStartTable), %d (buildLineIdx), %d (lineStartsOf)", text, table.Lines(), len(starts), len(idx), len(resolveStarts))
		}

		for i := range starts {
			if int(table[i]) != starts[i] || table[i] != idx[i] || resolveStarts[i] != starts[i] {
				t.Fatalf("%q: entry %d is %d, want %d", text, i, table[i], starts[i])
			}
		}

		for off := -2; off <= len(text)+2; off++ {
			got := table.LineOf(off)

			if want := refLineAtOffset(idx, off); got != want {
				t.Fatalf("%q: LineOf(%d) = %d, lineAtOffset %d", text, off, got, want)
			}

			if want := refParseResultLineAt(starts, off); got != want {
				t.Fatalf("%q: LineOf(%d) = %d, ParseResult.lineAt %d", text, off, got, want)
			}

			// lineOf answered 0 for a negative offset where the line-start
			// searches answer -1; its callers only ever pass an offset in
			// the text, where the two agree.
			if off >= 0 {
				if want := refLineOf(breaks, off); got != want {
					t.Fatalf("%q: LineOf(%d) = %d, lineIndex.lineOf %d", text, off, got, want)
				}
			}

			if off >= 0 && off <= len(text) {
				if want := strings.Count(text[:off], "\n"); got != want {
					t.Fatalf("%q: LineOf(%d) = %d, strings.Count %d", text, off, got, want)
				}
			}
		}

		for n := -2; n <= len(starts)+2; n++ {
			gotAt, gotOK := table.Start(n)
			if wantAt, wantOK := refLineOffset(resolveStarts, n); gotAt != wantAt || gotOK != wantOK {
				t.Fatalf("%q: Start(%d) = %d, %v; lineOffset %d, %v", text, n, gotAt, gotOK, wantAt, wantOK)
			}

			for m := -2; m <= len(starts)+2; m++ {
				gotStart, gotEnd := table.Span(n, m, len(text))
				if wantStart, wantEnd := refOffsetsFromTable(starts, len(text), n, m); gotStart != wantStart || gotEnd != wantEnd {
					t.Fatalf("%q: Span(%d, %d) = %d, %d; offsetsFromTable %d, %d", text, n, m, gotStart, gotEnd, wantStart, wantEnd)
				}
			}
		}
	}
}

// The ParseResult keeps its table for the content it was built from, and
// builds again once Content changes.
func TestParseResultLineTableFollowsContent(t *testing.T) {
	pr := &ParseResult{Content: "a\nb\n"}

	if got := pr.lineAt(2); got != 1 {
		t.Fatalf("lineAt(2) = %d, want 1", got)
	}

	pr.Content = "a\n\n\nb"

	if got := pr.lineAt(4); got != 3 {
		t.Fatalf("after the edit, lineAt(4) = %d, want 3", got)
	}

	if start, end := pr.lineOffsets(1, 2); start != 2 || end != 4 {
		t.Fatalf("lineOffsets(1, 2) = %d, %d, want 2, 4", start, end)
	}
}

// An edited table is the table of the edited text, for edits that add, remove
// and keep lines anywhere in it, including at its ends.
func TestEditedLineTableIsTheEditedTextsTable(t *testing.T) {
	r := rand.New(rand.NewSource(7))

	pieces := []string{"", "a", "\n", "ab\n", "\n\n", "x\ny", "é\n"}
	text := "line one\nline two\n\nline four"

	for range 2000 {
		off := r.Intn(len(text) + 1)
		removed := r.Intn(len(text) - off + 1)
		inserted := pieces[r.Intn(len(pieces))] + pieces[r.Intn(len(pieces))]

		before := NewLineTable(text)
		edited := text[:off] + inserted + text[off+removed:]

		got := before.Edited(off, removed, inserted)
		if want := NewLineTable(edited); !slices.Equal(got, want) {
			t.Fatalf("replacing %d bytes at %d of %q with %q: %v, want %v", removed, off, text, inserted, got, want)
		}

		text = edited
		if len(text) > 400 {
			text = text[:200]
		}
	}
}

// An edit inside a function carries the parse's line table over rather than
// building it again, and the table is still the text's.
func TestAnEditInAFunctionKeepsTheLineTable(t *testing.T) {
	pr := ParseWithOptions("file:///t.cfc", "component {\n\tfunction a() {\n\t\tx = 1;\n\t}\n}\n", &ParseOptions{})

	for i, e := range []struct {
		line, char int
		text       string
	}{{2, 8, "2"}, {2, 9, "\n\t\ty = 3;"}, {3, 0, "\t"}} {
		pr.ApplyEdit(e.line, e.char, e.line, e.char, e.text)

		pr.lineStartsMu.Lock()
		table, of := pr.lineStarts, pr.lineStartsContent
		pr.lineStartsMu.Unlock()

		if of != pr.Content {
			t.Fatalf("edit %d: the table is not the edited text's; it was built again or dropped", i)
		}

		if want := NewLineTable(pr.Content); !slices.Equal(table, want) {
			t.Fatalf("edit %d: table %v, want %v", i, table, want)
		}
	}
}
