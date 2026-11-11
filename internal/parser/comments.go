package parser

import "strings"

// The <!--- ---> comment rule, the grammar's (skip_cfml_comment_body in
// tree-sitter-cfml's scanner.h): an opener is "<!" and three or more dashes, a
// close is three or more dashes and ">", and comments nest, so a commented-out
// block holding a comment of its own stays one comment. The parser, its
// scanner and the resolver's tag walks each had a reader of their own: two
// closed a comment at its first "--->", so a tag after a nested comment's
// close was read as live, and the scanner closed one at "-->".

// SkipCFMLComment returns the offset just past the <!--- ---> comment opening
// at i, or len(s) when it is never closed.
func SkipCFMLComment(s string, i int) int {
	end, _ := CFMLCommentEnd(s, i)

	return end
}

// CFMLCommentEnd is SkipCFMLComment, and whether the comment closes.
func CFMLCommentEnd(s string, i int) (int, bool) {
	j := min(i+2, len(s)) // past "<!"
	for j < len(s) && s[j] == '-' {
		j++
	}

	for depth := 1; j < len(s); {
		k := strings.IndexAny(s[j:], "<-")
		if k < 0 {
			break
		}

		j += k

		if s[j] == '<' {
			j++
			if j >= len(s) || s[j] != '!' {
				continue
			}

			j++

			dashes := 0
			for j < len(s) && s[j] == '-' {
				dashes++
				j++
			}

			if dashes >= 3 {
				depth++
			}

			continue
		}

		dashes := 0
		for j < len(s) && s[j] == '-' {
			dashes++
			j++
		}

		if dashes >= 3 && j < len(s) && s[j] == '>' {
			j++

			depth--
			if depth == 0 {
				return j, true
			}
		}
	}

	return len(s), false
}
