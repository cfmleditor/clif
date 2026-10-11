package parser

import "strings"

// requestMappingBody admits only the outer statements before onRequest's
// include of its actual page parameter. Nested branch bodies contribute no
// mapping or alias facts. An unused method or a different include is no proof.
func requestMappingBody(content string) (string, bool) {
	sc := NewScanner(content)

	var tokens []Token

	for {
		t := sc.NextSkipComments()
		if t.Kind == TokEOF {
			break
		}

		tokens = append(tokens, t)
	}

	if len(tokens) < 7 {
		return "", false
	}

	i := 0
	for i < len(tokens) && !identEq(tokens[i].Value, "function") {
		i++
	}

	if i+3 >= len(tokens) || !identEq(tokens[i+1].Value, "onRequest") || tokens[i+2].Kind != TokLParen {
		return "", false
	}

	i += 3
	parameter := ""

	for i < len(tokens) && tokens[i].Kind != TokRParen {
		if tokens[i].Kind == TokComma || tokens[i].Kind == TokEquals {
			return "", false
		}

		if tokens[i].Kind == TokIdent {
			parameter = tokens[i].Value
		}

		i++
	}

	if parameter == "" || i+1 >= len(tokens) || tokens[i+1].Kind != TokLBrace {
		return "", false
	}

	start := tokens[i+1].Offset + 1
	projected := []byte(content)
	depth := 1

	for j := i + 2; j < len(tokens); j++ {
		t := tokens[j]
		if depth == 1 && identEq(t.Value, "include") && j+4 < len(tokens) && identEq(tokens[j+1].Value, "arguments") && tokens[j+2].Kind == TokDot && strings.EqualFold(tokens[j+3].Value, parameter) && tokens[j+4].Kind == TokSemicolon {
			return string(projected[start:t.Offset]), true
		}

		if t.Kind == TokLBrace {
			depth++
		}

		if depth > 1 {
			end := len(content)
			if j+1 < len(tokens) {
				end = tokens[j+1].Offset
			}

			for k := t.Offset; k < end; k++ {
				if projected[k] != '\n' && projected[k] != '\r' {
					projected[k] = ' '
				}
			}
		}

		if t.Kind == TokRBrace {
			depth--
			if depth == 0 {
				break
			}
		}
	}

	return "", false
}
