package parser

import (
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/lex"
)

type collectionWrite struct {
	target, expression, function string
	offset                       int
	element, local, unknown      bool
}

type collectionNode struct {
	sources                        map[string]ReturnSource
	structure, invalid, written    bool
	structSeed, arraySeed, escaped bool
	dependencies                   []collKey
	aliases                        []collKey
}

// collectionTarget reads static paths with at most one final subscript. The
// key may be dynamic: inference describes all values, never a particular key.
func collectionTarget(sc *Scanner) (root string, element bool) {
	t := sc.NextSkipComments()
	if t.Kind != TokIdent {
		return "", false
	}

	var path chainBuilder
	path.reset(t.Value)

	for sc.PeekSkipComments().Kind == TokDot {
		sc.NextSkipComments()

		t = sc.NextSkipComments()
		if t.Kind != TokIdent {
			return "", false
		}

		path.writeDot()
		path.writeString(t.Value)
	}

	root = path.String()
	if sc.PeekSkipComments().Kind != TokLBracket {
		return root, false
	}

	sc.NextSkipComments()

	depth := 1
	for depth > 0 {
		switch sc.NextSkipComments().Kind {
		case TokEOF:
			return "", false
		case TokLBracket:
			depth++
		case TokRBracket:
			depth--
		default:
		}
	}

	return root, true
}

func collectionRead(expression string) (root string, element bool) {
	held := collectionReads.get(expression, func(expression string) collectionReadAnswer {
		root, element := collectionReadAt(NewScanner(expression))

		return collectionReadAnswer{root: strings.Clone(root), element: element}
	})

	return held.root, held.element
}

type collectionReadAnswer struct {
	root    string
	element bool
}

var collectionReads exprMemo[collectionReadAnswer]

// collectionReadAt is collectionRead on a scanner at the expression's start.
func collectionReadAt(sc *Scanner) (root string, element bool) {
	root, element = collectionTarget(sc)
	if sc.NextSkipComments().Kind != TokEOF {
		return "", false
	}

	return root, element
}

// collectionReturnExpressions groups the return expressions by function, or
// is nil when none could be answered: the pass types a function that returns
// an element (`return x[k]`), and one returning a whole struct only in a file
// with no tag region, since a tag region marks every collection escaped
// (markCollectionEscapes). A tag component returning variables, the common
// case, skips the pass.
func collectionReturnExpressions(calls []pendingCall, tagFile bool) map[int][]string {
	candidates := false

	for i := range calls {
		c := &calls[i]
		if c.returnExpr {
			if root, element := collectionRead(c.varName); root != "" && (element || !tagFile) {
				candidates = true

				break
			}
		}
	}

	if !candidates {
		return nil
	}

	returns := map[int][]string{}

	for i := range calls {
		c := &calls[i]
		if c.returnExpr {
			start := -1
			if c.funcKey.in {
				start = int(c.funcKey.start)
			}

			returns[start] = append(returns[start], c.varName)
		}
	}

	return returns
}

func (pr *ParseResult) collectionKeys(writes []collectionWrite) (*localSet, map[int]string, func(string, string) collKey, *keyTable) {
	kt := newKeyTableCap(len(pr.Scopes) + len(writes)/2 + 8)
	locals := &localSet{kt: kt, m: make(map[collKey]bool, len(pr.Funcs)*2+len(writes)/4)}

	scopeKeys := make(map[int]string, len(pr.Scopes))
	for i := range pr.Scopes {
		scopeKeys[pr.Scopes[i].Start] = pr.scopeKey(i)
	}

	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		for _ai := range f.Arguments {
			a := &f.Arguments[_ai]
			locals.add(scopeKeys[int(f.Line)], a.Name)
		}
	}

	for _, w := range writes {
		if w.local && w.function != "" {
			target := w.target
			if lex.HasPrefixFold(target, "local.") {
				target = target[len("local."):]
			}

			locals.add(w.function, target)
		}
	}

	return locals, scopeKeys, func(root, function string) collKey {
		switch {
		case lex.HasPrefixFold(root, "variables."):
			return kt.field(idVariables, root[len("variables."):])
		case lex.HasPrefixFold(root, "this."):
			return kt.field(idThis, root[len("this."):])
		case lex.HasPrefixFold(root, "local."):
			return kt.local(function, root[len("local."):])
		case lex.HasPrefixFold(root, "arguments."):
			return kt.local(function, root[len("arguments."):])
		}

		base, _, _ := strings.Cut(root, ".")
		if locals.has(function, base) {
			return kt.local(function, root)
		}

		return kt.field(idVariables, root)
	}, kt
}

func (pr *ParseResult) applyCollectionReturns(calls []pendingCall) {
	returns := collectionReturnExpressions(calls, pr.hasTagRegion())
	if returns == nil {
		return
	}

	writes := pr.collectionWrites()

	// Each write's key, built once: three loops below read it, and so does
	// the member pass, and building one lowercases a string and looks the
	// name up among the locals.
	ks := pr.writeKeys(writes)
	locals, keys, key, targets, kt := ks.locals, ks.scopeKeys, ks.key, ks.targets, ks.kt
	roots := make(map[collKey]bool, len(writes)/4+len(returns))

	for i := range writes {
		if w := &writes[i]; w.element || emptyCollection(w.expression) || startsArrayLiteral(w.expression) {
			roots[targets[i]] = true
		}
	}

	for start, expressions := range returns {
		fnStr := ""

		if start >= 0 {
			fnStr = keys[start]
		}

		for _, expr := range expressions {
			if root, element := collectionRead(expr); element {
				roots[key(root, fnStr)] = true
			}
		}
	}

	var slab []collectionNode

	nodes := make(map[collKey]*collectionNode, len(roots))
	node := func(k collKey) *collectionNode {
		if n := nodes[k]; n != nil {
			return n
		}

		if len(slab) == 0 {
			slab = make([]collectionNode, 16)
		}

		n := &slab[0]
		slab = slab[1:]
		nodes[k] = n

		return n
	}

	// Whether the file declares a structNew of its own, asked once: it walks
	// every function and argument, and was asked for every empty-struct write.
	structNew := sync.OnceValue(func() bool { return pr.Declares("structNew") })

	for i := range writes {
		pr.recordCollectionWrite(writes[i], targets[i], roots, locals, key, node, structNew, kt)
	}

	invalidateClosureCollections(writes, pr.Scopes, nodes, kt)

	invalidateCollectionParents(writes, targets, nodes, kt)

	// A collection argument may contain caller-supplied values even when its
	// declaration has an empty-struct default.
	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		for _ai := range f.Arguments {
			arg := &f.Arguments[_ai]
			if n := nodes[kt.local(keys[int(f.Line)], arg.Name)]; n != nil {
				n.invalid = true
			}
		}
	}

	settleCollections(nodes)

	// An escape only vetoes a whole-struct contract, so the escape scan, which
	// reads every script region again, runs only when some function would
	// have one without it. It was a third of a script component's parse, for
	// an answer most files never have.
	escapes := false

	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		fnLine := int(f.Line)
		fnKeyStr := keys[fnLine]
		fnReturns := returns[fnLine]
		f.CollectionValues = wholeStructValues(f, fnReturns, fnKeyStr, key, nodes)

		if f.CollectionValues != nil && !escapes {
			escapes = true

			pr.markCollectionEscapes(key, nodes, kt)
			f.CollectionValues = wholeStructValues(f, fnReturns, fnKeyStr, key, nodes)
		}

		pr.applyCollectionFunction(f, fnReturns, fnKeyStr, key, nodes)
	}
}

// Closure writes still use their enclosing named function's identity. Withhold
// that function's locals and every shared field, but do not let an unrelated
// closure invalidate another named function's locals. Alias settlement carries
// invalidation from shared fields back to any local collection exposed there.
func invalidateClosureCollections(writes []collectionWrite, scopes []FuncScope, nodes map[collKey]*collectionNode, kt *keyTable) {
	functions := map[string]bool{}

	for _, w := range writes {
		if w.target == "" && w.unknown {
			functions[w.function] = true
		}
	}

	if len(functions) == 0 {
		return
	}

	// Nested named functions can capture enclosing locals. Their write scopes
	// are not yet lexical identities, so retain the old veto for overlaps,
	// including ambiguous functions that share a source line.
	overlapping, end := false, -1

	for _, scope := range scopes {
		overlapping = overlapping || scope.Start <= end
		end = max(end, scope.End)
	}

	// The scope ids of those functions. A function no key was built for has
	// none, and no local to withhold.
	owners := make(map[uint32]bool, len(functions))

	for function := range functions {
		if id, ok := kt.findLower(function); ok {
			owners[id] = true
		}
	}

	for k, n := range nodes {
		if overlapping || !k.local() || owners[k.scopeID()] {
			n.invalid = true
		}
	}
}

// Walking each node's static ancestors avoids comparing every write to every
// node. Dynamic writes into a scope invalidate any field in that scope.
func invalidateCollectionParents(writes []collectionWrite, targets []collKey, nodes map[collKey]*collectionNode, kt *keyTable) {
	parents := make(map[collKey]bool, len(writes))

	for i := range writes {
		w := &writes[i]
		if !w.element {
			parents[targets[i]] = true

			continue
		}

		switch {
		case strings.EqualFold(w.target, "variables"):
			parents[kt.fieldScope(idVariables)] = true
		case strings.EqualFold(w.target, "this"):
			parents[kt.fieldScope(idThis)] = true
		case strings.EqualFold(w.target, "local"), strings.EqualFold(w.target, "arguments"):
			parents[kt.localScope(w.function)] = true
		}
	}

	for k, n := range nodes {
		// A local is also under its function's whole scope, and a field
		// under the whole variables or this scope.
		if parents[scopeLevel(k)] {
			n.invalid = true

			continue
		}

		for parent := k; ; {
			above, _, found := strings.CutLast(kt.name(parent), ".")
			if !found {
				break
			}

			parent = kt.withName(parent, above)
			if parents[parent] {
				n.invalid = true

				break
			}
		}
	}
}

func (pr *ParseResult) recordCollectionWrite(w collectionWrite, target collKey, roots map[collKey]bool, locals *localSet, key func(string, string) collKey, node func(collKey) *collectionNode, structNew func() bool, kt *keyTable) {
	for prefix := target; ; {
		parent, _, found := strings.CutLast(kt.name(prefix), ".")
		if !found {
			break
		}

		prefix = kt.withName(prefix, parent)
		if roots[prefix] {
			target = prefix
			w.element = true

			break
		}
	}

	n := node(target)

	n.written = true
	if w.unknown {
		n.invalid = true

		return
	}

	if source, element := collectionRead(w.expression); source != "" {
		if element == w.element {
			dep := key(source, w.function)
			n.dependencies = append(n.dependencies, dep)
			node(dep)

			if !element {
				// Whole struct assignment shares storage. Element copies
				// only carry the source's value contract in one direction.
				n.aliases = append(n.aliases, dep)
				sourceNode := node(dep)
				sourceNode.dependencies = append(sourceNode.dependencies, target)
				sourceNode.aliases = append(sourceNode.aliases, target)
			}

			return
		}
	}

	if !w.element {
		switch {
		case emptyCollection(w.expression):
			n.structure = true
			n.arraySeed = n.arraySeed || strings.HasPrefix(strings.TrimSpace(w.expression), "[")
			n.structSeed = n.structSeed || strings.HasPrefix(strings.TrimSpace(w.expression), "{") || !n.arraySeed && !structNew()
		case startsArrayLiteral(w.expression):
			elems, ok := arrayLiteralElements(w.expression)
			if !ok {
				n.invalid = true

				return
			}

			n.structure = true
			n.arraySeed = true

			for _, el := range elems {
				if source, elem := collectionRead(el); source != "" && elem {
					dep := key(source, w.function)
					n.dependencies = append(n.dependencies, dep)
					node(dep)

					continue
				}

				source := pr.collectionValueSource(el, w.function, locals)
				if source.Component == "" || strings.HasPrefix(source.Component, "$") {
					n.invalid = true

					return
				}

				n.addSource(source)
			}
		default:
			n.invalid = true
		}

		return
	}

	source := pr.collectionValueSource(w.expression, w.function, locals)
	if source.Component == "" || strings.HasPrefix(source.Component, "$") {
		n.invalid = true
	} else {
		n.addSource(source)
	}
}
