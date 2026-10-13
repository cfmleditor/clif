package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// setCallSpan records a lexical origin only while the scanner is still at the
// argument list. Synthetic or already-consumed calls remain explicitly unknown.
func (p *scriptParser) setCallSpan(call *CallSite) {
	if call.HasSpan || p.baseOffset < 0 {
		return
	}

	open := p.sc.PeekSkipComments()
	if open.Kind != TokLParen {
		return
	}

	endName := open.Offset
	for endName > 0 && (p.sc.src[endName-1] == ' ' || p.sc.src[endName-1] == '\t' || p.sc.src[endName-1] == '\r' || p.sc.src[endName-1] == '\n') {
		endName--
	}

	start := endName - len(call.FuncName)
	if start < 0 || !strings.EqualFold(p.sc.src[start:endName], call.FuncName) {
		return
	}

	sc := *p.sc
	depth := 0

	for {
		t := sc.NextSkipComments()
		switch t.Kind {
		case TokEOF:
			return
		case TokLParen:
			depth++
		case TokRParen:
			depth--
			if depth == 0 {
				call.StartByte = conv.Uint32(p.baseOffset + start)
				call.EndByte = conv.Uint32(p.baseOffset + t.Offset + 1)
				call.HasSpan = true

				return
			}
		default:
		}
	}
}

// Tag expressions retain the expression parser's existing receiver semantics.
// A unique lexical occurrence can restore the span after a top-level parse has
// consumed its argument list. Repeated ambiguous occurrences remain unknown.
func expressionCallOrigin(call *CallSite, text string, offset, line int, inferLine ...bool) {
	correctLine := len(inferLine) > 0 && inferLine[0]
	expressionCallOriginDepth(call, text, offset, line, correctLine, 0)
}

func expressionCallOriginDepth(call *CallSite, text string, offset, line int, correctLine bool, depth int) int {
	if depth >= maxArgNesting {
		return 0
	}

	sc := NewScanner(text)
	sc.interpStrings = true
	previous := Token{}
	found := false
	originalLine := call.Line

	for {
		t := sc.NextSkipComments()
		if t.Kind == TokEOF {
			break
		}

		if t.Kind == TokString && strings.Contains(t.Value, "#") {
			spanLines := lineCounter{text: t.Value}

			for _, span := range interpolatedSpans(t.Value) {
				candidate := *call
				candidate.HasSpan = false

				count := expressionCallOriginDepth(&candidate, t.Value[span.start:span.end], offset+t.Offset+span.start, line+t.Line+spanLines.before(span.start), correctLine, depth+1)
				if count > 1 || count == 1 && found {
					call.HasSpan = false
					call.Line = originalLine

					return 2
				}

				if count == 1 {
					found = true
					call.StartByte, call.EndByte, call.HasSpan, call.Line = candidate.StartByte, candidate.EndByte, candidate.HasSpan, candidate.Line
				}
			}
		}

		declaration := identEq(previous.Value, "function")

		previous = t
		if t.Kind != TokIdent || !identEq(t.Value, call.FuncName) || !correctLine && line+t.Line != int(call.Line) || declaration || sc.PeekSkipComments().Kind != TokLParen {
			continue
		}

		if found {
			call.HasSpan = false
			call.Line = originalLine

			return 2
		}

		found = true
		clone := *sc
		parens := 0

		for {
			next := clone.NextSkipComments()
			if next.Kind == TokEOF {
				return 0
			}

			if next.Kind == TokLParen {
				parens++
			}

			if next.Kind == TokRParen {
				parens--
				if parens == 0 {
					call.StartByte = conv.Uint32(offset + t.Offset)
					call.EndByte = conv.Uint32(offset + next.Offset + 1)

					call.HasSpan = true
					if correctLine {
						call.Line = conv.Uint32(line + t.Line)
					}

					break
				}
			}
		}
	}

	if found {
		return 1
	}

	return 0
}

// A member-set subparse wraps real text in a synthetic function. Translate only
// calls wholly within that text; the wrapper itself has no source identity.
//
// lines counts text's newlines for every call of one expression, so a <cfset>
// holding thousands of calls does not count from the start for each.
func expressionWrappedCallOrigin(call *CallSite, text string, offset, line int, wrapped, correctLine bool, lines *lineCounter) {
	if wrapped && call.HasSpan {
		prefix := len("function " + memberSetWrapper + "(){")
		if int(call.StartByte) >= prefix && int(call.EndByte) <= prefix+len(text) {
			call.StartByte = conv.Uint32(offset + int(call.StartByte) - prefix)
			call.EndByte = conv.Uint32(offset + int(call.EndByte) - prefix)
		} else {
			call.HasSpan = false
		}
	}

	if !call.HasSpan {
		expressionCallOrigin(call, text, offset, line, correctLine)
	}

	if correctLine && call.HasSpan {
		start := int(call.StartByte) - offset
		if start >= 0 && start <= len(text) {
			call.Line = conv.Uint32(line + lines.before(start))
		}
	}
}

// lineCounter counts the newlines in text before an offset, carrying the
// count forward between calls asked in increasing order, as a token's spans
// and an expression's calls are. Counting from the start each time was
// quadratic in a string or a <cfset> holding thousands of them. An offset
// before the last one asked starts the count again.
type lineCounter struct {
	text      string
	at, lines int
}

func (l *lineCounter) before(i int) int {
	if i < l.at {
		l.at, l.lines = 0, 0
	}

	l.lines += strings.Count(l.text[l.at:i], "\n")
	l.at = i

	return l.lines
}
