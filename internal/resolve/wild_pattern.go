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

// accepts reports whether c may stand in a wildcard's run.
func (k wildKind) accepts(c byte) bool {
	switch k {
	case wildWord:
		return lex.IsWord(c)
	case wildSegment:
		return c != '/'
	default:
		return c != '\n'
	}
}

// least is how many characters a wildcard's run holds at the fewest.
func (k wildKind) least() int {
	if k == wildLine {
		return 1
	}

	return 0
}

// match reports whether s fits the pattern, ignoring case.
func (p *wildPattern) match(s string) bool {
	if !p.atSegment {
		return p.matchFrom(s, 0, 0)
	}

	for start := 0; start <= len(s); start++ {
		if (start == 0 || s[start-1] == '/') && p.matchFrom(s, 0, start) {
			return true
		}
	}

	return false
}

// matchFrom reports whether s from pos fits the pieces from piece on: the
// piece, a wildcard, the next piece, and the last piece ends the text.
func (p *wildPattern) matchFrom(s string, piece, pos int) bool {
	lit := p.pieces[piece]
	if pos+len(lit) > len(s) || !strings.EqualFold(s[pos:pos+len(lit)], lit) {
		return false
	}

	end := pos + len(lit)
	if piece == len(p.pieces)-1 {
		return end == len(s)
	}

	run := 0

	for {
		if run >= p.kind.least() && p.matchFrom(s, piece+1, end+run) {
			return true
		}

		if end+run >= len(s) || !p.kind.accepts(s[end+run]) {
			return false
		}

		run++
	}
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

			for i, part := range strings.Split(text, "#") {
				if i%2 == 1 {
					b.wild()

					continue
				}

				if trimLeadingSlash && b.empty() {
					part = strings.TrimPrefix(part, "/")
				}

				b.text(part)
			}
		case parser.TokAmpersand:
		default:
			b.wild()
		}
	}
}
