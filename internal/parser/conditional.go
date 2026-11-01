package parser

import "strings"

// ConditionalAt finds the top-level conditional in tokens: q is its `?` and
// colon the `:` that closes it, which is q+1 for `a ?: b`. Either is -1 when
// tokens hold no complete conditional at depth 0. It is the one reading of
// `?`, `:` and `?:` the parser's and the resolver's rules share.
func ConditionalAt(tokens []Token) (q, colon int) {
	q = -1
	depth := 0

	for i, t := range tokens {
		switch t.Kind {
		case TokLParen, TokLBracket, TokLBrace:
			depth++
		case TokRParen, TokRBracket, TokRBrace:
			depth--
		case TokQuestion:
			if depth == 0 && q < 0 {
				q = i
			}
		default:
		}
	}

	if q < 0 || q+1 >= len(tokens) {
		return q, -1
	}

	if tokens[q+1].Kind == TokColon {
		return q, q + 1
	}

	return q, ternaryColon(tokens, q)
}

// ternaryColon is the index of the `:` closing the `?` at q, stepping over
// brackets and over the `? :` of a conditional nested in the first branch,
// or -1.
func ternaryColon(tokens []Token, q int) int {
	depth, pending := 0, 0

	for i := q + 1; i < len(tokens); i++ {
		switch tokens[i].Kind {
		case TokLParen, TokLBracket, TokLBrace:
			depth++
		case TokRParen, TokRBracket, TokRBrace:
			depth--
		case TokQuestion:
			if depth == 0 {
				pending++
			}
		case TokColon:
			if depth != 0 {
				continue
			}

			if pending == 0 {
				return i
			}

			pending--
		default:
		}
	}

	return -1
}

// conditionalConstructions is what a returned `cond ? a : b` holds when each
// branch is a construction (`new X( … )` or `createObject( "component", "X" )`):
// their union, as the flow join writes it. "" for anything else, a branch of
// any other kind included, since it may hold anything. It reads expr on
// scanners of its own and records nothing, so either parser may ask it;
// resolve types a non-component createObject and applyImport, when not nil,
// qualifies a bare name.
func conditionalConstructions(expr string, resolve, applyImport func(string) string) string {
	if strings.IndexByte(expr, '?') < 0 {
		return ""
	}

	var tokens []Token

	for sc := NewScanner(expr); ; {
		t := sc.NextSkipComments()
		if t.Kind == TokEOF {
			break
		}

		tokens = append(tokens, t)
	}

	// `( c ? a : b )` is the conditional the parentheses hold.
	for len(tokens) > 2 && tokens[0].Kind == TokLParen && closingToken(tokens, 0) == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}

	q, colon := ConditionalAt(tokens)
	if q <= 0 || colon <= q+1 {
		return ""
	}

	a := constructionIn(expr[tokens[q].Offset+1:tokens[colon].Offset], resolve, applyImport)

	end := len(expr)
	if last := tokens[len(tokens)-1]; last.Offset+len(last.Value) < end {
		end = last.Offset + len(last.Value)
	}

	b := constructionIn(expr[tokens[colon].Offset+1:end], resolve, applyImport)
	if a == "" || b == "" {
		return ""
	}

	return unionComponents(a, b)
}

// closingToken is the index of the bracket closing the one at tokens[i], or -1.
func closingToken(tokens []Token, i int) int {
	depth := 0

	for j := i; j < len(tokens); j++ {
		switch tokens[j].Kind {
		case TokLParen, TokLBracket, TokLBrace:
			depth++
		case TokRParen, TokRBracket, TokRBrace:
			depth--
			if depth == 0 {
				return j
			}
		default:
		}
	}

	return -1
}

// constructionIn is the component text builds when it is exactly one
// construction with literal arguments naming it, or "".
func constructionIn(text string, resolve, applyImport func(string) string) string {
	comp, sc, _ := constructionHead(text, resolve, applyImport)
	if comp == "" || strings.HasPrefix(comp, "$") || sc.NextSkipComments().Kind != TokEOF {
		return ""
	}

	return comp
}

// constructionHead is the component the construction text starts with
// builds, and the scanner past it: `new X( … )`, or createObject, whose
// computed path is "$any" as readCreateObjectPath reads it. "" when text does
// not start with one. What follows (a chain, an operator) is the caller's.
// named is false when the component came from the resolvers (a Java class),
// whose chained calls return other things.
func constructionHead(text string, resolve, applyImport func(string) string) (comp string, sc *Scanner, named bool) {
	sc = NewScanner(strings.TrimSpace(text))

	if applyImport == nil {
		applyImport = func(s string) string { return s }
	}

	switch head := sc.NextSkipComments(); {
	case identEq(head.Value, "new"):
		peek := sc.PeekSkipComments()

		comp = newPathIn(sc, resolve, applyImport)
		if comp == "" || sc.NextSkipComments().Kind != TokLParen || skipToClose(sc, TokLParen, TokRParen).Kind != TokRParen {
			return "", sc, false
		}

		return comp, sc, !identEq(peek.Value, "java")
	case identEq(head.Value, "createObject"):
		if sc.NextSkipComments().Kind != TokLParen {
			return "", sc, false
		}

		kind, comma := sc.NextSkipComments(), sc.NextSkipComments()
		if kind.Kind != TokString || comma.Kind != TokComma {
			return "", sc, false
		}

		component := identEq(unquote(kind.Value), "component")

		path := sc.NextSkipComments()
		if path.Kind != TokString || sc.PeekSkipComments().Kind != TokRParen {
			// A path computed at run time names whichever component the
			// program picks.
			if component && path.Kind != TokEOF && path.Kind != TokRParen {
				return "$any", sc, true
			}

			return "", sc, false
		}

		sc.NextSkipComments()

		if component {
			return unquote(path.Value), sc, true
		}

		return resolve("createObject(\"" + unquote(kind.Value) + "\",\"" + unquote(path.Value) + "\")"), sc, false
	default:
		return "", sc, false
	}
}

// newPathIn reads the path after `new`, as readNewComponentPath does, without
// recording anything: a string, a dotted name, or one behind Lucee's `java:`
// or `cfml:` prefix. An inline `new component { … }` names nothing.
func newPathIn(sc *Scanner, resolve, applyImport func(string) string) string {
	tok := sc.NextSkipComments()
	if tok.Kind == TokString {
		return unquote(tok.Value)
	}

	if tok.Kind != TokIdent || identEq(tok.Value, "component") && sc.PeekSkipComments().Kind != TokLParen {
		return ""
	}

	prefix := ""

	if (identEq(tok.Value, "java") || identEq(tok.Value, "cfml")) && sc.PeekSkipComments().Kind == TokColon {
		sc.NextSkipComments()

		prefix, tok = tok.Value, sc.NextSkipComments()
		if tok.Kind != TokIdent {
			return ""
		}
	}

	var path strings.Builder

	path.WriteString(tok.Value)

	for sc.PeekSkipComments().Kind == TokDot {
		sc.NextSkipComments()

		next := sc.NextSkipComments()
		if next.Kind != TokIdent {
			return ""
		}

		path.WriteString(".")
		path.WriteString(next.Value)
	}

	if identEq(prefix, "java") {
		return resolve("createObject(\"java\",\"" + path.String() + "\")")
	}

	return applyImport(path.String())
}
