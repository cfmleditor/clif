package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

// A wildPattern is a name spelled in part: literal pieces, in order, with what
// is computed between them matching a run of characters. It replaces the
// regular expressions built for that from QuoteMeta and a wildcard: invoke's
// method selectors, ColdBox's interception points and Mura's template paths.
// A run that matches is anything at all but a newline, one character or more
// (a selector); a run of word characters, none or more (an interception
// point); or a run of anything but a slash, none or more (a path segment).

type wildKind uint8

const (
	wildLine    wildKind = iota // `.+`
	wildWord                    // `\w*`
	wildSegment                 // `[^/]*`
)

type wildPattern struct {
	// pieces are the literals; a wildcard stands between each two, so there is
	// one more piece than wildcards, and the first and last may be empty.
	pieces []string
	kind   wildKind
	// atSegment lets the pattern begin at the start of the text or just after
	// a slash, and ends it at the end: Mura finds a template under each
	// display-object directory, so the path's tail is what is matched.
	// Otherwise the pattern matches the whole text.
	atSegment bool
}

// accepts is the test of a byte that may stand in a wildcard's run. They are
// functions of the package, not method values, so that choosing one allocates
// nothing on the path of every match.
func (k wildKind) accepts() func(c byte) bool {
	switch k {
	case wildWord:
		return acceptWord
	case wildSegment:
		return acceptSegment
	default:
		return acceptLine
	}
}

func acceptWord(c byte) bool    { return lex.IsWord(c) }
func acceptSegment(c byte) bool { return c != '/' }
func acceptLine(c byte) bool    { return c != '\n' }

// least is how many characters a wildcard's run holds at the fewest.
func (k wildKind) least() int {
	if k == wildLine {
		return 1
	}

	return 0
}

// match reports whether s fits the pattern, ignoring case.
func (p *wildPattern) match(s string) bool {
	m := lex.Wild{Accept: p.kind.accepts(), Least: p.kind.least(), EndAnchored: true}

	var memo lex.WildMemo

	if !p.atSegment {
		_, ok := m.MatchAt(s, p.pieces, 0, &memo, false)

		return ok
	}

	for start := 0; start <= len(s); start++ {
		if start == 0 || s[start-1] == '/' {
			if _, ok := m.MatchAt(s, p.pieces, start, &memo, false); ok {
				return true
			}
		}
	}

	return false
}

// wildBuilder collects the literal and computed parts of a spelled name.
type wildBuilder struct {
	pieces   []string
	cur      strings.Builder
	lastWild bool
	literal  bool
}

// wild notes a computed part; two in a row are one.
func (b *wildBuilder) wild() {
	if b.lastWild {
		return
	}

	b.pieces = append(b.pieces, b.cur.String())
	b.cur.Reset()

	b.lastWild = true
}

// text notes a literal part.
func (b *wildBuilder) text(s string) {
	if s == "" {
		return
	}

	b.cur.WriteString(s)

	b.lastWild, b.literal = false, true
}

// empty reports whether nothing, not even a computed part, has been noted.
func (b *wildBuilder) empty() bool { return len(b.pieces) == 0 && b.cur.Len() == 0 }

// pattern is what was noted, or nil when no literal was: nothing is spelled.
func (b *wildBuilder) pattern(kind wildKind, atSegment bool) *wildPattern {
	if !b.literal {
		return nil
	}

	return &wildPattern{pieces: append(b.pieces, b.cur.String()), kind: kind, atSegment: atSegment}
}

// spelledParts walks the tokens of a value, calling text for each literal
// part of a string, and wild for what is computed (a `#expression#` in a
// string, or any token that is not a string or the `&` joining them).
func spelledParts(value []parser.Token, b *wildBuilder, trimLeadingSlash bool) {
	for _, t := range value {
		switch t.Kind {
		case parser.TokString:
			text := t.Value
			if len(text) >= 2 {
				text = text[1 : len(text)-1]
			}

			spelledString(text, b, trimLeadingSlash)
		case parser.TokAmpersand:
		default:
			b.wild()
		}
	}
}

// spelledString notes the parts of the text of one string: a run between hashes
// is computed, and `##` is a literal hash where a hash would open an expression,
// so `#a##b#` is two expressions, not one around a literal hash. The expression
// ends at the next hash outside a string of its own, as the rename's reading of
// the same text does.
func spelledString(text string, b *wildBuilder, trimLeadingSlash bool) {
	var lit strings.Builder

	flush := func() {
		part := lit.String()
		lit.Reset()

		if trimLeadingSlash && b.empty() {
			part = strings.TrimPrefix(part, "/")
		}

		b.text(part)
	}

	for i := 0; i < len(text); {
		switch {
		case text[i] != '#':
			lit.WriteByte(text[i])

			i++
		case i+1 < len(text) && text[i+1] == '#':
			lit.WriteByte('#')

			i += 2
		default:
			flush()
			b.wild()

			end := hashEnd(text, i+1)
			if end < 0 {
				return
			}

			i = end + 1
		}
	}

	flush()
}

// hashEnd is the offset of the hash that ends the expression which starts at
// from in text, a hash outside a string of the expression's own, or -1.
func hashEnd(text string, from int) int {
	var quote byte

	for k := from; k < len(text); k++ {
		switch c := text[k]; {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			return k
		}
	}

	return -1
}
