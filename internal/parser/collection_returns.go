package parser

import (
	"slices"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
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
	dependencies                   []string
	aliases                        []string
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
	return collectionReadAt(NewScanner(expression))
}

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
func collectionReturnExpressions(calls []pendingCall, tagFile bool) map[string][]string {
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

	returns := map[string][]string{}

	for i := range calls {
		c := &calls[i]
		if c.returnExpr {
			returns[c.funcKey] = append(returns[c.funcKey], c.varName)
		}
	}

	return returns
}

func (pr *ParseResult) collectionKeys(writes []collectionWrite) (map[string]bool, map[int]string, func(string, string) string) {
	locals := map[string]bool{}

	scopeKeys := map[int]string{}
	for _, scope := range pr.Scopes {
		scopeKeys[scope.Start] = funcKey(scope.Start, scope.End)
	}

	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		for _, a := range f.Arguments {
			locals[scopeKeys[int(f.Line)]+"\t"+strings.ToLower(a.Name)] = true
		}
	}

	for _, w := range writes {
		if w.local && w.function != "" {
			locals[w.function+"\t"+strings.TrimPrefix(strings.ToLower(w.target), "local.")] = true
		}
	}

	// The key is built lowercased in one allocation: lowercasing root first,
	// then joining it, was a second string for every mixed-case name.
	return locals, scopeKeys, func(root, function string) string {
		switch {
		case hasPrefixFold(root, "variables.") || hasPrefixFold(root, "this."):
			return concatLower("", root)
		case hasPrefixFold(root, "local."):
			return concatLower(function+"\t", root[len("local."):])
		case hasPrefixFold(root, "arguments."):
			return concatLower(function+"\t", root[len("arguments."):])
		}

		base, _, _ := strings.Cut(root, ".")
		if locals[concatLower(function+"\t", base)] {
			return concatLower(function+"\t", root)
		}

		return concatLower("variables.", root)
	}
}

// concatLower is prefix + strings.ToLower(s) in one allocation, and prefix
// itself when s is empty; s alone, uncopied, when prefix is empty and s has
// nothing to lower.
func concatLower(prefix, s string) string {
	upper := false

	for i := range len(s) {
		c := s[i]
		if c >= utf8.RuneSelf {
			return prefix + strings.ToLower(s)
		}

		upper = upper || c >= 'A' && c <= 'Z'
	}

	if !upper {
		if prefix == "" {
			return s
		}

		return prefix + s
	}

	var b strings.Builder

	b.Grow(len(prefix) + len(s))
	b.WriteString(prefix)

	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}

		b.WriteByte(c)
	}

	return b.String()
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
	locals, keys, key, targets := ks.locals, ks.scopeKeys, ks.key, ks.targets
	roots := map[string]bool{}

	for i := range writes {
		if w := &writes[i]; w.element || emptyCollection(w.expression) {
			roots[targets[i]] = true
		}
	}

	for function, expressions := range returns {
		for _, expr := range expressions {
			if root, element := collectionRead(expr); element {
				roots[key(root, function)] = true
			}
		}
	}

	nodes := map[string]*collectionNode{}
	node := func(k string) *collectionNode {
		if nodes[k] == nil {
			nodes[k] = &collectionNode{}
		}

		return nodes[k]
	}

	// Whether the file declares a structNew of its own, asked once: it walks
	// every function and argument, and was asked for every empty-struct write.
	structNew := sync.OnceValue(func() bool { return pr.Declares("structNew") })

	for i := range writes {
		pr.recordCollectionWrite(writes[i], targets[i], roots, locals, key, node, structNew)
	}

	invalidateClosureCollections(writes, pr.Scopes, nodes)

	invalidateCollectionParents(writes, targets, nodes)

	// A collection argument may contain caller-supplied values even when its
	// declaration has an empty-struct default.
	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		for _, arg := range f.Arguments {
			if n := nodes[key("arguments."+arg.Name, keys[int(f.Line)])]; n != nil {
				n.invalid = true
			}
		}
	}

	settleCollections(nodes)
	pr.markCollectionEscapes(key, nodes)

	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		f.CollectionValues = wholeStructValues(f, returns[keys[int(f.Line)]], keys[int(f.Line)], key, nodes)
		pr.applyCollectionFunction(f, returns[keys[int(f.Line)]], keys[int(f.Line)], key, nodes)
	}
}

// Closure writes still use their enclosing named function's identity. Withhold
// that function's locals and every shared field, but do not let an unrelated
// closure invalidate another named function's locals. Alias settlement carries
// invalidation from shared fields back to any local collection exposed there.
func invalidateClosureCollections(writes []collectionWrite, scopes []FuncScope, nodes map[string]*collectionNode) {
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

	for k, n := range nodes {
		function, _, local := strings.Cut(k, "\t")
		if overlapping || !local || functions[function] {
			n.invalid = true
		}
	}
}

// Walking each node's static ancestors avoids comparing every write to every
// node. Dynamic writes into a scope invalidate any field in that scope.
func invalidateCollectionParents(writes []collectionWrite, targets []string, nodes map[string]*collectionNode) {
	parents := map[string]bool{}

	for i := range writes {
		w := &writes[i]
		if !w.element {
			parents[targets[i]] = true

			continue
		}

		switch {
		case strings.EqualFold(w.target, "variables"), strings.EqualFold(w.target, "this"):
			parents[strings.ToLower(w.target)] = true
		case strings.EqualFold(w.target, "local"), strings.EqualFold(w.target, "arguments"):
			parents[w.function+"\t"] = true
		}
	}

	for k, n := range nodes {
		if function, _, ok := strings.Cut(k, "\t"); ok && parents[function+"\t"] {
			n.invalid = true
		}

		for parent := k; ; {
			index := strings.LastIndexByte(parent, '.')
			if index < 0 {
				break
			}

			parent = parent[:index]
			if parents[parent] {
				n.invalid = true

				break
			}
		}
	}
}

func (pr *ParseResult) recordCollectionWrite(w collectionWrite, target string, roots map[string]bool, locals map[string]bool, key func(string, string) string, node func(string) *collectionNode, structNew func() bool) {
	for prefix := target; ; {
		index := strings.LastIndexByte(prefix, '.')
		if index < 0 {
			break
		}

		prefix = prefix[:index]
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
		if emptyCollection(w.expression) {
			n.structure = true
			n.arraySeed = n.arraySeed || strings.HasPrefix(strings.TrimSpace(w.expression), "[")
			n.structSeed = n.structSeed || strings.HasPrefix(strings.TrimSpace(w.expression), "{") || !n.arraySeed && !structNew()
		} else {
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

func (pr *ParseResult) applyCollectionFunction(f *FunctionDef, expressions []string, function string, key func(string, string) string, nodes map[string]*collectionNode) {
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
	return emptyCollectionAt(NewScanner(expression))
}

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
func settleCollections(nodes map[string]*collectionNode) {
	reverse := map[string][]string{}

	for key, n := range nodes {
		if !n.written {
			n.invalid = true
		}

		for _, dep := range n.dependencies {
			reverse[dep] = append(reverse[dep], key)
		}
	}

	queue := make([]string, 0, len(nodes))
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
}

// writeKeySet is collectionKeys' answer and each write's key.
type writeKeySet struct {
	locals    map[string]bool
	scopeKeys map[int]string
	key       func(string, string) string
	targets   []string // key(w.target, w.function) for each write
}

// writeKeys is collectionKeys for writes, with each write's key built once
// for every pass that shares writesMemo.
func (pr *ParseResult) writeKeys(writes []collectionWrite) *writeKeySet {
	if m := pr.writesMemo; m != nil && m.keys != nil {
		return m.keys
	}

	locals, scopeKeys, key := pr.collectionKeys(writes)
	ks := &writeKeySet{locals: locals, scopeKeys: scopeKeys, key: key, targets: make([]string, len(writes))}

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

// scriptTokens is a script region's tokens, interpolated strings included,
// tokenised once for every pass that shares writesMemo.
func (pr *ParseResult) scriptTokens(r *Region) *[]Token {
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
		lines = buildLineIdx(pr.Content)
	}

	functionAt := func(offset int) string {
		line := sort.Search(len(lines), func(i int) bool { return int(lines[i]) > offset }) - 1

		idx := sort.Search(len(pr.Scopes), func(i int) bool { return pr.Scopes[i].Start > line }) - 1
		if idx >= 0 && line <= pr.Scopes[idx].End {
			return funcKey(pr.Scopes[idx].Start, pr.Scopes[idx].End)
		}

		return ""
	}

	for i := range pr.Regions {
		r := &pr.Regions[i]
		if r.Kind == RegionScript {
			toks := pr.scriptTokens(r)
			writes = placeWrites(scanExpressionWritesFrom(newReplayScanner(r.Text, toks), r.Text, *toks), r.Offset, functionAt, writes)

			continue
		}

		if r.Kind != RegionTag {
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

// scanExpressionWritesFrom is scanExpressionWrites reading sc, a scanner over
// expression, and toks, its tokens when sc replays them.
func scanExpressionWritesFrom(sc *Scanner, expression string, toks []Token) []expressionWrite {
	var writes []expressionWrite

	var ends *expressionEnds

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

		cursor := *sc
		if collectionMutator(t.Value) && cursor.PeekSkipComments().Kind == TokLParen {
			cursor.NextSkipComments()

			root, element := collectionTarget(&cursor)
			if root != "" && !element {
				writes = append(writes, expressionWrite{write: collectionWrite{target: root, unknown: true}, at: t.Offset, placed: true})
			}
		}

		cursor = *sc
		cursor.Restore(ScannerState{pos: t.Offset, line: t.Line})
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
				writes = append(writes, expressionWrite{write: collectionWrite{target: base, element: true, unknown: true}, at: t.Offset, placed: true})
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
				toks = tokenize(expression)
			}

			ends = newExpressionEndsFrom(expression, toks)
		}

		writes = append(writes, expressionWrite{write: collectionWrite{target: root, expression: ends.from(cursor.PeekSkipComments().Offset), element: element, local: local || hasPrefixFold(root, "local.")}, at: t.Offset, placed: true})
	}

	return writes
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
}

func newExpressionEnds(src string) *expressionEnds {
	return newExpressionEndsFrom(src, tokenize(src))
}

// newExpressionEndsFrom is newExpressionEnds from src's tokens.
func newExpressionEndsFrom(src string, toks []Token) *expressionEnds {
	var (
		offsets, depths []int
		terminal        []bool
	)

	depth := 0

	for _, t := range toks {
		if t.Kind == TokEOF {
			break
		}

		offsets = append(offsets, t.Offset)
		depths = append(depths, depth)
		terminal = append(terminal, t.Kind == TokSemicolon || t.Kind == TokRBrace)

		switch t.Kind {
		case TokLParen, TokLBracket, TokLBrace:
			depth++
		case TokRParen, TokRBracket, TokRBrace:
			depth--
		default:
		}
	}

	// Backwards: the nearest terminator at or after each token, at its depth.
	ends := make([]int, len(offsets))
	next := map[int]int{}

	for i := len(offsets) - 1; i >= 0; i-- {
		if terminal[i] {
			next[depths[i]] = offsets[i]
		}

		end, ok := next[depths[i]]
		if !ok {
			end = len(src)
		}

		ends[i] = end
	}

	return &expressionEnds{src: src, offsets: offsets, ends: ends}
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

func collectCollectionTags(text string, offset int, functionAt func(int) string, writes []collectionWrite, cache *expressionWriteCache) []collectionWrite {
	for pos := 0; pos < len(text); {
		idx := nextTagStart(text[pos:])
		if idx < 0 {
			break
		}

		idx += pos
		if strings.HasPrefix(text[idx:], "<!---") {
			pos = skipCFMLComment(text, idx)

			continue
		}

		end := tagEndIndex(text[idx:])
		if end < 0 {
			break
		}

		end += idx
		tag := text[idx : end+1]
		pos = end + 1

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
	}

	return writes
}

// Only whole expressions can supply an element contract.
func (pr *ParseResult) collectionValueSource(expression, function string, locals map[string]bool) ReturnSource {
	if !wholeCollectionValue(expression) {
		return ReturnSource{}
	}

	if root, methods := factoryReturnChain(expression); root != "" && pr.resolverSet != nil {
		if comp := pr.resolverSet.Resolve(root); comp != "" {
			return ReturnSource{Component: comp, Methods: methods}
		}
	}

	tp := &tagParser{fileURI: string(pr.URI), inFunc: function, resolvers: pr.Resolvers, resolverSet: pr.resolverSet, funcs: pr.Funcs}
	tp.checkSetRHSStr(expression, "$element", 0)

	refs := tp.componentRefs
	if function != "" {
		refs = tp.funcRefs[function]
	}

	if len(refs) > 0 {
		return ReturnSource{Component: refs[0].Component, Methods: refs[0].ChainRest}
	}

	for i := range tp.pendingCalls {
		c := &tp.pendingCalls[i]
		if c.alias {
			// A plain name is read by the caller from the refs it names.
			continue
		}

		if c.baseVar != "" {
			ref := firstRefIn(pr.funcRefsMap[function], c.baseVar, c.baseScope)
			if ref == nil && c.baseScope == RefAny && locals[function+"\t"+strings.ToLower(c.baseVar)] {
				return ReturnSource{}
			}

			if ref == nil {
				ref = firstRefIn(pr.ComponentRefs, c.baseVar, c.baseScope)
			}

			if ref != nil {
				return ReturnSource{Component: pr.settledComponent(ref), Methods: append([]string{collectionCallHop(c)}, c.rest...)}
			}
		} else {
			for i := range pr.Funcs {
				f := &pr.Funcs[i]
				if strings.EqualFold(f.Name, c.funcName) {
					return ReturnSource{Component: pr.URI.Path(), Methods: append([]string{collectionCallHop(c)}, c.rest...)}
				}
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

// expressionWriteCache keeps what scanExpressionWrites read from each <cfset>
// body, from one signature pass to the next. A reparse reads every <cfset> in
// the file again, and an edit changes one of them: on a 65,000-line tag
// component the rest were a tenth of the reparse. Only the bodies the last
// pass read are kept, so the cache is never larger than the file's <cfset>s.
type expressionWriteCache struct {
	prev, next map[string]cachedWrites
}

// cachedWrites is a body's writes with its text, which is the map key's own
// copy: a body carried over to the next pass is stored under that copy rather
// than under the slice of the document the pass found it in, so the cache
// keeps no document text of its own.
type cachedWrites struct {
	key    string
	writes []expressionWrite
}

// begin starts a pass: what the last one read is kept for this one to take.
func (c *expressionWriteCache) begin() {
	c.prev, c.next = c.next, map[string]cachedWrites{}
}

// scan is scanExpressionWrites, answered from the last pass when it read the
// same text. A nil cache scans.
func (c *expressionWriteCache) scan(expression string) []expressionWrite {
	if c == nil {
		return scanExpressionWrites(expression)
	}

	if e, ok := c.next[expression]; ok {
		return e.writes
	}

	e, ok := c.prev[expression]
	if !ok {
		// Read from a copy, so the strings kept refer to it and not to the
		// whole of this pass's content.
		e.key = strings.Clone(expression)
		e.writes = scanExpressionWrites(e.key)
	}

	c.next[e.key] = e

	return e.writes
}

// hasTagRegion reports whether the file has a tag region.
func (pr *ParseResult) hasTagRegion() bool {
	for i := range pr.Regions {
		if pr.Regions[i].Kind == RegionTag {
			return true
		}
	}

	return false
}
