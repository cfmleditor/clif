package server

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type paramHintSpec struct {
	name     string
	required bool
}

type callArgSpan struct {
	pos   protocol.Position
	named bool
}

func (s *Server) parameterInlayHints(docURI uri.URI, file, content string, want protocol.Range) []protocol.InlayHint {
	if content == "" {
		return nil
	}

	lines := parser.NewLineTable(content)
	posOf := func(off int) protocol.Position {
		if off < 0 {
			off = 0
		}

		if off > len(content) {
			off = len(content)
		}

		line := lines.LineOf(off)
		lineStart := lines[line]

		return protocol.Position{
			Line:      conv.Uint32(line),
			Character: conv.Uint32(lex.UTF16Len(content[lineStart:off])),
		}
	}

	baseDir := filepath.Dir(file)
	wantEndOff := positionToByteOffset(content, lines, want.End)

	var hints []protocol.InlayHint

	for _, r := range parser.ClassifyRegions(content) {
		endLine := lines.LineOf(min(r.Offset+len(r.Text), len(content)))
		if endLine < int(want.Start.Line) || r.StartLine > int(want.End.Line) {
			continue
		}

		switch r.Kind {
		case parser.RegionScript:
			hints = s.appendCallHints(hints, docURI, baseDir, r.Text, r.Offset, wantEndOff, posOf, want)
		case parser.RegionTag:
			hints = s.appendTagExprHints(hints, docURI, baseDir, r.Text, r.Offset, wantEndOff, posOf, want)
		default:
		}
	}

	return hints
}

func (s *Server) appendTagExprHints(hints []protocol.InlayHint, docURI uri.URI, baseDir, text string, baseOffset, wantEndOff int, posOf func(int) protocol.Position, want protocol.Range) []protocol.InlayHint {
	for i := 0; i < len(text); {
		lt := strings.IndexByte(text[i:], '<')
		if lt < 0 {
			break
		}

		tagStart := i + lt
		if baseOffset+tagStart > wantEndOff {
			break
		}

		// Skip CFML and HTML comments.
		if strings.HasPrefix(text[tagStart:], "<!---") {
			if end := strings.Index(text[tagStart+5:], "--->"); end >= 0 {
				i = tagStart + 5 + end + 4

				continue
			}

			break
		}

		if strings.HasPrefix(text[tagStart:], "<!--") {
			if end := strings.Index(text[tagStart+4:], "-->"); end >= 0 {
				i = tagStart + 4 + end + 3

				continue
			}

			break
		}

		exprStart, ok := expressionTagBodyStart(text, tagStart)
		if !ok {
			i = tagStart + 1

			continue
		}

		tagEnd := findExprTagClose(text, exprStart)
		if tagEnd < 0 {
			break
		}

		bodyEnd := tagEnd
		if bodyEnd > exprStart && text[bodyEnd-1] == '/' {
			bodyEnd--
		}

		if bodyEnd > exprStart {
			hints = s.appendCallHints(hints, docURI, baseDir, text[exprStart:bodyEnd], baseOffset+exprStart, wantEndOff, posOf, want)
		}

		i = tagEnd + 1
	}

	return hints
}

func expressionTagBodyStart(text string, tagStart int) (int, bool) {
	rest := text[tagStart+1:]
	for _, tag := range []string{"cfset", "cfif", "cfelseif", "cfreturn"} {
		if len(rest) > len(tag) && strings.EqualFold(rest[:len(tag)], tag) {
			next := rest[len(tag)]
			if next == ' ' || next == '\t' || next == '\n' || next == '\r' {
				return tagStart + 1 + len(tag) + 1, true
			}
		}
	}

	return 0, false
}

func findExprTagClose(text string, from int) int {
	var quote byte

	for i := from; i < len(text); i++ {
		c := text[i]
		if quote != 0 {
			if c == quote {
				if i+1 < len(text) && text[i+1] == quote {
					i++

					continue
				}

				quote = 0
			}

			continue
		}

		switch c {
		case '"', '\'':
			quote = c
		case '>':
			return i
		}
	}

	return -1
}

func (s *Server) appendCallHints(hints []protocol.InlayHint, docURI uri.URI, baseDir, text string, baseOffset, wantEndOff int, posOf func(int) protocol.Position, want protocol.Range) []protocol.InlayHint {
	toks := scanAllTokens(text)
	matchClose := buildBracketMatches(toks)
	padRight := true

	for i := range toks {
		if baseOffset+toks[i].Offset > wantEndOff {
			break
		}

		if toks[i].Kind != parser.TokIdent || i+1 >= len(toks) || toks[i+1].Kind != parser.TokLParen {
			continue
		}

		name := toks[i].Value
		if isInlaySkipKeyword(name) {
			continue
		}

		if i > 0 && toks[i-1].Kind == parser.TokIdent && strings.EqualFold(toks[i-1].Value, "function") {
			continue
		}

		closeIdx := matchClose[i+1]
		if closeIdx < 0 {
			continue
		}

		openPos := posOf(baseOffset + toks[i+1].Offset)
		closePos := posOf(baseOffset + toks[closeIdx].Offset)

		if posBefore(closePos, want.Start) || posBefore(want.End, openPos) {
			continue
		}

		args, ok := collectCallArgs(toks, i+2, closeIdx, matchClose, baseOffset, posOf)
		if !ok || len(args) == 0 {
			continue
		}

		hasVisiblePositional := false

		for _, a := range args {
			if !a.named && positionInRange(a.pos, want) {
				hasVisiblePositional = true

				break
			}
		}

		if !hasVisiblePositional {
			continue
		}

		callLine := posOf(baseOffset + toks[i].Offset).Line
		qualifier, isNew := callQualifier(toks, i)
		params := s.resolveCallParams(qualifier, name, isNew, docURI, baseDir, callLine)

		if len(params) == 0 || !arityMatches(args, params) {
			continue
		}

		for idx, arg := range args {
			if arg.named || !positionInRange(arg.pos, want) {
				continue
			}

			paramName := params[idx].name
			if paramName == "" {
				continue
			}

			hints = append(hints, protocol.InlayHint{
				Position:     arg.pos,
				Label:        protocol.String(paramName + ":"),
				Kind:         protocol.InlayHintKindParameter,
				PaddingRight: &padRight,
			})
		}
	}

	return hints
}

func scanAllTokens(text string) []parser.Token {
	sc := parser.NewSourceScanner(text)
	toks := make([]parser.Token, 0, len(text)/6)

	for {
		t := sc.NextSkipComments()
		if t.Kind == parser.TokEOF {
			return toks
		}

		toks = append(toks, t)
	}
}

func isInlaySkipKeyword(name string) bool {
	switch strings.ToLower(name) {
	case "if", "for", "while", "switch", "catch", "function", "return",
		"else", "try", "case", "default", "var", "new", "throw",
		"not", "and", "or", "xor", "eqv", "imp", "mod", "in", "contains":
		return true
	default:
		return false
	}
}

// buildBracketMatches precomputes the matching closing-token index for every
// `(`, `[`, and `{` in toks in a single O(N) pass (-1 when unmatched). This
// keeps both call matching and nested argument splitting strictly linear even
// on deeply nested closures or unclosed calls.
func buildBracketMatches(toks []parser.Token) []int {
	matches := make([]int, len(toks))
	for i := range matches {
		matches[i] = -1
	}

	type openItem struct {
		kind parser.TokenKind
		idx  int
	}

	var stack []openItem

	var openCount [3]int

	for i := range toks {
		switch toks[i].Kind {
		case parser.TokLParen:
			stack = append(stack, openItem{kind: parser.TokLParen, idx: i})
			openCount[0]++
		case parser.TokLBracket:
			stack = append(stack, openItem{kind: parser.TokLBracket, idx: i})
			openCount[1]++
		case parser.TokLBrace:
			stack = append(stack, openItem{kind: parser.TokLBrace, idx: i})
			openCount[2]++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			want := matchingOpenKind(toks[i].Kind)
			slot := bracketSlot(want)

			if openCount[slot] == 0 {
				continue
			}

			for j, op := range slices.Backward(stack) {
				openCount[bracketSlot(op.kind)]--
				if op.kind == want {
					matches[op.idx] = i
					stack = stack[:j]

					break
				}
			}
		default:
		}
	}

	return matches
}

func bracketSlot(openKind parser.TokenKind) int {
	switch openKind {
	case parser.TokLParen:
		return 0
	case parser.TokLBracket:
		return 1
	default:
		return 2
	}
}

func collectCallArgs(toks []parser.Token, start, end int, matchClose []int, baseOffset int, posOf func(int) protocol.Position) ([]callArgSpan, bool) {
	if start >= end {
		return nil, true
	}

	var args []callArgSpan

	argStart := start

	for i := start; i <= end; i++ {
		if i == end || toks[i].Kind == parser.TokComma {
			if argStart >= i {
				return nil, false
			}

			named := isNamedArgSlice(toks[argStart:i])
			args = append(args, callArgSpan{
				pos:   posOf(baseOffset + toks[argStart].Offset),
				named: named,
			})
			argStart = i + 1

			continue
		}

		// Jump over any matched inner (...), [...], or {...} in O(1) so nested
		// calls and closure bodies are never re-walked by outer calls.
		if c := matchClose[i]; c > i && c < end {
			i = c
		}
	}

	return args, true
}

func isNamedArgSlice(arg []parser.Token) bool {
	if len(arg) < 2 || arg[0].Kind != parser.TokIdent {
		return false
	}

	if arg[1].Kind == parser.TokEquals {
		return len(arg) < 3 || arg[2].Kind != parser.TokEquals
	}

	return arg[1].Kind == parser.TokColon
}

func callQualifier(toks []parser.Token, nameIdx int) (qualifier string, isNew bool) {
	// Collect a dotted prefix immediately before nameIdx: `a.b.foo(` or `new a.b.Foo(`.
	start := nameIdx
	for start >= 2 && toks[start-1].Kind == parser.TokDot && toks[start-2].Kind == parser.TokIdent {
		start -= 2
	}

	if start > 0 && toks[start-1].Kind == parser.TokIdent && strings.EqualFold(toks[start-1].Value, "new") {
		var b strings.Builder

		for j := start; j <= nameIdx; j += 2 {
			if j > start {
				b.WriteByte('.')
			}

			b.WriteString(toks[j].Value)
		}

		return b.String(), true
	}

	if start == nameIdx {
		return "", false
	}

	var b strings.Builder

	for j := start; j < nameIdx; j += 2 {
		if j > start {
			b.WriteByte('.')
		}

		b.WriteString(toks[j].Value)
	}

	return b.String(), false
}

func (s *Server) resolveCallParams(qualifier, name string, isNew bool, docURI uri.URI, baseDir string, line uint32) []paramHintSpec {
	if isNew {
		if def := s.getResolver().ResolveFunc(qualifier, "init", baseDir); def != nil {
			return userFuncParams(def)
		}

		return nil
	}

	if qualifier != "" {
		if def := s.resolveUserFunc(qualifier, name, docURI, line); def != nil {
			return userFuncParams(def)
		}

		return nil
	}

	if e, ok := docs.LookupFunction(name); ok {
		return builtinFuncParams(e)
	}

	def, inFile, total := s.index.LookupPreferred(name, docURI)
	if def != nil && (inFile || total == 1) {
		return userFuncParams(def)
	}

	return nil
}

func userFuncParams(def *parser.FunctionDef) []paramHintSpec {
	if len(def.Arguments) == 0 {
		return nil
	}

	out := make([]paramHintSpec, len(def.Arguments))
	for i, a := range def.Arguments {
		out[i] = paramHintSpec{name: a.Name, required: a.Required}
	}

	return out
}

func builtinFuncParams(e *docs.Entry) []paramHintSpec {
	if len(e.Params) == 0 {
		return nil
	}

	out := make([]paramHintSpec, len(e.Params))
	for i := range e.Params {
		p := &e.Params[i]
		out[i] = paramHintSpec{name: p.Name, required: p.Required}
	}

	return out
}

func arityMatches(args []callArgSpan, params []paramHintSpec) bool {
	if len(args) > len(params) {
		return false
	}

	required := 0
	for _, p := range params {
		if p.required {
			required++
		}
	}

	return len(args) >= required
}
