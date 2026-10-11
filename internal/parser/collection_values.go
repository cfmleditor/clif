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
	for _, region := range pr.Regions {
		if region.Kind == RegionTag {
			// Tag escapes need their own invocation/output ownership analysis.
			for _, n := range nodes {
				n.escaped = true
			}

			return
		}

		if region.Kind == RegionScript {
			pr.collectionCallEscapes(region.Text, region.Offset, key, nodes)
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

		pr.markCollectionArgumentEscape(write.expression, write.function, key, nodes)
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

func (pr *ParseResult) collectionCallEscapes(text string, offset int, key func(string, string) string, nodes map[string]*collectionNode) {
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
				pr.markCollectionArgumentEscape(embedded, pr.collectionFunctionAt(offset+t.Offset), key, nodes)
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
		if base, _, ok := strings.CutLast(callee, "."); ok {
			if n := nodes[key(base, function)]; n != nil {
				n.escaped = true
			}
		}

		arguments := collectionArguments(&cursor)
		pr.markCollectionArgumentEscape(arguments, function, key, nodes)
	}
}

func (pr *ParseResult) markCollectionArgumentEscape(expression, function string, key func(string, string) string, nodes map[string]*collectionNode) {
	pr.markCollectionArgumentEscapeDepth(expression, function, key, nodes, 0)
}

func (pr *ParseResult) markCollectionArgumentEscapeDepth(expression, function string, key func(string, string) string, nodes map[string]*collectionNode, depth int) {
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
				pr.markCollectionArgumentEscapeDepth(embedded, function, key, nodes, depth+1)
			}
		}

		if t.Kind != TokIdent || before == TokDot {
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
