package parser

// The lazy body readers as they were before they became the parse's walk
// (body_walk.go): the oracle TestLazyBodyReadsAreTheParses compares the new
// readers with. Each walked a function body with walkers of its own.

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

func (pr *ParseResult) oldTypeBodyPendingCalls(global []ComponentRef, local map[fnKey][]ComponentRef, calls []pendingCall) ([]ComponentRef, map[fnKey][]ComponentRef) {
	body := &ParseResult{
		URI: pr.URI, FuncLookup: pr.FuncLookup, Resolvers: pr.Resolvers, resolverSet: pr.resolverSet,
		ComponentRefs: global, funcRefsMap: local,
	}

	// The file's refs hold the body's own as well, from the parse or shifted
	// by an edit, so they are read as they are rather than copied with the
	// body's for every function asked about.
	if len(calls) > 0 {
		globals := pr.ComponentRefs
		if len(globals) == 0 {
			globals = global
		}

		body.typePendingCalls(calls, pr.funcReturnsFor(calls), globals)
	}

	body.applyChainedReturnLookup()

	return body.ComponentRefs, body.funcRefsMap
}

func (pr *ParseResult) oldFuncRefsUncached(funcStart, funcEnd int) ([]ComponentRef, []DocumentLink) {
	start, end := pr.lineOffsets(funcStart, funcEnd)
	if start < 0 {
		return nil, nil
	}

	body := pr.Content[start:end]

	var (
		refs  []ComponentRef
		links []DocumentLink
	)

	if pr.oldRegionKindAt(funcStart) == RegionScript {
		refs, links = pr.oldParseScriptBody(body, start, funcStart)
	} else {
		refs, links = pr.oldParseTagBody(body, start, funcStart)
	}

	refs = pr.oldResolveMethodReturnRefs(funcStart, funcEnd, refs)

	// This re-parse never reaches applyChainedReturnLookup, so a chained
	// assignment is typed by its rest here or it keeps its first call's type.
	typ := dynamicIfTyped
	if pr.FuncLookup != nil {
		typ = pr.walkChainRest
	}

	for i := range refs {
		if len(refs[i].ChainRest) > 0 {
			refs[i].Component = strictChainComponent(&refs[i], typ(refs[i].Component, refs[i].ChainRest))
			refs[i].ChainRest = nil
		}
	}

	if pr.hasMemberBinding(body) {
		// A fresh full parse owns its slices and computes member identities once
		// per edited document, rather than repeating whole-file work per function.
		if pr.memberSnapshot == nil || pr.memberSnapshotContent != pr.Content {
			pr.memberSnapshot = ParseWithOptions(pr.URI, pr.Content, &ParseOptions{
				Resolvers: pr.Resolvers, PropertyResolvers: pr.PropertyResolvers,
				FuncLookup: pr.FuncLookup, BeanLookup: pr.BeanLookup,
				SetterLookup: pr.SetterLookup, ConstructorLookup: pr.ConstructorLookup,
				PropertyBeanLookup: pr.PropertyBeanLookup, BuiltinReturnLookup: pr.BuiltinReturnLookup,
				ExpressionMappings: pr.expressionMappings, ServicePropertyResolvers: pr.ServicePropertyResolvers,
			})
			pr.memberSnapshotContent = pr.Content
		}

		refs = slices.Clone(pr.memberSnapshot.funcRefsMap[funcKey(funcStart, funcEnd)])
	}

	return refs, links
}

// regionKindAt is the kind of the region the function starting on line
// funcStart is in: the last region that starts at or before it.
func (pr *ParseResult) oldRegionKindAt(funcStart int) RegionKind {
	kind := RegionScript

	for _, r := range pr.Regions {
		if r.StartLine <= funcStart {
			kind = r.Kind
		}
	}

	return kind
}

// parseScriptBody is the refs and links of a function body written in
// CFScript, parsed on its own: its own and those of any function nested in it.
func (pr *ParseResult) oldParseScriptBody(body string, start, funcStart int) ([]ComponentRef, []DocumentLink) {
	sp := newScriptParser(body, string(pr.URI), funcStart, pr.Resolvers)
	sp.baseOffset = start
	sp.resolverSet = pr.resolverSet
	sp.argumentTypes = pr.argumentTypeLookup()
	sp.imports = pr.imports
	sp.parse()
	sp.fileLinks(oldRegionLinks(body, true, funcStart))
	sp.componentRefs, sp.funcRefs = pr.oldTypeBodyPendingCalls(sp.componentRefs, sp.funcRefs, sp.pendingCalls)

	refs, links := sp.componentRefs, sp.links

	// Include function-scoped refs (nested functions in body)
	for _, r := range sp.funcRefs {
		refs = append(refs, r...)
	}

	for _, l := range sp.funcLinks {
		links = append(links, l...)
	}

	return refs, links
}

// parseTagBody is parseScriptBody for a body written in tags. The tag parser
// counts lines from the body's first, so they are moved back by funcStart.
func (pr *ParseResult) oldParseTagBody(body string, start, funcStart int) ([]ComponentRef, []DocumentLink) {
	tp := newTagParser(body, string(pr.URI))
	tp.srcOffset = start
	tp.resolvers = pr.Resolvers
	tp.resolverSet = pr.resolverSet
	tp.argumentTypes = pr.argumentTypeLookup()
	tp.parse()
	tp.fileLinks(oldRegionLinks(body, false, 0))
	tp.componentRefs, tp.funcRefs = pr.oldTypeBodyPendingCalls(tp.componentRefs, tp.funcRefs, tp.pendingCalls)

	refs, links := tp.componentRefs, tp.links

	for _, r := range tp.funcRefs {
		refs = append(refs, r...)
	}

	for _, l := range tp.funcLinks {
		links = append(links, l...)
	}

	for i := range refs {
		refs[i].Line += conv.Uint32(funcStart)
	}

	for i := range links {
		links[i].Line += conv.Uint32(funcStart)
	}

	return refs, links
}

// resolveMethodReturnRefs scans a function body for x = y.method() assignments
// where y has a known component ref and method declares a return type.
func (pr *ParseResult) oldResolveMethodReturnRefs(funcStart, funcEnd int, existingRefs []ComponentRef) []ComponentRef {
	start, end := pr.lineOffsets(funcStart, funcEnd)
	if start < 0 {
		return existingRefs
	}

	body := pr.Content[start:end]
	lineNum := funcStart + 1

	// Combine global + existing function refs for lookup
	allRefs := make([]ComponentRef, 0, len(pr.ComponentRefs)+len(existingRefs))
	allRefs = append(allRefs, pr.ComponentRefs...)
	allRefs = append(allRefs, existingRefs...)

	for body != "" {
		nl := strings.IndexByte(body, '\n')

		var line string
		if nl < 0 {
			line = body
			body = ""
		} else {
			line = body[:nl]
			body = body[nl+1:]
		}

		// Look for: varName = something.method(
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 0 || (eqIdx+1 < len(line) && line[eqIdx+1] == '=') {
			lineNum++

			continue
		}

		rhs := strings.TrimSpace(line[eqIdx+1:])
		// Find pattern: ident.ident(
		dotIdx := strings.IndexByte(rhs, '.')
		if dotIdx <= 0 {
			lineNum++

			continue
		}

		parenIdx := strings.IndexByte(rhs[dotIdx:], '(')
		if parenIdx < 0 {
			lineNum++

			continue
		}

		baseVar := strings.TrimSpace(rhs[:dotIdx])
		methodName := strings.TrimSpace(rhs[dotIdx+1 : dotIdx+parenIdx])

		if baseVar == "" || methodName == "" || !isIdentifier(baseVar) || !isIdentifier(methodName) {
			lineNum++

			continue
		}

		// Find component for baseVar
		var baseComp string

		for i := range allRefs {
			ref := &allRefs[i]

			if strings.EqualFold(ref.Variable, baseVar) {
				baseComp = ref.Component

				break
			}
		}

		if baseComp == "" {
			lineNum++

			continue
		}

		// Look up the method's return type in the component
		retComp := pr.oldLookupMethodReturn(baseComp, methodName)
		if retComp == "" {
			lineNum++

			continue
		}

		// Extract LHS variable name
		lhs := strings.TrimSpace(line[:eqIdx])

		varName := lhs
		if _, after, ok := strings.CutLast(lhs, " "); ok {
			varName = strings.TrimSpace(after)
		}

		if dotIdx := strings.LastIndexByte(varName, '.'); dotIdx >= 0 {
			varName = varName[dotIdx+1:]
		}

		if varName != "" && isIdentifier(varName) {
			existingRefs = append(existingRefs, ComponentRef{
				Variable: varName, Component: retComp, URI: pr.URI, Line: conv.Uint32(lineNum),
			})
			allRefs = append(allRefs, existingRefs[len(existingRefs)-1])
		}

		lineNum++
	}

	return existingRefs
}

// lookupMethodReturn finds a method's ReturnComponent in a component.
// Uses FuncLookup callback if available (for cross-file resolution via index).
func (pr *ParseResult) oldLookupMethodReturn(component, methodName string) string {
	if pr.FuncLookup != nil {
		if ret := pr.FuncLookup(component, methodName); ret != "" {
			return ret
		}
	}

	// Fallback: check same-file functions
	for i := range pr.Funcs {
		f := &pr.Funcs[i]

		if strings.EqualFold(f.Name, methodName) {
			if f.ReturnComponent != "" {
				return f.ReturnComponent
			}

			if isComponentType(f.ReturnType) {
				return f.ReturnType
			}
		}
	}

	return ""
}

// oldCachedFuncRefs is cachedFuncRefs as it was: the parse's entry for the
// function when it filed one, else its body parsed on its own.
func (pr *ParseResult) oldCachedFuncRefs(funcStart, funcEnd int) []ComponentRef {
	if refs, ok := pr.funcRefsMap[funcKey(funcStart, funcEnd)]; ok {
		return refs
	}

	refs, _ := pr.oldFuncRefsUncached(funcStart, funcEnd)

	return refs
}

// computeAnyScopedVars scans every function body in the file for assignments in the
// given scope, so HasScopedAssignment can find a VARIABLES./THIS.-scoped assignment
// regardless of which function set it.
func (pr *ParseResult) oldComputeAnyScopedVars(scope Scope) []string {
	seen := make(map[string]bool)

	var names []string

	for _, fs := range pr.Scopes {
		start, end := pr.lineOffsets(fs.Start, fs.End)
		if start < 0 {
			continue
		}

		body := pr.Content[start:end]

		regionKind := RegionScript

		for _, r := range pr.Regions {
			if r.StartLine <= fs.Start {
				regionKind = r.Kind
			}
		}

		var bodyVars []VarDef

		if regionKind == RegionScript {
			sp := newScriptParser(body, "", fs.Start, nil)
			sp.parse()
			bodyVars = sp.vars
		} else {
			tp := newTagParser(body, "")
			tp.parse()
			bodyVars = tp.vars
		}

		for _, v := range bodyVars {
			if v.Scope == scope && !seen[v.Name] {
				seen[v.Name] = true

				names = append(names, v.Name)
			}
		}
	}

	return names
}

// parseFuncBody parses a single function body for variable declarations.
func (pr *ParseResult) oldParseFuncBody(funcStart, funcEnd int) (names []string) {
	start, end := pr.lineOffsets(funcStart, funcEnd)
	if start < 0 {
		return nil
	}

	body := pr.Content[start:end]

	sp := newScriptParser(body, "", 0, nil)
	sp.parse()

	seen := make(map[string]bool)

	for _, v := range sp.vars {
		if v.Scope == ScopeLocal || v.Scope == ScopeArguments {
			if !seen[v.Name] {
				seen[v.Name] = true

				names = append(names, v.Name)
			}
		}
	}

	return names
}

// initBodyVars is every variable the init() function's body declares, nil
// when the file has no init().
func (pr *ParseResult) oldInitBodyVars() []VarDef {
	initScope := pr.initFuncScope()
	if initScope.Start == -1 {
		return nil
	}

	start, end := pr.lineOffsets(initScope.Start, initScope.End)
	if start < 0 {
		return nil
	}

	body := pr.Content[start:end]
	regionKind := RegionScript

	for _, r := range pr.Regions {
		if r.StartLine <= initScope.Start {
			regionKind = r.Kind
		}
	}

	if regionKind == RegionScript {
		sp := newScriptParser(body, "", initScope.Start, nil)
		sp.parse()

		return sp.vars
	}

	tp := newTagParser(body, "")
	tp.parse()

	return tp.vars
}

// funcCallsUncached parses a function body on-demand to extract call sites.
func (pr *ParseResult) oldFuncCallsUncached(funcStart, funcEnd int) []CallSite {
	start, end := pr.lineOffsets(funcStart, funcEnd)
	if start < 0 {
		return nil
	}

	body := pr.Content[start:end]

	regionKind := RegionScript

	for _, r := range pr.Regions {
		if r.StartLine <= funcStart {
			regionKind = r.Kind
		}
	}

	var calls []CallSite

	if regionKind == RegionScript {
		sp := newScriptParser(body, string(pr.URI), funcStart, pr.Resolvers)
		sp.baseOffset = start
		sp.resolverSet = pr.resolverSet
		sp.extractCalls = true
		sp.parse()
		calls = append(calls, sp.calls...)

		for _, c := range sp.funcCalls {
			calls = append(calls, c...)
		}
	} else {
		tp := newTagParser(body, string(pr.URI))
		tp.srcOffset = start
		tp.resolvers = pr.Resolvers
		tp.resolverSet = pr.resolverSet
		tp.extractCalls = true
		tp.parse()
		calls = append(calls, tp.calls...)

		for _, c := range tp.funcCalls {
			calls = append(calls, c...)
		}

		for i := range calls {
			calls[i].Line += conv.Uint32(funcStart)
		}
	}

	return calls
}

// funcReturnsFor is funcReturns limited to the functions calls name, for a
// body typed on its own: building the whole map for every function asked
// about was most of what typing one cost.
func (pr *ParseResult) funcReturnsFor(calls []pendingCall) map[string]string {
	out := map[string]string{}

	for i := range calls {
		if c := &calls[i]; c.funcName != "" && !c.returnExpr && c.callsOwnFunction() {
			out[strings.ToLower(c.funcName)] = ""
		}
	}

	for i := range pr.Funcs {
		f := &pr.Funcs[i]

		key := strings.ToLower(f.Name)
		if _, ok := out[key]; !ok {
			continue
		}

		comp := f.ReturnComponent
		if comp == "" && isComponentType(f.ReturnType) {
			comp = f.ReturnType
		}

		if comp != "" {
			out[key] = comp
		}
	}

	return out
}
