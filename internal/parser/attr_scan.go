// One reader for a tag's attributes, in the two ways a value is written.

package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// attrMode is how a tag's attribute values are read.
type attrMode uint8

const (
	// cfAttrs is a CF tag's, a custom tag's included: a doubled quote is an
	// escaped one ("a""b" is a"b), and a #...# is an expression, whose own
	// strings do not end the value ("#f("x")#").
	cfAttrs attrMode = iota
	// htmlAttrs is markup's: a value ends at its closing quote.
	htmlAttrs
)

// tagAttr is one attribute an attrScanner read: where its name and its value
// stand in the text, the value without its quotes.
type tagAttr struct {
	nameStart, nameEnd int
	valStart, valEnd   int
}

// attrScanner reads a tag's attributes in order. Its text is the tag from
// its '<' and name, or its attributes alone; it stops at the tag's '>'.
// Judged against the tree-sitter grammar over the corpus, this is the reading
// five separate readers each got wrong in a way of its own
// (.ai/docs/SCANNER-CONSOLIDATION.md, section 7).
type attrScanner struct {
	s    string
	i    int
	mode attrMode
}

func newAttrScanner(s string, mode attrMode) attrScanner {
	i := 0
	if i < len(s) && s[i] == '<' {
		i++
		if i < len(s) && s[i] == '/' {
			i++
		}

		for i < len(s) && isTagNameByte(s[i]) {
			i++
		}
	}

	return attrScanner{s: s, i: i, mode: mode}
}

// next is the next attribute with a value, false at the tag's end. An
// attribute with no value (`<input checked>`) is passed over.
func (a *attrScanner) next() (tagAttr, bool) {
	s := a.s

	for a.i < len(s) {
		c := s[a.i]

		switch {
		case c == '>':
			a.i = len(s)

			return tagAttr{}, false
		case lex.IsQuote(c):
			// A value with no name before it: step over it whole.
			if end := quotedValueEnd(s, a.i, a.mode); end >= 0 {
				a.i = end + 1
			} else {
				a.i = len(s)
			}

			continue
		case !isTagNameByte(c):
			a.i++

			continue
		}

		start := a.i
		for a.i < len(s) && isTagNameByte(s[a.i]) {
			a.i++
		}

		attr := tagAttr{nameStart: start, nameEnd: a.i}

		j := a.i
		for j < len(s) && lex.IsRegexpSpace(s[j]) {
			j++
		}

		if j >= len(s) || s[j] != '=' {
			continue
		}

		j++
		for j < len(s) && lex.IsRegexpSpace(s[j]) {
			j++
		}

		if j >= len(s) {
			a.i = j

			return tagAttr{}, false
		}

		if lex.IsQuote(s[j]) {
			end := quotedValueEnd(s, j, a.mode)
			if end < 0 {
				// A value whose quote never closes ends the reading.
				a.i = len(s)

				return tagAttr{}, false
			}

			attr.valStart, attr.valEnd = j+1, end
			a.i = end + 1

			return attr, true
		}

		k := j
		for k < len(s) && !endsUnquotedValue(s, k) {
			k++
		}

		a.i = k

		if k > j {
			attr.valStart, attr.valEnd = j, k

			return attr, true
		}
	}

	return tagAttr{}, false
}

// endsUnquotedValue reports whether s[k] ends an unquoted value: a space, the
// tag's '>', or the '/' of its "/>".
func endsUnquotedValue(s string, k int) bool {
	switch s[k] {
	case '>':
		return true
	case '/':
		return k+1 == len(s) || s[k+1] == '>'
	default:
		return lex.IsRegexpSpace(s[k])
	}
}

// quotedValueEnd is the offset of the quote closing the value s opens at
// open, or -1 when it never closes.
func quotedValueEnd(s string, open int, mode attrMode) int {
	q := s[open]

	for i := open + 1; i < len(s); i++ {
		switch c := s[i]; {
		case c == q:
			if mode == cfAttrs && i+1 < len(s) && s[i+1] == q {
				i++ // "" is an escaped quote

				continue
			}

			return i
		case c == '#' && mode == cfAttrs:
			if i+1 < len(s) && s[i+1] == '#' {
				i++ // ## is a literal #

				continue
			}

			if end := hashExprEnd(s, i, q); end > i {
				i = end
			}
		}
	}

	return -1
}

// hashExprEnd is the offset of the '#' closing the expression opening at
// open, inside a value quoted with q, or -1 when it does not close as an
// expression. A string in it may use q itself; q after a name, a ) or a ]
// is the value's closing quote, which ends the reading as no expression, so
// a stray # (`color="#fff"`) reads as text.
func hashExprEnd(s string, open int, q byte) int {
	prev := byte('#')

	for i := open + 1; i < len(s); i++ {
		c := s[i]

		switch {
		case c == '#':
			return i
		case lex.IsQuote(c):
			if c == q && (isIdentPart(prev) || prev == ')' || prev == ']') {
				return -1
			}

			end := strings.IndexByte(s[i+1:], c)
			for end >= 0 && i+2+end < len(s) && s[i+2+end] == c {
				// A doubled quote inside the string escapes it.
				next := strings.IndexByte(s[i+3+end:], c)
				if next < 0 {
					return -1
				}

				end += 2 + next
			}

			if end < 0 {
				return -1
			}

			i += 1 + end
			prev = c
		case c == '\n' && i-open > 4096:
			return -1
		default:
			if !lex.IsRegexpSpace(c) {
				prev = c
			}
		}
	}

	return -1
}

// tagAttrValue is the value of attr in s, a tag or its attributes, read in
// mode, the first time it is given; ok is false when it is not.
func tagAttrValue(s, attr string, mode attrMode) (start, end int, ok bool) {
	sc := newAttrScanner(s, mode)

	for {
		a, more := sc.next()
		if !more {
			return 0, 0, false
		}

		if strings.EqualFold(s[a.nameStart:a.nameEnd], attr) {
			return a.valStart, a.valEnd, true
		}
	}
}

// tagAttrMap is every attribute of s with a value, read in mode, by name
// lowercased; a name given twice keeps its first value, as the grammar does.
func tagAttrMap(s string, mode attrMode) map[string]string {
	out := map[string]string{}
	sc := newAttrScanner(s, mode)

	for {
		a, more := sc.next()
		if !more {
			return out
		}

		name := strings.ToLower(s[a.nameStart:a.nameEnd])
		if _, seen := out[name]; !seen {
			out[name] = s[a.valStart:a.valEnd]
		}
	}
}
