package server

import (
	"context"
	"encoding/json/v2"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
)

type byteSpan struct {
	start int
	end   int
}

// handleSelectionRange answers textDocument/selectionRange with nested
// selection chains covering words, strings, `#...#` interpolations, balanced
// brackets/blocks, CFML/HTML tags and elements, lines, and the full document.
func (s *Server) handleSelectionRange(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.SelectionRangeParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI
	if _, open := s.getDocument(docURI); !open {
		return nil, nil
	}

	defer s.lockDocOnly(docURI)()

	content, ok := s.getDocument(docURI)
	if !ok {
		return nil, nil
	}

	lines := parser.NewLineTable(content)
	columns := newUTF16Columns(content, lines)
	docSpans := collectSelectionSpans(content)

	out := make([]protocol.SelectionRange, len(params.Positions))
	for i, pos := range params.Positions {
		off := positionToByteOffset(content, lines, pos)
		out[i] = buildSelectionChain(content, lines, columns, docSpans, off, pos)
	}

	return out, nil
}

func positionToByteOffset(content string, lines parser.LineTable, pos protocol.Position) int {
	if content == "" || len(lines) == 0 {
		return 0
	}

	lineIdx := int(pos.Line)
	if lineIdx < 0 {
		return 0
	}

	if lineIdx >= len(lines) {
		return len(content)
	}

	lineStart := int(lines[lineIdx])
	lineEnd := len(content)

	if lineIdx+1 < len(lines) {
		lineEnd = int(lines[lineIdx+1])
		if lineEnd > lineStart && content[lineEnd-1] == '\n' {
			lineEnd--
		}

		if lineEnd > lineStart && content[lineEnd-1] == '\r' {
			lineEnd--
		}
	}

	col := utf16ColToByte(content[lineStart:lineEnd], int(pos.Character))

	return lineStart + col
}

func utf16ColToByte(line string, utf16Col int) int {
	if utf16Col <= 0 {
		return 0
	}

	units := 0
	for i := 0; i < len(line); {
		if units >= utf16Col {
			return i
		}

		r, size := utf8.DecodeRuneInString(line[i:])
		if r > 0xFFFF {
			units += 2
		} else {
			units++
		}

		i += size
	}

	return len(line)
}

func byteSpanToRange(content string, lines parser.LineTable, columns *utf16Columns, sp byteSpan) protocol.Range {
	if len(lines) == 0 {
		return protocol.Range{}
	}

	start := max(0, min(sp.start, len(content)))
	end := max(start, min(sp.end, len(content)))
	sLine := lines.LineOf(start)
	eLine := lines.LineOf(end)

	return protocol.Range{
		Start: protocol.Position{
			Line:      conv.Uint32(sLine),
			Character: conv.Uint32(columns.column(sLine, start)),
		},
		End: protocol.Position{
			Line:      conv.Uint32(eLine),
			Character: conv.Uint32(columns.column(eLine, end)),
		},
	}
}

func buildSelectionChain(content string, lines parser.LineTable, columns *utf16Columns, docSpans []byteSpan, off int, pos protocol.Position) protocol.SelectionRange {
	cands := make([]byteSpan, 0, 12)

	if wStart, wEnd, ok := wordSpanAt(content, off); ok {
		cands = append(cands, byteSpan{start: wStart, end: wEnd})
	}

	for _, sp := range docSpans {
		if sp.start <= off && off <= sp.end && sp.end > sp.start {
			cands = append(cands, sp)
		}
	}

	if lineIdx := lines.LineOf(off); lineIdx >= 0 && lineIdx < len(lines) {
		lStart := int(lines[lineIdx])
		lEnd := len(content)

		if lineIdx+1 < len(lines) {
			lEnd = int(lines[lineIdx+1])
			if lEnd > lStart && content[lEnd-1] == '\n' {
				lEnd--
			}

			if lEnd > lStart && content[lEnd-1] == '\r' {
				lEnd--
			}
		}

		if lEnd > lStart {
			cands = append(cands, byteSpan{start: lStart, end: lEnd})
		}
	}

	if content != "" {
		cands = append(cands, byteSpan{start: 0, end: len(content)})
	}

	if len(cands) == 0 {
		return protocol.SelectionRange{Range: protocol.Range{Start: pos, End: pos}}
	}

	slices.SortFunc(cands, func(a, b byteSpan) int {
		da, db := a.end-a.start, b.end-b.start
		if da != db {
			return da - db
		}

		if a.start != b.start {
			return a.start - b.start
		}

		return a.end - b.end
	})

	// Build a strictly nested sequence from innermost to outermost.
	nested := make([]byteSpan, 0, len(cands))
	for _, sp := range cands {
		if len(nested) == 0 {
			nested = append(nested, sp)

			continue
		}

		prev := nested[len(nested)-1]
		if sp.start <= prev.start && sp.end >= prev.end && (sp.start < prev.start || sp.end > prev.end) {
			nested = append(nested, sp)
		}
	}

	var cur *protocol.SelectionRange
	for _, sp := range slices.Backward(nested) {
		sr := protocol.SelectionRange{
			Range:  byteSpanToRange(content, lines, columns, sp),
			Parent: cur,
		}
		cur = &sr
	}

	return *cur
}

func wordSpanAt(content string, off int) (start, end int, ok bool) {
	if content == "" {
		return 0, 0, false
	}

	i := off
	if i == len(content) || !isWordByte(content[i]) {
		if i > 0 && isWordByte(content[i-1]) {
			i--
		} else {
			return 0, 0, false
		}
	}

	start = i
	for start > 0 && isWordByte(content[start-1]) {
		start--
	}

	end = i + 1
	for end < len(content) && isWordByte(content[end]) {
		end++
	}

	return start, end, end > start
}

func isWordByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$'
}

func collectSelectionSpans(content string) []byteSpan {
	var spans []byteSpan

	for _, r := range parser.ClassifyRegions(content) {
		switch r.Kind {
		case parser.RegionScript:
			spans = appendScriptSelectionSpans(spans, r.Text, r.Offset)
		case parser.RegionTag:
			spans = appendTagSelectionSpans(spans, r.Text, r.Offset)
		default:
		}
	}

	return spans
}

func appendScriptSelectionSpans(spans []byteSpan, text string, baseOffset int) []byteSpan {
	type openBracket struct {
		kind parser.TokenKind
		off  int
	}

	var stack []openBracket

	var openCount [3]int

	sc := parser.NewSourceScanner(text)

	for {
		t := sc.NextSkipComments()
		if t.Kind == parser.TokEOF {
			break
		}

		switch t.Kind {
		case parser.TokString:
			s := baseOffset + t.Offset
			e := s + len(t.Value)
			spans = append(spans, byteSpan{start: s, end: e})

			if len(t.Value) >= 2 {
				spans = append(spans, byteSpan{start: s + 1, end: e - 1})
			}
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			stack = append(stack, openBracket{kind: t.Kind, off: baseOffset + t.Offset})
			openCount[bracketSlot(t.Kind)]++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			want := matchingOpenKind(t.Kind)
			if openCount[bracketSlot(want)] == 0 {
				continue
			}

			for idx, op := range slices.Backward(stack) {
				openCount[bracketSlot(op.kind)]--
				if op.kind == want {
					openOff := op.off
					closeOff := baseOffset + t.Offset
					spans = append(spans, byteSpan{start: openOff, end: closeOff + 1})

					if closeOff > openOff+1 {
						spans = append(spans, byteSpan{start: openOff + 1, end: closeOff})
					}

					stack = stack[:idx]

					break
				}
			}
		default:
		}
	}

	return spans
}

func matchingOpenKind(closeKind parser.TokenKind) parser.TokenKind {
	switch closeKind {
	case parser.TokRParen:
		return parser.TokLParen
	case parser.TokRBracket:
		return parser.TokLBracket
	case parser.TokRBrace:
		return parser.TokLBrace
	default:
		return parser.TokEOF
	}
}

func appendTagSelectionSpans(spans []byteSpan, text string, baseOffset int) []byteSpan {
	type openTag struct {
		name      string
		openStart int
		openEnd   int
	}

	var stack []openTag

	openCount := make(map[string]int)

	spans = appendInterpolationSpans(spans, text, baseOffset)

	for i := 0; i < len(text); {
		lt := strings.IndexByte(text[i:], '<')
		if lt < 0 {
			break
		}

		tagStart := i + lt
		if strings.HasPrefix(text[tagStart:], "<!---") {
			if end := strings.Index(text[tagStart+5:], "--->"); end >= 0 {
				closeEnd := tagStart + 5 + end + 4
				spans = append(spans, byteSpan{start: baseOffset + tagStart, end: baseOffset + closeEnd})
				i = closeEnd

				continue
			}

			break
		}

		if strings.HasPrefix(text[tagStart:], "<!--") {
			if end := strings.Index(text[tagStart+4:], "-->"); end >= 0 {
				closeEnd := tagStart + 4 + end + 3
				spans = append(spans, byteSpan{start: baseOffset + tagStart, end: baseOffset + closeEnd})
				i = closeEnd

				continue
			}

			break
		}

		tagEnd := findExprTagClose(text, tagStart+1)
		if tagEnd < 0 {
			break
		}

		raw := text[tagStart+1 : tagEnd]
		trimmed := strings.TrimSpace(raw)

		if trimmed == "" {
			i = tagEnd + 1

			continue
		}

		fullStart := baseOffset + tagStart
		fullEnd := baseOffset + tagEnd + 1
		spans = append(spans, byteSpan{start: fullStart, end: fullEnd})

		if trimmed[0] == '/' {
			name := strings.ToLower(tagIdent(trimmed[1:]))
			if name != "" && openCount[name] > 0 {
				for idx, op := range slices.Backward(stack) {
					openCount[op.name]--
					if op.name == name {
						spans = append(spans, byteSpan{start: op.openStart, end: fullEnd})

						if fullStart > op.openEnd {
							spans = append(spans, byteSpan{start: op.openEnd, end: fullStart})
						}

						stack = stack[:idx]

						break
					}
				}
			}
		} else if trimmed[0] != '!' && !strings.HasSuffix(trimmed, "/") {
			name := strings.ToLower(tagIdent(trimmed))
			if name != "" && !isVoidSelectionTag(name) {
				stack = append(stack, openTag{name: name, openStart: fullStart, openEnd: fullEnd})
				openCount[name]++
			}
		}

		i = tagEnd + 1
	}

	return spans
}

func appendInterpolationSpans(spans []byteSpan, text string, baseOffset int) []byteSpan {
	for i := 0; i < len(text); i++ {
		if text[i] != '#' {
			continue
		}

		if i+1 < len(text) && text[i+1] == '#' {
			i++

			continue
		}

		// Find matching closing `#` on the same line, respecting quoted strings.
		var quote byte

		closeIdx := -1

		for j := i + 1; j < len(text); j++ {
			c := text[j]
			if c == '\n' || c == '\r' {
				break
			}

			if quote != 0 {
				if c == quote {
					if j+1 < len(text) && text[j+1] == quote {
						j++

						continue
					}

					quote = 0
				}

				continue
			}

			switch c {
			case '"', '\'':
				quote = c
			case '#':
				closeIdx = j
			}

			if closeIdx >= 0 {
				break
			}
		}

		if closeIdx > i+1 {
			s := baseOffset + i
			e := baseOffset + closeIdx + 1
			spans = append(spans, byteSpan{start: s, end: e}, byteSpan{start: s + 1, end: e - 1})
			i = closeIdx
		}
	}

	return spans
}

func tagIdent(s string) string {
	s = strings.TrimLeft(s, " \t\r\n")
	end := 0

	for end < len(s) {
		c := s[end]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == ':' {
			end++

			continue
		}

		break
	}

	return s[:end]
}

func isVoidSelectionTag(name string) bool {
	switch name {
	case "cfset", "cfreturn", "cfparam", "cfargument", "cfproperty", "cfinclude",
		"cfbreak", "cfcontinue", "cfabort", "cfexit", "cfrethrow", "cfthrow",
		"cfqueryparam", "cfhttpparam", "cfprocparam", "cfprocresult", "cflocation",
		"cfheader", "cfcontent", "cfcookie", "cfdump", "cflog", "cfsetting", "cfimport",
		"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}
