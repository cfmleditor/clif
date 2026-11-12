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
//
// A comment of dashes alone, `<!------------>`, closes at its '>' as the
// grammar closes it: it counts the dashes after "<!--", and three of them
// before a '>' close the comment, so five or more in all. Reading every dash
// as the opener's left such a comment open to the end of the file, and the
// code after it unread (16 corpus files). A nested one counts three dashes to
// open and three more to close, so one of six or more opens and closes in
// place.
func CFMLCommentEnd(s string, i int) (int, bool) {
	j := min(i+2, len(s)) // past "<!"
	for j < len(s) && s[j] == '-' {
		j++
	}

	if j-(i+2) >= 5 && j < len(s) && s[j] == '>' {
		return j + 1, true
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

			if dashes >= 6 && j < len(s) && s[j] == '>' {
				j++

				continue
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
