package parser

// ComponentDeclarationEnd is the byte offset where a component's declaration
// ends: the `{` that opens a script component's body, or the `>` of a
// <cfcomponent> tag. ok is false for text that declares none. The declaration
// is read by the script scanner, so a brace or `>` in a quoted value, and
// anything in a comment before it, is no end of it.
func ComponentDeclarationEnd(content string) (offset int, ok bool) {
	offset, _, ok = componentDeclaration(content)

	return offset, ok
}

// ComponentDeclarationBounds is ComponentDeclarationEnd with the byte offset
// just past the last word of the declaration: before the blanks and any comment
// that lead to the `{` on a line of its own, and just past the `>` of a tag
// component. It is where a mark of the declaration belongs.
func ComponentDeclarationBounds(content string) (open, textEnd int, ok bool) {
	return componentDeclaration(content)
}

// componentDeclaration is the offset of the `{` or `>` that ends a component's
// declaration, and the offset just past the last token before it.
func componentDeclaration(content string) (open, textEnd int, ok bool) {
	sc := NewScriptScanner(content)
	inDeclaration := false
	depth := 0 // braces open before the declaration: `<cfset x = {}>`, a struct

	var prev Token

	prevEnd := 0

	for budget := declarationTokens; budget > 0; budget-- {
		t := sc.NextSkipComments()

		switch {
		case t.Kind == TokEOF:
			return 0, 0, false
		case t.Kind == TokLBrace && inDeclaration:
			return t.Offset, prevEnd, true
		case t.Kind == TokGT && inDeclaration:
			return t.Offset, t.Offset + 1, true
		case t.Kind == TokLBrace:
			depth++
		case t.Kind == TokRBrace:
			depth = max(depth-1, 0)
		case t.Kind == TokIdent && !inDeclaration && depth == 0:
			inDeclaration = opensDeclaration(t, prev)
		}

		prev, prevEnd = t, sc.Pos()
	}

	return 0, 0, false
}

// declarationTokens is how many tokens ComponentDeclarationEnd reads before it
// decides a text declares no component: a header is a few, however long a
// struct literal before it, and a text that declares none is not read to its end
// on each request.
const declarationTokens = 2000

// opensDeclaration reports whether the word t, which follows prev, names the
// component or interface the text declares: the tag `<cfcomponent`, or the
// script word where it is not a member, a value or an argument.
func opensDeclaration(t, prev Token) bool {
	switch {
	case identEq(t.Value, "cfcomponent"), identEq(t.Value, "cfinterface"):
		return prev.Kind == TokLT
	case identEq(t.Value, "component"), identEq(t.Value, "interface"):
		switch prev.Kind {
		case TokDot, TokEquals, TokLParen, TokComma, TokColon, TokLT:
			return false
		default:
			return true
		}
	default:
		return false
	}
}

// ComponentBodyEnd is the byte offset where a component's body ends: the `}`
// that closes a script component, or the last `</cfcomponent>` of a tag one,
// and whether the component is written as tags. ok is false when the text
// declares no component or never closes it.
func ComponentBodyEnd(content string) (offset int, tags, ok bool) {
	open, ok := ComponentDeclarationEnd(content)
	if !ok {
		return 0, false, false
	}

	if content[open] == '>' {
		// The last closing tag that is one: not a word in a comment.
		at := -1

		for _, tag := range CFTags(content, "cfcomponent", "cfinterface") {
			if tag.Close {
				at = tag.Start
			}
		}

		if at < open {
			return 0, true, false
		}

		return at, true, true
	}

	sc := NewScriptScanner(content[open:])
	depth := 0

	for {
		switch t := sc.NextSkipComments(); t.Kind {
		case TokEOF:
			return 0, false, false
		case TokLBrace:
			depth++
		case TokRBrace:
			depth--

			if depth == 0 {
				return open + t.Offset, false, true
			}
		default:
		}
	}
}
