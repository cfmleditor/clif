package parser

import (
	"slices"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/lex"
)

// scanExpressionWritesFrom is scanExpressionWrites reading sc, a scanner over
// expression, and toks, its tokens when sc replays them.
func scanExpressionWritesFrom(sc *Scanner, expression string, toks []Token) []expressionWrite {
	writes := newExpressionWrites(toks)

	// pooled is the token list borrowed from tokenPool, if one was.
	var pooled *[]Token

	var ends *expressionEnds

	defer func() { releaseWriteScratch(pooled, ends) }()

	previous := TokEOF

	for {
		t := sc.NextSkipComments()
		before := previous

		previous = t.Kind
		if t.Kind == TokEOF {
			break
		}

		// Closure-local collection bindings need their own lexical identities.
		// Withhold collection inference rather than attributing them to the parent.
		// The marker carries the function the closure is written in, which is
		// what applyMemberBindings narrows its withholding to.
		if ((t.Kind == TokEquals || t.Kind == TokMinus) && sc.PeekSkipComments().Kind == TokGT) ||
			(t.Kind == TokIdent && identEq(t.Value, "function") && sc.PeekSkipComments().Kind == TokLParen) {
			writes = append(writes, expressionWrite{write: collectionWrite{unknown: true}, at: t.Offset})
		}

		if t.Kind != TokIdent || before == TokDot || before == TokDoubleColon {
			continue
		}

		// Every write below reads on from the name to a ".", a "[", a "(",
		// or an operator; only "var" is followed by another name. Testing the
		// token after it, which the scanner caches for the next step anyway,
		// spares most names the two scanner copies and the path read.
		if !mayStartWrite(sc.PeekSkipComments().Kind) && !identEq(t.Value, "var") {
			continue
		}

		cursor := *sc
		if collectionMutator(t.Value) && cursor.PeekSkipComments().Kind == TokLParen {
			cursor.NextSkipComments()

			root, element := collectionTarget(&cursor)
			if root != "" && !element {
				writes = append(writes, expressionWrite{write: collectionMutatorWrite(t.Value, root, &cursor, expression), at: t.Offset, placed: true})
			}
		}

		cursor = *sc
		cursor.Restore(ScannerState{pos: t.Offset, line: int(t.Line)})
		root, element := collectionTarget(&cursor)

		local := identEq(t.Value, "var")
		if local {
			root, element = collectionTarget(&cursor)
		}

		if root == "" {
			continue
		}

		// x.append( v ), x.update( k, v ): an in-place change to what x holds,
		// never a new value for x. Read as replacing x, it withheld the type
		// of every member a component calls such a method on —
		// variables.instance.DAO.update( bean ) on Masa's DAOs.
		if cursor.PeekSkipComments().Kind == TokLParen {
			if base, method, ok := strings.CutLast(root, "."); ok && collectionMemberMutator(method) {
				writes = append(writes, expressionWrite{write: memberMutatorWrite(method, base, &cursor, expression), at: t.Offset, placed: true})
			}
		}

		operator := cursor.NextSkipComments()
		if operator.Kind == TokPlus || operator.Kind == TokMinus || operator.Kind == TokStar || operator.Kind == TokSlash || operator.Kind == TokAmpersand || operator.Kind == TokPercent || operator.Kind == TokCaret {
			next := cursor.PeekSkipComments().Kind
			if next == TokEquals || next == operator.Kind {
				writes = append(writes, expressionWrite{write: collectionWrite{target: root, element: element, unknown: true}, at: t.Offset, placed: true})
			}

			continue
		}

		if operator.Kind != TokEquals || cursor.PeekSkipComments().Kind == TokEquals {
			continue
		}

		if ends == nil {
			if toks == nil {
				// The tokens are needed only to find where each right-hand
				// side ends, so the list is borrowed and given back.
				pooled, _ = tokenPool.Get().(*[]Token)
				*pooled = tokenizeInto(*pooled, expression)
				toks = *pooled
			}

			ends, _ = expressionEndsPool.Get().(*expressionEnds)
			ends.fill(expression, toks)
		}

		writes = append(writes, expressionWrite{write: collectionWrite{target: root, expression: ends.from(cursor.PeekSkipComments().Offset), element: element, local: local || lex.HasPrefixFold(root, "local.")}, at: t.Offset, placed: true})
	}

	return writes
}

// releaseWriteScratch gives back what scanExpressionWritesFrom borrowed.
func releaseWriteScratch(pooled *[]Token, ends *expressionEnds) {
	if pooled != nil {
		tokenPool.Put(pooled)
	}

	if ends != nil {
		ends.src = ""
		expressionEndsPool.Put(ends)
	}
}

// newExpressionWrites is the list scanExpressionWritesFrom fills, sized from
// toks when it has them: a script region holds hundreds of writes, nearly all
// at an "=", and growing the list by doubling was the largest allocation of a
// parse.
func newExpressionWrites(toks []Token) []expressionWrite {
	if toks == nil {
		return nil
	}

	n := 0

	for i := range toks {
		if toks[i].Kind == TokEquals {
			n++
		}
	}

	return make([]expressionWrite, 0, n)
}

// mayStartWrite reports whether a name followed by a token of kind k could
// begin a write scanExpressionWritesFrom records.
func mayStartWrite(k TokenKind) bool {
	switch k {
	case TokDot, TokLBracket, TokLParen, TokEquals,
		TokPlus, TokMinus, TokStar, TokSlash, TokAmpersand, TokPercent, TokCaret:
		return true
	default:
		return false
	}
}

// expressionEnds answers scriptReturnExpression for every token of one
// expression from a single pass. scriptReturnExpression scans from its start
// to the first ";" or "}" at the start's depth, and CFScript does not need
// the ";": in a function written without them every assignment scanned on to
// the function's closing brace, quadratic in its length (most of a reparse
// of tassweb's largest components in the editor).
type expressionEnds struct {
	src     string
	offsets []int // each token's offset
	ends    []int // where the expression starting at that token ends

	// Scratch for fill, kept so a pooled value reuses it.
	depths            []int
	terminal, newline []bool
	next, nextLine    []int
}

// expressionEndsPool holds expressionEnds for scanExpressionWritesFrom, which
// builds one per script region and keeps none of it: what it keeps is text
// from the source. Five slices sized by the region's tokens were a tenth of
// what a script component's parse allocated.
var expressionEndsPool = sync.Pool{New: func() any { return &expressionEnds{} }}

func newExpressionEnds(src string) *expressionEnds {
	return newExpressionEndsFrom(src, tokenize(src))
}

// newExpressionEndsFrom is newExpressionEnds from src's tokens.
func newExpressionEndsFrom(src string, toks []Token) *expressionEnds {
	e := &expressionEnds{}
	e.fill(src, toks)

	return e
}

// fill makes e answer for src, whose tokens are toks, reusing e's slices.
func (e *expressionEnds) fill(src string, toks []Token) {
	// Sized from the tokens, and the nearest terminator per depth kept in a
	// slice rather than a map: built for every assignment a <cfset> holds,
	// four growing slices and a map apiece were most of what collecting a
	// file's collection writes allocated.
	n := len(toks)
	offsets := slices.Grow(e.offsets[:0], n)
	depths := slices.Grow(e.depths[:0], n)
	terminal := slices.Grow(e.terminal[:0], n)
	newline := slices.Grow(e.newline[:0], n)

	depth, lo, hi := 0, 0, 0
	prev := Token{Kind: TokEOF}

	for _, t := range toks {
		if t.Kind == TokEOF {
			break
		}

		offsets = append(offsets, t.Offset)
		depths = append(depths, depth)
		terminal = append(terminal, t.Kind == TokSemicolon || t.Kind == TokRBrace)
		newline = append(newline, newlineEndsStatement(src, prev, t))
		prev = t

		switch t.Kind {
		case TokLParen, TokLBracket, TokLBrace:
			depth++
		case TokRParen, TokRBracket, TokRBrace:
			depth--
		default:
		}

		lo, hi = min(lo, depth), max(hi, depth)
	}

	// Backwards: the nearest terminator at or after each token, at its depth.
	// next is indexed by depth less the lowest depth reached, -1 for none.
	// A line break ends an expression only after its first token, as it does
	// for scriptReturnExpression, so it is kept apart and taken from past i.
	ends := slices.Grow(e.ends[:0], len(offsets))[:len(offsets)]
	next := slices.Grow(e.next[:0], hi-lo+1)[:hi-lo+1]
	nextLine := slices.Grow(e.nextLine[:0], hi-lo+1)[:hi-lo+1]

	for i := range next {
		next[i], nextLine[i] = -1, -1
	}

	for i := len(offsets) - 1; i >= 0; i-- {
		d := depths[i] - lo
		if terminal[i] {
			next[d] = offsets[i]
		}

		end := next[d]
		if l := nextLine[d]; l >= 0 && (end < 0 || l < end) {
			end = l
		}

		if end < 0 {
			end = len(src)
		}

		ends[i] = end

		if newline[i] {
			nextLine[d] = offsets[i]
		}
	}

	e.src, e.offsets, e.ends = src, offsets, ends
	e.depths, e.terminal, e.newline, e.next, e.nextLine = depths, terminal, newline, next, nextLine
}

// at is from when the caller already has the token's index in e.offsets.
func (e *expressionEnds) at(i int) string {
	if i < 0 || i >= len(e.ends) {
		return ""
	}

	return strings.TrimSpace(e.src[e.offsets[i]:e.ends[i]])
}

// from is scriptReturnExpression for a scanner whose next token starts at
// offset.
func (e *expressionEnds) from(offset int) string {
	i, found := slices.BinarySearch(e.offsets, offset)
	if !found {
		return strings.TrimSpace(e.src[min(offset, len(e.src)):])
	}

	return strings.TrimSpace(e.src[offset:e.ends[i]])
}
