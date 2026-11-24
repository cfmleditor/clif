package parser

import (
	"slices"
	"sort"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// scriptTokens is a script region's tokens, interpolated strings included,
// tokenised once for every pass that shares writesMemo, and a large region's
// once for every parse of its text (scriptTokensOf): that list is not the
// memo's to release, nor anyone's to change.
func (pr *ParseResult) scriptTokens(r *Region) *[]Token {
	if len(r.Text) >= sharedTokenMin {
		return scriptTokensOf(r.Text)
	}

	m := pr.writesMemo
	if m == nil {
		toks := tokenize(r.Text)

		return &toks
	}

	if toks, ok := m.tokens[r.Offset]; ok {
		return toks
	}

	if m.tokens == nil {
		m.tokens = map[int]*[]Token{}
	}

	toks, _ := tokenPool.Get().(*[]Token)
	*toks = tokenizeInto(*toks, r.Text)
	m.tokens[r.Offset] = toks

	return toks
}

func (pr *ParseResult) collectionWrites() []collectionWrite {
	if m := pr.writesMemo; m != nil {
		if !m.done {
			if pr.tagWrites == nil {
				pr.tagWrites = &expressionWriteCache{}
			}

			pr.tagWrites.begin()
			m.writes, m.done = pr.collectWrites(pr.tagWrites), true
		}

		return m.writes
	}

	return pr.collectWrites(nil)
}

// collectWrites reads every collection write in the file. cache, when not
// nil, answers a <cfset> whose text an earlier pass already read.
func (pr *ParseResult) collectWrites(cache *expressionWriteCache) []collectionWrite {
	var writes []collectionWrite

	lines := pr.contentLineIdx
	if lines == nil {
		lines = NewLineTable(pr.Content)
	}

	lc := lineCursor{idx: lines}
	scopeIdx := 0

	functionAt := func(offset int) string {
		line := lc.at(offset)
		scopes := pr.Scopes

		if len(scopes) == 0 {
			return ""
		}

		if line < scopes[scopeIdx].Start {
			if scopeIdx == 0 {
				return ""
			}

			idx := sort.Search(scopeIdx, func(i int) bool { return scopes[i].Start > line }) - 1
			if idx < 0 {
				scopeIdx = 0

				return ""
			}

			scopeIdx = idx
		} else {
			for scopeIdx+1 < len(scopes) && scopes[scopeIdx+1].Start <= line {
				scopeIdx++
			}
		}

		if line <= scopes[scopeIdx].End {
			return pr.scopeKey(scopeIdx)
		}

		return ""
	}

	for i := range pr.Regions {
		r := &pr.Regions[i]
		if r.Kind == RegionScript {
			toks := pr.scriptTokens(r)
			writes = placeWrites(scanTokenWrites(*toks, r.Text), r.Offset, functionAt, writes)

			continue
		}

		if r.Kind != RegionTag {
			continue
		}

		if sites, ok := pr.keptCollectionTags(r.Offset); ok {
			writes = collectCollectionTagsAt(r.Text, sites, r.Offset, functionAt, writes, cache)

			continue
		}

		writes = collectCollectionTags(r.Text, r.Offset, functionAt, writes, cache)
	}

	return writes
}

// expressionWrite is a collection write as scanExpressionWrites reads it,
// before it is placed in the file: at is the offset in the expression of the
// token the write's function is read at, and placed says the write carries
// that offset as its own. A write read this way depends on nothing but the
// expression's text, which is what lets expressionWriteCache keep it.
type expressionWrite struct {
	write  collectionWrite
	at     int
	placed bool
}

// placeWrites places writes read from an expression starting at offset.
func placeWrites(scanned []expressionWrite, offset int, functionAt func(int) string, writes []collectionWrite) []collectionWrite {
	writes = slices.Grow(writes, len(scanned))

	for i := range scanned {
		e := &scanned[i]

		w := e.write
		w.function = functionAt(offset + e.at)

		if e.placed {
			w.offset = offset + e.at
		}

		writes = append(writes, w)
	}

	return writes
}

// scanExpressionWrites reads the collection writes in an expression.
func scanExpressionWrites(expression string) []expressionWrite {
	sc := NewScanner(expression)
	sc.interpStrings = true

	return scanExpressionWritesFrom(sc, expression, nil)
}

func tokenKindAt(toks []Token, i int) TokenKind {
	if i < len(toks) {
		return toks[i].Kind
	}

	return TokEOF
}

func isCompoundWriteOp(k TokenKind) bool {
	switch k {
	case TokPlus, TokMinus, TokStar, TokSlash, TokAmpersand, TokPercent, TokCaret:
		return true
	default:
		return false
	}
}

// scanTokenWrites is scanExpressionWritesFrom walking a pre-tokenized script
// region directly by slice index — avoiding replay-scanner copies, Restore
// binary searches, and expressionEnds offset binary searches.
func scanTokenWrites(toks []Token, expression string) []expressionWrite {
	writes := newExpressionWrites(toks)

	var ends *expressionEnds

	defer func() {
		if ends != nil {
			ends.src = ""
			expressionEndsPool.Put(ends)
		}
	}()

	prev := TokEOF

	for i := range toks {
		t := toks[i]
		if t.Kind == TokEOF {
			break
		}

		before := prev
		prev = t.Kind

		nextKind := tokenKindAt(toks, i+1)

		// Closure-local collection bindings need their own lexical identities.
		// Withhold collection inference rather than attributing them to the parent.
		if ((t.Kind == TokEquals || t.Kind == TokMinus) && nextKind == TokGT) ||
			(t.Kind == TokIdent && identEq(t.Value, "function") && nextKind == TokLParen) {
			writes = append(writes, expressionWrite{write: collectionWrite{unknown: true}, at: t.Offset})
		}

		if t.Kind != TokIdent || before == TokDot || before == TokDoubleColon {
			continue
		}

		if !mayStartWrite(nextKind) && !identEq(t.Value, "var") {
			continue
		}

		if collectionMutator(t.Value) && nextKind == TokLParen {
			if root, element, after := collectionTargetAt(toks, i+2); root != "" && !element {
				writes = append(writes, expressionWrite{write: collectionMutatorWriteAt(t.Value, root, toks, after, expression), at: t.Offset, placed: true})
			}
		}

		local := identEq(t.Value, "var")
		startIdx := i

		if local {
			startIdx = i + 1
		}

		root, element, after := collectionTargetAt(toks, startIdx)
		if root == "" {
			continue
		}

		afterKind := tokenKindAt(toks, after)
		if afterKind == TokLParen {
			if base, method, ok := strings.CutLast(root, "."); ok && collectionMemberMutator(method) {
				writes = append(writes, expressionWrite{write: memberMutatorWriteAt(method, base, toks, after, expression), at: t.Offset, placed: true})
			}
		}

		if isCompoundWriteOp(afterKind) {
			if afterNext := tokenKindAt(toks, after+1); afterNext == TokEquals || afterNext == afterKind {
				writes = append(writes, expressionWrite{write: collectionWrite{target: root, element: element, unknown: true}, at: t.Offset, placed: true})
			}

			continue
		}

		if afterKind != TokEquals || tokenKindAt(toks, after+1) == TokEquals {
			continue
		}

		if ends == nil {
			ends, _ = expressionEndsPool.Get().(*expressionEnds)
			ends.fill(expression, toks)
		}

		writes = append(writes, expressionWrite{write: collectionWrite{target: root, expression: ends.at(after + 1), element: element, local: local || lex.HasPrefixFold(root, "local.")}, at: t.Offset, placed: true})
	}

	return writes
}

// collectionTargetAt is collectionTarget over a token slice starting at i,
// returning the index of the first token after the target.
func collectionTargetAt(toks []Token, i int) (root string, element bool, next int) {
	n := len(toks)
	if i >= n || toks[i].Kind != TokIdent {
		return "", false, i
	}

	var path chainBuilder
	path.reset(toks[i].Value)
	i++

	for i < n && toks[i].Kind == TokDot {
		i++
		if i >= n || toks[i].Kind != TokIdent {
			return "", false, i
		}

		path.writeDot()
		path.writeString(toks[i].Value)
		i++
	}

	root = path.String()
	if i >= n || toks[i].Kind != TokLBracket {
		return root, false, i
	}

	i++
	depth := 1

	for depth > 0 {
		if i >= n {
			return "", false, i
		}

		switch toks[i].Kind {
		case TokEOF:
			return "", false, i
		case TokLBracket:
			depth++
		case TokRBracket:
			depth--
		default:
		}

		i++
	}

	return root, true, i
}

func collectionMutatorWriteAt(name, root string, toks []Token, i int, expression string) collectionWrite {
	if isArrayAppendMutator(name) {
		if val, ok := mutatorArgExprAt(toks, i, expression, true); ok {
			return collectionWrite{target: root, expression: val, element: true}
		}
	}

	return collectionWrite{target: root, unknown: true}
}

func memberMutatorWriteAt(method, base string, toks []Token, i int, expression string) collectionWrite {
	if isMemberAppendMutator(method) {
		if val, ok := mutatorArgExprAt(toks, i, expression, false); ok {
			return collectionWrite{target: base, expression: val, element: true}
		}
	}

	return collectionWrite{target: base, element: true, unknown: true}
}

func mutatorArgExprAt(toks []Token, i int, expression string, afterComma bool) (string, bool) {
	n := len(toks)
	if afterComma {
		if i >= n || toks[i].Kind != TokComma {
			return "", false
		}
	} else {
		if i >= n || toks[i].Kind != TokLParen {
			return "", false
		}
	}

	i++
	if i >= n || toks[i].Kind == TokEOF || toks[i].Kind == TokRParen {
		return "", false
	}

	start := toks[i].Offset
	depth := 0

	for ; i < n; i++ {
		tok := toks[i]
		switch tok.Kind {
		case TokEOF:
			return "", false
		case TokLParen, TokLBracket, TokLBrace:
			depth++
		case TokRParen:
			if depth == 0 {
				if start < 0 || tok.Offset < start || tok.Offset > len(expression) {
					return "", false
				}

				val := strings.TrimSpace(expression[start:tok.Offset])

				return val, val != ""
			}

			depth--
		case TokRBracket, TokRBrace:
			if depth == 0 {
				return "", false
			}

			depth--
		case TokComma, TokEquals:
			if depth == 0 {
				return "", false
			}
		default:
		}
	}

	return "", false
}
