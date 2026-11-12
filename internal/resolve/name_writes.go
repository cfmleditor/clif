package resolve

import (
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
// They read the parse now: the writes are its assignments (assignments.go)
// and the guards its calls (guardCalls, guarded_call.go). fileCode, still
// read by other readers, gives a text's tokens with every comment skipped and
// each string read by the parse's rule, and each #...# expression inside a
// string as a list of its own.

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

// isWordRune is \w: an ASCII letter, digit or underscore.
func isWordRune(r rune) bool {
	return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}
