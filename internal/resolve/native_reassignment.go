package resolve

import (
	"github.com/cfmleditor/clif/internal/parser"
)

// Assignment recovery also matches named arguments and struct keys. Those
// names belong to their expression, not the caller's variable. This guard is
// bounded to separate statement lines; it is not branch dominance analysis.
func nativeReassignedAt(content, name string, start, header int) bool {
	for header > start {
		_, at, assigned := localAssignmentAt(content, name, start, header)
		if !assigned {
			return false
		}

		shape, ok := statementShape(contentBeforeLine(content, at))
		if !ok {
			return false
		}

		if shape.parens == 0 && shape.brackets == 0 && !shape.expressionKey() {
			return true
		}

		header = at
	}

	return false
}

// textShape is what nativeReassignedAt reads of the text before an
// assignment: whether its brackets balance and how it ends.
type textShape struct {
	parens, brackets int
	last, prev       parser.TokenKind
	tokens           int
}

// statementShape scans source once, keeping only its shape: it used to build
// a token slice of everything above the assignment to read three facts from
// it, half of producerTokens' allocation. Like producerTokens it gives up past
// 4,096 tokens.
func statementShape(source string) (textShape, bool) {
	var shape textShape

	scanner := parser.NewScanner(source)

	for {
		tok := scanner.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			return shape, true
		}

		shape.tokens++
		if shape.tokens > 4096 {
			return textShape{}, false
		}

		switch tok.Kind {
		case parser.TokLParen:
			shape.parens++
		case parser.TokRParen:
			shape.parens--
		case parser.TokLBracket:
			shape.brackets++
		case parser.TokRBracket:
			shape.brackets--
		default:
		}

		shape.prev, shape.last = shape.last, tok.Kind
	}
}

// expressionKey reports whether the text ends where an assignment would be a
// named argument or a struct key rather than a statement.
func (s textShape) expressionKey() bool {
	if s.tokens == 0 {
		return false
	}

	switch s.last {
	case parser.TokComma, parser.TokLParen, parser.TokLBracket, parser.TokEquals:
		return true
	case parser.TokLBrace:
		if s.tokens > 1 {
			switch s.prev {
			case parser.TokEquals, parser.TokComma, parser.TokLParen, parser.TokLBracket:
				return true
			default:
			}
		}
	default:
	}

	return false
}
