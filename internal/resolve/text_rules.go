package resolve

import (
	"regexp"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
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
	return lex.IsWord(c) || c == '.' || c == '$'
}

// foldAt reports whether s holds lit at offset at, ignoring ASCII case.
func foldAt(s string, at int, lit string) bool {
	if at < 0 || at+len(lit) > len(s) {
		return false
	}

	for i := range len(lit) {
		if a, b := s[at+i], lit[i]; a != b && lex.LowerASCII(a) != lex.LowerASCII(b) {
			return false
		}
	}

	return true
}

// skipSpaceBack returns the offset just past the last non-whitespace byte
// before i: the whitespace run ending at i is s[result:i].
func skipSpaceBack(s string, i int) int {
	for i > 0 && lex.IsRegexpSpace(s[i-1]) {
		i--
	}

	return i
}

// startsWord reports whether a \b falls before offset at, for a word byte
// there: nothing before it, or a byte that is not a word byte.
func startsWord(s string, at int) bool {
	return at == 0 || !lex.IsWord(s[at-1])
}

// endsWord reports whether a \b falls at offset at, after a byte that is a word
// byte: the text ends there or the next byte is not one.
func endsWord(s string, at int) bool {
	return at >= len(s) || !lex.IsWord(s[at])
}

// nextFold is the offset of the next occurrence of lit at or after from, or -1.
func nextFold(s, lit string, from int) int {
	return parser.IndexFoldFrom(s, lit, from)
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
	if o == 0 || !lex.IsRegexpSpace(s[o-1]) {
		return false
	}

	w := skipSpaceBack(s, o)
	if w < 3 || !foldAt(s, w-3, "var") || !startsWord(s, w-3) {
		return false
	}

	end := o + len(name)

	return lex.IsWord(name[len(name)-1]) != (end < len(s) && lex.IsWord(s[end]))
}

// loopsOverAt is `\bfor\s*\(\s*(?:var\s+)?NAME\s+in\b` for the occurrence at o.
func loopsOverAt(s string, o int, name string) bool {
	after := o + len(name)
	if after >= len(s) || !lex.IsRegexpSpace(s[after]) {
		return false
	}

	in := lex.SkipRegexpSpace(s, after)
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
		if assignOpAt(s, lex.SkipRegexpSpace(s, o+len(name))) && nameStartsHere(s, o) {
			return true
		}

		if declaresVarAt(s, o, name) || loopsOverAt(s, o, name) || namesVariableAt(s, o, name) {
			return true
		}
	}

	return false
}

// quotedNameStarts returns the offsets of the non-overlapping matches of
// `\bNAME["']` in s: the name, on a word boundary, with a quote straight after.
func quotedNameStarts(s, name string) []int {
	var starts []int

	lastEnd := 0

	for o := nextFold(s, name, 0); o >= 0; o = nextFold(s, name, o+1) {
		end := o + len(name)
		if end >= len(s) || s[end] != '"' && s[end] != '\'' || o < lastEnd {
			continue
		}

		if lex.IsWord(name[0]) != (o > 0 && lex.IsWord(s[o-1])) {
			starts = append(starts, o)
			lastEnd = end + 1
		}
	}

	return starts
}

// matchesAt is re.FindAllStringSubmatchIndex(s, -1) for an expression whose
// anchored form is re, given every offset it can match at in order: the
// leftmost matches that do not overlap, with indices in s.
func matchesAt(re *regexp.Regexp, s string, starts []int) [][]int {
	var out [][]int

	lastEnd := 0

	for _, start := range starts {
		if start < lastEnd {
			continue
		}

		m := re.FindStringSubmatchIndex(s[start:])
		if m == nil {
			continue
		}

		for i := range m {
			if m[i] >= 0 {
				m[i] += start
			}
		}

		out = append(out, m)
		lastEnd = m[1]
	}

	return out
}

// lineWritesName is the expression initStoredArg asks of one line,
// `(?:^|[^\w.])(?:variables\.)?NAME\s*=[^=]|\bsetNAME\s*\(`, for a plain name.
func lineWritesName(line, name string) bool {
	for o := nextFold(line, name, 0); o >= 0; o = nextFold(line, name, o+1) {
		if j := lex.SkipRegexpSpace(line, o+len(name)); j+1 < len(line) && line[j] == '=' && line[j+1] != '=' {
			if o == 0 || !lex.IsWord(line[o-1]) && line[o-1] != '.' {
				return true
			}

			if p := o - len("variables."); p >= 0 && foldAt(line, p, "variables.") && (p == 0 || !lex.IsWord(line[p-1]) && line[p-1] != '.') {
				return true
			}
		}

		if o >= 3 && foldAt(line, o-3, "set") && startsWord(line, o-3) {
			if open := lex.SkipRegexpSpace(line, o+len(name)); open < len(line) && line[open] == '(' {
				return true
			}
		}
	}

	return false
}

// memberAssignMatches is memberAssignRe(container, name).FindAllStringSubmatchIndex(body, -1).
//
// The expression opens with `^\s*(?:<cfset\s+)?(?:arguments\.)?`, and `\s*` crosses
// lines, so a match may begin at any line start in the whitespace before the
// optional prefixes. For each place `container.name` is written, those line starts
// are the candidates, and the expression, anchored with \A, decides at each. A
// name that is not plain takes the expression.
func memberAssignMatches(body, container, name string) [][]int {
	if !plainName(container) || !plainName(name) {
		return memberAssignRe(container, name).FindAllStringSubmatchIndex(body, -1)
	}

	return matchesAt(memberAssignAnchored(container, name), body, memberAssignStarts(body, container+"."+name))
}

// memberAssignAnchored is memberAssignRe with its leading `^` replaced by \A.
func memberAssignAnchored(container, name string) *regexp.Regexp {
	p := memberAssignRe(container, name).String()

	const head = `(?im)^`

	if !strings.HasPrefix(p, head) {
		panic("memberAssignAnchored: " + p + " does not begin with " + head)
	}

	return compiledPattern(`(?im)\A` + p[len(head):])
}

// memberAssignStarts are the offsets, in order, at which a match of
// memberAssignRe for the written text target can begin.
func memberAssignStarts(body, target string) []int {
	var starts []int

	for o := nextFold(body, target, 0); o >= 0; o = nextFold(body, target, o+1) {
		i := o
		if i >= len("arguments.") && foldAt(body, i-len("arguments."), "arguments.") {
			i -= len("arguments.")
		}

		cfset := -1
		if j := skipSpaceBack(body, i); j < i && j >= len("<cfset") && foldAt(body, j-len("<cfset"), "<cfset") {
			cfset = j - len("<cfset")
		}

		// The <cfset is optional, so the line starts before it count as well as
		// those before the written text itself.
		for _, end := range []int{i, cfset} {
			if end < 0 {
				continue
			}

			for q := skipSpaceBack(body, end); q <= end; q++ {
				if q == 0 || body[q-1] == '\n' {
					starts = append(starts, q)
				}
			}
		}
	}

	slices.Sort(starts)

	return slices.Compact(starts)
}
