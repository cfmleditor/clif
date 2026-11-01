package parser

import "strings"

// WithoutTrailingComment is s up to the end of its last token, read as
// CFScript: a trailing `//` or `/* */` comment is left off. Strings are read
// as the parser reads them, a doubled quote, a `#...#` holding strings of its
// own and a lone `#` included, so a `//` in one is not a comment.
func WithoutTrailingComment(s string) string {
	if !strings.Contains(s, "//") && !strings.Contains(s, "/*") {
		return s
	}

	sc := NewScanner(s)
	sc.interpStrings = true

	end := 0

	for {
		t := sc.NextSkipComments()
		if t.Kind == TokEOF {
			return s[:end]
		}

		end = t.Offset + len(t.Value)
	}
}
