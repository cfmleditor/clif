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
