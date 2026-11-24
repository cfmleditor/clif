package parser

import (
	"sort"
	"strings"
)

func (pr *ParseResult) applyCollectionFunction(f *FunctionDef, expressions []string, function string, key func(string, string) collKey, nodes map[collKey]*collectionNode) {
	combined := &collectionNode{}

	candidate, invalid := false, false

	for _, expression := range expressions {
		root, element := collectionRead(expression)
		if !element {
			invalid = true

			continue
		}

		candidate = true

		n := nodes[key(root, function)]
		if n == nil || n.invalid {
			invalid = true

			continue
		}

		for _, source := range n.sources {
			combined.addSource(source)
		}
	}

	if !candidate {
		return
	}

	f.returnVar = ""
	f.ReturnComponent = ""
	f.ReturnSources = nil

	if invalid || combined.invalid {
		return
	}

	var buf foldScratch
	switch string(buf.lowerFold(f.ReturnType)) {
	case "", "any", "component", "object":
	default:
		return
	}

	sources := make([]ReturnSource, 0, len(combined.sources))
	for _, source := range combined.sources {
		sources = append(sources, source)
	}

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Component+"\t"+strings.Join(sources[i].Methods, "\t") < sources[j].Component+"\t"+strings.Join(sources[j].Methods, "\t")
	})
	f.ReturnSources = sources
	answer := ""

	for _, source := range sources {
		comp := source.Component
		if pr.FuncLookup != nil {
			comp = pr.walkChainRest(comp, source.Methods)
		} else {
			comp = dynamicIfTyped(comp, source.Methods)
		}

		if comp == "" || strings.HasPrefix(comp, "$") || answer != "" && !strings.EqualFold(answer, comp) {
			answer = ""

			break
		}

		answer = comp
	}

	f.ReturnComponent = pr.componentReturnFor(f, answer)
}

func emptyCollection(expression string) bool {
	return emptyCollections.get(expression, func(expression string) bool { return emptyCollectionAt(NewScanner(expression)) })
}

var emptyCollections exprMemo[bool]

// emptyCollectionAt is emptyCollection on a scanner at the expression's start.
func emptyCollectionAt(sc *Scanner) bool {
	t := sc.NextSkipComments()
	switch {
	case t.Kind == TokLBrace || t.Kind == TokLBracket:
		closing := TokRBrace
		if t.Kind == TokLBracket {
			closing = TokRBracket
		}

		if sc.NextSkipComments().Kind != closing {
			return false
		}
	case t.Kind == TokIdent && identEq(t.Value, "structNew"):
		if sc.NextSkipComments().Kind != TokLParen {
			return false
		}

		if sc.NextSkipComments().Kind != TokRParen {
			return false
		}
	default:
		return false
	}

	return sc.NextSkipComments().Kind == TokEOF
}

// Element-copy dependencies are directed; whole aliases share writes.
// A seed can ground a cycle, but an empty or
// uninitialized source cannot acquire a type from a collection that reads it.
func settleCollections(nodes map[collKey]*collectionNode) {
	reverse := map[collKey][]collKey{}

	for key, n := range nodes {
		if !n.written {
			n.invalid = true
		}

		for _, dep := range n.dependencies {
			reverse[dep] = append(reverse[dep], key)
		}
	}

	queue := make([]collKey, 0, len(nodes))
	for key := range nodes {
		queue = append(queue, key)
	}

	propagate := func() {
		for len(queue) > 0 {
			key := queue[0]
			queue = queue[1:]

			n := nodes[key]
			for _, target := range reverse[key] {
				dst := nodes[target]

				count, structure, invalid := len(dst.sources), dst.structure, dst.invalid

				structSeed, arraySeed := dst.structSeed, dst.arraySeed
				if n.invalid {
					dst.invalid = true
				}

				for _, source := range n.sources {
					dst.addSource(source)
				}

				for _, alias := range dst.aliases {
					if alias == key && n.structure {
						dst.structure = true
						dst.structSeed = dst.structSeed || n.structSeed
						dst.arraySeed = dst.arraySeed || n.arraySeed
					}
				}

				if len(dst.sources) != count || dst.structure != structure || dst.invalid != invalid || dst.structSeed != structSeed || dst.arraySeed != arraySeed {
					queue = append(queue, target)
				}
			}
		}
	}
	propagate()

	for key, n := range nodes {
		if len(n.sources) == 0 || !n.structure {
			n.invalid = true

			queue = append(queue, key)
		}
	}

	propagate()
}

type writesMemo struct {
	writes []collectionWrite
	done   bool
	// tokens are each script region's, by its offset (scriptTokens), from
	// tokenPool until release.
	tokens map[int]*[]Token
	// keys is collectionKeys' answer for writes and each write's key, which
	// the member and collection passes both read (writeKeys).
	keys *writeKeySet
	// sources is readSourceExpression's answer by expression and whether it
	// is read inside a function, and funcNames the file's function names,
	// lowercased: both passes ask about the same expressions.
	sources   map[sourceKey]sourceRead
	funcNames map[string]bool
	// scopeKeys is funcKey of each of pr.Scopes (scopeKey).
	scopeKeys []string
}

// scopeKey is funcKey for pr.Scopes[i], built once for the passes sharing
// writesMemo: every collection write asks for its function's key, and
// formatting one is an allocation each time.
func (pr *ParseResult) scopeKey(i int) string {
	m := pr.writesMemo
	if m == nil {
		return funcKey(pr.Scopes[i].Start, pr.Scopes[i].End).String()
	}

	if m.scopeKeys == nil {
		m.scopeKeys = make([]string, len(pr.Scopes))
		for j := range pr.Scopes {
			m.scopeKeys[j] = funcKey(pr.Scopes[j].Start, pr.Scopes[j].End).String()
		}
	}

	return m.scopeKeys[i]
}

type sourceKey struct {
	expression string
	inFunction bool
}

// sourceRead is what a tag parser reads from an assignment's right-hand side:
// the first ref it makes, and its pending calls.
type sourceRead struct {
	ref     *ComponentRef
	pending []pendingCall
}

// readSourceExpression reads expression as the right-hand side of an
// assignment in function, as collectionValueSource needs it. What it reads
// does not depend on which function beyond whether there is one, so it is
// read once per expression for the passes sharing writesMemo: the member and
// collection passes ask about the same right-hand sides, and the resolver
// matching each read pays for was 4.5ms of an edit on a 58,000-line component.
func (pr *ParseResult) readSourceExpression(expression, function string) sourceRead {
	key := sourceKey{expression, function != ""}

	m := pr.writesMemo
	if m != nil {
		if read, ok := m.sources[key]; ok {
			return read
		}
	}

	tp := &tagParser{fileURI: string(pr.URI), inFunc: parseFnKey(function), resolvers: pr.Resolvers, resolverSet: pr.resolverSet, funcs: pr.Funcs}
	tp.checkSetRHSStr(expression, "$element", 0)

	refs := tp.componentRefs
	if tp.inFunc.in {
		refs = tp.funcRefs[tp.inFunc]
	}

	var read sourceRead
	if len(refs) > 0 {
		read.ref = &refs[0]
	} else {
		read.pending = tp.pendingCalls
	}

	if m != nil {
		if m.sources == nil {
			m.sources = map[sourceKey]sourceRead{}
		}

		m.sources[key] = read
	}

	return read
}

// declaresFunction reports whether the file declares a function named name,
// from a set built once for the passes sharing writesMemo rather than a walk
// of every function for each call asked about.
func (pr *ParseResult) declaresFunction(name string) bool {
	m := pr.writesMemo
	if m == nil {
		for i := range pr.Funcs {
			if strings.EqualFold(pr.Funcs[i].Name, name) {
				return true
			}
		}

		return false
	}

	if m.funcNames == nil {
		m.funcNames = make(map[string]bool, len(pr.Funcs))
		for i := range pr.Funcs {
			m.funcNames[strings.ToLower(pr.Funcs[i].Name)] = true
		}
	}

	var buf foldScratch

	return m.funcNames[string(buf.lowerFold(name))]
}

// writeKeySet is collectionKeys' answer and each write's key.
type writeKeySet struct {
	locals    *localSet
	scopeKeys map[int]string
	key       func(string, string) collKey
	kt        *keyTable
	targets   []collKey // key(w.target, w.function) for each write
}

// writeKeys is collectionKeys for writes, with each write's key built once
// for every pass that shares writesMemo.
func (pr *ParseResult) writeKeys(writes []collectionWrite) *writeKeySet {
	if m := pr.writesMemo; m != nil && m.keys != nil {
		return m.keys
	}

	locals, scopeKeys, key, kt := pr.collectionKeys(writes)
	ks := &writeKeySet{locals: locals, scopeKeys: scopeKeys, key: key, kt: kt, targets: make([]collKey, len(writes))}

	for i := range writes {
		ks.targets[i] = key(writes[i].target, writes[i].function)
	}

	if m := pr.writesMemo; m != nil {
		m.keys = ks
	}

	return ks
}

// release returns the memo's token lists to tokenPool. Nothing the passes
// keep refers to a token list: what they keep is text from the source.
func (m *writesMemo) release() {
	for _, toks := range m.tokens {
		tokenPool.Put(toks)
	}

	m.tokens = nil
}
