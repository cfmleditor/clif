package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// LineTable is the byte offset at which each line of a text starts: entry 0
// is always 0, and every '\n' starts another line, so a text ending in a
// newline has an empty last line and the empty text has one line. Text is
// normalised before it gets here (lex.NormalizeLineBreaks), so a bare CR is
// already a newline and only '\n' is counted.
//
// It is the one line table the parser, the resolver and the server build:
// six copies of it once answered the same question with different types and
// edge rules, and line_table_test.go holds this one to each of them.
//
// int32 rather than int: this is one entry per line of every file parsed,
// built several times over, and it was the single largest allocation site in
// a tag parse. A CFML file large enough to overflow int32 is 2GB of source,
// which no engine would load and this parser holds entirely in memory anyway.
type LineTable []int32

// NewLineTable builds the line table of text.
//
// Both passes go through the stdlib's byte scanners rather than a loop over
// every byte: strings.Count and strings.IndexByte are vectorised, and this runs
// once per parsed file over the whole source, which made it 3.9% of tag parsing
// in a profile. The count pass stays: sizing the slice up front is what keeps
// the fill pass from reallocating a dozen times on a large file.
func NewLineTable(text string) LineTable {
	idx := make(LineTable, 1, strings.Count(text, "\n")+1)

	for off := 0; off < len(text); {
		i := strings.IndexByte(text[off:], '\n')
		if i < 0 {
			break
		}

		off += i + 1

		idx = append(idx, conv.Int32(off))
	}

	return idx
}

// Edited is the table of the text made from t's by replacing the removed
// bytes at off with inserted. The lines before the edit keep their starts,
// those after it move by the change in length, and only inserted is read: an
// edit inside a function used to build the whole document's table again on
// every keystroke.
func (t LineTable) Edited(off, removed int, inserted string) LineTable {
	first := t.LineOf(off) + 1
	last := t.LineOf(off + removed)
	shift := len(inserted) - removed

	out := make(LineTable, 0, len(t)-(last+1-first)+strings.Count(inserted, "\n"))
	out = append(out, t[:first]...)

	for at := 0; ; {
		i := strings.IndexByte(inserted[at:], '\n')
		if i < 0 {
			break
		}

		at += i + 1
		out = append(out, conv.Int32(off+at))
	}

	for _, start := range t[last+1:] {
		out = append(out, conv.Int32(int(start)+shift))
	}

	return out
}

// Lines is the number of lines in the text: one more than its newlines.
func (t LineTable) Lines() int { return len(t) }

// LineOf is the 0-based line holding byte offset off: the number of line
// starts at or before off, less one. That is strings.Count(text[:off], "\n")
// for 0 <= off <= len(text); an offset past the end is on the last line, and
// a negative one answers -1.
func (t LineTable) LineOf(off int) int {
	lo, hi := 0, len(t)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if int(t[mid]) <= off {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	return lo - 1
}

// Start is the byte offset at which 0-based line n starts, and false when the
// text has no line n.
func (t LineTable) Start(n int) (int, bool) {
	if n < 0 || n >= len(t) {
		return 0, false
	}

	return int(t[n]), true
}

// Span is the byte range of 0-based lines startLine through endLine of a text
// size bytes long: start is where startLine begins and end is just past
// endLine's newline, or size when endLine is the last line or beyond it. A
// negative startLine reads as 0, an endLine before startLine gives the empty
// range at start, and a startLine past the last line answers -1, -1.
func (t LineTable) Span(startLine, endLine, size int) (start, end int) {
	if startLine < 0 {
		startLine = 0
	}

	if startLine >= len(t) {
		return -1, -1
	}

	start = int(t[startLine])

	switch {
	case endLine < startLine:
		end = start
	case endLine+1 < len(t):
		end = int(t[endLine+1])
	default:
		end = size
	}

	return start, end
}
