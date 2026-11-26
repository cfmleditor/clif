package server

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

// utf16Columns turns byte offsets of one text into LSP columns, which count
// UTF-16 code units from the start of the line.
//
// Measured from the line's start for every offset, a long line (a data literal
// written on one line, minified or generated code) made any pass that converts
// many offsets on it quadratic: one measurement of the line per offset. A line
// of ASCII, which is nearly every line, has a column that is the byte distance,
// so whether a line is ASCII is learned once and kept. A line with a
// multi-byte character is measured exactly, as before.
type utf16Columns struct {
	content string
	lines   parser.LineTable
	ascii   map[int]bool
	lineLen map[int]uint32
}

func newUTF16Columns(content string, lines parser.LineTable) *utf16Columns {
	return &utf16Columns{content: content, lines: lines, ascii: map[int]bool{}, lineLen: map[int]uint32{}}
}

// lineLength is the length of line in UTF-16 units, as an LSP range that ends
// at the line's end counts it: the text without its line break, and without the
// "\r" of a CRLF one. Kept per line: a pass that asks for it once per symbol
// (the outline gives each function the end of its last line) measured a line
// again for every function on it.
func (c *utf16Columns) lineLength(line int) uint32 {
	if n, ok := c.lineLen[line]; ok {
		return n
	}

	n := conv.Uint32(lex.UTF16Len(lineTextFromTable(c.content, c.lines, line)))
	c.lineLen[line] = n

	return n
}

// column is the column of byte offset off, which lies on line.
func (c *utf16Columns) column(line, off int) int {
	lineStart := int(c.lines[line])

	ascii, known := c.ascii[line]
	if !known {
		ascii = isASCII(c.content[lineStart:lineEnd(c.content, lineStart)])
		c.ascii[line] = ascii
	}

	if ascii {
		return off - lineStart
	}

	return lex.UTF16Len(c.content[lineStart:off])
}

// lineEnd is the offset of the newline that ends the line starting at start, or
// the end of content.
func lineEnd(content string, start int) int {
	if i := strings.IndexByte(content[start:], '\n'); i >= 0 {
		return start + i
	}

	return len(content)
}

// isASCII reports whether s holds no byte of a multi-byte character.
func isASCII(s string) bool {
	for i := range len(s) {
		if s[i] >= 0x80 {
			return false
		}
	}

	return true
}
