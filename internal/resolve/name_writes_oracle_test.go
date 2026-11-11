package resolve

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// The token readers the write readers used before they read the parse's
// assignments (parser.Assignment), kept as the oracle the differential and
// the tests hold the records to.

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
	return t.Line + int32(strings.Count(t.Value, "\n"))
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

// tokenDeclaresName reports whether lists declare name rather than assign it: `var
// name`, a for-in loop over it, or a tag attribute or named argument that
// names it as the variable it fills (item, index, name, returnVariable,
// variable, result).
func tokenDeclaresName(lists [][]parser.Token, name string) bool {
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

// tokenVariableAttributes are the attributes and arguments whose string value
// names the variable a tag or call fills.
var tokenVariableAttributes = []string{"item", "index", "name", "returnvariable", "variable", "result"}

// namesVariableBefore reports whether the string at list[i] is the value of
// one of variableAttributes.
func namesVariableBefore(list []parser.Token, i int) bool {
	if i < 2 || !isAssignAt(list, i-1) || list[i-2].Kind != parser.TokIdent {
		return false
	}

	return slices.ContainsFunc(tokenVariableAttributes, func(a string) bool { return strings.EqualFold(a, list[i-2].Value) })
}
