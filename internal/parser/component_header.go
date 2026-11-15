package parser

// ComponentDeclarationEnd is the byte offset where a component's declaration
// ends: the `{` that opens a script component's body, or the `>` of a
// <cfcomponent> tag. ok is false for text that declares none. The declaration
// is read by the script scanner, so a brace or `>` in a quoted value, and
// anything in a comment before it, is no end of it.
func ComponentDeclarationEnd(content string) (offset int, ok bool) {
	sc := NewScriptScanner(content)
	inDeclaration := false

	for {
		t := sc.NextSkipComments()

		switch {
		case t.Kind == TokEOF:
			return 0, false
		case t.Kind == TokLBrace && inDeclaration, t.Kind == TokGT && inDeclaration:
			return t.Offset, true
		case t.Kind == TokLBrace:
			// A brace before any declaration: not a component.
			return 0, false
		case t.Kind == TokIdent && !inDeclaration:
			inDeclaration = identEq(t.Value, "component") || identEq(t.Value, "cfcomponent")
		}
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

		for _, tag := range CFTags(content, "cfcomponent") {
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
