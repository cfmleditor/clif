package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// Assignment recovery also matches named arguments and struct keys. Those
// names belong to their expression, not the caller's variable. This guard is
// bounded to separate statement lines; it is not branch dominance analysis.
func nativeReassignedAt(content, name string, start, header int) bool {
	lines := strings.Split(content, "\n")
	for header > start {
		_, at, assigned := localAssignmentAt(content, name, start, header)
		if !assigned {
			return false
		}

		tokens := producerTokens(strings.Join(lines[:at], "\n"))
		if tokens == nil {
			return false
		}

		parens, brackets := 0, 0

		for _, tok := range tokens {
			switch tok.Kind {
			case parser.TokLParen:
				parens++
			case parser.TokRParen:
				parens--
			case parser.TokLBracket:
				brackets++
			case parser.TokRBracket:
				brackets--
			default:
			}
		}

		if parens == 0 && brackets == 0 && !nativeAssignmentExpressionKey(tokens) {
			return true
		}

		header = at
	}

	return false
}

func nativeAssignmentExpressionKey(tokens []parser.Token) bool {
	if len(tokens) == 0 {
		return false
	}

	switch tokens[len(tokens)-1].Kind {
	case parser.TokComma, parser.TokLParen, parser.TokLBracket, parser.TokEquals:
		return true
	case parser.TokLBrace:
		if len(tokens) > 1 {
			switch tokens[len(tokens)-2].Kind {
			case parser.TokEquals, parser.TokComma, parser.TokLParen, parser.TokLBracket:
				return true
			default:
			}
		}
	default:
	}

	return false
}
