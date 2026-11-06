package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// The rules that ask a question of a file's text — does it declare a function
// of this name, does it assign this variable — were regular expressions run over
// the whole text, once per call site and name. On ColdBox and ContentBox that
// was 27% and 29% of an `unresolved` scan's CPU, nearly all of it matching: the
// name is in most files, so the prefilter in front of each passed, and a
// case-insensitive alternation then ran an automaton over every byte.
//
// These find the name instead, with the vectorised search `mentions` already
// uses, and check what must stand before and after it. The name is the rare
// token; the keywords around it are checked at the few places it occurs.
//
// Each is the regular expression it replaces and nothing more, held to it by a
// differential test over random text (text_rules_test.go), which keeps the
// expressions as the reference. They are exact for a name that is plain —
// ASCII, no whitespace, no '>' — which is every name a call site produces; any
// other name takes the expression as before. ASCII case folding only, as the
// prefilter before each already had: a rune that folds to an ASCII letter is
// longer than one byte, so it never stood in a slice the name's length.

// plainName reports whether name can be searched for byte for byte: non-empty,
// ASCII, and holding no whitespace or '>' that the expressions treat as
// structure.
func plainName(name string) bool {
	if name == "" {
		return false
	}

	for i := range len(name) {
		if c := name[i]; c <= ' ' || c >= 0x80 || c == '>' {
			return false
		}
	}

	return true
}

// isNameByte is the class a variable's name continues in, [\w.$].
func isNameByte(c byte) bool {
	return isWordByte(c) || c == '.' || c == '$'
}

func lowerByte(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}

	return c
}

// foldAt reports whether s holds lit at offset at, ignoring ASCII case.
func foldAt(s string, at int, lit string) bool {
	if at < 0 || at+len(lit) > len(s) {
		return false
	}

	for i := range len(lit) {
		if a, b := s[at+i], lit[i]; a != b && lowerByte(a) != lowerByte(b) {
			return false
		}
	}

	return true
}

// skipRegexSpace returns the first offset at or after i that is not \s as RE2
// reads it, which leaves out \v: the package's skipSpace does not.
func skipRegexSpace(s string, i int) int {
	for i < len(s) && isRegexSpace(s[i]) {
		i++
	}

	return i
}

// skipSpaceBack returns the offset just past the last non-whitespace byte
// before i: the whitespace run ending at i is s[result:i].
func skipSpaceBack(s string, i int) int {
	for i > 0 && isRegexSpace(s[i-1]) {
		i--
	}

	return i
}

// startsWord reports whether a \b falls before offset at, for a word byte
// there: nothing before it, or a byte that is not a word byte.
func startsWord(s string, at int) bool {
	return at == 0 || !isWordByte(s[at-1])
}

// endsWord reports whether a \b falls at offset at, after a byte that is a word
// byte: the text ends there or the next byte is not one.
func endsWord(s string, at int) bool {
	return at >= len(s) || !isWordByte(s[at])
}

// nextFold is the offset of the next occurrence of lit at or after from, or -1.
func nextFold(s, lit string, from int) int {
	return parser.IndexFoldFrom(s, lit, from)
}

// scriptFunctionOpens returns the end offsets (just past the parenthesis) of up
// to limit non-overlapping matches of `\bfunction\s+NAME\s*\(` in s.
func scriptFunctionOpens(s, name string, limit int) []int {
	var ends []int

	lastEnd := 0

	for o := nextFold(s, name, 0); o >= 0 && len(ends) < limit; o = nextFold(s, name, o+1) {
		if o == 0 || !isRegexSpace(s[o-1]) {
			continue
		}

		open := skipRegexSpace(s, o+len(name))
		if open >= len(s) || s[open] != '(' {
			continue
		}

		w := skipSpaceBack(s, o)
		if w < 8 || !foldAt(s, w-8, "function") || !startsWord(s, w-8) || w-8 < lastEnd {
			continue
		}

		ends = append(ends, open+1)
		lastEnd = open + 1
	}

	return ends
}

// tagFunctionNamed reports whether seg, the text of a <cffunction> tag between
// its name and its '>', holds `\bname\s*=\s*["']NAME["']`.
func tagFunctionNamed(seg, name string) bool {
	for k := nextFold(seg, "name", 1); k >= 0; k = nextFold(seg, "name", k+1) {
		if isWordByte(seg[k-1]) {
			continue
		}

		j := skipRegexSpace(seg, k+4)
		if j >= len(seg) || seg[j] != '=' {
			continue
		}

		j = skipRegexSpace(seg, j+1)
		if j >= len(seg) || seg[j] != '"' && seg[j] != '\'' || !foldAt(seg, j+1, name) {
			continue
		}

		if end := j + 1 + len(name); end < len(seg) && (seg[end] == '"' || seg[end] == '\'') {
			return true
		}
	}

	return false
}

// tagFunctionBodiesNamed returns the text between the '>' of up to limit
// non-overlapping `<cffunction\b[^>]*\bname\s*=\s*["']NAME["'][^>]*>(.*?)</cffunction>`
// matches in s and their </cffunction>.
func tagFunctionBodiesNamed(s, name string, limit int) []string {
	const closer = "</cffunction>"

	var bodies []string

	for c := nextFold(s, "<cffunction", 0); c >= 0 && len(bodies) < limit; {
		e := c + len("<cffunction")
		if e < len(s) && isWordByte(s[e]) {
			c = nextFold(s, "<cffunction", c+1)

			continue
		}

		gt := strings.IndexByte(s[e:], '>')
		if gt < 0 {
			break
		}

		tagEnd := e + gt

		if !tagFunctionNamed(s[e:tagEnd], name) {
			c = nextFold(s, "<cffunction", c+1)

			continue
		}

		end := nextFold(s, closer, tagEnd+1)
		if end < 0 {
			break
		}

		bodies = append(bodies, s[tagEnd+1:end])
		c = nextFold(s, "<cffunction", end+len(closer))
	}

	return bodies
}

// writesFunction reports whether s holds
// `\bfunction\s+NAME\s*\(|<cffunction[^>]+name\s*=\s*["']NAME["']`.
func writesFunction(s, name string) bool {
	if len(scriptFunctionOpens(s, name, 1)) == 1 {
		return true
	}

	for o := nextFold(s, name, 1); o >= 0; o = nextFold(s, name, o+1) {
		if s[o-1] != '"' && s[o-1] != '\'' {
			continue
		}

		if end := o + len(name); end >= len(s) || s[end] != '"' && s[end] != '\'' {
			continue
		}

		// ["'] NAME ["'] is preceded by `name\s*=\s*` inside a <cffunction> tag.
		eq := skipSpaceBack(s, o-1)
		if eq == 0 || s[eq-1] != '=' {
			continue
		}

		nameEnd := skipSpaceBack(s, eq-1)
		if nameEnd < 4 || !foldAt(s, nameEnd-4, "name") {
			continue
		}

		// `<cffunction[^>]+name`: the tag opens after the last '>' before the
		// attribute, with at least one byte between it and the attribute.
		nameStart := nameEnd - 4
		from := strings.LastIndexByte(s[:nameStart], '>') + 1

		if t := nextFold(s[:nameStart], "<cffunction", from); t >= 0 && t+len("<cffunction") < nameStart {
			return true
		}
	}

	return false
}

// assignOpAt reports whether s holds `[-+*/&]?=[^=]|\+\+|--` at offset j.
func assignOpAt(s string, j int) bool {
	if j >= len(s) {
		return false
	}

	switch c := s[j]; c {
	case '+', '-':
		if j+1 < len(s) && s[j+1] == c {
			return true
		}

		fallthrough
	case '*', '/', '&':
		return j+2 < len(s) && s[j+1] == '=' && s[j+2] != '='
	case '=':
		return j+1 < len(s) && s[j+1] != '='
	default:
		return false
	}
}

// nameStartsHere is `(?:^|[^\w.$])(?:local\.|variables\.)?` ending at offset o.
func nameStartsHere(s string, o int) bool {
	if o == 0 || !isNameByte(s[o-1]) {
		return true
	}

	for _, prefix := range [...]string{"local.", "variables."} {
		if p := o - len(prefix); p >= 0 && foldAt(s, p, prefix) && (p == 0 || !isNameByte(s[p-1])) {
			return true
		}
	}

	return false
}

// declaresVarAt is `\bvar\s+NAME\b` for the occurrence of name at o.
func declaresVarAt(s string, o int, name string) bool {
	if o == 0 || !isRegexSpace(s[o-1]) {
		return false
	}

	w := skipSpaceBack(s, o)
	if w < 3 || !foldAt(s, w-3, "var") || !startsWord(s, w-3) {
		return false
	}

	end := o + len(name)

	return isWordByte(name[len(name)-1]) != (end < len(s) && isWordByte(s[end]))
}

// loopsOverAt is `\bfor\s*\(\s*(?:var\s+)?NAME\s+in\b` for the occurrence at o.
func loopsOverAt(s string, o int, name string) bool {
	after := o + len(name)
	if after >= len(s) || !isRegexSpace(s[after]) {
		return false
	}

	in := skipRegexSpace(s, after)
	if !foldAt(s, in, "in") || !endsWord(s, in+2) {
		return false
	}

	p := skipSpaceBack(s, o)

	// `\(\s*NAME` and `\(\s*var\s+NAME`.
	open := -1

	switch {
	case p > 0 && s[p-1] == '(':
		open = p - 1
	case p < o && p >= 3 && foldAt(s, p-3, "var"):
		if q := skipSpaceBack(s, p-3); q > 0 && s[q-1] == '(' {
			open = q - 1
		}
	}

	if open < 0 {
		return false
	}

	f := skipSpaceBack(s, open)

	return f >= 3 && foldAt(s, f-3, "for") && startsWord(s, f-3)
}

// namesVariableAt is `\b(?:item|index|name|returnvariable|variable|result)\s*=\s*["']NAME["']`.
func namesVariableAt(s string, o int, name string) bool {
	if o == 0 || s[o-1] != '"' && s[o-1] != '\'' {
		return false
	}

	if end := o + len(name); end >= len(s) || s[end] != '"' && s[end] != '\'' {
		return false
	}

	eq := skipSpaceBack(s, o-1)
	if eq == 0 || s[eq-1] != '=' {
		return false
	}

	w := skipSpaceBack(s, eq-1)

	for _, word := range [...]string{"item", "index", "name", "returnvariable", "variable", "result"} {
		if start := w - len(word); start >= 0 && foldAt(s, start, word) && startsWord(s, start) {
			return true
		}
	}

	return false
}

// assignsName reports whether s holds, case-insensitively, any of
//
//	(?:^|[^\w.$])(?:local\.|variables\.)?NAME\s*(?:[-+*/&]?=[^=]|\+\+|--)
//	\bvar\s+NAME\b
//	\bfor\s*\(\s*(?:var\s+)?NAME\s+in\b
//	\b(?:item|index|name|returnvariable|variable|result)\s*=\s*["']NAME["']
func assignsName(s, name string) bool {
	for o := nextFold(s, name, 0); o >= 0; o = nextFold(s, name, o+1) {
		if assignOpAt(s, skipRegexSpace(s, o+len(name))) && nameStartsHere(s, o) {
			return true
		}

		if declaresVarAt(s, o, name) || loopsOverAt(s, o, name) || namesVariableAt(s, o, name) {
			return true
		}
	}

	return false
}
