package resolve

import (
	"sort"
	"strings"
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
}

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

// adjacent reports whether b is written straight after a.
func adjacent(a, b parser.Token) bool {
	return a.Offset+len(a.Value) == b.Offset
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
