// Where a #...# span in a script string is a call, and the scans that find its ends.

package parser

import (
	"strings"
)

// scanInterpolation records the calls written inside a string's #...# spans.
//
// The scanner takes a quoted string as one token, so everything in it was
// invisible: `writeOutput("id #svc.getName()#")` recorded only writeOutput.
// That is idiomatic CFML rather than an edge case — interpolation is how a
// computed value reaches a string — and it cost the same as the argument-list
// gap did: no edge into the callee, so the code map read it as unreachable and
// `unresolved` never checked it.
//
// Each span is scanned by the same parser over a scanner of its own, so the
// calls land in the enclosing function's bucket with the resolver machinery
// that a call outside a string gets. Line numbers within a multi-line string
// collapse onto the line the string starts at.
func (p *scriptParser) scanInterpolation(tok Token) {
	// A span with no '(' in it cannot hold a call, and a call is the only thing
	// this records — `"#user.name#"` and `"#i#"` are most of the interpolation
	// in any CFML file, so rejecting them on one byte scan is what keeps this
	// off the profile.
	if strings.IndexByte(tok.Value, '#') < 0 || strings.IndexByte(tok.Value, '(') < 0 ||
		p.argNesting >= maxArgNesting {
		return
	}

	outerScanner, outerBase, outerOffset := p.sc, p.baseLine, p.baseOffset

	p.argNesting++

	defer func() {
		p.sc, p.baseLine = outerScanner, outerBase
		p.baseOffset = outerOffset
		p.argNesting--
	}()

	tokBase := p.baseLine + int(tok.Line)

	// lines counts the newlines before counted, carried from span to span:
	// counting from the token's start for each span was quadratic in a
	// string holding thousands of them.
	lines, counted := 0, 0

	for _, span := range interpolatedSpans(tok.Value) {
		// A string token can span lines — CFML's `""` escape is what makes a
		// multi-line one ordinary — so the span's own line is the token's start
		// plus the newlines before it, not the token's start.
		lines += countNewlines(tok.Value[counted:span.start])
		counted = span.start
		p.baseLine = tokBase + lines

		if outerOffset >= 0 {
			p.baseOffset = outerOffset + tok.Offset + span.start
		}

		p.sc = NewScanner(tok.Value[span.start:span.end])
		p.sc.interpStrings = true

		for {
			t := p.sc.NextSkipComments()
			if t.Kind == TokEOF {
				break
			}

			switch t.Kind {
			case TokIdent:
				p.scanNestedCall(t)
			case TokString, TokRBracket:
				p.handleLiteralToken(t)
			default:
				// Any other token is passed over.
			}
		}
	}
}

// countNewlines reports how many lines a byte range spans past its first.
func countNewlines(s string) int {
	return strings.Count(s, "\n")
}

// hashSpan is the half-open byte range between a pair of interpolation hashes.
// The tag parser needs the offsets to place a line number, so the ranges rather
// than the substrings are what this returns.
type hashSpan struct{ start, end int }

// interpolatedSpans returns the range inside each #...# of a piece of source.
//
// `##` is CFML's escape for a literal hash and opens nothing, which is what
// stops `"a ## b"` being read as a span containing " ".
func interpolatedSpans(s string) []hashSpan {
	var spans []hashSpan

	for i := 0; i < len(s); i++ {
		if s[i] != '#' {
			continue
		}

		if i+1 < len(s) && s[i+1] == '#' {
			i++ // an escaped hash: skip the pair

			continue
		}

		end := matchingHash(s, i+1)
		if end < 0 {
			// A lone `#` — a CSS colour, a jQuery id selector, prose. It opens
			// nothing, and the spans after it are still spans: this used to
			// give up on the whole remainder, which lost every interpolation
			// later in the chunk.
			continue
		}

		if end > i+1 {
			spans = append(spans, hashSpan{start: i + 1, end: end})
		}

		i = end
	}

	return spans
}

// matchingHash finds the `#` that closes a span opened just before from,
// stepping over any quoted string on the way.
//
// A string inside a span may hold a span of its own — CFML nests them:
//
//	var u = "#html.elixirPath( root = '#cb.themeRoot()#/includes' )#";
//
// and taking the first `#` as the close made the *inner* opening hash the
// outer's terminator, so the outer call was sub-parsed from a fragment and
// every pairing after it on the line was inverted. Scanner.scanHashExpr already
// reads CFScript this way; this is the same rule for the text a tag parser
// hands over.
func matchingHash(s string, from int) int {
	return matchingHashAt(s, from, 0)
}

// maxHashNesting bounds how many spans may sit inside one another. Real code
// goes one deep; the bound is what keeps a run of unclosed `#f(` from costing
// more than a few hash scans each.
const maxHashNesting = 4

// matchingHashAt is matchingHash for a span nested level deep.
//
// A hash met outside a string with a parenthesis still open does not close the
// span, because an expression cannot end inside a call: it opens another span,
// which CFML allows to stand in an argument list,
//
//	#event.buildLink( prc.xehEditor & "?t=" & #encodeForHTMLAttribute( id )# )#
//
// and the closing hash is the one after that span, with the parentheses shut.
// Counted in the text outside strings only, since a string may hold any
// parenthesis. A span nested past maxHashNesting, or one that never closes, is
// no span: -1.
func matchingHashAt(s string, from, level int) int {
	depth := 0 // parentheses open outside a string

	// Both scans are IndexByte: a quote before the next `#` is what makes
	// nesting possible, and only then is there anything to step over. A span
	// with no quote in it — which is nearly all of them — costs the same two
	// byte scans the single loop this replaces cost as one.
	for i := from; ; {
		h := strings.IndexByte(s[i:], '#')
		if h < 0 {
			return -1
		}

		end := i + h

		q := indexQuote(s[i:end])
		if q < 0 {
			// Nearly every span, `#user.name#`, holds no call at all.
			if depth == 0 && strings.IndexByte(s[i:end], '(') < 0 {
				return end
			}

			depth += parenBalance(s[i:end])
			if depth <= 0 {
				return end
			}

			if level >= maxHashNesting {
				return -1
			}

			inner := matchingHashAt(s, end+1, level+1)
			if inner < 0 {
				return -1
			}

			i = inner + 1

			continue
		}

		depth += parenBalance(s[i : i+q])

		j := skipQuotedIn(s, i+q)
		if j < 0 {
			return -1
		}

		i = j + 1
	}
}

// parenBalance is the number of `(` in s less the number of `)`.
func parenBalance(s string) int {
	n := 0

	for i := range len(s) {
		switch s[i] {
		case '(':
			n++
		case ')':
			n--
		}
	}

	return n
}

// indexQuote returns the offset of the first single or double quote, or -1.
func indexQuote(s string) int {
	d := strings.IndexByte(s, '"')

	sq := strings.IndexByte(s, '\'')
	if d < 0 {
		return sq
	}

	if sq < 0 || d < sq {
		return d
	}

	return sq
}

// skipQuotedIn returns the index of the quote closing the string opened at i,
// honouring CFML's doubled-quote escape, or -1 if it never closes.
func skipQuotedIn(s string, i int) int {
	q := s[i]

	for j := i + 1; j < len(s); j++ {
		if s[j] != q {
			continue
		}

		if j+1 < len(s) && s[j+1] == q {
			j++

			continue
		}

		return j
	}

	return -1
}
