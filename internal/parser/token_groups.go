package parser

// GroupEnd finds the close balancing open from start, or -1. It counts only
// the requested delimiter pair; other token kinds (including EOF) are ignored.
// Callers with an EOF sentinel or mixed-delimiter policy keep that policy local.
func GroupEnd(tokens []Token, start int, open, closing TokenKind) int {
	depth := 0

	for i := start; i < len(tokens); i++ {
		if tokens[i].Kind == open {
			depth++
		}

		if tokens[i].Kind == closing {
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}
