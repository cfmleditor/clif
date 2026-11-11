package parser

import (
	"slices"
	"strings"
)

// tagFrame is one open tag in tagStructure's stack.
type tagFrame struct {
	name string
	// branches are the lines of a `<cfif>`'s `<cfelse>` and `<cfelseif>`.
	branches []int
	// line is the line of the tag's `<`.
	line int
}

// spanLineRatio is how many lines of a page tagStructure expects one span
// from, to size its result. An estimate: a page with more spans still grows
// the slice, and one with fewer has spare room.
const spanLineRatio = 4

// maxSpanPresize caps that estimate, so a file of blank lines at the size
// limit does not ask for a slice as large as it is.
const maxSpanPresize = 1 << 14

// maxTagSearch bounds how far down the stack a closing tag looks for its
// opener. Unclosed tags pile up — a `<p>` or `<li>` whose close is implied —
// and a close with no opener at all would otherwise search all of them, once
// per close.
const maxTagSearch = 64

// tagStructure appends the spans of the markup in content to spans: every
// comment, every element or CF tag spanning lines, every opening tag whose
// attributes wrap, and every `<cfelse>` and `<cfelseif>` branch. script lists
// the byte ranges of the script regions, which scriptStructure has read and
// this walk steps over.
//
// Tags pair on a stack. A closing tag pops to the nearest opener of the same
// name, case-insensitively, and whatever is left open above it is discarded:
// that is how `<cfset>`, `<br>` and an unclosed `<p>` fall out without a list
// of which tags have bodies. isBodilessTag is only there to keep the stack
// short.
func tagStructure(content string, script [][2]int, idx []int32, spans []Span) []Span {
	var stack []tagFrame

	// A page has about one span in four lines. Growing the slice from nothing
	// copied it over and over.
	spans = slices.Grow(spans, min(len(idx)/spanLineRatio, maxSpanPresize))

	ends := newTagEnds(content)
	lines := lineCursor{idx: idx}

	for pos := 0; pos < len(content); {
		if len(script) > 0 && pos >= script[0][0] {
			pos = max(pos, script[0][1])
			script = script[1:]

			continue
		}

		k := strings.IndexByte(content[pos:], '<')
		if k < 0 {
			break
		}

		lt := pos + k
		if len(script) > 0 && lt >= script[0][0] {
			pos = lt

			continue
		}

		rest := content[lt:]

		switch {
		case strings.HasPrefix(rest, "<!--"):
			end := len(content)

			if strings.HasPrefix(rest, "<!---") {
				end = SkipCFMLComment(content, lt)
			} else if c := strings.Index(rest[4:], "-->"); c >= 0 {
				end = lt + 4 + c + 3
			}

			if start, last := lines.at(lt), lines.at(end-1); last > start {
				spans = append(spans, Span{Start: start, End: last, Kind: SpanComment})
			}

			pos = end
		case len(rest) > 1 && rest[1] == '/':
			name := tagName(rest[2:])
			if name == "" {
				pos = lt + 2

				continue
			}

			pos = tagEnd(&ends, lt+2+len(name))
			stack = closeTag(stack, name, lines.at(lt), lines.at(lastContent(content, lt)), &spans)
		default:
			name := tagName(rest[1:])
			if name == "" {
				pos = lt + 1

				continue
			}

			gt := tagEnd(&ends, lt+1+len(name))
			pos = gt
			line := lines.at(lt)

			// An opening tag whose attributes wrap folds by itself, to the
			// line before its `>`, whether or not it has a body: `<button`
			// with an attribute a line, a twenty-line `<cfset x = {`.
			if last := lines.at(gt - 1); last > line {
				spans = append(spans, Span{Start: line, End: last, Kind: SpanBlock})
			}

			if selfClosing(content, lt, gt) || isBodilessTag(name) {
				continue
			}

			if isBranchTag(name) {
				branchCFIf(stack, line)

				continue
			}

			if raw := rawTextEnd(content, gt, name); raw >= 0 {
				// <script> and <style> hold JavaScript and CSS, where
				// `a<b` is not a tag: step to the closing tag, which the
				// next turn of the walk pairs.
				pos = raw
			}

			stack = pushTag(stack, name, line)
		}
	}

	return spans
}

// pushTag pushes an open tag onto stack. A frame popped earlier leaves its
// slot, and its branches list, in the slice's spare room: the new frame takes
// the slot and keeps the list's storage, so a page of many `<cfif>`s with an
// `<cfelse>` each does not allocate a list per `<cfif>`. Nothing else holds a
// popped frame's list; closeTag has read it before it pops.
func pushTag(stack []tagFrame, name string, line int) []tagFrame {
	n := len(stack)
	if n == cap(stack) {
		return append(stack, tagFrame{name: name, line: line})
	}

	stack = stack[:n+1]
	f := &stack[n]
	*f = tagFrame{name: name, line: line, branches: f.branches[:0]}

	return stack
}

// tagName returns the name at the start of s: a letter or underscore, then
// letters, digits, `_`, `-`, `.` and `:` (a cfimport prefix, `cf_` custom
// tags). Empty when s does not start a tag — `a < b`, `<=`, `<#`.
func tagName(s string) string {
	if s == "" || !isTagNameStart(s[0]) {
		return ""
	}

	i := 1
	for i < len(s) && isTagNamePart(s[i]) {
		i++
	}

	return s[:i]
}

func isTagNameStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
}

func isTagNamePart(c byte) bool {
	return isTagNameStart(c) || c >= '0' && c <= '9' || c == '-' || c == '.' || c == ':'
}

// tagEnd returns the offset just past the `>` ending the tag whose attributes
// begin at from, stepping over quoted values; the end of content when the
// tag is never closed. A `<` starting a tag before that `>` ends the tag where
// it stands, so the walk reads what it opens: `<input <cfif a>checked</cfif>>`
// puts a CF tag inside an HTML one, and swallowing it left the `</cfif>`
// unpaired.
//
// A tag name running to the end of content (`<a`, `<cf`) has no attributes
// at all, and from is then len(content): the arithmetic below made that a
// negative length and the slice of it panicked.
func tagEnd(ends *tagEnds, from int) int {
	content := ends.s
	if from >= len(content) {
		return len(content)
	}

	gt := ends.at(from)
	if gt < 0 {
		gt = len(content) - from - 1
	}

	var lt int

	if ends.ltFrom == from && ends.ltGT == gt {
		lt = ends.ltAt
	} else {
		lt = strings.IndexByte(content[from:from+gt], '<')
	}

	if lt >= 0 {
		lt = tagStartOutsideQuotes(content[from:from+gt], lt)
	}

	if lt >= 0 {
		return from + lt
	}

	return from + gt + 1
}

// tagStartOutsideQuotes is the offset in span, a tag's text before its '>',
// of the first '<' at or after first that starts a tag outside the tag's
// quoted values, or -1. A '<' inside a value is the value's text:
// hint="… <br />" and output="</cfif>" ended the tag there, and every
// attribute after it was lost to every reader. When the quotes do not pair
// up within the span, which is not markup but text, any '<' that starts a tag
// ends it, as before.
func tagStartOutsideQuotes(span string, first int) int {
	var quote byte

	inQuote := -1 // the first '<' starting a tag inside a quote, for when quotes do not pair

	for i := range len(span) {
		c := span[i]

		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			} else if c == '<' && inQuote < 0 && i >= first && startsTag(span[i+1:]) {
				inQuote = i
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '<' && i >= first && startsTag(span[i+1:]):
			return i
		}
	}

	if quote != 0 {
		return inQuote
	}

	return -1
}

// startsTag reports whether s, the text after a `<`, begins an opening or
// closing tag.
func startsTag(s string) bool {
	if s != "" && s[0] == '/' {
		s = s[1:]
	}

	return s != "" && isTagNameStart(s[0])
}

// lastContent returns the offset of the last non-space byte before lt, or lt
// when there is none.
func lastContent(content string, lt int) int {
	for i := lt - 1; i >= 0; i-- {
		switch content[i] {
		case ' ', '\t', '\r', '\n':
		default:
			return i
		}
	}

	return lt
}

// selfClosing reports whether the tag from lt to just past its `>` at gt ends
// in `/>`.
func selfClosing(content string, lt, gt int) bool {
	return gt-2 > lt && content[gt-1] == '>' && content[gt-2] == '/'
}

// closeTag pops stack to the nearest open tag named name, folding it to the
// closing tag on line. With no opener within maxTagSearch, the close is
// ignored and the stack left alone.
//
// A `<cfif>`'s branches end at contentLine, the line of the last thing before
// the `</cfif>`. A branch has no closing tag of its own, so, like a switch
// case, it keeps its last line on screen; and each runs to the `</cfif>`,
// since a `<cfelseif>` holds the branches after it, as tree-sitter read it.
func closeTag(stack []tagFrame, name string, line, contentLine int, spans *[]Span) []tagFrame {
	for i := len(stack) - 1; i >= 0 && i >= len(stack)-maxTagSearch; i-- {
		f := &stack[i]
		if !strings.EqualFold(f.name, name) {
			continue
		}

		if line > f.line {
			*spans = append(*spans, Span{Start: f.line, End: line, Kind: SpanBlock})
		}

		for _, b := range f.branches {
			if contentLine > b {
				*spans = append(*spans, Span{Start: b, End: contentLine, Kind: SpanCase})
			}
		}

		return stack[:i]
	}

	return stack
}

// branchCFIf records a `<cfelse>` or `<cfelseif>` branch, on line, of the
// nearest open `<cfif>`, for closeTag to fold. The `<cfif>`'s own first
// branch is not folded apart from the whole `<cfif>`.
func branchCFIf(stack []tagFrame, line int) {
	for i := len(stack) - 1; i >= 0 && i >= len(stack)-maxTagSearch; i-- {
		if f := &stack[i]; strings.EqualFold(f.name, "cfif") {
			f.branches = append(f.branches, line)

			return
		}
	}
}

// rawTextEnd returns, for a <script> or <style> whose open tag ends at from,
// the offset of its closing tag, or the end of content; -1 for any other tag,
// and for a block holding a CF tag, which ClassifyRegions does not treat as
// raw either: its `<cfif>`s and `<cfoutput>`s are walked like any others.
func rawTextEnd(content string, from int, name string) int {
	var closing string

	switch {
	case strings.EqualFold(name, "script"):
		closing = "</script"
	case strings.EqualFold(name, "style"):
		closing = "</style"
	default:
		return -1
	}

	end := len(content)
	if i := indexFoldASCII(content[from:], closing); i >= 0 {
		end = from + i
	}

	if indexFoldASCII(content[from:end], "<cf") >= 0 || indexFoldASCII(content[from:end], "</cf") >= 0 {
		return -1
	}

	return end
}

// indexFoldASCII is strings.Index, ASCII case-insensitive, for a needle
// starting with `<`.
func indexFoldASCII(s, needle string) int {
	for i := 0; i+len(needle) <= len(s); i++ {
		j := strings.IndexByte(s[i:], '<')
		if j < 0 {
			return -1
		}

		i += j
		if i+len(needle) <= len(s) && strings.EqualFold(s[i:i+len(needle)], needle) {
			return i
		}
	}

	return -1
}

// isBranchTag reports whether name starts a branch of a `<cfif>`.
func isBranchTag(name string) bool {
	return strings.EqualFold(name, "cfelse") || strings.EqualFold(name, "cfelseif")
}

// isBodilessTag reports whether name never has a closing tag: an HTML void
// element, or a CF tag that takes no body. Pushing one would only lengthen
// the stack until something popped past it.
func isBodilessTag(name string) bool {
	if len(name) > len("cfprocessingdirective") {
		return false
	}

	var buf foldScratch

	switch string(buf.lowerFold(name)) {
	case "br", "img", "input", "meta", "link", "hr", "area", "base", "col", "embed",
		"param", "source", "track", "wbr",
		"cfset", "cfreturn", "cfparam", "cfargument", "cfproperty", "cfinclude",
		"cfabort", "cfbreak", "cfcontinue", "cflocation", "cfqueryparam", "cfdump",
		"cfthrow", "cfrethrow", "cfheader", "cfcontent", "cfcookie", "cfinvokeargument",
		"cflog", "cfsetting", "cfprocparam", "cfprocresult", "cfexit", "cfflush",
		"cfimport", "cfprocessingdirective":
		return true
	default:
		return false
	}
}
