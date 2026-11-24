package parser

import (
	"strings"
)

func collectCollectionTags(text string, offset int, functionAt func(int) string, writes []collectionWrite, cache *expressionWriteCache) []collectionWrite {
	ends := newTagEnds(text)

	for pos := 0; pos < len(text); {
		idx := nextTagStart(text[pos:])
		if idx < 0 {
			break
		}

		idx += pos
		if strings.HasPrefix(text[idx:], "<!---") {
			pos = SkipCFMLComment(text, idx)

			continue
		}

		var next int

		writes, next = collectCollectionTag(&ends, idx, offset, functionAt, writes, cache)
		if next < 0 {
			break
		}

		pos = next
	}

	return writes
}

// collectCollectionTagsAt is collectCollectionTags over the tags at sites,
// where the parse's own tag walk found them (collectionTagName), rather than a
// walk of its own. It reads each as collectCollectionTags does, and stops
// where that would: at a tag with no end.
func collectCollectionTagsAt(text string, sites []int, offset int, functionAt func(int) string, writes []collectionWrite, cache *expressionWriteCache) []collectionWrite {
	ends := newTagEnds(text)

	for _, idx := range sites {
		var next int

		if writes, next = collectCollectionTag(&ends, idx, offset, functionAt, writes, cache); next < 0 {
			break
		}
	}

	return writes
}

// collectionTagName reports whether a tag named name can hold a collection
// write: <cfset>, <cfparam>, or one with collectionOutputAttributes.
func collectionTagName(name string) bool {
	return strings.EqualFold(name, "cfset") || strings.EqualFold(name, "cfparam") || collectionOutputAttributes(name) != nil
}

// collectCollectionTag reads the writes of the tag opening at ends.s[idx],
// returning where the tag ends, or -1 when it has no end.
func collectCollectionTag(ends *tagEnds, idx, offset int, functionAt func(int) string, writes []collectionWrite, cache *expressionWriteCache) ([]collectionWrite, int) {
	text := ends.s

	end := ends.at(idx)
	if end < 0 {
		return writes, -1
	}

	end += idx
	tag := text[idx : end+1]
	pos := end + 1

	name := extractIdent(tag[1:])
	if strings.EqualFold(name, "cfset") {
		body := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(tag[1+len(name):], ">"), "/"))
		writes = placeWrites(cache.scan(body), offset+idx+strings.Index(tag, body), functionAt, writes)
	}

	for _, attribute := range collectionOutputAttributes(name) {
		if root, element := collectionRead(getAttr(tag, attribute)); root != "" {
			writes = append(writes, collectionWrite{offset: offset + idx, target: root, element: element, function: functionAt(offset + idx), unknown: true})
		}
	}

	if strings.EqualFold(name, "cfparam") {
		name := getAttr(tag, "name")
		def := strings.Trim(getAttr(tag, "default"), "#")

		if name != "" {
			root, element := collectionRead(name)
			if root != "" {
				writes = append(writes, collectionWrite{offset: offset + idx, target: root, expression: def, element: element, unknown: element, function: functionAt(offset + idx)})
			}
		}
	}

	return writes, pos
}

// Only whole expressions can supply an element contract.
func (pr *ParseResult) collectionValueSource(expression, function string, locals *localSet) ReturnSource {
	if !wholeCollectionValue(expression) {
		return ReturnSource{}
	}

	if root, methods := factoryReturnChain(expression); root != "" && pr.resolverSet != nil {
		if comp := pr.resolverSet.Resolve(root); comp != "" {
			return ReturnSource{Component: comp, Methods: methods}
		}
	}

	read := pr.readSourceExpression(expression, function)
	if read.ref != nil {
		return ReturnSource{Component: read.ref.Component, Methods: read.ref.ChainRest}
	}

	for i := range read.pending {
		c := &read.pending[i]
		if c.alias {
			// A plain name is read by the caller from the refs it names.
			continue
		}

		switch {
		case c.baseVar != "":
			ref := firstRefIn(pr.funcRefsMap[parseFnKey(function)], c.baseVar, c.baseScope)
			if ref == nil && c.baseScope == RefAny && locals.has(function, c.baseVar) {
				return ReturnSource{}
			}

			if ref == nil {
				ref = firstRefIn(pr.ComponentRefs, c.baseVar, c.baseScope)
			}

			if ref != nil {
				return ReturnSource{Component: pr.settledComponent(ref), Methods: append([]string{collectionCallHop(c)}, c.rest...)}
			}
		case pr.declaresFunction(c.funcName):
			return ReturnSource{Component: pr.URI.Path(), Methods: append([]string{collectionCallHop(c)}, c.rest...)}
		case pr.BuiltinReturnLookup != nil:
			// A native global the file does not shadow, as a pending call on a
			// plain variable is typed (resolvePendingCalls): `instance.q =
			// queryNew( "a" )` is a query.
			if comp := pr.BuiltinReturnLookup(c.funcName); comp != "" {
				return ReturnSource{Component: comp, Methods: c.rest}
			}
		}
	}

	return ReturnSource{}
}

func (n *collectionNode) addSource(source ReturnSource) {
	if n.sources == nil {
		n.sources = map[string]ReturnSource{}
	}

	key := source.Component + "\t" + strings.Join(source.Methods, "\t")
	if len(n.sources) >= 16 {
		if _, exists := n.sources[key]; !exists {
			n.invalid = true

			return
		}
	}

	n.sources[key] = source
}

func wholeCollectionValue(expression string) bool {
	sc := NewScanner(expression)

	first := sc.PeekSkipComments()
	if first.Kind == TokIdent && identEq(first.Value, "new") {
		sc.NextSkipComments()
	}

	if path, _ := collectionTarget(sc); path == "" {
		return false
	}

	for {
		switch sc.NextSkipComments().Kind {
		case TokEOF:
			return true
		case TokLParen:
			depth := 1
			for depth > 0 {
				switch sc.NextSkipComments().Kind {
				case TokEOF:
					return false
				case TokLParen:
					depth++
				case TokRParen:
					depth--
				default:
				}
			}

			if sc.PeekSkipComments().Kind == TokDot {
				sc.NextSkipComments()

				if sc.NextSkipComments().Kind != TokIdent {
					return false
				}

				if sc.PeekSkipComments().Kind != TokLParen {
					return false
				}
			}
		default:
			return false
		}
	}
}

func collectionMutatorWrite(name, root string, cursor *Scanner, expression string) collectionWrite {
	if isArrayAppendMutator(name) {
		if val, ok := mutatorArgExpr(cursor, expression, true); ok {
			return collectionWrite{target: root, expression: val, element: true}
		}
	}

	return collectionWrite{target: root, unknown: true}
}

func memberMutatorWrite(method, base string, cursor *Scanner, expression string) collectionWrite {
	if isMemberAppendMutator(method) {
		c := *cursor
		if val, ok := mutatorArgExpr(&c, expression, false); ok {
			return collectionWrite{target: base, expression: val, element: true}
		}
	}

	return collectionWrite{target: base, element: true, unknown: true}
}

func isArrayAppendMutator(name string) bool {
	return identEq(name, "arrayappend") || identEq(name, "arrayprepend")
}

func isMemberAppendMutator(name string) bool {
	return identEq(name, "append") || identEq(name, "prepend")
}

// mutatorArgExpr extracts the single positional value expression passed to
// arrayAppend(arr, expr) / arrayPrepend(arr, expr) (afterComma=true) or
// arr.append(expr) / arr.prepend(expr) (afterComma=false). Calls with a merge
// flag, extra arguments, or named arguments return false so the caller falls
// back to invalidating the collection.
func mutatorArgExpr(sc *Scanner, expression string, afterComma bool) (string, bool) {
	if afterComma {
		if sc.NextSkipComments().Kind != TokComma {
			return "", false
		}
	} else {
		if sc.NextSkipComments().Kind != TokLParen {
			return "", false
		}
	}

	first := sc.PeekSkipComments()
	if first.Kind == TokEOF || first.Kind == TokRParen {
		return "", false
	}

	start := first.Offset
	depth := 0

	for {
		tok := sc.NextSkipComments()
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
}

func startsArrayLiteral(expression string) bool {
	s := strings.TrimSpace(expression)

	return len(s) >= 3 && s[0] == '[' && s[len(s)-1] == ']'
}

// arrayLiteralElements splits a top-level non-empty array literal `[e1, e2]`
// into its element expressions. Compound expressions like `[a] + [b]` return
// false.
func arrayLiteralElements(expression string) ([]string, bool) {
	s := strings.TrimSpace(expression)
	if len(s) < 3 || s[0] != '[' || s[len(s)-1] != ']' {
		return nil, false
	}

	sc := NewScanner(s)
	if sc.NextSkipComments().Kind != TokLBracket {
		return nil, false
	}

	depth := 1
	elemStart := sc.PeekSkipComments().Offset

	var elems []string

	for {
		tok := sc.NextSkipComments()
		switch tok.Kind {
		case TokEOF:
			return nil, false
		case TokLBracket, TokLParen, TokLBrace:
			depth++
		case TokRBracket, TokRParen, TokRBrace:
			depth--
			if depth == 0 {
				if tok.Kind != TokRBracket || sc.PeekSkipComments().Kind != TokEOF {
					return nil, false
				}

				el := strings.TrimSpace(s[elemStart:tok.Offset])
				if el == "" {
					return nil, false
				}

				elems = append(elems, el)

				return elems, true
			}
		case TokComma:
			if depth == 1 {
				el := strings.TrimSpace(s[elemStart:tok.Offset])
				if el == "" {
					return nil, false
				}

				elems = append(elems, el)
				elemStart = sc.PeekSkipComments().Offset
			}
		default:
		}
	}
}

func collectionMutator(name string) bool {
	var buf foldScratch
	switch string(buf.lowerFold(name)) {
	case "structinsert", "structappend", "structupdate", "arrayappend", "arrayprepend", "arrayset", "arrayinsertat":
		return true
	default:
		return false
	}
}

func collectionMemberMutator(name string) bool {
	var buf foldScratch
	switch string(buf.lowerFold(name)) {
	case "insert", "append", "update", "prepend", "set", "insertat":
		return true
	default:
		return false
	}
}

// Output bindings assign non-element contracts. Their contents are not modeled.
func collectionOutputAttributes(name string) []string {
	var buf foldScratch
	switch string(buf.lowerFold(name)) {
	case "cfquery", "cfstoredproc":
		return []string{"name", "result"}
	case "cfobject", "cfsearch", "cfldap", "cfdbinfo", "cfprocresult", "cfpop", "cfimap", "cfspreadsheet", "cfdocument":
		return []string{"name"}
	case "cfsavecontent", "cfxml", "cfexecute", "cfprocparam":
		return []string{"variable"}
	case "cfwddx":
		return []string{"output"}
	case "cfinvoke":
		return []string{"returnvariable"}
	case "cfhttp":
		return []string{"result", "name"}
	case "cffile", "cfdirectory", "cfimage", "cfzip":
		return []string{"variable", "name", "result"}
	case "cfloop":
		return []string{"index", "item"}
	default:
		return nil
	}
}

func collectionCallHop(c *pendingCall) string {
	if c.expression != "" {
		return CallHop(c.expression)
	}

	return c.funcName
}
