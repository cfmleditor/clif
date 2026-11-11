package resolve

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// The readers that look for a framework's calls in a file — setView(),
// hasMany(), map().to(), super.init(), dspObject_include() and the rest — take
// them from the file's parse (parsedCalls): the parse records the calls of
// readerCallNames, where they are written, with the receiver and the chain
// they are made on, and a reader reads a call's arguments from the tokens of
// its span. A regular expression over the text matched a call in a comment
// (`// setView( "old/view" )`) or inside a string, and then a walk of each
// file's tokens of its own read a function's declaration as a call of it.
//
// The guards and the windows of code the write readers walk are still tokens
// (codeLists, codeCalls). Code also runs inside a string:
// `"#renderView( 'x' )#"`. codeLists hands a reader the tokens of each such
// expression as a list of its own, so a call in one is read and a call is
// never split across a string's edge.

// maxInterpolationDepth bounds how deep codeLists follows expressions inside
// strings inside expressions.
const maxInterpolationDepth = 4

// codeLists is the code of tokens, a text's tokens as the scanner gives them:
// the tokens themselves with newlines dropped, then the tokens of each #...#
// expression inside one of their strings, and of the expressions inside
// those. Every token's offset and line are the text's.
func codeLists(tokens []parser.Token) [][]parser.Token {
	top := tokens
	if slices.ContainsFunc(tokens, func(t parser.Token) bool { return t.Kind == parser.TokNewline }) {
		top = make([]parser.Token, 0, len(tokens))

		for _, t := range tokens {
			if t.Kind != parser.TokNewline {
				top = append(top, t)
			}
		}
	}

	lists := [][]parser.Token{top}

	var walk func(list []parser.Token, depth int)

	walk = func(list []parser.Token, depth int) {
		if depth >= maxInterpolationDepth {
			return
		}

		for _, t := range list {
			if t.Kind != parser.TokString || strings.IndexByte(t.Value, '#') < 0 {
				continue
			}

			for _, span := range parser.InterpolatedSpans(t.Value) {
				inner := interpolationTokens(t, span)
				if len(inner) == 0 {
					continue
				}

				lists = append(lists, inner)
				walk(inner, depth+1)
			}
		}
	}

	walk(top, 0)

	return lists
}

// interpolationTokens are the tokens of the expression at span inside the
// string token t, placed where they are in t's text.
func interpolationTokens(t parser.Token, span [2]int) []parser.Token {
	base := t.Offset + span[0]
	line := t.Line + int32(strings.Count(t.Value[:span[0]], "\n")) //nolint:gosec // a newline count of one token

	var out []parser.Token

	sc := parser.NewScriptScanner(t.Value[span[0]:span[1]])
	for tok := sc.NextSkipComments(); tok.Kind != parser.TokEOF; tok = sc.NextSkipComments() {
		if tok.Kind == parser.TokNewline {
			continue
		}

		tok.Offset += base
		tok.Line += line
		out = append(out, tok)
	}

	return out
}

// textCode is codeLists of a whole file's text, kept for the texts asked
// about most recently (codeMemo): a file is asked by one reader after another
// (computed includes, view renders, Mura objects, element returns, a name's
// writes), and each tokenized it again, which took a Mura scan from 14 to 21
// passes over its text. The lists are shared; a caller does not change them.
func textCode(text string) [][]parser.Token {
	return codeMemo.of(text, textCodeOf).lists
}

// rawTokens is text's tokens as the scanner gives them, newlines kept, from
// the same entry as textCode: one read of the file for both.
func rawTokens(text string) []parser.Token {
	return codeMemo.of(text, textCodeOf).raw
}

// textCalls is codeCalls(textCode(text), names...), answered from the text's
// call index: the file's tokens are walked once for every name asked about.
// No reader asks it now; the framework readers take the parse's calls
// (parsedCalls).
func textCalls(text string, names ...string) []codeCall {
	entry := codeMemo.of(text, textCodeOf)
	entry.callsOnce.Do(func() { entry.calls = indexCalls(entry.lists) })

	var out []codeCall

	for _, name := range names {
		for _, c := range entry.calls[strings.ToLower(name)] {
			list := entry.lists[c.list]
			if closing := producerGroupEnd(list, int(c.at)+1, parser.TokLParen, parser.TokRParen); closing > 0 {
				out = append(out, codeCall{tokens: list, at: int(c.at), closing: closing})
			}
		}
	}

	if len(names) > 1 || len(entry.lists) > 1 {
		slices.SortStableFunc(out, func(a, b codeCall) int { return a.name().Offset - b.name().Offset })
	}

	return out
}

// indexCalls is every `name (` in lists, by lowercased name, in order.
func indexCalls(lists [][]parser.Token) map[string][]callAt {
	calls := map[string][]callAt{}

	for l, list := range lists {
		for i := 0; i+1 < len(list); i++ {
			if list[i].Kind == parser.TokIdent && list[i+1].Kind == parser.TokLParen {
				name := strings.ToLower(list[i].Value)
				calls[name] = append(calls[name], callAt{int32(l), int32(i)})
			}
		}
	}

	return calls
}

// readerCallNames are the calls the resolver's framework readers read:
// renders, announcements, Wheels associations, WireBox's map().to(), cborm's
// super.init(), includeUDF, Lucee's package listings and Mura's display
// objects. Every plain parse records them (plainParseText), so a reader takes
// a file's calls from the parse it already has, not from a walk of its own.
var readerCallNames = slices.Concat(viewRenderNames, interceptAnnouncers, []string{
	"hasMany", "belongsTo", "hasOne", "map", "to", "init", "includeUDF",
	"ComponentListPackageAsStruct", "dspObject_include",
})

// plainParseText is the plain parse of content, the text of file: no
// resolver, lookup or option but the readers' calls (readerCallNames).
func plainParseText(file, content string) *parser.ParseResult {
	return parser.ParseWithOptions(cfpath.ToURI(file), content, &parser.ParseOptions{FindCalls: readerCallNames})
}

// parsedCalls are the calls of names pr recorded, in the order they are
// written, each read from its span in pr's text: its name, its parenthesis,
// its arguments' tokens and the parenthesis closing them, at their offsets
// and lines in the file. A call without a span (the parse could not place
// it) is not one.
func parsedCalls(pr *parser.ParseResult, names ...string) []parsedCall {
	if pr == nil {
		return nil
	}

	sites := pr.CallsWritten(names...)
	out := make([]parsedCall, 0, len(sites))

	for i := range sites {
		if call, ok := spanCall(pr.Content, &sites[i]); ok {
			out = append(out, call)
		}
	}

	return out
}

// parsedCall is a call the parse recorded, as a codeCall over the tokens of
// its own span, with the parse's record of it.
type parsedCall struct {
	codeCall

	site *parser.CallSite
}

// spanCall reads the call site c from its span in content.
func spanCall(content string, c *parser.CallSite) (parsedCall, bool) {
	start, end := int(c.StartByte), int(c.EndByte)
	if !c.HasSpan || start >= end || end > len(content) {
		return parsedCall{}, false
	}

	tokens := make([]parser.Token, 0, 8)

	sc := parser.NewScriptScanner(content[start:end])
	for tok := sc.NextSkipComments(); tok.Kind != parser.TokEOF; tok = sc.NextSkipComments() {
		if tok.Kind == parser.TokNewline {
			continue
		}

		tok.Offset += start
		tok.Line += int32(c.Line) //nolint:gosec // a line number of one file
		tokens = append(tokens, tok)
	}

	if len(tokens) < 3 || tokens[0].Kind != parser.TokIdent || tokens[1].Kind != parser.TokLParen {
		return parsedCall{}, false
	}

	closing := producerGroupEnd(tokens, 1, parser.TokLParen, parser.TokRParen)
	if closing != len(tokens)-1 {
		return parsedCall{}, false
	}

	return parsedCall{tokens: tokens, at: 0, closing: closing, site: c}, true
}

// textCodeOf is the text's entry, uncached. The raw list keeps no spare
// capacity: tokenize sizes it from the text's length, and a file of mostly
// markup or comments has far fewer tokens.
func textCodeOf(text string) *textTokens {
	raw := tokenize(text)
	if cap(raw) > len(raw)+len(raw)/4 {
		raw = slices.Clip(slices.Clone(raw))
	}

	return &textTokens{raw: raw, lists: codeLists(raw)}
}

// codeCall is one call in a token list: the name at tokens[at], its
// parenthesis at at+1, the one closing it at closing.
type codeCall struct {
	tokens      []parser.Token
	at, closing int
}

// name is the called name's token.
func (c codeCall) name() parser.Token { return c.tokens[c.at] }

// args are the tokens between the call's parentheses.
func (c codeCall) args() []parser.Token { return c.tokens[c.at+2 : c.closing] }

// pieces are the call's arguments, split at the commas between them; none
// for an empty list.
func (c codeCall) pieces() [][]parser.Token {
	if c.closing == c.at+2 {
		return nil
	}

	return producerSplit(c.args())
}

// before is the token n before the called name, or one of kind EOF.
func (c codeCall) before(n int) parser.Token {
	if c.at-n >= 0 {
		return c.tokens[c.at-n]
	}

	return parser.Token{Kind: parser.TokEOF}
}

// end is the offset just past the closing parenthesis.
func (c codeCall) end() int { return c.tokens[c.closing].Offset + 1 }

// codeCalls are the calls in lists to any of names, ignoring case, or to any
// name when none is given, in the order they are written. A call whose
// parenthesis is never closed is not one.
func codeCalls(lists [][]parser.Token, names ...string) []codeCall {
	var out []codeCall

	for _, list := range lists {
		for i := 0; i+1 < len(list); i++ {
			if list[i].Kind != parser.TokIdent || list[i+1].Kind != parser.TokLParen {
				continue
			}

			if len(names) > 0 && !slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, list[i].Value) }) {
				continue
			}

			if closing := producerGroupEnd(list, i+1, parser.TokLParen, parser.TokRParen); closing > 0 {
				out = append(out, codeCall{tokens: list, at: i, closing: closing})
			}
		}
	}

	if len(lists) > 1 {
		slices.SortStableFunc(out, func(a, b codeCall) int { return a.name().Offset - b.name().Offset })
	}

	return out
}

// stringLiteral is what the string token t holds between its quotes.
func stringLiteral(t parser.Token) (string, bool) {
	if t.Kind != parser.TokString || len(t.Value) < 2 {
		return "", false
	}

	return t.Value[1 : len(t.Value)-1], true
}

// soleString is the literal an argument is when it is one string and nothing
// else.
func soleString(piece []parser.Token) (string, bool) {
	if len(piece) != 1 {
		return "", false
	}

	return stringLiteral(piece[0])
}

// namedArg splits an argument written `name = value`: its name as written and
// the value's tokens. The value is not empty.
func namedArg(piece []parser.Token) (string, []parser.Token, bool) {
	if len(piece) < 3 || piece[0].Kind != parser.TokIdent || piece[1].Kind != parser.TokEquals || piece[2].Kind == parser.TokEquals {
		return "", nil, false
	}

	return piece[0].Value, piece[2:], true
}

// firstArgValue is the value of the first argument, given positionally or as
// `name = value`; nil when it is named something else.
func firstArgValue(pieces [][]parser.Token, name string) []parser.Token {
	if len(pieces) == 0 {
		return nil
	}

	if n, value, ok := namedArg(pieces[0]); ok {
		if !strings.EqualFold(n, name) {
			return nil
		}

		return value
	}

	return pieces[0]
}

// namedArgValue is the value of the argument written `name = value`, or nil.
func namedArgValue(pieces [][]parser.Token, name string) []parser.Token {
	for _, piece := range pieces {
		if n, value, ok := namedArg(piece); ok && strings.EqualFold(n, name) {
			return value
		}
	}

	return nil
}

// isWord reports whether t is the identifier word, ignoring case.
func isWord(t parser.Token, word string) bool {
	return t.Kind == parser.TokIdent && strings.EqualFold(t.Value, word)
}

// isAssignAt reports whether tokens[i] is an `=` that assigns: not part of
// `==`, `!=`, `<=` or `>=`.
func isAssignAt(tokens []parser.Token, i int) bool {
	if i >= len(tokens) || tokens[i].Kind != parser.TokEquals {
		return false
	}

	if i+1 < len(tokens) && tokens[i+1].Kind == parser.TokEquals && tokens[i+1].Offset == tokens[i].Offset+1 {
		return false
	}

	if i > 0 && tokens[i-1].Offset+len(tokens[i-1].Value) == tokens[i].Offset {
		switch tokens[i-1].Kind {
		case parser.TokEquals, parser.TokBang, parser.TokLT, parser.TokGT:
			return false
		default:
		}
	}

	return true
}

// dottedAt reads the name ident(.ident)* starting at tokens[i], and the index
// after it; "" when tokens[i] is no identifier.
func dottedAt(tokens []parser.Token, i int) (string, int) {
	if i >= len(tokens) || tokens[i].Kind != parser.TokIdent {
		return "", i
	}

	parts := []string{tokens[i].Value}
	i++

	for i+1 < len(tokens) && tokens[i].Kind == parser.TokDot && tokens[i+1].Kind == parser.TokIdent {
		parts = append(parts, tokens[i+1].Value)
		i += 2
	}

	return strings.Join(parts, "."), i
}
