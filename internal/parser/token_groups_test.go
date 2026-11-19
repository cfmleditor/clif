package parser

import (
	"math/rand"
	"testing"
)

func TestGroupEndKeepsDelimiterPolicies(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kinds []TokenKind
		want  int
	}{
		{"nested", []TokenKind{TokLBrace, TokLBrace, TokRBrace, TokRBrace}, 3},
		{"other delimiters", []TokenKind{TokLBrace, TokLParen, TokRBrace}, 2},
		{"EOF is not a delimiter", []TokenKind{TokLBrace, TokEOF, TokRBrace}, 2},
		{"unclosed", []TokenKind{TokLBrace, TokIdent}, -1},
		{"unmatched closer", []TokenKind{TokRBrace, TokLBrace, TokRBrace}, -1},
		{"empty", nil, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tokens := make([]Token, len(tc.kinds))
			for i, kind := range tc.kinds {
				tokens[i].Kind = kind
			}

			if got := GroupEnd(tokens, 0, TokLBrace, TokRBrace); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}

	tokens := []Token{{Kind: TokLBracket}, {Kind: TokEOF}, {Kind: TokRBracket}}
	if got := bracketEnd(tokens, 0); got != -1 {
		t.Fatalf("bracketEnd must still stop at EOF: %d", got)
	}

	tokens = []Token{{Kind: TokLBrace}, {Kind: TokLParen}, {Kind: TokRBrace}, {Kind: TokRParen}}
	if got := closingToken(tokens, 0); got != 3 {
		t.Fatalf("closingToken must still count all group kinds: %d", got)
	}
}

// Keep the removed switch-based brace reader as a differential oracle, including
// malformed token streams and starts beyond the end of a slice.
func TestGroupEndMatchesPreviousBraceReader(t *testing.T) {
	rng := rand.New(rand.NewSource(19))
	kinds := []TokenKind{TokEOF, TokIdent, TokLBrace, TokRBrace, TokLParen, TokRParen, TokLBracket, TokRBracket}

	for range 2000 {
		tokens := make([]Token, rng.Intn(40))
		for i := range tokens {
			tokens[i].Kind = kinds[rng.Intn(len(kinds))]
		}

		start := rng.Intn(len(tokens) + 2)
		want, depth := -1, 0

		for i := start; i < len(tokens); i++ {
			switch tokens[i].Kind {
			case TokLBrace:
				depth++
			case TokRBrace:
				depth--
				if depth == 0 {
					want = i
				}
			default:
			}

			if want >= 0 {
				break
			}
		}

		if got := GroupEnd(tokens, start, TokLBrace, TokRBrace); got != want {
			t.Fatalf("start %d, tokens %v: got %d, want %d", start, tokens, got, want)
		}
	}
}
