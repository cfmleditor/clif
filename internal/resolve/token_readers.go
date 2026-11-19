package resolve

import (
	"slices"

	"github.com/cfmleditor/clif/internal/parser"
)

// withoutNewlines shares tokens when no filtering is needed, and never writes
// to the input: the slice may belong to one of the shared token caches.
func withoutNewlines(tokens []parser.Token) []parser.Token {
	if !slices.ContainsFunc(tokens, func(t parser.Token) bool { return t.Kind == parser.TokNewline }) {
		return tokens
	}

	out := make([]parser.Token, 0, len(tokens))

	for _, t := range tokens {
		if t.Kind != parser.TokNewline {
			out = append(out, t)
		}
	}

	return out
}

// callArgumentScanner reads through the first opening parenthesis. Keeping this
// streaming lets a zero-argument check avoid allocating an argument token list.
func callArgumentScanner(expression string) *parser.Scanner {
	scanner := parser.NewSourceScanner(expression)

	for {
		switch scanner.NextSkipComments().Kind {
		case parser.TokEOF:
			return nil
		case parser.TokLParen:
			return scanner
		default:
		}
	}
}
