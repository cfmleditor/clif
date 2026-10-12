package parser

import (
	"sort"
	"strings"
)

func wholeStructValues(f *FunctionDef, expressions []string, function string, key func(string, string) string, nodes map[string]*collectionNode) *CollectionValues {
	var fold foldScratch
	switch string(fold.lowerFold(f.ReturnType)) {
	case "", "any", "struct":
	default:
		return nil
	}

	combined := &collectionNode{}

	for _, expression := range expressions {
		root, element := collectionRead(expression)
		if root == "" || element {
			return nil
		}

		n := nodes[key(root, function)]
		if n == nil || n.invalid || n.escaped || !n.structSeed || n.arraySeed || hasPrefixFold(root, "this.") {
			return nil
		}

		for _, source := range n.sources {
			combined.addSource(source)
		}
	}

	if combined.invalid || len(combined.sources) == 0 {
		return nil
	}

	sources := make([]ReturnSource, 0, len(combined.sources))
	for _, source := range combined.sources {
		sources = append(sources, source)
	}

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Component+"\t"+strings.Join(sources[i].Methods, "\t") < sources[j].Component+"\t"+strings.Join(sources[j].Methods, "\t")
	})

	return &CollectionValues{Sources: sources}
}

// Unknown calls receiving the whole struct can mutate it. This additional
// veto applies to whole-struct contracts; existing indexed-return behavior is
// retained. Whole aliases propagate escapes in both directions.
func (pr *ParseResult) markCollectionEscapes(key func(string, string) string, nodes map[string]*collectionNode) {
	if len(nodes) == 0 {
		return
	}

	names := collectionNames(nodes)

	for _, region := range pr.Regions {
		if region.Kind == RegionTag {
			// Tag escapes need their own invocation/output ownership analysis.
			for _, n := range nodes {
				n.escaped = true
			}

			return
		}

		if region.Kind == RegionScript {
			pr.collectionCallEscapes(region.Text, region.Offset, key, nodes, names)
		}
	}

	for _, write := range pr.collectionWrites() {
		root, element := collectionRead(write.expression)
		if root != "" && !element && !write.element {
			if hasPrefixFold(write.target, "this.") {
				if n := nodes[key(root, write.function)]; n != nil {
					n.escaped = true
				}
			}

			continue // already modeled as a whole alias
		}

		pr.markCollectionArgumentEscape(write.expression, write.function, key, nodes, names)
	}

	queue := []string{}

	for k, n := range nodes {
		if n.escaped {
			queue = append(queue, k)
		}
	}

	for len(queue) > 0 {
		k := queue[0]
		queue = queue[1:]

		for _, alias := range nodes[k].aliases {
			if !nodes[alias].escaped {
				nodes[alias].escaped = true
				queue = append(queue, alias)
			}
		}
	}
}

// collectionNames is every name a node's key holds, lowercased, and the scope
// words a key may begin with. A target whose first name is not among them
// has no node, so the escape scans test this before building its key: on a
// large component that test replaced a lowercased key and a map lookup for
// every identifier in the file, which was most of what a parse cost.
func collectionNames(nodes map[string]*collectionNode) map[string]struct{} {
	names := map[string]struct{}{"variables": {}, "this": {}, "local": {}, "arguments": {}}

	for identity := range nodes {
		if _, rest, ok := strings.Cut(identity, "\t"); ok {
			identity = rest
		}

		for name := range strings.SplitSeq(identity, ".") {
			names[name] = struct{}{}
		}
	}

	return names
}

// mayName reports whether ident could begin a target with a node.
func mayName(names map[string]struct{}, ident string) bool {
	var buf foldScratch

	_, ok := names[string(buf.lowerFold(ident))]

	return ok
}

func (pr *ParseResult) collectionCallEscapes(text string, offset int, key func(string, string) string, nodes map[string]*collectionNode, names map[string]struct{}) {
	sc := NewScanner(text)
	previous := TokEOF

	for {
		t := sc.NextSkipComments()
		before := previous

		previous = t.Kind
		if t.Kind == TokEOF {
			return
		}

		if t.Kind == TokString {
			for _, embedded := range InterpolatedExpressions(t.Value) {
				pr.markCollectionArgumentEscape(embedded, pr.collectionFunctionAt(offset+t.Offset), key, nodes, names)
			}
		}

		if t.Kind != TokIdent || before == TokDot {
			continue
		}

		cursor := *sc
		cursor.Restore(ScannerState{pos: t.Offset, line: t.Line})

		callee, element := collectionTarget(&cursor)
		if callee == "" || element || cursor.NextSkipComments().Kind != TokLParen {
			continue
		}

		function := pr.collectionFunctionAt(offset + t.Offset)
		if base, _, ok := strings.CutLast(callee, "."); ok && mayName(names, t.Value) {
			if n := nodes[key(base, function)]; n != nil {
				n.escaped = true
			}
		}

		arguments := collectionArguments(&cursor)
		pr.markCollectionArgumentEscape(arguments, function, key, nodes, names)
	}
}

func (pr *ParseResult) markCollectionArgumentEscape(expression, function string, key func(string, string) string, nodes map[string]*collectionNode, names map[string]struct{}) {
	pr.markCollectionArgumentEscapeDepth(expression, function, key, nodes, names, 0)
}

func (pr *ParseResult) markCollectionArgumentEscapeDepth(expression, function string, key func(string, string) string, nodes map[string]*collectionNode, names map[string]struct{}, depth int) {
	if depth > 8 {
		for _, n := range nodes {
			n.escaped = true
		}

		return
	}

	sc := NewScanner(expression)
	previous := TokEOF

	for {
		t := sc.NextSkipComments()
		before := previous

		previous = t.Kind
		if t.Kind == TokEOF {
			return
		}

		if t.Kind == TokString {
			for _, embedded := range InterpolatedExpressions(t.Value) {
				pr.markCollectionArgumentEscapeDepth(embedded, function, key, nodes, names, depth+1)
			}
		}

		if t.Kind != TokIdent || before == TokDot || !mayName(names, t.Value) {
			continue
		}

		cursor := *sc
		cursor.Restore(ScannerState{pos: t.Offset, line: t.Line})

		root, indexed := collectionTarget(&cursor)
		if root != "" && !indexed {
			if n := nodes[key(root, function)]; n != nil {
				n.escaped = true
			}
		}
	}
}

func (pr *ParseResult) collectionFunctionAt(offset int) string {
	line := pr.lineAt(offset)

	i := sort.Search(len(pr.Scopes), func(i int) bool { return pr.Scopes[i].Start > line }) - 1
	if i >= 0 && line <= pr.Scopes[i].End {
		return funcKey(pr.Scopes[i].Start, pr.Scopes[i].End)
	}

	return ""
}

func collectionArguments(sc *Scanner) string {
	start := sc.PeekSkipComments().Offset

	depth := 1
	for depth > 0 {
		t := sc.NextSkipComments()
		switch t.Kind {
		case TokEOF:
			return ""
		case TokLParen:
			depth++
		case TokRParen:
			depth--
			if depth == 0 {
				return sc.src[start:t.Offset]
			}
		default:
		}
	}

	return ""
}
