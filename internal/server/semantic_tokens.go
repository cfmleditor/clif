package server

import (
	"context"
	"encoding/json/v2"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// Semantic token type indices matching semanticTokensLegend.TokenTypes.
const (
	semTokNamespace uint32 = iota
	semTokType
	semTokFunction
	semTokMethod
	semTokParameter
	semTokVariable
)

// Semantic token modifier bitmasks matching semanticTokensLegend.TokenModifiers.
const (
	semModDeclaration uint32 = 1 << iota
	semModDefaultLibrary
)

var semanticTokensLegend = protocol.SemanticTokensLegend{
	TokenTypes: []string{
		"namespace",
		"type",
		"function",
		"method",
		"parameter",
		"variable",
	},
	TokenModifiers: []string{
		"declaration",
		"defaultLibrary",
	},
}

func (s *Server) semanticTokensProvider() protocol.SemanticTokensProvider {
	if !s.Features.SemanticTokens {
		return nil
	}

	return &protocol.SemanticTokensOptions{
		Legend: semanticTokensLegend,
		Full:   protocol.Boolean(true),
		Range:  protocol.Boolean(true),
	}
}

func (s *Server) handleSemanticTokensFull(_ context.Context, rawParams []byte) (any, error) {
	if !s.Features.SemanticTokens {
		return nil, nil
	}

	var params protocol.SemanticTokensParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	return s.semanticTokensForDocument(params.TextDocument.URI, nil)
}

func (s *Server) handleSemanticTokensRange(_ context.Context, rawParams []byte) (any, error) {
	if !s.Features.SemanticTokens {
		return nil, nil
	}

	var params protocol.SemanticTokensRangeParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	want := params.Range

	return s.semanticTokensForDocument(params.TextDocument.URI, &want)
}

func (s *Server) semanticTokensForDocument(docURI uri.URI, want *protocol.Range) (*protocol.SemanticTokens, error) {
	if _, open := s.getDocument(docURI); !open {
		return nil, nil
	}

	defer s.lockDoc(docURI)()

	content, ok := s.getDocument(docURI)
	if !ok {
		return nil, nil
	}

	s.mu.RLock()
	pr := s.parseResults[docURI]
	s.mu.RUnlock()

	file := cfpath.FromURI(string(docURI))
	data := computeSemanticTokens(content, file, pr, want)

	return &protocol.SemanticTokens{Data: data}, nil
}

type semFuncScope struct {
	start int
	end   int
	args  []parser.Argument
}

type semTokenEncoder struct {
	content string
	lines   parser.LineTable
	want    *protocol.Range
	scopes  []semFuncScope
	scopeIx int
	isCFC   bool

	// Incremental UTF-16 column cursor keeps column calculation O(N) even on
	// arbitrarily long single lines.
	curLine        int
	curLineByteOff int
	curLineUTF16   int

	// LSP delta state.
	hasPrev  bool
	lastLine uint32
	lastCol  uint32

	data []uint32
}

// computeSemanticTokens produces LSP delta-encoded semantic tokens in a single
// O(N) buffer-local pass with no tree-sitter, disk I/O, or cross-file lookups.
func computeSemanticTokens(content, file string, pr *parser.ParseResult, want *protocol.Range) []uint32 {
	if content == "" {
		return []uint32{}
	}

	lines := parser.NewLineTable(content)

	var regions []parser.Region
	if pr != nil && len(pr.Regions) > 0 {
		regions = pr.Regions
	} else {
		regions = parser.ClassifyRegions(content)
	}

	enc := semTokenEncoder{
		content: content,
		lines:   lines,
		want:    want,
		scopes:  buildSemFuncScopes(pr),
		isCFC:   strings.EqualFold(filepath.Ext(file), ".cfc"),
		curLine: -1,
		data:    make([]uint32, 0, min(len(content)/8, 4096)),
	}

	for _, r := range regions {
		if want != nil {
			endLine := lines.LineOf(min(r.Offset+len(r.Text), len(content)))
			if endLine < int(want.Start.Line) || r.StartLine > int(want.End.Line) {
				continue
			}
		}

		switch r.Kind {
		case parser.RegionScript:
			enc.scanScript(r.Text, r.Offset)
		case parser.RegionTag:
			enc.scanTagRegion(r.Text, r.Offset)
		default:
		}
	}

	return enc.data
}

func buildSemFuncScopes(pr *parser.ParseResult) []semFuncScope {
	if pr == nil || len(pr.Scopes) == 0 || len(pr.Funcs) == 0 {
		return nil
	}

	// Map non-generated functions by declaration line and by lowercase name so
	// we can attach parameters to each sorted FuncScope in O(F + S).
	byLine := make(map[int]*parser.FunctionDef, len(pr.Funcs))
	byName := make(map[string]*parser.FunctionDef, len(pr.Funcs))

	for i := range pr.Funcs {
		fn := &pr.Funcs[i]
		if fn.Generated || len(fn.Arguments) == 0 {
			continue
		}

		byLine[int(fn.Line)] = fn

		if fn.Name != "" {
			byName[strings.ToLower(fn.Name)] = fn
		}
	}

	if len(byLine) == 0 {
		return nil
	}

	out := make([]semFuncScope, 0, len(pr.Scopes))

	for _, sc := range pr.Scopes {
		fn := byLine[sc.Start]
		if fn == nil && sc.Name != "" {
			fn = byName[strings.ToLower(sc.Name)]
		}

		if fn == nil || len(fn.Arguments) == 0 {
			continue
		}

		out = append(out, semFuncScope{
			start: sc.Start,
			end:   sc.End,
			args:  fn.Arguments,
		})
	}

	return out
}

func (e *semTokenEncoder) isParamAtLine(line int, name string) bool {
	for e.scopeIx < len(e.scopes) && e.scopes[e.scopeIx].end < line {
		e.scopeIx++
	}

	if e.scopeIx < len(e.scopes) {
		sc := &e.scopes[e.scopeIx]
		if line >= sc.start && line <= sc.end {
			for i := range sc.args {
				if strings.EqualFold(sc.args[i].Name, name) {
					return true
				}
			}
		}
	}

	return false
}

func (e *semTokenEncoder) emit(byteStart, byteLen int, tokType, mods uint32) {
	if byteLen <= 0 || byteStart < 0 || byteStart+byteLen > len(e.content) {
		return
	}

	line := e.lines.LineOf(byteStart)

	if e.want != nil {
		if line < int(e.want.Start.Line) || line > int(e.want.End.Line) {
			return
		}
	}

	if line != e.curLine {
		e.curLine = line
		e.curLineByteOff = int(e.lines[line])
		e.curLineUTF16 = 0
	}

	if byteStart < e.curLineByteOff {
		// Defensive fallback if an offset ever arrives out of order on the same line.
		e.curLineByteOff = int(e.lines[line])
		e.curLineUTF16 = 0
	}

	if byteStart > e.curLineByteOff {
		e.curLineUTF16 += lex.UTF16Len(e.content[e.curLineByteOff:byteStart])
		e.curLineByteOff = byteStart
	}

	col := conv.Uint32(e.curLineUTF16)
	uLine := conv.Uint32(line)
	utf16Len := conv.Uint32(lex.UTF16Len(e.content[byteStart : byteStart+byteLen]))

	if utf16Len == 0 {
		return
	}

	if e.want != nil {
		if uLine == e.want.Start.Line && col+utf16Len <= e.want.Start.Character {
			return
		}

		if uLine == e.want.End.Line && col >= e.want.End.Character {
			return
		}
	}

	// LSP requires strictly non-decreasing token positions.
	if e.hasPrev {
		if uLine < e.lastLine || (uLine == e.lastLine && col <= e.lastCol) {
			return
		}
	}

	var deltaLine, deltaStart uint32

	switch {
	case !e.hasPrev:
		deltaLine = uLine
		deltaStart = col
	case uLine == e.lastLine:
		deltaLine = 0
		deltaStart = col - e.lastCol
	default:
		deltaLine = uLine - e.lastLine
		deltaStart = col
	}

	e.data = append(e.data, deltaLine, deltaStart, utf16Len, tokType, mods)
	e.hasPrev = true
	e.lastLine = uLine
	e.lastCol = col
}

// scanScript streams tokens from text in a single pass using a 3-token sliding
// window (prev, cur, next), avoiding any slice allocation for tokens.
func (e *semTokenEncoder) scanScript(text string, baseOffset int) {
	sc := parser.NewSourceScanner(text)

	var prev parser.Token

	cur := sc.NextSkipComments()
	if cur.Kind == parser.TokEOF {
		return
	}

	next := sc.NextSkipComments()

	inNewType := false
	expectFuncParams := false
	paramParenDepth := 0

	for cur.Kind != parser.TokEOF {
		switch cur.Kind {
		case parser.TokLParen:
			if expectFuncParams {
				paramParenDepth = 1
				expectFuncParams = false
			} else if paramParenDepth > 0 {
				paramParenDepth++
			}
		case parser.TokRParen:
			if paramParenDepth > 0 {
				paramParenDepth--
			}
		case parser.TokLBrace:
			expectFuncParams = false
			paramParenDepth = 0
		case parser.TokIdent:
			e.classifyScriptIdent(prev, cur, next, baseOffset, &inNewType, &expectFuncParams, paramParenDepth > 0)
		default:
			if cur.Kind != parser.TokDot {
				inNewType = false
			}
		}

		prev = cur
		cur = next

		if next.Kind != parser.TokEOF {
			next = sc.NextSkipComments()
		}
	}
}

func (e *semTokenEncoder) classifyScriptIdent(
	prev, cur, next parser.Token,
	baseOffset int,
	inNewType, expectFuncParams *bool,
	inFuncParams bool,
) {
	val := cur.Value
	byteStart := baseOffset + cur.Offset

	// Continuation of a dotted type path after `new` (e.g. `new models.User()`).
	if *inNewType && prev.Kind == parser.TokDot {
		e.emit(byteStart, len(val), semTokType, 0)

		if next.Kind != parser.TokDot {
			*inNewType = false
		}

		return
	}

	*inNewType = false

	// Function declaration name: `function foo(...)`.
	if prev.Kind == parser.TokIdent && strings.EqualFold(prev.Value, "function") {
		tokType := semTokFunction
		if e.isCFC {
			tokType = semTokMethod
		}

		e.emit(byteStart, len(val), tokType, semModDeclaration)

		*expectFuncParams = true

		return
	}

	// Anonymous function: `function(...)`.
	if strings.EqualFold(val, "function") && next.Kind == parser.TokLParen {
		*expectFuncParams = true

		return
	}

	// `new ComponentName` or `new path.to.Component`.
	if prev.Kind == parser.TokIdent && strings.EqualFold(prev.Value, "new") {
		e.emit(byteStart, len(val), semTokType, 0)

		if next.Kind == parser.TokDot {
			*inNewType = true
		}

		return
	}

	// Member access or member call: `obj.method()` or `obj.prop`.
	if prev.Kind == parser.TokDot {
		if next.Kind == parser.TokLParen {
			e.emit(byteStart, len(val), semTokMethod, 0)
		}

		return
	}

	// Bare function call: `foo(...)`.
	if next.Kind == parser.TokLParen {
		if isSemSkipCallKeyword(val) {
			return
		}

		if _, ok := docs.LookupFunction(val); ok {
			e.emit(byteStart, len(val), semTokFunction, semModDefaultLibrary)
		} else {
			e.emit(byteStart, len(val), semTokFunction, 0)
		}

		return
	}

	// CFML scope prefix: `variables.foo`, `local.x`, `arguments.id`, etc.
	if next.Kind == parser.TokDot && isCFMLScopeIdent(val) {
		e.emit(byteStart, len(val), semTokNamespace, 0)

		return
	}

	// Parameter of the enclosing function (declaration or reference).
	line := e.lines.LineOf(byteStart)

	if e.isParamAtLine(line, val) {
		var mods uint32
		if inFuncParams {
			mods = semModDeclaration
		}

		e.emit(byteStart, len(val), semTokParameter, mods)
	}
}

// scanTagRegion scans tag-based CFML regions for <cffunction name="...">,
// <cfargument name="...">, and expression tag bodies (<cfset>, <cfif>,
// <cfelseif>, <cfreturn>).
func (e *semTokenEncoder) scanTagRegion(text string, baseOffset int) {
	for i := 0; i < len(text); {
		lt := strings.IndexByte(text[i:], '<')
		if lt < 0 {
			break
		}

		tagStart := i + lt

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

		if exprStart, ok := expressionTagBodyStart(text, tagStart); ok {
			tagEnd := findExprTagClose(text, exprStart)
			if tagEnd < 0 {
				break
			}

			bodyEnd := tagEnd
			if bodyEnd > exprStart && text[bodyEnd-1] == '/' {
				bodyEnd--
			}

			if bodyEnd > exprStart {
				e.scanScript(text[exprStart:bodyEnd], baseOffset+exprStart)
			}

			i = tagEnd + 1

			continue
		}

		if attrFrom, tokType, ok := declTagAttrStart(text, tagStart, e.isCFC); ok {
			tagEnd := findExprTagClose(text, attrFrom)
			if tagEnd < 0 {
				break
			}

			if valStart, valEnd, found := findTagAttrValue(text, attrFrom, tagEnd, "name"); found {
				e.emit(baseOffset+valStart, valEnd-valStart, tokType, semModDeclaration)
			}

			i = tagEnd + 1

			continue
		}

		i = tagStart + 1
	}
}

func declTagAttrStart(text string, tagStart int, isCFC bool) (int, uint32, bool) {
	rest := text[tagStart+1:]

	if len(rest) > len("cffunction") && strings.EqualFold(rest[:len("cffunction")], "cffunction") && isTagWhitespace(rest[len("cffunction")]) {
		tokType := semTokFunction
		if isCFC {
			tokType = semTokMethod
		}

		return tagStart + 1 + len("cffunction") + 1, tokType, true
	}

	if len(rest) > len("cfargument") && strings.EqualFold(rest[:len("cfargument")], "cfargument") && isTagWhitespace(rest[len("cfargument")]) {
		return tagStart + 1 + len("cfargument") + 1, semTokParameter, true
	}

	return 0, 0, false
}

func isTagWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// findTagAttrValue locates a quoted single-line identifier value for attrName
// within text[from:to].
func findTagAttrValue(text string, from, to int, attrName string) (valStart, valEnd int, found bool) {
	i := from
	for i < to {
		for i < to && isTagWhitespace(text[i]) {
			i++
		}

		nameStart := i
		for i < to && (isSemIdentByte(text[i]) || text[i] == '-') {
			i++
		}

		name := text[nameStart:i]

		for i < to && isTagWhitespace(text[i]) {
			i++
		}

		if i >= to || text[i] != '=' {
			if i == nameStart {
				i++
			}

			continue
		}

		i++ // skip '='

		for i < to && isTagWhitespace(text[i]) {
			i++
		}

		if i >= to {
			break
		}

		if q := text[i]; q == '"' || q == '\'' {
			i++
			valStart = i

			for i < to && text[i] != q {
				i++
			}

			valEnd = i
			if i < to {
				i++
			}
		} else {
			valStart = i
			for i < to && !isTagWhitespace(text[i]) && text[i] != '/' {
				i++
			}

			valEnd = i
		}

		if strings.EqualFold(name, attrName) && valEnd > valStart && isSingleLineIdent(text[valStart:valEnd]) {
			return valStart, valEnd, true
		}
	}

	return 0, 0, false
}

func isSemIdentByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$'
}

func isSingleLineIdent(s string) bool {
	if s == "" {
		return false
	}

	for i := range len(s) {
		if !isSemIdentByte(s[i]) {
			return false
		}
	}

	return true
}

// asciiLowerSmall lowercases ASCII identifiers without allocating for already-
// lowercase inputs, which is the common case for parameter lookups.
func asciiLowerSmall(s string) string {
	for i := range len(s) {
		if s[i] >= 'A' && s[i] <= 'Z' {
			return strings.ToLower(s)
		}
	}

	return s
}

func isCFMLScopeIdent(name string) bool {
	switch asciiLowerSmall(name) {
	case "variables", "local", "arguments", "this", "super",
		"request", "session", "application", "cgi", "form",
		"url", "cookie", "server", "client", "thread",
		"attributes", "caller", "cfcatch", "file":
		return true
	default:
		return false
	}
}

func isSemSkipCallKeyword(name string) bool {
	switch asciiLowerSmall(name) {
	case "if", "elseif", "for", "while", "switch", "catch", "function",
		"return", "lock", "transaction", "savecontent", "thread", "param",
		"throw", "abort", "exit", "include", "new", "var", "case":
		return true
	default:
		return false
	}
}
