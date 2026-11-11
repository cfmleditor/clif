package parser

// InterpolatedExpressions returns executable hash expressions in source using
// the parser's quote/nesting and escaped-hash rules. Callers analyzing a quoted
// token must inspect these expressions as well as the surrounding literal.
func InterpolatedExpressions(source string) []string {
	spans := interpolatedSpans(source)
	if len(spans) == 0 {
		return nil
	}

	expressions := make([]string, 0, len(spans))
	for _, span := range spans {
		expressions = append(expressions, source[span.start:span.end])
	}

	return expressions
}

// InterpolatedSpans is where each expression InterpolatedExpressions returns
// lies in source: [start, end), the hashes around it excluded.
func InterpolatedSpans(source string) [][2]int {
	spans := interpolatedSpans(source)
	if len(spans) == 0 {
		return nil
	}

	out := make([][2]int, 0, len(spans))
	for _, span := range spans {
		out = append(out, [2]int{span.start, span.end})
	}

	return out
}
