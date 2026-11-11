// The whitespace-only guard and the byte-level scans of source it relies on: strings, script spans, comments and tag headers.

package formatter

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/cfmleditor/clif/internal/lex"
)

// checkWhitespaceOnly walks both byte slices skipping whitespace, comparing
// non-whitespace characters case-insensitively. If allowSelfClose is true,
// a "/" immediately before ">" in either side is also skipped; if allowRequote
// is true, the attribute re-quoting described below is skipped too.
// Returns nil if only whitespace differs, or an error describing the first mismatch.
func checkWhitespaceOnly(a, b []byte, allowSelfClose, allowRequote bool) error {
	i, j := 0, 0

	// The formatter canonicalises cfscript as it goes: a statement written
	// without its optional semicolon gains one, and a single-statement body
	// gains braces. Both are deliberate insertions, so an extra ";", "{" or
	// "}" on the output side is not a defect. The allowance is one-directional
	// — a token the formatter *dropped* is still reported — and the braces are
	// counted, so an unmatched one is still caught below.
	openedAdded, closedAdded := 0, 0

	// Comment bodies are gathered as the walk steps over them and compared
	// once both sides are exhausted. See skipWSAndComments for why comments
	// cannot be compared in line with the surrounding code.
	var commentsA, commentsB []byte

	scriptA, scriptB := scriptRegionsOf(a), scriptRegionsOf(b)
	strsA, strsB := stringSpansOf(a, scriptA), stringSpansOf(b, scriptB)

	for {
		i = skipWSAndComments(a, i, &commentsA, scriptA, strsA)
		j = skipWSAndComments(b, j, &commentsB, scriptB, strsB)

		if j < len(b) && isNormalizationToken(b[j]) &&
			(i >= len(a) || lex.LowerASCII(a[i]) != lex.LowerASCII(b[j])) {
			switch b[j] {
			case '{':
				openedAdded++
			case '}':
				closedAdded++
			}

			j++

			continue
		}

		if i == len(a) && j == len(b) {
			if openedAdded != closedAdded {
				return fmt.Errorf("formatter added %d unmatched brace(s)", openedAdded-closedAdded)
			}

			return compareCommentBodies(a, commentsA, commentsB)
		}

		if i == len(a) || j == len(b) {
			line := byteOffsetToLine(a, i)

			return fmt.Errorf("formatter made non-whitespace changes near line %d (content length mismatch)", line)
		}

		if allowSelfClose {
			if a[i] == '/' && i+1 < len(a) && a[i+1] == '>' && b[j] == '>' {
				i++

				continue
			}

			if b[j] == '/' && j+1 < len(b) && b[j+1] == '>' && a[i] == '>' {
				j++

				continue
			}
		}

		// Attribute re-quoting is a deliberate canonicalisation, and only ever
		// runs in one of two shapes (normaliseAttrValue): an unquoted value
		// gains quotes, or a single-quoted one is upgraded to double. Those are
		// a quote the output has and the source does not, and a quote on each
		// side that differ.
		//
		// A quote the formatter *dropped* is neither, and allowing it is what
		// blinded the guard: the allowance was written as "any mismatched quote
		// on either side", so the formatter stripping the quotes off a CFML
		// string — `<cfset msg = "hello world">` coming back as
		// `<cfset msg = hello world>`, or `SELECT 'a'` as `SELECT a` — passed
		// as whitespace-only. A removal is compared like any other byte now.
		if allowRequote {
			if lex.IsQuote(a[i]) && lex.IsQuote(b[j]) && a[i] != b[j] {
				i++
				j++

				continue
			}

			if lex.IsQuote(b[j]) && !lex.IsQuote(a[i]) {
				j++

				continue
			}
		}

		if lex.LowerASCII(a[i]) != lex.LowerASCII(b[j]) {
			line := byteOffsetToLine(a, i)
			ctx := snippetAt(a, i)

			return fmt.Errorf("formatter made non-whitespace changes at line %d near %q", line, ctx)
		}

		i++
		j++
	}
}

// isNormalizationToken reports whether c is one of the tokens the formatter
// inserts as part of canonicalising cfscript.
func isNormalizationToken(c byte) bool {
	return c == ';' || c == '{' || c == '}'
}

func byteOffsetToLine(src []byte, offset int) int {
	line := 1

	for k := 0; k < offset && k < len(src); k++ {
		if src[k] == '\n' {
			line++
		}
	}

	return line
}

func snippetAt(src []byte, offset int) string {
	start := max(offset-10, 0)

	end := min(offset+10, len(src))

	return string(src[start:end])
}

func isWS(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// spaceLen returns the byte length of the whitespace character at pos, or 0 if
// there is none. Unlike isWS it decodes multi-byte runes, because indentation
// is not always ASCII: source pasted from a word processor or a browser arrives
// indented with U+2003 EM SPACE and the like. Re-indenting such a line is a
// whitespace change, but measuring it a byte at a time made it look like the
// formatter had deleted content, and the file was refused.
//
// Only *breaking* spaces count. U+00A0 NO-BREAK SPACE and U+202F NARROW NO-BREAK
// SPACE render differently from an ordinary space and are load-bearing in
// markup, so they stay content: losing one has to be reported, not waved through.
func spaceLen(src []byte, pos int) int {
	if pos < 0 || pos >= len(src) {
		return 0
	}

	if c := src[pos]; c < utf8.RuneSelf {
		if isWS(c) || c == '\v' || c == '\f' {
			return 1
		}

		return 0
	}

	r, size := utf8.DecodeRune(src[pos:])

	switch {
	case r >= 0x2000 && r <= 0x200A: // EN QUAD … HAIR SPACE
		return size
	case r == 0x0085, // NEXT LINE
		r == 0x1680, // OGHAM SPACE MARK
		r == 0x2028, // LINE SEPARATOR
		r == 0x2029, // PARAGRAPH SEPARATOR
		r == 0x205F, // MEDIUM MATHEMATICAL SPACE
		r == 0x3000: // IDEOGRAPHIC SPACE
		return size
	}

	return 0
}

// skipSpace advances past a run of whitespace, ASCII or otherwise.
func skipSpace(src []byte, pos int) int {
	for {
		n := spaceLen(src, pos)
		if n == 0 {
			return pos
		}

		pos += n
	}
}

// scriptSpans marks the byte ranges of a document that hold script rather than
// markup. Only inside those does "//" open a comment: in markup the same two
// characters are ordinary content, and appear in places as mundane as a
// DOCTYPE ("-//W3C//DTD XHTML 1.0 Strict//EN"). Reading one of those as a
// comment would swallow the rest of its line and reject a perfectly good
// reformat of it.
type scriptSpans []struct{ start, end int }

func (s scriptSpans) contains(pos int) bool {
	for _, sp := range s {
		if pos >= sp.start && pos < sp.end {
			return true
		}
	}

	return false
}

// containsSorted is contains for a slice built in ascending order, which the
// string spans are. The guard consults them once per byte of the file, and the
// linear scan above is O(spans) per call — fine for the handful of script
// regions a file has, but a large component holds hundreds of string literals.
func (s scriptSpans) containsSorted(pos int) bool {
	i := sort.Search(len(s), func(i int) bool { return s[i].end > pos })

	return i < len(s) && pos >= s[i].start
}

// stringSpansOf locates the quoted string literals inside src's script regions.
//
// The guard has to know about them because a string is allowed to contain the
// characters that open a comment, and CFML code is full of globs that do.
// ColdBox's own build script has `path = "/#libBuildDir#/**"`, whose `/*` was
// taken as the start of a block comment: everything from there to the next
// `*/`, some forty lines of code below, was collected as comment body.
//
// The effect is a false rejection rather than a blind spot, which is worth
// being precise about. Code swallowed this way is still compared — as comment
// text — and that comparison is the stricter of the two: it folds whitespace
// and case exactly as the main loop does, but it has none of the main loop's
// allowances for the canonicalisation the formatter performs on purpose. So a
// semicolon the formatter legitimately adds to `.run()` forty lines down lands
// inside a "comment body", the two sinks differ, and a correct format is
// refused with a message about comment text that names neither the string nor
// the statement. Seven files in the corpus, four of them reporting changed
// comment text whose reported bodies were plainly code.
//
// Quotes are only tracked inside script regions. In markup the same bytes are
// attribute delimiters and ordinary prose, and `//` and an apostrophe carry no
// such meaning there.
func stringSpansOf(src []byte, script scriptSpans) scriptSpans {
	var spans scriptSpans

	for _, region := range script {
		pos := max(region.start, 0)
		for pos < region.end && pos < len(src) {
			switch {
			case hasBytesAt(src, pos, "<!---"):
				end := indexBytesFrom(src, pos+5, "--->")
				if end < 0 {
					pos = region.end

					continue
				}

				pos = end + 4
			case hasBytesAt(src, pos, "/*"):
				end := indexBytesFrom(src, pos+2, "*/")
				if end < 0 {
					pos = region.end

					continue
				}

				pos = end + 2
			case isLineCommentStart(src, pos):
				for pos < len(src) && src[pos] != '\n' && src[pos] != '\r' {
					pos++
				}
			case src[pos] == '"' || src[pos] == '\'':
				start := pos
				pos = endOfString(src, pos)

				spans = append(spans, struct{ start, end int }{start, pos})
			default:
				pos++
			}
		}
	}

	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })

	return spans
}

// endOfString returns the offset just past the string literal opening at pos.
//
// CFML escapes a quote by doubling it ("say ""hi"""), and *only* that way — a
// backslash is an ordinary character. Treating "\\" as an escape, the way most
// C-family languages would, runs the literal past its own closing quote and on
// to the next one: ContentBox has `replace( inPath, "\\", "/", "all" )`, a string
// holding one backslash, and every Windows path written `"C:\\dir\\"` ends the
// same way. The span would then cover code, and no comment could be recognised
// inside it.
func endOfString(src []byte, pos int) int {
	quote := src[pos]
	pos++

	for pos < len(src) {
		switch {
		// "##" is a literal hash; a single "#" opens an interpolation, which
		// may hold strings of its own in either quote style. Lucee's admin has
		// `"timezone:'#replace(ds.timezone,"'","''","all")#' // ..."` — a
		// double-quoted string whose interpolation contains three more of them.
		// Ending the outer literal at the first of those put the rest of the
		// line outside any string, where its "//" was read as a comment.
		case src[pos] == '#' && pos+1 < len(src) && src[pos+1] == '#':
			pos += 2
		case src[pos] == '#':
			if end, ok := endOfInterpolation(src, pos); ok {
				pos = end
			} else {
				pos++
			}
		case src[pos] == quote && pos+1 < len(src) && src[pos+1] == quote:
			pos += 2
		case src[pos] == quote:
			return pos + 1
		default:
			pos++
		}
	}

	return pos
}

// endOfInterpolation returns the offset just past the #...# opening at pos, and
// whether pos opened one at all. It and endOfString call each other, since
// either may nest inside the other; both always advance, so the pair terminates
// on any input.
//
// A lone "#" is not an interpolation. CFML itself requires the escaped "##" for
// a literal hash, but a <script> region is JavaScript, where CFML's rules do not
// apply and `var s = "#id"` — a CSS selector — is ordinary. Treating that "#" as
// an opener sent the scan hunting for a close that never comes, and the
// enclosing literal then ran to end of file: every later position looked like it
// was inside a string, so no comment was recognised in the rest of the file and
// the guard stopped checking comment extents entirely.
func endOfInterpolation(src []byte, pos int) (int, bool) {
	if !closingHashOnLine(src, pos+1) {
		return pos, false
	}

	p := pos + 1

	for p < len(src) {
		switch src[p] {
		case '"', '\'':
			p = endOfString(src, p)
		case '#':
			return p + 1, true
		case '\n':
			// An interpolation closes on the line it opened on. Bounding the
			// walk here keeps a nested literal that runs away from taking the
			// rest of the file with it.
			return pos, false
		default:
			p++
		}
	}

	return pos, false
}

// closingHashOnLine reports whether a "#" appears before the end of the line
// starting at pos. It is a flat scan, deliberately: the nested walk above can
// step over a newline through a string literal, so the cheap bound has to be
// taken before that walk starts.
func closingHashOnLine(src []byte, pos int) bool {
	for ; pos < len(src) && src[pos] != '\n'; pos++ {
		if src[pos] == '#' {
			return true
		}
	}

	return false
}

// lowerASCIIBytes lowercases A-Z and copies every other byte through.
//
// bytes.ToLower cannot be used here: it folds by Unicode rune, and some runes
// change byte length doing so — "İ" (U+0130, two bytes) lowercases to one. The
// offsets found in the lowered copy are used to slice the *original*, so a
// single such character anywhere earlier in the file shifts every subsequent
// span, and the whitespace-only guard then compares the wrong regions. Tag
// names are ASCII, so folding only A-Z loses nothing.
func lowerASCIIBytes(src []byte) []byte {
	out := make([]byte, len(src))

	for i, c := range src {
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}

		out[i] = c
	}

	return out
}

// scriptRegionsOf locates the <cfscript> and <script> blocks in src, or spans
// the whole file when it is a script-syntax component.
func scriptRegionsOf(src []byte) scriptSpans {
	if isScriptSyntaxComponent(src) {
		return scriptSpans{{0, len(src)}}
	}

	return tagBodiesOf(src, "cfscript", "script")
}

// tagBodiesOf locates the bodies of every element named in tags, from just past
// the opening tag's `>` to the start of its closing tag, or to the end of src
// when the element is never closed.
func tagBodiesOf(src []byte, tags ...string) scriptSpans {
	var spans scriptSpans

	lower := lowerASCIIBytes(src)

	for _, tag := range tags {
		open, closeTag := "<"+tag, "</"+tag

		for pos := 0; ; {
			start := indexBytesFrom(lower, pos, open)
			if start < 0 {
				break
			}

			// Require a tag boundary so <script> does not also match
			// <scripting> and <cfscript> is not matched twice.
			after := start + len(open)
			if after < len(lower) && !isTagNameEnd(lower[after]) {
				pos = after

				continue
			}

			body := indexBytesFrom(lower, after, ">")
			if body < 0 {
				break
			}

			end := indexBytesFrom(lower, body+1, closeTag)
			if end < 0 {
				end = len(src)
			}

			spans = append(spans, struct{ start, end int }{body + 1, end})
			pos = end
		}
	}

	return spans
}

func isTagNameEnd(c byte) bool {
	return isWS(c) || c == '>' || c == '/'
}

// importEnd returns the offset just past an import statement whose keyword ends
// at pos, or -1 when it does not terminate.
//
// A semicolon ends it, and so does a newline: Lucee accepts `import a.b.C`
// without one, and stopping at the line end also keeps a malformed file from
// swallowing the rest of the source looking for a `;` that never comes.
func importEnd(src []byte, pos int) int {
	for i := pos; i < len(src); i++ {
		switch src[i] {
		case ';':
			return i + 1
		case '\n':
			return i + 1
		}
	}

	return -1
}

// isScriptSyntaxComponent reports whether src is a script-syntax component,
// which has no <cfscript> tag to key off because the entire file is script.
func isScriptSyntaxComponent(src []byte) bool {
	// Probe with the whole file marked as script so a leading /** */ doc block
	// is stepped over; only the keyword that follows it matters here.
	all := scriptSpans{{0, len(src)}}

	// A UTF-8 BOM is not whitespace, so skipWSAndComments stops on it and the
	// keyword check below then fails on a file that is plainly a script
	// component. Everything downstream keys off this answer: with no script
	// region the guard stops recognising `//` as a comment anywhere in the
	// file, and compares comment text as though it were code. That is how a
	// leading-comma struct with a commented-out entry — `//, bundleVersion:
	// '3.2.2.54'` — came back as a non-whitespace change, reported against the
	// entry before it. Two files in the corpus; 554 in it carry a BOM.
	start := 0
	if bytes.HasPrefix(src, utf8BOM) {
		start = len(utf8BOM)
	}

	// No string spans: this probe only steps over a leading doc block, before
	// any string literal can appear, and computing them would recurse — string
	// spans need the script regions this function is being called to determine.
	pos := skipWSAndComments(src, start, nil, all, nil)

	// `import a.b.C;` is legal between the doc block and the component, and a
	// file may open with several. Not stepping over them cost the same as the
	// BOM above: the keyword check failed on a file that is plainly a script
	// component, the file got no script region, and the guard then stopped
	// recognising `//` anywhere in it and compared comment text as code. Five
	// corpus files, and it surfaced as a rejection only under
	// `commaPosition: "before"`, where a comma legitimately crosses a comment.
	for hasWordAt(src, pos, "import") {
		end := importEnd(src, pos+len("import"))
		if end < 0 {
			return false
		}

		pos = skipWSAndComments(src, end, nil, all, nil)
	}

	for _, kw := range []string{"abstract", "final"} {
		if hasWordAt(src, pos, kw) {
			pos = skipWSAndComments(src, pos+len(kw), nil, all, nil)
		}
	}

	return hasWordAt(src, pos, "component") || hasWordAt(src, pos, "interface")
}

// hasWordAt reports whether word sits at pos, case-insensitively, and is not
// merely the prefix of a longer identifier.
func hasWordAt(src []byte, pos int, word string) bool {
	if pos < 0 || pos+len(word) > len(src) {
		return false
	}

	for k := range len(word) {
		if lex.LowerASCII(src[pos+k]) != word[k] {
			return false
		}
	}

	after := pos + len(word)

	return after >= len(src) || !lex.IsIdent(src[after])
}

func hasBytesAt(src []byte, pos int, lit string) bool {
	return pos >= 0 && pos+len(lit) <= len(src) && string(src[pos:pos+len(lit)]) == lit
}

func indexBytesFrom(src []byte, pos int, lit string) int {
	if pos < 0 || pos > len(src) {
		return -1
	}

	idx := bytes.Index(src[pos:], []byte(lit))
	if idx < 0 {
		return -1
	}

	return pos + idx
}

// headerEnd returns the offset just past the ">" closing the opening tag that
// begins at start. Quoted values and comments are stepped over so that a ">"
// inside either does not end the header early.
//
// Node shape cannot answer this on its own: a tag's attributes and its body
// can be siblings under one node (cf_output_tag holds body comments as direct
// children), so only the position of the ">" separates the two.
func headerEnd(src []byte, start int) int {
	i := start
	for i < len(src) {
		switch {
		case hasBytesAt(src, i, "<!---"):
			end := indexBytesFrom(src, i+5, "--->")
			if end < 0 {
				return len(src)
			}

			i = end + 4
		case src[i] == '"' || src[i] == '\'':
			q := src[i]
			i++

			for i < len(src) && src[i] != q {
				i++
			}

			i++
		case src[i] == '>':
			return i + 1
		default:
			i++
		}
	}

	return len(src)
}

// hasMultiLineComment reports whether any entry is a comment spanning more than
// one line. Such a comment cannot share a line with the attributes around it,
// so the whole list has to go one-per-line.
func hasMultiLineComment(attrs []cfAttr) bool {
	for _, a := range attrs {
		if a.comment && strings.Contains(a.name, "\n") {
			return true
		}
	}

	return false
}

// collectCommentBody appends body's non-whitespace bytes, folded to lower case,
// so that reindenting or rewrapping a comment does not register as a change.
//
// A "/" directly before a ">" is dropped as well. Commented-out markup is
// common, and the self-closing pass rewrites "<cfargument ...>" to
// "<cfargument ... />" wherever it appears, comments included. The code side of
// the comparison already tolerates that same difference.
func collectCommentBody(sink *[]byte, body []byte) {
	if sink == nil {
		return
	}

	for k := 0; k < len(body); {
		if n := spaceLen(body, k); n > 0 {
			k += n

			continue
		}

		c := body[k]
		k++

		if c == '/' && nextNonWS(body, k) == '>' {
			continue
		}

		*sink = append(*sink, lex.LowerASCII(c))
	}
}

// nextNonWS returns the first non-whitespace byte at or after pos, or 0.
func nextNonWS(src []byte, pos int) byte {
	pos = skipSpace(src, pos)
	if pos < len(src) {
		return src[pos]
	}

	return 0
}

// isLineCommentStart reports whether a "//" comment opens at pos. The caller
// has already established that pos sits in script, where "//" is a comment
// rather than the content it would be in markup.
func isLineCommentStart(src []byte, pos int) bool {
	if pos+1 >= len(src) || src[pos] != '/' || src[pos+1] != '/' {
		return false
	}

	// A scheme separator immediately before the slashes marks a URL
	// ("http://host") rather than a comment. Reading it as one would swallow
	// the rest of the line and turn any reflow of that line into a spurious
	// rejection.
	return pos == 0 || src[pos-1] != ':'
}

// compareCommentBodies reports whether the comment text survived the format
// unchanged. Both sides arrive stripped of whitespace and folded to lower case,
// so reindenting and rewrapping are already accounted for; anything left is a
// real edit to what a comment says.
func compareCommentBodies(src, a, b []byte) error {
	if bytes.Equal(a, b) {
		return nil
	}

	at := 0
	for at < len(a) && at < len(b) && a[at] == b[at] {
		at++
	}

	// Locate the divergence in the original so the message points somewhere
	// the reader can actually look.
	line := byteOffsetToLine(src, commentBodyOffset(src, at))

	return fmt.Errorf("formatter changed comment text near line %d (%q became %q)",
		line, commentSnippet(a, at), commentSnippet(b, at))
}

// commentBodyOffset maps an index into the collected comment bytes back to a
// byte offset in src, by re-walking src and counting comment body bytes.
func commentBodyOffset(src []byte, target int) int {
	script := scriptRegionsOf(src)
	strs := stringSpansOf(src, script)
	seen, pos := 0, 0

	for pos < len(src) {
		var body []byte

		next := skipWSAndComments(src, pos, &body, script, strs)
		if next == pos {
			pos++

			continue
		}

		if seen+len(body) > target {
			return next
		}

		seen += len(body)
		pos = next
	}

	return len(src)
}

func commentSnippet(s []byte, at int) string {
	start := max(at-10, 0)

	end := min(at+20, len(s))

	return string(s[start:end])
}

// skipWSAndComments advances past whitespace and comments, appending the
// non-whitespace body of every comment it skips to sink when that is non-nil.
//
// Comments are stepped over rather than compared character by character
// because a comment's *extent* is what carries meaning. A "//" comment runs to
// the end of its line, so deleting that newline silently pulls whatever
// followed on the next line into the comment. A raw character walk cannot see
// that — joining two lines only removes whitespace, leaving the non-whitespace
// sequence identical — whereas skipping each comment and then requiring the
// surrounding code to line up catches it on the spot. The bodies collected in
// sink are compared separately, so a comment whose text was mangled is still
// reported even though its characters no longer take part in the walk.
func skipWSAndComments(src []byte, pos int, sink *[]byte, script scriptSpans, strs scriptSpans) int {
	for {
		pos = skipSpace(src, pos)

		// A comment cannot open inside a string literal. Without this the
		// guard reads a glob like "/#dir#/**" as a block comment and compares
		// everything up to the next "*/" as comment text — which refuses the
		// formatter's own deliberate insertions. See stringSpansOf.
		if strs.containsSorted(pos) {
			return pos
		}

		switch {
		case hasBytesAt(src, pos, "<!---"):
			end := indexBytesFrom(src, pos+5, "--->")
			if end < 0 {
				return pos
			}

			collectCommentBody(sink, src[pos+5:end])
			pos = end + 4
		case hasBytesAt(src, pos, "<!--"):
			// An HTML comment. It has to be recognised in its own right, not
			// left to the "//" rule below, because its body routinely contains
			// slashes ("<!-- // end-of-template -->") that would otherwise be
			// read as a line comment and swallow the rest of the line.
			end := indexBytesFrom(src, pos+4, "-->")
			if end < 0 {
				return pos
			}

			collectCommentBody(sink, src[pos+4:end])
			pos = end + 3
		case script.contains(pos) && hasBytesAt(src, pos, "/*"):
			end := indexBytesFrom(src, pos+2, "*/")
			if end < 0 {
				return pos
			}

			collectCommentBody(sink, src[pos+2:end])
			pos = end + 2
		case script.contains(pos) && isLineCommentStart(src, pos):
			// A "//" comment ends at the end of its line, and a line does not
			// always end in "\n": classic Mac files separate lines with a bare
			// "\r", and CFML written on one is not rare — TestBox ships a
			// fixture with 81 of them and no newline at all. Scanning for "\n"
			// alone ran the comment to end of file and collected every
			// remaining line as its body. CRLF is unaffected either way, since
			// stopping at the "\r" leaves the "\n" as the whitespace it is.
			end := pos + 2
			for end < len(src) && src[end] != '\n' && src[end] != '\r' {
				end++
			}

			collectCommentBody(sink, src[pos+2:end])
			pos = end
		default:
			return pos
		}
	}
}
