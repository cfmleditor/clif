package parser

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// SignatureCall describes the call and argument at a signature-help cursor.
type SignatureCall struct {
	Name, Qualifier string
	ArgumentName    string
	ArgumentIndex   int
}

// SignatureCallAt scans through the cursor so strings, comments and nested
// literals cannot contribute argument separators. The cursor column is in
// UTF-16. A current parse bounds the scan to the enclosing function.
func (pr *ParseResult) SignatureCallAt(content string, line int, char uint32) SignatureCall {
	var cursor int

	start, limit := 0, len(content)
	if pr != nil && pr.Content == content {
		cursor = pr.offsetAt(line, int(char))
		if scope := pr.FuncScopeAt(line); scope.Start >= 0 {
			start, limit = pr.lineStartsTable().Span(scope.Start, scope.End, len(content))
		}
	} else {
		cursor = PositionToOffset(content, line, int(char))
	}

	if cursor <= start {
		return SignatureCall{}
	}

	if pr != nil && pr.regionsOf == content {
		for i := len(pr.Regions) - 1; i >= 0; i-- {
			region := &pr.Regions[i]
			end := region.Offset + len(region.Text)

			if region.Offset > cursor || end < cursor {
				continue
			}

			lo, hi := max(start, region.Offset), min(max(cursor, limit), end)
			text := region.Text[lo-region.Offset : hi-region.Offset]

			switch region.Kind {
			case RegionScript:
				return scanSignatureCall(text, cursor-lo, NewScriptScanner(text))
			case RegionTag:
				return scanTagSignatureCall(text, cursor-lo)
			case RegionSkip:
				return SignatureCall{}
			}
		}
	}

	text := content[start:max(cursor, limit)]

	return scanSignatureCall(text, cursor-start, NewSourceScanner(text))
}

// Prefer the enclosing CF tag to the surrounding markup. Quotes in HTML
// prose have no bearing on the expression the tag contains.
func scanTagSignatureCall(text string, cursor int) SignatureCall {
	at := strings.LastIndexByte(text[:cursor], '<')
	if at >= 0 && lex.HasPrefixFold(text[at:], "<cf") {
		tag := text[at:]
		if end := tagEndIndex(tag); end < 0 || cursor <= at+end {
			if end >= 0 {
				tag = tag[:end]
			}

			return scanSignatureCall(tag, cursor-at, NewScriptScanner(tag))
		}
	}

	return scanSignatureCall(text, cursor, NewScanner(text))
}

type argumentFrame struct {
	close    TokenKind
	open     int
	argStart int
	argIndex int
}

func scanSignatureCall(text string, cursor int, scanner *Scanner) SignatureCall {
	var local [16]argumentFrame

	stack := local[:0]

	for tok := scanner.NextSkipComments(); tok.Kind != TokEOF && tok.Offset < cursor; tok = scanner.NextSkipComments() {
		switch tok.Kind {
		case TokLParen:
			stack = append(stack, argumentFrame{close: TokRParen, open: tok.Offset, argStart: tok.Offset + 1})
		case TokLBrace:
			stack = append(stack, argumentFrame{close: TokRBrace})
		case TokLBracket:
			stack = append(stack, argumentFrame{close: TokRBracket})
		case TokRParen, TokRBrace, TokRBracket:
			if n := len(stack); n > 0 && stack[n-1].close == tok.Kind {
				stack = stack[:n-1]
			}
		case TokString:
			if cursor < tok.Offset+len(tok.Value) {
				for _, span := range interpolatedSpans(tok.Value[1:]) {
					from, to := tok.Offset+1+span.start, tok.Offset+1+span.end
					if from <= cursor && cursor <= to {
						expr := text[from:to]

						return scanSignatureCall(expr, cursor-from, NewScriptScanner(expr))
					}
				}
			}
		case TokComma:
			if n := len(stack); n > 0 && stack[n-1].close == TokRParen {
				stack[n-1].argStart = tok.Offset + 1
				stack[n-1].argIndex++
			}
		default:
		}
	}

	for _, frame := range slices.Backward(stack) {
		if frame.close != TokRParen {
			continue
		}

		before := strings.TrimRight(text[:frame.open], " \t\r\n")

		name, qualifier := calleeBefore(before, len(before))
		if name != "" {
			return SignatureCall{
				Name: name, Qualifier: qualifier,
				ArgumentName:  signatureArgumentName(text[frame.argStart:cursor]),
				ArgumentIndex: frame.argIndex,
			}
		}
	}

	return SignatureCall{}
}

func signatureArgumentName(text string) string {
	scanner := NewScriptScanner(text)
	name := scanner.NextSkipComments()
	equals := scanner.NextSkipComments()

	if name.Kind != TokIdent || equals.Kind != TokEquals {
		return ""
	}

	// A second equals sign or an arrow belongs to the value expression.
	if following := equals.Offset + 1; following < len(text) && (text[following] == '=' || text[following] == '>') {
		return ""
	}

	return name.Value
}
