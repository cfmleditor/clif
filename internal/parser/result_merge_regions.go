package parser

import (
	"maps"
	"slices"

	"github.com/cfmleditor/clif/internal/conv"
)

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
