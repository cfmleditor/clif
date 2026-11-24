package parser

import (
	"maps"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/log"
)

// extractSignatures does a shallow parse: function names/args, component refs, scopes.
// tagFunctionScopes are the scopes of the tag functions of the whole file, and
// whether it has a tag region at all.
//
// Pre-computed from the whole file (not per region) so a function whose body is
// interrupted by a nested <cfscript> block (which splits the file into separate
// Tag/Script/Tag... regions) still gets a correct end line and in-function
// tracking in every region it spans, not just the region containing its
// opening <cffunction> tag.
func (pr *ParseResult) tagFunctionScopes() (scopes []FuncScope, hasTagRegion bool) {
	if ctx := pr.bodies.context; ctx != nil {
		return ctx.walkTagScopes()
	}

	for _, r := range pr.Regions {
		if r.Kind == RegionTag {
			return findTagFuncScopesIdx(pr.Content, 0, pr.contentLineIdx, pr.Regions), true
		}
	}

	return nil, false
}

// mergeRegions parses each region of the file into the result, in file order,
// and returns the calls whose receivers the later passes still have to type.
func (pr *ParseResult) mergeRegions(tagScopes []FuncScope) []pendingCall {
	var allPendingCalls []pendingCall

	// The locals of a tag function a region split, carried to the regions
	// after the split.
	var open openLocals

	// The tag regions share one output state, in file order, from the top.
	pr.output = outputState{}
	pr.collectionTags, pr.collectionTagsOf = nil, pr.Content
	pr.bodies.vars, pr.bodies.outputAt = pr.bodies.vars[:0], nil

	regions := pr.Regions
	if body := pr.bodies.only; body != nil {
		regions = pr.bodyRegions(body)
		pr.output = pr.outputAtBody(regions)
	}

	for _, r := range regions {
		// A RegionSkip is a literal <script> block, left out of the script
		// regions so its JavaScript is never fed to the CFScript scanner. Its
		// `#...#` spans are CFML all the same — a <script> body inside
		// <cfoutput> is where a page writes `var id = "#o.getContentID()#";` —
		// and dropping the region dropped those with it.
		//
		// It goes to the tag parser below, which needs no mode of its own:
		// findScriptSkipSpans only makes a span of a block that holds no `<cf`
		// tag at all, so the walk can find nothing in one *but* its
		// interpolation. A flag restricting it to the spans measured
		// identically over the corpus and against tag-shaped text inside a JS
		// string, which is the case it was written for — and which is not a
		// skip region, because it contains `<cf`.
		if r.Kind == RegionSkip && !pr.walksCalls() {
			continue
		}

		// The tag functions' scopes and spans come from tagScopes, below.
		if pr.scopesOnly && r.Kind != RegionScript {
			continue
		}

		switch {
		case r.Kind == RegionScript:
			allPendingCalls = append(allPendingCalls, pr.mergeScriptRegion(&r, tagScopes, &open)...)
		case r.Island:
			// A tag island's code is the script function's around it, whose
			// scope the script region before it has just filed.
			allPendingCalls = append(allPendingCalls, pr.mergeTagRegion(&r, slices.Concat(tagScopes, pr.Scopes), &open)...)
		default:
			allPendingCalls = append(allPendingCalls, pr.mergeTagRegion(&r, tagScopes, &open)...)
		}
	}

	return allPendingCalls
}

// addTagScopes adds the tag functions' scopes the regions did not record. They
// are keyed by start line rather than found by a scan of pr.Scopes per
// candidate: both lists are one entry per function, so the pair was quadratic
// in a component's method count.
func (pr *ParseResult) addTagScopes(tagScopes []FuncScope) {
	seen := make(map[int]struct{}, len(pr.Scopes))
	for _, existing := range pr.Scopes {
		seen[existing.Start] = struct{}{}
	}

	for _, sc := range tagScopes {
		// A body's walk keeps the functions on its lines alone.
		if body := pr.bodies.only; body != nil && (sc.Start < body.Start || sc.End > body.End) {
			continue
		}

		// Every tag function's span is kept, whichever walk filed its scope.
		pr.keepSpan(&sc)

		if _, dup := seen[sc.Start]; dup {
			continue
		}

		seen[sc.Start] = struct{}{}

		pr.Scopes = append(pr.Scopes, sc)
	}
}

// settleSignatures is the second half of a full parse, once every region is in:
// the expression mappings and service properties, the accessors and delegates
// properties generate, the member writes and factory and collection returns,
// and the calls left untyped.
func (pr *ParseResult) settleSignatures(allPendingCalls []pendingCall) {
	pr.applyExpressionMappings()
	pr.applyServiceProperties()
	pr.generatePropertyAccessors()
	pr.collectDelegates()
	pr.appendResolverRefs()
	// Both passes read the same writes, and nothing between them changes
	// the regions or scopes those are read from.
	pr.writesMemo = &writesMemo{}
	hasBindings := pr.hasMemberBinding(pr.Content)
	pr.applyMemberBindings(hasBindings)
	pr.applyFactoryReturnCalls(allPendingCalls)
	pr.applyCollectionReturns(allPendingCalls)
	pr.writesMemo.release()
	pr.writesMemo = nil
	pr.resolvePendingCalls(allPendingCalls)
	pr.flow = nil
	pr.applyChainedReturnLookup()

	if hasBindings {
		pr.memberSnapshot = &ParseResult{funcRefsMap: maps.Clone(pr.funcRefsMap)}
		pr.memberSnapshotContent = pr.Content
	}
}

func (pr *ParseResult) extractSignatures() {
	defer func() {
		if r := recover(); r != nil {
			log.Recovered(pr.log, "parse panic in extractSignatures", r, "uri", string(pr.URI))
		}
	}()

	pr.startRefs()

	if pr.fileRefsOnly {
		pr.refsOnlyRegions()
		pr.endRefs()

		return
	}

	if ctx := pr.bodies.context; ctx != nil {
		// A body's walk takes the metadata its file's parse read.
		pr.managedSetterLookup = ctx.managedSetterLookup
	} else {
		pr.prepareManagedSetterLookup()
	}

	if pr.FuncLookup != nil {
		lookup := pr.FuncLookup
		pr.FuncLookup = memoFuncLookup(lookup)

		defer func() { pr.FuncLookup = lookup }()
	}

	if !pr.shallow {
		pr.flow = newFlowBlocks(0)
	}

	pr.spans = &functionSpans{of: pr.Content}
	defer pr.spans.sort()

	tagScopes, hasTagRegion := pr.tagFunctionScopes()

	allPendingCalls := pr.mergeRegions(tagScopes)
	pr.endRefs()

	// Detect tag-based function scopes from full content (handles functions
	// that span region boundaries, e.g. containing <cfscript> blocks).
	//
	// tagScopes and hasTagRegion are the ones computed at the top of this
	// function. findTagFuncScopes used to run here as well, over the same
	// pr.Content, under a guard testing the same condition — 18% of everything a
	// tag parse allocated, spent scanning the file a second time for an answer
	// already in hand.
	if hasTagRegion {
		pr.addTagScopes(tagScopes)
	}

	// Sort scopes by start line to match function order.
	sortScopes(pr.Scopes)

	if pr.keepVars {
		pr.allVars, pr.allVarsDone, pr.foundVars = varDefs(pr.foundVars, pr.Scopes), true, nil
	}

	pr.bodies.walked(pr)

	if pr.bodies.only == nil {
		pr.keepSplit(tagScopes, hasTagRegion)
	}

	if pr.extractCalls {
		pr.fillCallers()
	}

	pr.markStructReturns(allPendingCalls)
	pr.discardPrimitiveReturnComponents()

	// Generate synthetic accessor functions for properties (skip if explicit function exists).
	if pr.shallow {
		// A shallow parse cannot afford FuncLookup, and a ref typed by the
		// first call of a longer chain is wrong, so such a ref is dynamic.
		dropChainRest(pr, dynamicIfTyped)

		return
	}

	if pr.bodies.only != nil {
		pr.settleBody(allPendingCalls)

		return
	}

	pr.settleSignatures(allPendingCalls)
	pr.bodies.settled = true
}

// markStructReturns flags each function one of whose returns is a struct
// literal or structNew(), from the return expressions both parsers record.
func (pr *ParseResult) markStructReturns(calls []pendingCall) {
	var byKey map[fnKey][]int

	for i := range calls {
		c := &calls[i]
		if !c.returnExpr || !c.funcKey.in {
			continue
		}

		expr := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(c.varName), "/"))
		if !strings.HasPrefix(expr, "{") && !lex.HasPrefixFold(expr, "structnew(") {
			continue
		}

		if byKey == nil {
			byKey = make(map[fnKey][]int, len(pr.Funcs))
			scopeAt := scopeIndex(pr.Scopes, true)

			for j := range pr.Funcs {
				if k := scopeAt(int(pr.Funcs[j].Line)); k >= 0 {
					scope := pr.Scopes[k]
					key := funcKey(scope.Start, scope.End)
					byKey[key] = append(byKey[key], j)
				}
			}
		}

		for _, j := range byKey[c.funcKey] {
			pr.Funcs[j].returnsStruct = true
		}
	}
}

// discardPrimitiveReturnComponents keeps an inference from overriding a
// declared primitive return type. Without this, a string method's local
// assigned from a receiver call can make the method and its callers return
// the receiver's component. Generic any/component/struct declarations still
// infer.
func (pr *ParseResult) discardPrimitiveReturnComponents() {
	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		f.ReturnComponent = pr.componentReturnFor(f, f.ReturnComponent)
	}
}

// componentReturnFor also guards returns settled after resolver refs become
// available. A runtime-created component remains dynamic; it must not acquire
// a concrete component type.
//
// `struct` admits a component instance, because to the engine a component is
// a struct, so it keeps the inference as `any` does. tassweb leans on that
// throughout: kernel2's GetPageTools, GetLockBroker and getController all
// declare struct and hand back components, and treating struct as primitive
// left every caller of getTASSWebUser() holding nothing.
func (pr *ParseResult) componentReturnFor(f *FunctionDef, comp string) string {
	if strings.EqualFold(f.ReturnType, "struct") {
		if pr.replaceExpressions(comp) == "$any" {
			return "$any"
		}

		// One path returns a plain struct: Wheels' mapper writes
		// `if (flag) { return {}; } return this;`.
		if f.returnsStruct {
			return ""
		}

		return comp
	}

	if f.ReturnType != "" && !strings.EqualFold(f.ReturnType, "any") && !looksLikeCFCType(f.ReturnType) {
		if pr.replaceExpressions(comp) == "$any" {
			return "$any"
		}

		return ""
	}

	return comp
}

// openLocals are the names declared local — `var`, `local.` — in the tag
// function still open where a region ends, keyed by the function's file-level
// funcKey. A <cfscript> block inside a <cffunction> splits the file into
// regions, and the parser of each region starts with only the function's
// arguments as its locals: `<cfset var conn = "">` above the block, and
// `conn = uri.openConnection();` inside it read as an unscoped assignment,
// which is a variables-scope one, so the variable was filed for the whole
// component.
type openLocals struct {
	key   fnKey
	names []string
}

// namesFor is the names declared in the function keyed key, if they are the
// ones held.
func (o *openLocals) namesFor(key fnKey) []string {
	if o.key != key {
		return nil
	}

	return o.names
}

// mergeScriptRegion parses a script region and merges what it found into pr,
// returning its pending calls for resolvePendingCalls.
func (pr *ParseResult) mergeScriptRegion(r *Region, tagScopes []FuncScope, open *openLocals) []pendingCall {
	sp := newScriptParser(r.Text, string(pr.URI), r.StartLine, pr.Resolvers).asCFScript()
	sp.baseOffset = r.Offset
	sp.keepVars = pr.keepVars
	sp.resolverSet = pr.resolverSet
	sp.extractCalls = pr.walksCalls()
	sp.setCallNames(pr.callNames())
	sp.builtinReturnLookup = pr.BuiltinReturnLookup
	sp.argumentTypes = pr.argumentTypeLookup()

	if pr.flow != nil {
		sp.flow = pr.flow.region(r.Offset, 0)
	}

	// If this <cfscript> region sits inside a tag <cffunction> body
	// (nested script island — ClassifyRegions splits the file there),
	// seed inFunc so refs/pending calls in it route to that function's
	// scope instead of global. scriptParser bakes baseLine into its own
	// keys, so the seed must be the function's absolute funcKey — unlike
	// the tag-region continuation seed below, which stays region-relative.
	seed := -1

	if s, ok := pr.regionTagScope(tagScopes, r.StartLine); ok {
		sp.inFunc = funcKey(s.Start, s.End)
		seed, sp.funcs = pr.seedRegionFunction(s.Start, s.Name)
		sp.localVarSet = &localNames{}

		for _, arg := range pr.argsOfFuncAt(s.Start) {
			sp.localVarSet.add(arg.Name)
		}

		for _, name := range open.namesFor(sp.inFunc) {
			sp.localVarSet.add(name)
		}
	}

	key := sp.inFunc

	// A body's walk starts where the whole walk had read the file's imports.
	if ctx := pr.bodies.context; ctx != nil && len(ctx.imports) > 0 {
		sp.imports = maps.Clone(ctx.imports)
	}

	refsFrom := pr.refsMark()
	watch := pr.watchScript(sp.sc, r)

	sp.parse()
	finishScript(sp.sc, watch)

	if pr.extractLinks {
		sp.fileLinks(linksOf(pr.refsSince(refsFrom), r.Text, r.Offset, r.StartLine, nil))
	}

	if !pr.scopesOnly && r.Kind == RegionScript {
		pr.bodies.keep(sp.vars, false)
	}

	if pr.keepVars {
		pr.foundVars = append(pr.foundVars, sp.vars)
	}

	// What the block declared local stays local in the regions after it.
	if key.in {
		names := slices.Clone(open.namesFor(key))
		if sp.localVarSet != nil {
			names = append(names, sp.localVarSet.names...)
		}

		*open = openLocals{key: key, names: names}
	}

	if seed >= 0 {
		pr.Funcs[seed] = sp.funcs[0]
		sp.funcs = sp.funcs[1:]
	}

	if prefixCols := linePrefixUTF16(pr.Content, r.Offset); prefixCols > 0 {
		startLine := conv.Uint32(r.StartLine)
		for i := range sp.funcs {
			if sp.funcs[i].HasNameRange() && sp.funcs[i].NameLine == startLine {
				sp.funcs[i].NameCol += prefixCols
				sp.funcs[i].NameEndCol += prefixCols
			}
		}

		for i := range sp.properties {
			prop := &sp.properties[i]
			if prop.line == startLine {
				prop.col += prefixCols
			}

			if prop.endLine == startLine {
				prop.endCol += prefixCols
			}

			if prop.nameEndCol > prop.nameCol && prop.nameLine == startLine {
				prop.nameCol += prefixCols
				prop.nameEndCol += prefixCols
			}
		}
	}

	pr.Funcs = appendSlice(pr.Funcs, sp.funcs)
	pr.ComponentRefs = appendSlice(pr.ComponentRefs, sp.componentRefs)

	for i := range sp.scopes {
		pr.keepSpan(&sp.scopes[i])
	}

	pr.Scopes = appendSlice(pr.Scopes, sp.scopes)
	pr.Properties = appendSlice(pr.Properties, sp.properties)

	if len(sp.imports) > 0 {
		pr.imports = maps.Clone(sp.imports)
	}

	// Merge function-scoped refs, links, and calls from script parser
	pr.funcRefsMap = mergeKeyedMap(pr.funcRefsMap, sp.funcRefs)
	pr.Links = appendSlice(pr.Links, sp.links)
	pr.funcLinksMap = mergeKeyedMap(pr.funcLinksMap, sp.funcLinks)
	pr.Calls = appendSlice(pr.Calls, sp.calls)
	pr.funcCallsMap = mergeKeyedMap(pr.funcCallsMap, sp.funcCalls)

	pr.Instantiations = append(pr.Instantiations, sp.instantiationsFound()...)

	pr.dropCallsIndex()

	if sp.extends != "" {
		pr.Extends = sp.extends
	}

	if sp.extras != nil && sp.extras.implements != "" {
		pr.Implements = SplitList(sp.extras.implements)
	}

	pr.Accessors = pr.Accessors || sp.accessors

	if sp.persistent {
		pr.Persistent = true
	}

	return sp.pendingCalls
}

// newRegionTagParser is a tag parser set up to read region r of the file: the
// result's resolvers and lookups, the file's tag function scopes, the output
// state the tag regions share, and the flow blocks at the region's offset.
func (pr *ParseResult) newRegionTagParser(r *Region, tagScopes []FuncScope) *tagParser {
	tp := newTagParser(r.Text, string(pr.URI))
	tp.resolvers = pr.Resolvers
	tp.resolverSet = pr.resolverSet
	tp.extractCalls = pr.walksCalls()
	tp.callNames = pr.callNames()
	tp.builtinReturnLookup = pr.BuiltinReturnLookup
	tp.argumentTypes = pr.argumentTypeLookup()
	tp.baseLine = r.StartLine
	tp.knownScopes = tagScopes
	tp.output, tp.importPrefixes, tp.gated = pr.outputGate()
	tp.srcOffset = r.Offset
	tp.refs = pr.tagReader(r, tp.tagEnds())

	if pr.flow != nil {
		tp.flow = pr.flow.region(r.Offset, conv.Uint32(r.StartLine))
	}

	return tp
}

// mergeTagFindings adds the links, function links, function refs and pending
// calls a region's tag parser found, with their lines and function keys moved
// from the region's to the file's by the region's start line.
func (pr *ParseResult) mergeTagFindings(tp *tagParser, startLine int) {
	// Merge links from tag parser
	if startLine > 0 {
		for i := range tp.links {
			tp.links[i].Line += conv.Uint32(startLine)
		}
	}

	pr.Links = appendSlice(pr.Links, tp.links)

	if startLine == 0 && pr.funcLinksMap == nil {
		pr.funcLinksMap = tp.funcLinks
	} else {
		for k, links := range tp.funcLinks {
			if startLine > 0 {
				k = shiftFuncKey(k, startLine)

				for i := range links {
					links[i].Line += conv.Uint32(startLine)
				}
			}

			pr.funcLinksMap = appendKeyed(pr.funcLinksMap, k, links)
		}
	}

	// Merge function-scoped refs from tag parser
	if startLine == 0 && pr.funcRefsMap == nil {
		pr.funcRefsMap = tp.funcRefs
	} else {
		for k, refs := range tp.funcRefs {
			// Offset the key and ref lines by region start line
			if startLine > 0 {
				k = shiftFuncKey(k, startLine)

				for i := range refs {
					refs[i].Line += conv.Uint32(startLine)
				}
			}

			pr.funcRefsMap = appendKeyed(pr.funcRefsMap, k, refs)
		}
	}

	// Collect pending calls from tag parser (offset lines and funcKey)
	for i := range tp.pendingCalls {
		tp.pendingCalls[i].line += conv.Uint32(startLine)
		if startLine > 0 && tp.pendingCalls[i].funcKey.in {
			tp.pendingCalls[i].funcKey = shiftFuncKey(tp.pendingCalls[i].funcKey, startLine)
		}
	}
}

// mergeTagRegion parses a tag region and merges what it found into pr, with
// its lines and function keys moved from region-relative to file lines,
// returning its pending calls for resolvePendingCalls.
func (pr *ParseResult) mergeTagRegion(r *Region, tagScopes []FuncScope, open *openLocals) []pendingCall {
	tp := pr.newRegionTagParser(r, tagScopes)

	// If this region starts partway through a function whose opening
	// <cffunction> tag was in an earlier region (interrupted by a
	// nested <cfscript> region split), seed inFunc/localVars so refs
	// in this region still route to function scope instead of global.
	seed := -1

	if s, ok := pr.regionTagScope(tagScopes, r.StartLine); ok {
		tp.inFunc = funcKey(s.Start-r.StartLine, s.End-r.StartLine)
		seed, tp.funcs = pr.seedRegionFunction(s.Start, s.Name)
		tp.localVars, tp.localSet = nil, nil

		for _, arg := range pr.argsOfFuncAt(s.Start) {
			tp.markVarLocal(arg.Name)
		}

		for _, name := range open.namesFor(funcKey(s.Start, s.End)) {
			tp.markVarLocal(name)
		}
	}

	seededReturn := seed >= 0 && tp.funcs[0].returnVar != ""

	refsFrom := pr.refsMark()

	tp.parse()

	if pr.extractLinks && tp.refs != nil {
		tp.fileLinks(linksOf(pr.refsSince(refsFrom), r.Text, r.Offset, 0, tp.lineIndex))
	}

	// The walk's declarations, on file lines: the tag parser is done with
	// them, and nothing after this reads them on the region's.
	vars := tp.vars
	for i := range vars {
		vars[i].Line += conv.Uint32(r.StartLine)
	}

	if pr.keepVars {
		pr.foundVars = append(pr.foundVars, vars)
	}

	if r.Kind == RegionTag {
		pr.bodies.keep(vars, true)
	}

	for _, at := range tp.outputAt {
		pr.bodies.noteOutput(at.line+r.StartLine, at.state)
	}

	if seed >= 0 {
		// A return this region found for the function it continues is at a
		// line within the region, like everything else it recorded.
		if !seededReturn && tp.funcs[0].returnVar != "" {
			tp.funcs[0].returnLine += conv.Uint32(r.StartLine)
		}

		pr.Funcs[seed] = tp.funcs[0]
		tp.funcs = tp.funcs[1:]
	}

	// The function still open where the region ends continues in the next.
	if tp.inFunc.in {
		*open = openLocals{key: shiftFuncKey(tp.inFunc, r.StartLine), names: slices.Clone(tp.localVars)}
	}

	prefixCols := linePrefixUTF16(pr.Content, r.Offset)

	for i := range tp.funcs {
		if prefixCols > 0 && tp.funcs[i].HasNameRange() && tp.funcs[i].NameLine == 0 {
			tp.funcs[i].NameCol += prefixCols
			tp.funcs[i].NameEndCol += prefixCols
		}

		tp.funcs[i].Line += conv.Uint32(r.StartLine)
		if tp.funcs[i].HasNameRange() {
			tp.funcs[i].NameLine += conv.Uint32(r.StartLine)
		}

		tp.funcs[i].returnLine += conv.Uint32(r.StartLine)
	}

	for i := range tp.componentRefs {
		tp.componentRefs[i].Line += conv.Uint32(r.StartLine)
	}

	for i := range tp.properties {
		prop := &tp.properties[i]
		if prefixCols > 0 {
			if prop.line == 0 {
				prop.col += prefixCols
			}

			if prop.endLine == 0 {
				prop.endCol += prefixCols
			}

			if prop.nameEndCol > prop.nameCol && prop.nameLine == 0 {
				prop.nameCol += prefixCols
				prop.nameEndCol += prefixCols
			}
		}

		prop.line += conv.Uint32(r.StartLine)
		prop.endLine += conv.Uint32(r.StartLine)

		if prop.nameEndCol > prop.nameCol {
			prop.nameLine += conv.Uint32(r.StartLine)
		}
	}

	pr.Funcs = append(pr.Funcs, tp.funcs...)
	pr.ComponentRefs = append(pr.ComponentRefs, tp.componentRefs...)
	pr.Properties = append(pr.Properties, tp.properties...)
	pr.keepRegionFacts(r, tp)

	pr.mergeTagFindings(tp, r.StartLine)

	pr.mergeTagCalls(tp, r.StartLine)

	pr.dropCallsIndex()

	// Tag scopes are computed from full content after region processing
	// to handle functions spanning region boundaries.
	if tp.extends != "" {
		pr.Extends = tp.extends
	}

	if tp.implements != "" {
		pr.Implements = SplitList(tp.implements)
	}

	pr.Accessors = pr.Accessors || tp.accessors

	if tp.persistent {
		pr.Persistent = true
	}

	return tp.pendingCalls
}

// mergeTagCalls takes the calls and instantiations a tag region's parser
// recorded, its lines moved from the region's to the file's.
func (pr *ParseResult) mergeTagCalls(tp *tagParser, startLine int) {
	if startLine > 0 {
		for i := range tp.calls {
			tp.calls[i].Line += conv.Uint32(startLine)
		}
	}

	pr.Calls = append(pr.Calls, tp.calls...)

	for _, in := range tp.instantiations {
		in.Line += conv.Uint32(startLine)
		pr.Instantiations = append(pr.Instantiations, in)
	}

	if startLine == 0 && pr.funcCallsMap == nil {
		pr.funcCallsMap = tp.funcCalls
	} else {
		for k, calls := range tp.funcCalls {
			if startLine > 0 {
				k = shiftFuncKey(k, startLine)

				for i := range calls {
					calls[i].Line += conv.Uint32(startLine)
				}
			}

			pr.funcCallsMap = appendKeyed(pr.funcCallsMap, k, calls)
		}
	}
}

// enclosingTagScope is the tag function a region starting at startLine opens
// inside of: one whose opening <cffunction> is in an earlier region, the file
// having been split around a nested <cfscript>.
func enclosingTagScope(tagScopes []FuncScope, startLine int) (FuncScope, bool) {
	for _, s := range tagScopes {
		if s.Start < startLine && startLine <= s.End {
			return s, true
		}
	}

	return FuncScope{}, false
}

// argsOfFuncAt is the argument list of the first function declared at line.
func (pr *ParseResult) argsOfFuncAt(line int) []Argument {
	for i := range pr.Funcs {
		if int(pr.Funcs[i].Line) == line {
			return pr.Funcs[i].Arguments
		}
	}

	return nil
}

// parseFnKey is the key String wrote as s, for the collection readers, which
// intern a function's key as text with the names declared in it.
func parseFnKey(s string) fnKey {
	start, end, ok := strings.Cut(s, ":")
	if !ok {
		return noFn
	}

	return funcKey(atoi(start), atoi(end))
}

// shiftFuncKey moves a function key by delta lines.
func shiftFuncKey(k fnKey, delta int) fnKey {
	if !k.in {
		return k
	}

	return funcKey(int(k.start)+delta, int(k.end)+delta)
}

// appendSlice returns src directly when dst is empty, avoiding an extra copy
// for single-region files where src is already an owned slice.
func appendSlice[T any](dst, src []T) []T {
	if len(src) == 0 {
		return dst
	}

	if len(dst) == 0 {
		return src
	}

	return append(dst, src...)
}

// mergeKeyedMap moves src directly into dst when dst is nil, otherwise appends
// each entry through appendKeyed.
func mergeKeyedMap[T any](dst, src map[fnKey][]T) map[fnKey][]T {
	if len(src) == 0 {
		return dst
	}

	if dst == nil {
		return src
	}

	for k, v := range src {
		dst = appendKeyed(dst, k, v)
	}

	return dst
}

// appendKeyed appends v to m[k], making the map on first use, and returns it.
func appendKeyed[T any](m map[fnKey][]T, k fnKey, v []T) map[fnKey][]T {
	if len(v) == 0 {
		return m
	}

	if m == nil {
		m = make(map[fnKey][]T)
	}

	if cur := m[k]; len(cur) == 0 {
		m[k] = v
	} else {
		m[k] = append(cur, v...)
	}

	return m
}
