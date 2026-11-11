package resolve

import (
	"slices"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"github.com/cfmleditor/clif/internal/parser"
)

// The resolver asks of a file, a function or a few lines whether they write a
// name — assign rc.x or prc.x, a member of a local, variables.field, a
// variable a template reads — or guard a call on it with structKeyExists,
// isDefined, keyExists, isNull or an instance check. Those questions were
// regular expressions built per name at run time and run over the text, so a
// write or a guard in a comment, or spelled inside a string, counted as code.
//
// They read the code now, as the call readers in code_calls.go do: codeOf and
// fileCode give a text's tokens with every comment skipped and each string
// read by the parse's rule, and each #...# expression inside a string as a
// list of its own. codeWrites finds the writes to a name in those lists under
// a writeRule saying which spellings count; the guards are codeCalls.

// codeMemo keeps, for the texts asked about most recently, their tokens
// (textCodeOf): the same file, view or function body is asked about once per
// reader and once per name it might write.
var codeMemo = &textCache[*textTokens]{size: textTokensSize}

// textTokens is one text read once as code: its tokens as the scanner gives
// them (newlines kept, comments skipped), and the code lists codeLists makes
// of them. Every whole-file reader in the resolver takes one or the other.
type textTokens struct {
	raw   []parser.Token
	lists [][]parser.Token

	// calls indexes every `name (` in lists by lowercased name, built on the
	// first textCalls: each reader asked the whole file for its own names, and
	// walked every token to find them.
	callsOnce sync.Once
	calls     map[string][]callAt
}

// callAt is where one call's name stands: lists[list][at].
type callAt struct{ list, at int32 }

func textTokensSize(t *textTokens) int {
	n := 48 + tokenBytes*cap(t.raw) + codeListsSize(t.lists)
	if len(t.lists) > 0 && len(t.lists[0]) > 0 && len(t.raw) > 0 && &t.lists[0][0] == &t.raw[0] {
		n -= tokenBytes * cap(t.lists[0]) // the top list is raw itself
	}

	return n
}

// tokenBytes is what a parser.Token holds.
const tokenBytes = int(unsafe.Sizeof(parser.Token{}))

func codeListsSize(lists [][]parser.Token) int {
	n := 0
	for _, l := range lists {
		n += 24 + tokenBytes*cap(l)
	}

	return n
}

// codeOf is textCode(text), remembered.
func codeOf(text string) [][]parser.Token {
	return textCode(text)
}

// fileCode is the code of pr's whole file, read once per parse.
func fileCode(pr *parser.ParseResult) [][]parser.Token {
	return textCode(pr.Content)
}

// fileCodeOn is the code of pr's lines from through to, 0-based and both
// included: the file's tokens, read once per parse, so that a comment opened
// above from is still a comment.
func fileCodeOn(pr *parser.ParseResult, from, to int) [][]parser.Token {
	tokens := fileTokens(pr)
	lo := sort.Search(len(tokens), func(i int) bool { return int(tokens[i].Line) >= from })
	hi := sort.Search(len(tokens), func(i int) bool { return int(tokens[i].Line) > to })

	if lo >= hi {
		return nil
	}

	return codeLists(tokens[lo:hi])
}

// fileCodeBetween is the code of pr's tokens that start at offsets from up to
// to.
func fileCodeBetween(pr *parser.ParseResult, from, to int) [][]parser.Token {
	tokens := fileTokens(pr)
	lo := sort.Search(len(tokens), func(i int) bool { return tokens[i].Offset >= from })
	hi := sort.Search(len(tokens), func(i int) bool { return tokens[i].Offset >= to })

	if lo >= hi {
		return nil
	}

	return codeLists(tokens[lo:hi])
}

// writeRule says which writes to a name a reader takes.
type writeRule struct {
	// scopes may stand before the name, at most one of them: variables.x
	// for x.
	scopes []string
	// qualified takes the name after any other qualifier as well: a.rc.x for
	// rc.x. Otherwise a name after a dot, other than one of scopes, is some
	// other variable's member.
	qualified bool
	// member takes a write to one member of the name, x.y = or x[k] =, and
	// no other.
	member bool
	// compound takes +=, -=, *=, /=, &=, ++ and -- as well as =.
	compound bool
}

// codeWrite is one write to a name in a list of tokens.
type codeWrite struct {
	tokens []parser.Token
	start  int // the written name's first token, its scope included
	op     int // the operator's first token
}

// at is the write's offset in the text read.
func (w codeWrite) at() int { return w.tokens[w.start].Offset }

// line is the write's line in the text read, 0-based.
func (w codeWrite) line() int { return int(w.tokens[w.start].Line) }

// codeWrites are the writes to name in lists that rule takes, in the order
// they are written. name is a variable's name as written, dotted or not.
func codeWrites(lists [][]parser.Token, name string, rule writeRule) []codeWrite {
	parts := strings.Split(name, ".")

	var out []codeWrite

	for _, list := range lists {
		for i := range list {
			end, ok := spelledAt(list, i, parts)
			if !ok {
				continue
			}

			start, ok := writtenFrom(list, i, rule)
			if !ok {
				continue
			}

			if rule.member {
				if end = memberEnd(list, end); end < 0 {
					continue
				}
			}

			if writeOpAt(list, end, rule.compound) {
				out = append(out, codeWrite{tokens: list, start: start, op: end})
			}
		}
	}

	if len(lists) > 1 {
		slices.SortStableFunc(out, func(a, b codeWrite) int { return a.at() - b.at() })
	}

	return out
}

// spelledAt reports whether the tokens from list[i] are the dotted name parts,
// ignoring case, and the index after them.
func spelledAt(list []parser.Token, i int, parts []string) (int, bool) {
	for k, part := range parts {
		j := i + 2*k
		if j >= len(list) || list[j].Kind != parser.TokIdent || !strings.EqualFold(list[j].Value, part) {
			return 0, false
		}

		if k > 0 && list[j-1].Kind != parser.TokDot {
			return 0, false
		}
	}

	return i + 2*len(parts) - 1, true
}

// writtenFrom is the first token of the name at list[i] as rule takes it: a
// scope of rule's before it, or the name itself.
func writtenFrom(list []parser.Token, i int, rule writeRule) (int, bool) {
	start := i
	if i >= 2 && list[i-1].Kind == parser.TokDot && list[i-2].Kind == parser.TokIdent &&
		slices.ContainsFunc(rule.scopes, func(s string) bool { return strings.EqualFold(s, list[i-2].Value) }) {
		start = i - 2
	}

	if rule.qualified || start == 0 {
		return start, true
	}

	switch list[start-1].Kind {
	case parser.TokDot, parser.TokDoubleColon:
		return 0, false
	default:
		return start, true
	}
}

// memberEnd is the index after one member step at list[i], .y or [k], or -1.
func memberEnd(list []parser.Token, i int) int {
	if i+1 < len(list) && list[i].Kind == parser.TokDot && list[i+1].Kind == parser.TokIdent {
		return i + 2
	}

	if i < len(list) && list[i].Kind == parser.TokLBracket {
		if closing := producerGroupEnd(list, i, parser.TokLBracket, parser.TokRBracket); closing > 0 {
			return closing + 1
		}
	}

	return -1
}

// adjacent reports whether b is written straight after a.
func adjacent(a, b parser.Token) bool {
	return a.Offset+len(a.Value) == b.Offset
}

// writeOpAt reports whether list[i] opens an assignment with a value after
// it, or with compound one of +=, -=, *=, /=, &= (with a value), ++ or --.
func writeOpAt(list []parser.Token, i int, compound bool) bool {
	if i >= len(list) {
		return false
	}

	switch t := list[i]; t.Kind {
	case parser.TokEquals:
		return i+1 < len(list) && isAssignAt(list, i)
	case parser.TokPlus, parser.TokMinus:
		if compound && i+1 < len(list) && list[i+1].Kind == t.Kind && adjacent(t, list[i+1]) {
			return true
		}

		return compound && compoundAt(list, i)
	case parser.TokStar, parser.TokSlash, parser.TokAmpersand:
		return compound && compoundAt(list, i)
	default:
		return false
	}
}

// compoundAt reports whether list[i] is the operator of op= with a value
// after it, the = not part of ==.
func compoundAt(list []parser.Token, i int) bool {
	if i+2 >= len(list) || list[i+1].Kind != parser.TokEquals || !adjacent(list[i], list[i+1]) {
		return false
	}

	return list[i+2].Kind != parser.TokEquals || !adjacent(list[i+1], list[i+2])
}

// endLine is the line the token t ends on.
func endLine(t parser.Token) int32 {
	return t.Line + int32(strings.Count(t.Value, "\n")) //nolint:gosec // a newline count of one token
}

// inTag reports whether the write is a <cfset>'s, `var` or not.
func (w codeWrite) inTag() bool {
	j := w.start - 1
	if j >= 0 && isWord(w.tokens[j], "var") {
		j--
	}

	return j >= 1 && isWord(w.tokens[j], "cfset") && w.tokens[j-1].Kind == parser.TokLT && adjacent(w.tokens[j-1], w.tokens[j])
}

// startsStatement reports whether the write is the first thing its statement
// does: the first code on its line, or after a ;, a brace or a <cfset.
func (w codeWrite) startsStatement() bool {
	j := w.start - 1
	if j < 0 {
		return true
	}

	before := w.tokens[j]
	if endLine(before) < w.tokens[w.start].Line {
		return true
	}

	switch before.Kind {
	case parser.TokSemicolon, parser.TokLBrace, parser.TokRBrace:
		return true
	default:
		return j >= 1 && isWord(before, "cfset") && w.tokens[j-1].Kind == parser.TokLT && adjacent(w.tokens[j-1], before)
	}
}

// value is the tokens of the value the write assigns: up to the ; or the
// closing bracket its statement ends at, a <cfset>'s closing > (a / before it
// left out), or the end of a line outside any bracket.
func (w codeWrite) value() []parser.Token {
	tokens, from := w.tokens, w.op+1
	if from < len(tokens) && tokens[w.op].Kind != parser.TokEquals {
		from++ // the = of op=, or the second + of ++
	}

	tag, depth := w.inTag(), 0

	for i := from; i < len(tokens); i++ {
		t := tokens[i]
		if depth == 0 && i > from && t.Line > endLine(tokens[i-1]) {
			return tokens[from:i]
		}

		switch t.Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			if depth == 0 {
				return tokens[from:i]
			}

			depth--
		case parser.TokSemicolon:
			if depth == 0 {
				return tokens[from:i]
			}
		case parser.TokGT:
			if tag && depth == 0 {
				if i > from && tokens[i-1].Kind == parser.TokSlash && adjacent(tokens[i-1], t) {
					return tokens[from : i-1]
				}

				return tokens[from:i]
			}
		default:
		}
	}

	return tokens[from:]
}

// tokensText is the text tokens span in content, from the first one's start to
// the last one's end.
func tokensText(content string, tokens []parser.Token) string {
	if len(tokens) == 0 {
		return ""
	}

	last := tokens[len(tokens)-1]

	return content[tokens[0].Offset : last.Offset+len(last.Value)]
}

// declaresName reports whether lists declare name rather than assign it: `var
// name`, a for-in loop over it, or a tag attribute or named argument that
// names it as the variable it fills (item, index, name, returnVariable,
// variable, result).
func declaresName(lists [][]parser.Token, name string) bool {
	parts := strings.Split(name, ".")

	for _, list := range lists {
		for i, t := range list {
			if t.Kind == parser.TokString {
				if lit, ok := stringLiteral(t); ok && strings.EqualFold(lit, name) && namesVariableBefore(list, i) {
					return true
				}

				continue
			}

			end, ok := spelledAt(list, i, parts)
			if !ok || i == 0 {
				continue
			}

			if isWord(list[i-1], "var") {
				return true
			}

			// for ( [var] name in
			j := i - 1
			if isWord(list[j], "var") {
				j--
			}

			if j >= 1 && list[j].Kind == parser.TokLParen && isWord(list[j-1], "for") && end < len(list) && isWord(list[end], "in") {
				return true
			}
		}
	}

	return false
}

// variableAttributes are the attributes and arguments whose string value
// names the variable a tag or call fills.
var variableAttributes = []string{"item", "index", "name", "returnvariable", "variable", "result"}

// namesVariableBefore reports whether the string at list[i] is the value of
// one of variableAttributes.
func namesVariableBefore(list []parser.Token, i int) bool {
	if i < 2 || !isAssignAt(list, i-1) || list[i-2].Kind != parser.TokIdent {
		return false
	}

	return slices.ContainsFunc(variableAttributes, func(a string) bool { return strings.EqualFold(a, list[i-2].Value) })
}

// spells reports whether tokens are text as written without spaces between
// them, ignoring case: arguments . event for arguments.event.
func spells(tokens []parser.Token, text string) bool {
	if len(tokens) == 0 {
		return false
	}

	rest := text

	for _, t := range tokens {
		if len(t.Value) > len(rest) || !strings.EqualFold(rest[:len(t.Value)], t.Value) {
			return false
		}

		rest = rest[len(t.Value):]
	}

	return rest == ""
}

// spelledBefore is the index of the first of the tokens ending just before
// list[end] that spell text, as spells reads it.
func spelledBefore(list []parser.Token, end int, text string) (int, bool) {
	rest := text

	for k := end - 1; k >= 0 && rest != ""; k-- {
		v := list[k].Value
		if len(v) > len(rest) || !strings.EqualFold(rest[len(rest)-len(v):], v) {
			return 0, false
		}

		if rest = rest[:len(rest)-len(v)]; rest == "" {
			return k, true
		}
	}

	return 0, false
}

// soleStringIs reports whether the argument is one string holding text,
// ignoring case.
func soleStringIs(piece []parser.Token, text string) bool {
	lit, ok := soleString(piece)

	return ok && strings.EqualFold(lit, text)
}

// existsGuards are the [start, end) offsets of the checks in lists that
// variable has a member funcName: structKeyExists( variable, "funcName" ),
// variable.keyExists( "funcName" ) and isDefined( "variable.funcName" ). A
// check's start is its function's name, or the variable's for keyExists.
func existsGuards(lists [][]parser.Token, variable, funcName string) [][2]int {
	var out [][2]int

	for _, c := range codeCalls(lists, "structKeyExists", "keyExists", "isDefined") {
		pieces := c.pieces()

		switch strings.ToLower(c.name().Value) {
		case "structkeyexists":
			if len(pieces) == 2 && spells(pieces[0], variable) && soleStringIs(pieces[1], funcName) {
				out = append(out, [2]int{c.name().Offset, c.end()})
			}
		case "keyexists":
			if c.before(1).Kind != parser.TokDot || len(pieces) != 1 || !soleStringIs(pieces[0], funcName) {
				continue
			}

			if start, ok := spelledBefore(c.tokens, c.at-1, variable); ok {
				out = append(out, [2]int{c.tokens[start].Offset, c.end()})
			}
		default:
			if len(pieces) == 1 && soleStringIs(pieces[0], variable+"."+funcName) {
				out = append(out, [2]int{c.name().Offset, c.end()})
			}
		}
	}

	return out
}

// instanceGuard is one isInstanceOf or checkForInstanceOf check of a
// variable: where it is, and the component it names.
type instanceGuard struct {
	start, end int
	comp       string
}

// instanceGuards are the checks in lists that variable is an instance of a
// component: isInstanceOf( variable, "a.b.C" ) or Mura's
// checkForInstanceOf( variable, "a.b.C" ).
func instanceGuards(lists [][]parser.Token, variable string) []instanceGuard {
	var out []instanceGuard

	for _, c := range codeCalls(lists, "isInstanceOf", "checkForInstanceOf") {
		pieces := c.pieces()
		if len(pieces) != 2 || !spells(pieces[0], variable) {
			continue
		}

		if comp, ok := soleString(pieces[1]); ok && comp != "" && strings.IndexFunc(comp, func(r rune) bool { return r != '.' && !isWordRune(r) }) < 0 {
			out = append(out, instanceGuard{start: c.name().Offset, end: c.end(), comp: comp})
		}
	}

	return out
}

// isWordRune is \w: an ASCII letter, digit or underscore.
func isWordRune(r rune) bool {
	return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}
