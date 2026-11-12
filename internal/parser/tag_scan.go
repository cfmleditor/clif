// Finding where a tag ends and where the next one starts, stepping over comments, quotes and #...# spans.

package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// looksLikeCallSpan reports whether the text between two hashes is worth
// treating as opaque: a CFML expression holding a call, rather than two
// unrelated hashes that happened to pair.
//
// Two tests, and each is here for a shape the corpus produced:
//
//   - a `(`, because a call is the only thing a span contributes to the tag
//     walk and scanInterpolatedText rejects the rest on the same byte scan;
//   - balanced quotes, because a lone `#` inside a quoted string is how an
//     embedded page writes a jQuery id selector — `$( '#search' ).typeahead(`
//     has both a quote and a paren, and pairing its hash with the next one in
//     the file swallowed four tags of Lucee's doc pages. A CSS `#01798A` needs
//     no test of its own: it has no paren.
func looksLikeCallSpan(body string) bool {
	if strings.IndexByte(body, '(') < 0 {
		return false
	}

	return strings.Count(body, `"`)%2 == 0 && strings.Count(body, "'")%2 == 0
}

// tagEndIndex finds the '>' that closes a tag, skipping over quoted strings.
//
// A bare IndexByte stops at the first '>' anywhere, and a CF tag holds an
// expression that may contain one:
//
//	<cfset fields = array( field( "Host:Port&lt;new line><br>" ), field( "b" ) )>
//
// cut there, the second `field` is in no tag and in no text the walk scans, so
// it is recorded nowhere. Lucee's cache-driver components are written this way.
//
// It falls back to the plain scan when a quote never closes, so a malformed tag
// cannot swallow the rest of the file — the tag is read exactly as it was
// before, which is the behaviour this replaces.
func tagEndIndex(s string) int {
	end := tagEndIndexRaw(s)

	for end > 0 && arrowContinues(s, end) {
		next := tagEndIndexRaw(s[end+1:])
		if next < 0 {
			break
		}

		end += 1 + next
	}

	return end
}

// arrowContinues reports whether the '>' at gt in s is an arrow function's
// `=>`, which ends no tag: `<cfset double = (x) => x * 2>` is one tag, as the
// grammar reads it, where the walk ended it at the arrow and read `x * 2>`
// as text. In markup `=>` is no attribute the grammar reads either. A tag with
// no '>' after its arrow keeps the arrow's, as it ended before.
func arrowContinues(s string, gt int) bool {
	return gt > 0 && s[gt-1] == '='
}

// tagEndIndexRaw is tagEndIndex, reading every '>' outside a quote or comment
// as the end.
func tagEndIndexRaw(s string) int {
	// The bulk of the scan stays an IndexByte: a quote before the next '>' is
	// what makes a false end possible, and only then is there anything to skip.
	// A byte-at-a-time version of this cost 9% of a tag parse — it ran over
	// every tag in the file, most of which hold no quote at all.
	gt := strings.IndexByte(s, '>')
	if gt < 0 {
		return -1
	}

	// A '>' with every quote before it already closed is the tag's end, and
	// two counts settle that for the whole attribute list at once. The walk
	// below costs three calls per attribute instead, on short strings where
	// the call is most of the cost — and a tag whose attributes are all
	// ordinarily quoted is nearly every tag in a file. Parity is exact rather
	// than a heuristic: CFML escapes a quote by doubling it, which adds two.
	//
	// A comment between attributes is the other way that '>' is not the end,
	// and parity says nothing about it: the comment's text may hold no quote.
	// It is rare enough that its test is one Contains over the short prefix.
	if strings.Contains(s[:gt], "<!--") {
		return tagEndWithComments(s)
	}

	if strings.Count(s[:gt], `"`)%2 == 0 && strings.Count(s[:gt], "'")%2 == 0 {
		return gt
	}

	return tagEndWalk(s)
}

// tagEndWithComments is the '>' closing a tag whose attribute list holds
// `<!--- … --->` comments, which are not string-aware (a `--->` inside a quote
// still ends one, as in an engine) and hide what they hold. A quoted string is
// stepped over, and one that never closes or a comment left open falls back to
// the plain scan, as tagEndIndex's other paths do.
func tagEndWithComments(s string) int {
	for pos := 0; pos < len(s); pos++ {
		switch s[pos] {
		case '>':
			return pos
		case '"', '\'':
			j := skipQuotedIn(s, pos)
			if j < 0 {
				return strings.IndexByte(s, '>')
			}

			pos = j
		case '<':
			if strings.HasPrefix(s[pos:], "<!---") {
				c := strings.Index(s[pos+5:], "--->")
				if c < 0 {
					return strings.IndexByte(s, '>')
				}

				pos += 5 + c + 3
			}
		default:
		}
	}

	return -1
}

// tagEndWalk is the quote-by-quote scan tagEndIndex falls back to. It is a
// function of its own so the fast path above can be compared against it
// directly — restating it in a test would be the parallel list this codebase
// keeps warning about.
func tagEndWalk(s string) int {
	for pos := 0; ; {
		gt := strings.IndexByte(s[pos:], '>')
		if gt < 0 {
			return -1
		}

		end := pos + gt

		q := indexQuote(s[pos:end])
		if q < 0 {
			return end
		}

		j := skipQuotedIn(s, pos+q)
		if j < 0 {
			return strings.IndexByte(s, '>')
		}

		pos = j + 1
	}
}

// nextTagStart finds the next '<' that could open a tag, stepping over a
// `#...#` span rather than through it.
//
// Markup written inside a span is an argument, not a tag:
//
//	#ETH.author( content = "<strong>@name@</strong>" )#
//
// Splitting there hands scanInterpolatedText a chunk with an unterminated `#`,
// so the call is lost — and the hashes left over mis-pair with the next span,
// so the call after it is lost too.
//
// Two rules, and getting either wrong loses tags rather than finding calls:
//
//   - **A span's hashes pair whether or not its contents are stepped over.** A
//     first version advanced only past the *opening* hash of a span it declined,
//     so the closing hash became the next opening one and everything after was
//     inverted: `#local.iconType#` left its second hash to pair with the `#`
//     three lines later, and the `</i><cfif isSimpleValue( … )>` between them
//     was swallowed. That alone was 141 files.
//   - **A span holding no `(` does not hide a tag.** Hashes interpolate inside
//     <cfoutput> and in a tag's attributes and this walk tracks neither, so in
//     prose a `#` is just a character and `item #1 <b>x</b> #2` must keep its
//     markup. A call is the only thing a span contributes here —
//     scanInterpolatedText rejects the rest on the same byte scan — so a span
//     with no `(` still pairs, but a `<` inside it is returned as a tag.
func nextTagStart(s string) int {
	// The bulk of the scan stays an IndexByte. A '#' before the next '<' is
	// what makes a span possible, and only then is there anything to resolve —
	// on a file with no interpolation this is the same two byte scans the walk
	// always did.
	//
	// Memoising the '#' scan across calls — one scan per hash rather than one
	// per tag — is the obvious next step and was measured: it changed nothing
	// on the tag benchmarks, because what this costs is the '<' scan's call
	// overhead on short strings, not the hash scan.
	//
	// The next '<' is found once and kept until a span carries the walk past
	// it. Searching again for every '#' scanned the same run once per hash, so
	// a run of hashes far from any '<' (`<cfoutput>` then forty thousand
	// `#`, or text full of spans) was quadratic.
	ltAt := -1 // the next '<' at or after pos, or len(s) for none; -1 for unknown

	for pos := 0; ; {
		if ltAt < pos {
			ltAt = len(s)
			if lt := strings.IndexByte(s[pos:], '<'); lt >= 0 {
				ltAt = pos + lt
			}
		}

		h := strings.IndexByte(s[pos:ltAt], '#')
		if h < 0 {
			if ltAt == len(s) {
				return -1
			}

			return ltAt
		}

		at := pos + h

		// `##` is CFML's escaped hash and opens nothing.
		if at+1 < len(s) && s[at+1] == '#' {
			pos = at + 2

			continue
		}

		// The next '#', not matchingHash: the two scans legitimately differ,
		// because they optimise different errors. A wrong span here costs a
		// *tag* — everything between the hashes stops being markup — so this
		// one stays conservative and pairs hashes as they come. A wrong span
		// in interpolatedSpans costs at most a call, so that one follows CFML's
		// nesting through quoted strings. Making this share that rule fixed one
		// corpus site and broke thirty-six: skipping quoted strings runs a span
		// much further in markup, and Lucee's doc pages lost their tags to it.
		rel := strings.IndexByte(s[at+1:], '#')
		if rel < 0 {
			// Unterminated: the chunk is a fragment of some tag's attribute
			// list, and the '#' is just a byte.
			pos = at + 1

			continue
		}

		spanEnd := at + 1 + rel

		body := s[at+1 : spanEnd]
		if !looksLikeCallSpan(body) {
			// Declined, but it may have been a span after all: one that opens as
			// a call is closed the way CFML closes it, through its strings.
			// `#ETH.text( "<p>Don't</p><i style='color: ##f2dede'>" )#` has a
			// `##` inside its string, which pairs with the opening hash here,
			// and apostrophes that unbalance the quote count. Only a declined
			// span is asked, so the common one costs what it always did.
			if end := callSpanEnd(s, at); end >= 0 {
				pos = end + 1

				continue
			}

			if i := strings.IndexByte(body, '<'); i >= 0 {
				return at + 1 + i
			}
		}

		pos = spanEnd + 1
	}
}

// callSpanEnd returns the index of the hash closing the span that the hash at
// s[at] opens, when that span is a call written through its strings, and -1
// when the walk should pair hashes as it always has.
//
// It is narrow on purpose, because the walk loses tags, not calls, when it
// steps over something that was never a span (see nextTagStart's second rule):
//
//   - the hash must open as a call, a name or dotted name and then `(`. A CSS
//     colour, an id selector, a jQuery `'#search'` and prose hashes never do, so
//     none of them reach the quote-aware scan that ran too far in markup;
//   - every string in it must close, which matchingHash reports as -1 when one
//     does not, leaving the markup after an unclosed quote as markup;
//   - it must not hold a CF tag. An expression does not, and a span that would
//     run across `<cfif …>` has swallowed real tags, whatever its quotes say.
func callSpanEnd(s string, at int) int {
	if !opensAsCall(s[at+1:]) {
		return -1
	}

	end := matchingHash(s, at+1)
	if end < 0 || holdsCFTag(s[at:end]) {
		return -1
	}

	return end
}

// opensAsCall reports whether s begins with a name or dotted name and then `(`,
// allowing space before the parenthesis: `ETH.text(`, `local.svc.get (`.
func opensAsCall(s string) bool {
	i := 0

	for i < len(s) && (lex.IsIdent(s[i]) || s[i] == '.') {
		i++
	}

	if i == 0 || s[0] == '.' || s[0] >= '0' && s[0] <= '9' {
		return false
	}

	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}

	return i < len(s) && s[i] == '('
}

// holdsCFTag reports whether s contains the start or end of a CF tag, `<cf` or
// `</cf` in any case.
func holdsCFTag(s string) bool {
	for {
		i := strings.IndexByte(s, '<')
		if i < 0 {
			return false
		}

		s = s[i+1:]

		j := 0
		if s != "" && s[0] == '/' {
			j = 1
		}

		if len(s) > j+1 && (s[j] == 'c' || s[j] == 'C') && (s[j+1] == 'f' || s[j+1] == 'F') {
			return true
		}
	}
}
