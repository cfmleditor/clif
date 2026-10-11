package resolve

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

func internalRequestLocalInitializer(declaration, name, function string, prefix []parser.Token) bool {
	if strings.HasPrefix(declaration, "var ") || strings.HasPrefix(declaration, "local.") {
		return true
	}
	// Explicit nonlocal scopes must never borrow a bare local contract.
	if !strings.HasPrefix(declaration, name) {
		return false
	}

	rest := strings.TrimSpace(strings.TrimPrefix(declaration, name))
	if !strings.HasPrefix(rest, "=") {
		return false
	}

	return resultModernFunction(prefix, function) || resultDominatingLocal(prefix, name)
}

// Read a literal function-level setting. Parent functions, application/admin
// defaults, callbacks, dynamic expressions and duplicate attributes supply no
// proof here. The prefix ends before the selected initializer.
func resultModernFunction(tokens []parser.Token, name string) bool {
	for i := 0; i+2 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokIdent || !strings.EqualFold(tokens[i].Value, "function") || !strings.EqualFold(tokens[i+1].Value, name) || tokens[i+2].Kind != parser.TokLParen {
			continue
		}

		params := producerGroupEnd(tokens, i+2, parser.TokLParen, parser.TokRParen)
		if params < 0 {
			return false
		}

		found, modern := 0, false

		for at := params + 1; at < len(tokens) && tokens[at].Kind != parser.TokLBrace; {
			if at+2 >= len(tokens) || tokens[at].Kind != parser.TokIdent || tokens[at+1].Kind != parser.TokEquals {
				return false
			}

			value := tokens[at+2]
			if value.Kind != parser.TokIdent && value.Kind != parser.TokString && value.Kind != parser.TokNumber {
				return false
			}

			if strings.EqualFold(tokens[at].Value, "localmode") {
				found++
				text := strings.ToLower(strings.Trim(value.Value, "\"'"))
				modern = text == "true" || (value.Kind == parser.TokString && text == "modern")
			}

			at += 3
			if at >= len(tokens) {
				return false
			}

			if tokens[at].Kind != parser.TokLBrace && (at+1 >= len(tokens) || tokens[at].Kind != parser.TokIdent || tokens[at+1].Kind != parser.TokEquals) {
				return false
			}
		}

		return found == 1 && modern
	}

	return false
}

// A previous explicit local initializer in the same lexical block establishes
// the destination of a later bare assignment even in classic mode. Declarations
// in closed/sibling branches and callbacks cannot supply that destination.
func resultDominatingLocal(tokens []parser.Token, name string) bool {
	var (
		stack  []int
		owners [][]int
	)

	frame := 0

	for i, t := range tokens {
		switch t.Kind {
		case parser.TokLBrace:
			frame++
			stack = append(stack, frame)
		case parser.TokRBrace:
			if len(stack) == 0 {
				return false
			}

			stack = stack[:len(stack)-1]
		case parser.TokIdent:
			// A whole-scope escape or operation can remove the established key.
			// Only the explicit local initializer itself is admitted here.
			if strings.EqualFold(t.Value, "local") && (i+3 >= len(tokens) || tokens[i+1].Kind != parser.TokDot || !strings.EqualFold(tokens[i+2].Value, name) || tokens[i+3].Kind != parser.TokEquals) {
				return false
			}

			if !queryOutputStatement(tokens, i) {
				continue
			}

			var at int

			switch {
			case strings.EqualFold(t.Value, "var") && i+2 < len(tokens) && strings.EqualFold(tokens[i+1].Value, name):
				at = i + 2
			case strings.EqualFold(t.Value, "local") && i+3 < len(tokens) && tokens[i+1].Kind == parser.TokDot && strings.EqualFold(tokens[i+2].Value, name):
				at = i + 3
			default:
				continue
			}

			if tokens[at].Kind == parser.TokEquals {
				owners = append(owners, slices.Clone(stack))
			}
		default:
		}
	}

	for _, owner := range owners {
		if slices.Equal(owner, stack) {
			return true
		}
	}

	return false
}
