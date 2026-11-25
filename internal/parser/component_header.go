package parser

// ComponentDeclarationBounds is where a component's declaration ends: open is
// the byte offset of the `{` that opens a script component's body, or the `>` of
// a <cfcomponent> tag, and textEnd the offset just past the last word of the
// declaration, before the blanks and any comment that lead to a `{` on a line of
// its own, and just past a tag's `>`: where a mark of the declaration belongs. ok
// is false for text that declares none. The declaration is read by the script
// scanner, so a brace or `>` in a quoted value, and anything in a comment before
// it, is no end of it.
func ComponentDeclarationBounds(content string) (open, textEnd int, ok bool) {
	_, open, textEnd, ok = componentDeclaration(content)

	return open, textEnd, ok
}

// ComponentDeclarationSpan is where a component's declaration starts and ends:
// start is the offset of its keyword (`component`, `interface`, or the name in
// `<cfcomponent`) and open the offset of the `{` or `>` that ends it, as
// ComponentDeclarationBounds finds it. Unlike ComponentOutline it does not
// look for the end of the body, so it reads a bounded number of tokens
// whatever the size of the file; it is what a request made on every cursor
// move should ask. ok is false for text that declares none.
func ComponentDeclarationSpan(content string) (start, open int, ok bool) {
	start, open, _, ok = componentDeclaration(content)

	return start, open, ok
}

// componentDeclaration is the offset of the keyword that opens a component's
// declaration, the offset of the `{` or `>` that ends it, and the offset just
// past the last token before that.
func componentDeclaration(content string) (start, open, textEnd int, ok bool) {
	sc := NewScriptScanner(content)
	inDeclaration := false
	depth := 0 // braces open before the declaration: `<cfset x = {}>`, a struct

	var prev Token

	prevEnd := 0
	inTag := false // between `<name` and its `>`: a `component=` there is an attribute

	for budget := declarationTokens; budget > 0; budget-- {
		t := sc.NextSkipComments()

		switch {
		case t.Kind == TokEOF:
			return 0, 0, 0, false
		case t.Kind == TokLBrace && inDeclaration:
			return start, t.Offset, prevEnd, true
		case t.Kind == TokGT && !inDeclaration:
			inTag = false
		case t.Kind == TokGT && inDeclaration:
			return start, t.Offset, t.Offset + 1, true
		case t.Kind == TokLBrace:
			depth++
		case t.Kind == TokRBrace:
			depth = max(depth-1, 0)
		case t.Kind == TokIdent && !inDeclaration && depth == 0:
			if inTag {
				break
			}

			inDeclaration = opensDeclaration(t, prev)
			if inDeclaration {
				start = t.Offset
			}

			inTag = !inDeclaration && prev.Kind == TokLT
		}

		prev, prevEnd = t, sc.Pos()
	}

	return 0, 0, 0, false
}

// declarationTokens is how many tokens ComponentDeclarationBounds reads before it
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

// ComponentHeader locates a component or interface declaration and its body
// for document-symbol outlines.
type ComponentHeader struct {
	IsInterface bool   // true for <cfinterface>, interface {}, or component interface {}
	InitMethod  string // explicit initmethod attribute, or ""
	Start       int    // byte offset of the opening token
	NameStart   int    // byte offset of the declaration keyword
	NameEnd     int    // byte offset just past that keyword
	End         int    // byte offset just past the closing `}` or closing tag, or len(content)
}

// ComponentOutline finds a component or interface declaration and its closing
// delimiter in content. ok is false for templates that declare neither.
func ComponentOutline(content string) (ComponentHeader, bool) {
	sc := NewScriptScanner(content)
	inDeclaration := false
	isTag := false
	depth := 0
	inTag := false

	var (
		prev   Token
		header ComponentHeader
	)

	for budget := declarationTokens; budget > 0; budget-- {
		t := sc.NextSkipComments()

		switch {
		case t.Kind == TokEOF:
			return ComponentHeader{}, false
		case t.Kind == TokLBrace && inDeclaration && !isTag:
			header.End = scriptComponentEnd(content, t.Offset)

			return header, true
		case t.Kind == TokGT && !inDeclaration:
			inTag = false
		case t.Kind == TokGT && inDeclaration && isTag:
			tag := content[header.Start : t.Offset+1]
			header.InitMethod = getAttr(tag, "initmethod")
			header.End = tagComponentEnd(content, t.Offset)

			return header, true
		case t.Kind == TokLBrace:
			depth++
		case t.Kind == TokRBrace:
			depth = max(depth-1, 0)
		case inDeclaration && !isTag:
			if t.Kind == TokIdent && prev.Kind != TokEquals && prev.Kind != TokColon {
				if identEq(t.Value, "interface") {
					header.IsInterface = true
				} else if identEq(t.Value, "initmethod") {
					if next := sc.PeekSkipComments(); next.Kind == TokEquals || next.Kind == TokColon {
						sc.NextSkipComments()

						if val := sc.PeekSkipComments(); val.Kind == TokString || val.Kind == TokIdent {
							sc.NextSkipComments()

							header.InitMethod = unquote(val.Value)
							prev = val

							continue
						}
					}
				}
			}
		case t.Kind == TokIdent && !inDeclaration && depth == 0:
			if inTag {
				break
			}

			if opensDeclaration(t, prev) {
				inDeclaration = true
				isTag = prev.Kind == TokLT
				header = ComponentHeader{
					IsInterface: identEq(t.Value, "interface") || identEq(t.Value, "cfinterface"),
					Start:       t.Offset,
					NameStart:   t.Offset,
					NameEnd:     t.Offset + len(t.Value),
				}

				if isTag {
					header.Start = prev.Offset
				}
			} else if prev.Kind == TokLT {
				inTag = true
			}
		}

		prev = t
	}

	return ComponentHeader{}, false
}

func tagComponentEnd(content string, openGT int) int {
	at := -1

	for _, tag := range CFTags(content, "cfcomponent", "cfinterface") {
		if tag.Close && tag.Start >= openGT {
			at = tag.End
		}
	}

	if at >= 0 {
		return at
	}

	return len(content)
}

func scriptComponentEnd(content string, openBrace int) int {
	sc := NewScriptScanner(content[openBrace:])
	depth := 0

	for {
		switch t := sc.NextSkipComments(); t.Kind {
		case TokEOF:
			return len(content)
		case TokLBrace:
			depth++
		case TokRBrace:
			depth--

			if depth == 0 {
				return openBrace + t.Offset + 1
			}
		default:
		}
	}
}

// ComponentBodyEnd is the byte offset where a component's body ends: the `}`
// that closes a script component, or the last `</cfcomponent>` of a tag one,
// and whether the component is written as tags. ok is false when the text
// declares no component or never closes it.
func ComponentBodyEnd(content string) (offset int, tags, ok bool) {
	open, _, ok := ComponentDeclarationBounds(content)
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
