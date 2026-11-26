package parser

import (
	"strings"
	"unsafe"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
)

// columnCursor turns the byte offsets a parse meets, in the order it meets them,
// into UTF-16 columns on their lines.
//
// Measured from the start of the line each time, as nameRangeAt and
// linePrefixUTF16 do, a line holding many declarations (generated or minified
// code, a data component written on a line) costs the line once for each: 16,000
// functions on one line took two seconds to parse, on every edit. The cursor
// remembers the last offset and its column. An offset later on the same line
// adds only the stretch since the last one, and one on a later line measures
// only from that line's start, so a run of offsets in increasing order costs the
// text once; one a little earlier on the same line steps back the same way. An
// offset the cursor cannot extend from (another text, or back across a line
// break) is measured from its line's start, which is exactly what the cursor
// replaces: the answer is the same either way.
//
// Not safe for concurrent use: a parser owns one.
type columnCursor struct {
	src   string
	off   int
	col   int
	ready bool
}

// column is the UTF-16 column of offset in src, 0 <= offset <= len(src).
func (c *columnCursor) column(src string, offset int) int {
	// The same text is the same string, by where it is held: comparing the bytes
	// would cost the file on every call. The cursor holds c.src, so that address
	// is not reused for another string while it is compared with.
	if c.ready && len(src) == len(c.src) && unsafe.StringData(src) == unsafe.StringData(c.src) { //nolint:gosec // string identity, see above
		if offset >= c.off {
			since := src[c.off:offset]

			if _, after, crossed := strings.CutLast(since, "\n"); crossed {
				// A later line: its column is the text after its line break.
				c.col = lex.UTF16Len(after)
			} else {
				c.col += lex.UTF16Len(since)
			}

			c.off = offset

			return c.col
		}

		// Back along the same line, as a property asks for its start, its end
		// and then its name between them: UTF-16 lengths add, so the column
		// falls by the length of what is stepped over.
		if back := src[offset:c.off]; strings.IndexByte(back, '\n') < 0 {
			c.col -= lex.UTF16Len(back)
			c.off = offset

			return c.col
		}
	}

	lineStart := strings.LastIndexByte(src[:offset], '\n') + 1

	c.src, c.off, c.col, c.ready = src, offset, lex.UTF16Len(src[lineStart:offset]), true

	return c.col
}

// nameRange is nameRangeAt, with the column found from the last offset asked.
func (c *columnCursor) nameRange(src string, offset, line int, name string) (nameLine, nameCol, nameEndCol uint32) {
	if offset < 0 || offset > len(src) || name == "" || strings.IndexByte(name, '\n') >= 0 {
		return 0, 0, 0
	}

	col := c.column(src, offset)

	return conv.Uint32(line), conv.Uint32(col), conv.Uint32(col + lex.UTF16Len(name))
}

// prefix is linePrefixUTF16, with the column found from the last offset asked.
func (c *columnCursor) prefix(src string, offset int) uint32 {
	if offset <= 0 || offset > len(src) || src[offset-1] == '\n' {
		return 0
	}

	return conv.Uint32(c.column(src, offset))
}
