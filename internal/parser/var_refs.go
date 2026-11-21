package parser

import (
	"slices"
	"strings"
)

// VarOccurrence is a byte range [Start, End) in a document naming a variable.
type VarOccurrence struct {
	Start int
	End   int
}

// FindVariableOccurrences finds every occurrence of variable name belonging to
// scope in content.
//
// When funcSpan is non-nil (for function-scoped bindings such as local or
// arguments), the search stays within that function. When funcSpan is nil (for
// file/page-scoped bindings such as variables, this, url, etc.), the whole
// document is searched while skipping bare occurrences inside shadowSpans (any
// function that declares a local or argument of the same name).
func FindVariableOccurrences(content, name string, scope Scope, funcSpan *FunctionSpan, allSpans, shadowSpans []FunctionSpan) []VarOccurrence {
	if name == "" || content == "" {
		return nil
	}

	w := &varRefFinder{
		content:     content,
		name:        name,
		scope:       scope,
		shadowSpans: shadowSpans,
		outputSpans: OutputSpans(content),
	}

	if funcSpan != nil {
		w.nested = nestedRanges(allSpans, funcSpan.Start, funcSpan.End)
		w.walkFunction(funcSpan)
	} else {
		w.walkDocument()
	}

	if len(w.occs) == 0 {
		return nil
	}

	slices.SortFunc(w.occs, func(a, b VarOccurrence) int {
		if a.Start != b.Start {
			return a.Start - b.Start
		}

		return a.End - b.End
	})

	return slices.Compact(w.occs)
}

type varRefFinder struct {
	content     string
	name        string
	scope       Scope
	nested      [][2]int
	shadowSpans []FunctionSpan
	outputSpans [][2]int
	occs        []VarOccurrence
}

func (f *varRefFinder) walkFunction(fn *FunctionSpan) {
	if fn.Kind != FunctionTag {
		if f.scope == ScopeArguments && fn.Params >= 0 && fn.ParamsEnd > fn.Params {
			for _, p := range parseParams(fn.ParamsText(f.content)) {
				if strings.EqualFold(p.name, f.name) {
					start := fn.Params + p.nameAt
					f.occs = append(f.occs, VarOccurrence{Start: start, End: start + len(p.name)})
				}
			}
		}

		if fn.Body >= 0 && fn.BodyEnd > fn.Body {
			f.walkScript(f.content[fn.Body:fn.BodyEnd], fn.Body, false)
		}

		return
	}

	// Tag function: walk each classified region clipped to [fn.Start, fn.End).
	for _, r := range ClassifyRegions(f.content) {
		rStart := max(r.Offset, fn.Start)
		rEnd := min(r.Offset+len(r.Text), fn.End)

		if rStart >= rEnd {
			continue
		}

		sub := f.content[rStart:rEnd]

		switch r.Kind {
		case RegionScript:
			f.walkScript(sub, rStart, false)
		case RegionTag:
			f.walkTagRegion(sub, rStart)
		case RegionSkip:
		}
	}
}

func (f *varRefFinder) walkDocument() {
	for _, r := range ClassifyRegions(f.content) {
		switch r.Kind {
		case RegionScript:
			f.walkScript(r.Text, r.Offset, false)
		case RegionTag:
			f.walkTagRegion(r.Text, r.Offset)
		case RegionSkip:
		}
	}
}

func (f *varRefFinder) acceptsBare() bool {
	switch f.scope {
	case ScopeLocal, ScopeArguments, ScopeVariables:
		return true
	default:
		return false
	}
}

func (f *varRefFinder) matchesPrefix(prefix string) bool {
	sc, ok := ScopeForPrefix(prefix)

	return ok && sc == f.scope
}

func (f *varRefFinder) inNested(off int) bool {
	for _, r := range f.nested {
		if off >= r[0] && off < r[1] {
			return true
		}
	}

	return false
}

func (f *varRefFinder) isShadowedBare(off int) bool {
	for i := range f.shadowSpans {
		sp := &f.shadowSpans[i]
		if off >= sp.Start && off < sp.End {
			return true
		}
	}

	return false
}

func (f *varRefFinder) inOutput(off int) bool {
	for _, span := range f.outputSpans {
		if off < span[0] {
			return false
		}

		if off < span[1] {
			return true
		}
	}

	return false
}

func (f *varRefFinder) walkScript(text string, base int, expr bool) {
	toks := scriptTokens(text)
	st := bracketState{types: map[int]bool{}, expr: expr}

	// Collect closures/arrows that shadow f.name via their own params or var decls.
	var shadowedClosures [][2]int

	for i, t := range toks {
		if f.inNested(base + t.Offset) {
			continue
		}

		switch t.Kind {
		case TokLParen, TokLBracket, TokLBrace:
			st.open(toks, i, expr)
		case TokRParen, TokRBracket, TokRBrace:
			st.shut(t.Kind)
		case TokQuestion:
			if top := st.top(); top != nil {
				top.ternary++
			}
		case TokColon:
			if top := st.top(); top != nil && top.ternary > 0 {
				top.ternary--
			}
		case TokComma:
			if top := st.top(); top != nil {
				top.inDefault = false
			}
		case TokEquals:
			if i+1 < len(toks) && toks[i+1].Kind == TokGT && toks[i+1].Offset == t.Offset+1 {
				if i+2 < len(toks) && toks[i+2].Kind == TokLBrace {
					end := matching(toks, i+2, TokLBrace, TokRBrace)
					if end < 0 {
						end = len(toks)
					}

					st.arrows = append(st.arrows, [2]int{i + 2, end})
					if arrowShadowsName(toks, i, end, f.name) {
						shadowedClosures = append(shadowedClosures, [2]int{i + 2, end})
					}
				}
			}

			if top := st.top(); top != nil && top.kind == rnParams && !equalsPair(toks, i) {
				top.inDefault = true
			}
		case TokString:
			f.walkString(t, base, shadowedClosures, i)
		case TokIdent:
			f.walkScriptWord(toks, i, &st, base, &shadowedClosures)
		default:
		}
	}
}

func (f *varRefFinder) walkScriptWord(toks []Token, i int, st *bracketState, base int, shadowedClosures *[][2]int) {
	t := toks[i]
	lower := strings.ToLower(t.Value)

	var prev, next Token
	if i > 0 {
		prev = toks[i-1]
	}

	if i+1 < len(toks) {
		next = toks[i+1]
	}

	switch lower {
	case "function":
		if prev.Kind != TokDot {
			st.pendFunc = true

			if span, ok := closureShadowSpan(toks, i, f.name); ok {
				*shadowedClosures = append(*shadowedClosures, span)
			}
		}
	case "var":
		if prev.Kind != TokDot {
			markVarTypes(toks, i, st)
		}
	case "catch":
		markCatchTypes(toks, i, st)
	}

	if st.types[i] || !strings.EqualFold(t.Value, f.name) {
		return
	}

	// Skip nested function names (`function foo(`) and closure parameter names.
	if prev.Kind == TokIdent && strings.EqualFold(prev.Value, "function") && next.Kind == TokLParen {
		return
	}

	if top := st.top(); top != nil && top.kind == rnParams && !top.inDefault {
		return
	}

	if prev.Kind == TokDot {
		if i >= 2 && toks[i-2].Kind == TokIdent && (i < 3 || toks[i-3].Kind != TokDot) && f.matchesPrefix(toks[i-2].Value) {
			// Inside a closure or arrow function, `local.x` belongs to the closure.
			if f.scope == ScopeLocal && (st.fnDepth > 0 || st.inArrow(i)) {
				return
			}

			f.occs = append(f.occs, VarOccurrence{Start: base + t.Offset, End: base + t.Offset + len(t.Value)})
		}

		return
	}

	if !f.acceptsBare() || inTokenSpans(*shadowedClosures, i) || f.isShadowedBare(base+t.Offset) {
		return
	}

	// Skip function calls `name(` unless preceded by `var` (unlikely) or `.` (handled above).
	if next.Kind == TokLParen && !declaresOrCreates(prev) {
		return
	}

	if classifyWord(toks, i, st) == wordOther {
		return
	}

	f.occs = append(f.occs, VarOccurrence{Start: base + t.Offset, End: base + t.Offset + len(t.Value)})
}

func (f *varRefFinder) walkString(t Token, base int, shadowed [][2]int, tokIdx int) {
	if len(t.Value) < 2 || inTokenSpans(shadowed, tokIdx) {
		return
	}

	inner := stringContent(t.Value)
	for _, seg := range hashSegments(inner) {
		f.walkScript(inner[seg[0]:seg[1]], base+t.Offset+1+seg[0], true)
	}
}

func markVarTypes(toks []Token, i int, st *bracketState) {
	last := -1

	for j := i + 1; j < len(toks); j++ {
		t := toks[j]
		if t.Kind == TokDot {
			continue
		}

		if t.Kind != TokIdent || strings.EqualFold(t.Value, "in") {
			break
		}

		if last >= 0 && t.Line != toks[last].Line {
			break
		}

		last = j
	}

	for j := i + 1; j < last; j++ {
		st.types[j] = true
	}
}

func markCatchTypes(toks []Token, i int, st *bracketState) {
	last := -1

	for j := i + 1; j < len(toks) && toks[j].Kind != TokRParen; j++ {
		if toks[j].Kind == TokIdent {
			last = j
		}
	}

	for j := i + 1; j < last; j++ {
		st.types[j] = true
	}
}

func inTokenSpans(spans [][2]int, idx int) bool {
	for _, sp := range spans {
		if idx > sp[0] && idx < sp[1] {
			return true
		}
	}

	return false
}

func closureShadowSpan(toks []Token, fnIdx int, name string) ([2]int, bool) {
	if fnIdx+1 >= len(toks) || toks[fnIdx+1].Kind != TokLParen {
		return [2]int{}, false
	}

	rparen := matching(toks, fnIdx+1, TokLParen, TokRParen)
	if rparen < 0 {
		return [2]int{}, false
	}

	open := rparen + 1
	for open < len(toks) && toks[open].Kind != TokLBrace && toks[open].Kind != TokSemicolon && toks[open].Kind != TokRParen {
		open++
	}

	if open >= len(toks) || toks[open].Kind != TokLBrace {
		return [2]int{}, false
	}

	end := matching(toks, open, TokLBrace, TokRBrace)
	if end < 0 {
		return [2]int{}, false
	}

	// Check parameters or `var name` inside closure body.
	for j := fnIdx + 2; j < rparen; j++ {
		if toks[j].Kind == TokIdent && strings.EqualFold(toks[j].Value, name) {
			return [2]int{open, end}, true
		}
	}

	if bodyDeclaresVar(toks, open+1, end, name) {
		return [2]int{open, end}, true
	}

	return [2]int{}, false
}

func arrowShadowsName(toks []Token, arrowIdx, bodyEnd int, name string) bool {
	if arrowIdx > 0 {
		lo := arrowIdx - 1
		if toks[lo].Kind == TokRParen {
			if open := matchingBack(toks, lo, TokLParen, TokRParen); open >= 0 {
				lo = open
			}
		}

		for _, t := range toks[lo:arrowIdx] {
			if t.Kind == TokIdent && strings.EqualFold(t.Value, name) {
				return true
			}
		}
	}

	return bodyDeclaresVar(toks, arrowIdx+3, bodyEnd, name)
}

func bodyDeclaresVar(toks []Token, from, to int, name string) bool {
	for j := from; j+1 < to; j++ {
		if toks[j].Kind == TokIdent && strings.EqualFold(toks[j].Value, "var") {
			if j > 0 && toks[j-1].Kind == TokDot {
				continue
			}

			last := -1

			for k := j + 1; k < to; k++ {
				if toks[k].Kind == TokDot {
					continue
				}

				if toks[k].Kind != TokIdent || strings.EqualFold(toks[k].Value, "in") {
					break
				}

				if last >= 0 && toks[k].Line != toks[last].Line {
					break
				}

				last = k
			}

			if last >= 0 && strings.EqualFold(toks[last].Value, name) {
				return true
			}
		}
	}

	return false
}

func (f *varRefFinder) walkTagRegion(text string, base int) {
	ends := newTagEnds(text)

	for pos := 0; pos < len(text); {
		rel := strings.IndexByte(text[pos:], '<')
		if rel < 0 {
			f.walkOutputText(text[pos:], base+pos)

			break
		}

		tagStart := pos + rel
		if tagStart > pos {
			f.walkOutputText(text[pos:tagStart], base+pos)
		}

		if strings.HasPrefix(text[tagStart:], "<!---") {
			pos = SkipCFMLComment(text, tagStart)

			continue
		}

		endRel := ends.at(tagStart)
		if endRel < 0 {
			break
		}

		tagEnd := tagStart + endRel + 1
		tag := text[tagStart:tagEnd]
		f.walkTag(tag, base+tagStart)

		pos = tagEnd
	}
}

func (f *varRefFinder) walkOutputText(text string, base int) {
	if !strings.ContainsRune(text, '#') {
		return
	}

	for _, seg := range hashSegments(text) {
		if f.inOutput(base + seg[0]) {
			f.walkScript(text[seg[0]:seg[1]], base+seg[0], true)
		}
	}
}

func (f *varRefFinder) walkTag(tag string, base int) {
	if len(tag) < 3 || tag[1] == '/' || tag[1] == '!' {
		return
	}

	name := extractIdent(tag[1:])
	if name == "" {
		return
	}

	lower := strings.ToLower(name)
	if !strings.HasPrefix(lower, "cf") {
		if f.inOutput(base) {
			f.walkOutputText(tag, base)
		}

		return
	}

	switch lower {
	case "cfset", "cfif", "cfelseif", "cfreturn":
		bodyStart := 1 + len(name)
		bodyEnd := len(tag) - 1

		if bodyEnd > bodyStart && tag[bodyEnd-1] == '/' {
			bodyEnd--
		}

		if bodyEnd > bodyStart {
			f.walkScript(tag[bodyStart:bodyEnd], base+bodyStart, lower != "cfset")
		}

		return
	}

	f.walkTagAttrs(lower, tag, base)
}

func (f *varRefFinder) walkTagAttrs(tagLower, tag string, base int) {
	sc := newAttrScanner(tag, cfAttrs)

	for {
		a, ok := sc.next()
		if !ok {
			break
		}

		if a.valStart < 0 || a.valEnd < a.valStart {
			continue
		}

		attrName := strings.ToLower(tag[a.nameStart:a.nameEnd])
		valStart := base + a.valStart
		valText := tag[a.valStart:a.valEnd]

		if isVarBindingAttr(tagLower, attrName, f.scope) {
			f.matchAttrBinding(valText, valStart)
		}

		if strings.ContainsRune(valText, '#') {
			for _, seg := range hashSegments(valText) {
				f.walkScript(valText[seg[0]:seg[1]], valStart+seg[0], true)
			}
		}
	}
}

func isVarBindingAttr(tagLower, attrLower string, scope Scope) bool {
	switch tagLower {
	case "cfargument":
		return attrLower == "name" && scope == ScopeArguments
	case "cfparam", "cfquery", "cfobject", "cfstoredproc", "cfhttp", "cffile", "cfdirectory", "cfsearch":
		return attrLower == "name" || attrLower == "result" || attrLower == "variable"
	case "cfsavecontent", "cfxml", "cfexecute":
		return attrLower == "variable"
	case "cfloop":
		return attrLower == "index" || attrLower == "item"
	case "cfinvoke":
		return attrLower == "returnvariable"
	default:
		return false
	}
}

func (f *varRefFinder) matchAttrBinding(val string, valStart int) {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" || strings.ContainsRune(trimmed, '#') {
		return
	}

	lead := strings.Index(val, trimmed)
	if dot := strings.IndexByte(trimmed, '.'); dot >= 0 {
		prefix := trimmed[:dot]
		rest := trimmed[dot+1:]

		if strings.IndexByte(rest, '.') < 0 && f.matchesPrefix(prefix) && strings.EqualFold(rest, f.name) {
			start := valStart + lead + dot + 1
			f.occs = append(f.occs, VarOccurrence{Start: start, End: start + len(rest)})
		}

		return
	}

	if f.acceptsBare() && !f.isShadowedBare(valStart+lead) && strings.EqualFold(trimmed, f.name) {
		start := valStart + lead
		f.occs = append(f.occs, VarOccurrence{Start: start, End: start + len(trimmed)})
	}
}
