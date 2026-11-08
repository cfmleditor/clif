// The component refs and links of a function body, and the return types its calls settle.

package parser

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// FuncComponentRefs returns cached component refs for a function scope.
//
// A ref a closure in the function declared is left out: it exists only inside
// the closure. FuncComponentRefsAt includes the ones in force at a line.
func (pr *ParseResult) FuncComponentRefs(funcStart, funcEnd int) []ComponentRef {
	refs, _ := pr.cachedFuncRefs(funcStart, funcEnd)

	scoped := 0

	for i := range refs {
		if refs[i].VisibleTo != 0 || refs[i].HasClosureSpan {
			scoped++
		}
	}

	if scoped == 0 {
		return refs
	}

	wide := make([]ComponentRef, 0, len(refs)-scoped)

	for i := range refs {
		if refs[i].VisibleTo == 0 && !refs[i].HasClosureSpan {
			wide = append(wide, refs[i])
		}
	}

	return wide
}

// FuncComponentRefsAt returns the function's refs in force at line: the ones
// the whole function sees, and the ones declared by a closure holding line.
// The slice is the parse's own storage when no closure declared anything, so
// callers must not write to it.
func (pr *ParseResult) FuncComponentRefsAt(funcStart, funcEnd int, line uint32) []ComponentRef {
	refs, _ := pr.cachedFuncRefs(funcStart, funcEnd)

	for i := range refs {
		if !refs[i].VisibleAt(line) {
			out := slices.Clone(refs[:i])

			for j := i + 1; j < len(refs); j++ {
				if refs[j].VisibleAt(line) {
					out = append(out, refs[j])
				}
			}

			return out
		}
	}

	return refs
}

// FuncRefs returns cached component refs and document links for a function body.
func (pr *ParseResult) FuncRefs(funcStart, funcEnd int) ([]ComponentRef, []DocumentLink) {
	return pr.cachedFuncRefs(funcStart, funcEnd)
}

// TopLevelClosureRefs returns the full parse's anonymous-function bindings.
// They are keyed separately from named functions and never become file globals.
func (pr *ParseResult) TopLevelClosureRefs() []ComponentRef {
	pr.funcRefsMu.Lock()
	defer pr.funcRefsMu.Unlock()

	return slices.Clone(pr.funcRefsMap[""])
}

// FuncLinks returns document links for a function body from the parse-time cache.
func (pr *ParseResult) FuncLinks(funcStart, funcEnd int) []DocumentLink {
	key := funcKey(funcStart, funcEnd)

	pr.funcRefsMu.Lock()
	defer pr.funcRefsMu.Unlock()

	if pr.funcLinksMap != nil {
		return pr.funcLinksMap[key]
	}

	return nil
}

func (pr *ParseResult) cachedFuncRefs(funcStart, funcEnd int) ([]ComponentRef, []DocumentLink) {
	key := funcKey(funcStart, funcEnd)

	pr.funcRefsMu.Lock()
	defer pr.funcRefsMu.Unlock()

	if pr.funcRefsMap != nil {
		if cached, ok := pr.funcRefsMap[key]; ok {
			links := pr.funcLinksMap[key]

			return cached, links
		}
	}

	refs, links := pr.funcRefsUncached(funcStart, funcEnd)

	if pr.funcRefsMap == nil {
		pr.funcRefsMap = make(map[string][]ComponentRef)
	}

	if pr.funcLinksMap == nil {
		pr.funcLinksMap = make(map[string][]DocumentLink)
	}

	pr.funcRefsMap[key] = refs
	pr.funcLinksMap[key] = links

	return refs, links
}

// typeBodyPendingCalls types the assignments a function body parsed on its
// own made from calls, as the full parse types them: without it
// `variables.y = variables.y.next()` and `z = make()` added no ref, and the
// variable kept its first type wherever FuncRefs parsed the body itself:
// for a function the parse filed no ref under, and for every function after
// an edit. Chained refs are settled as applyChainedReturnLookup settles them.
// The body's refs are its own, keyed as its parser keyed them; a receiver is
// also looked up among the file's.
func (pr *ParseResult) typeBodyPendingCalls(global []ComponentRef, local map[string][]ComponentRef, calls []pendingCall) ([]ComponentRef, map[string][]ComponentRef) {
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

func (pr *ParseResult) funcRefsUncached(funcStart, funcEnd int) ([]ComponentRef, []DocumentLink) {
	start, end := pr.lineOffsets(funcStart, funcEnd)
	if start < 0 {
		return nil, nil
	}

	body := pr.Content[start:end]

	// Determine region kind for this function
	regionKind := RegionScript

	for _, r := range pr.Regions {
		if r.StartLine <= funcStart {
			regionKind = r.Kind
		}
	}

	var (
		refs  []ComponentRef
		links []DocumentLink
	)

	if regionKind == RegionScript {
		sp := newScriptParser(body, string(pr.URI), funcStart, pr.Resolvers)
		sp.baseOffset = start
		sp.resolverSet = pr.resolverSet
		sp.extractLinks = true
		sp.argumentTypes = pr.argumentTypeLookup()
		sp.imports = pr.imports
		sp.parse()
		sp.componentRefs, sp.funcRefs = pr.typeBodyPendingCalls(sp.componentRefs, sp.funcRefs, sp.pendingCalls)
		refs = sp.componentRefs
		links = sp.links
		// Include function-scoped refs (nested functions in body)
		for _, r := range sp.funcRefs {
			refs = append(refs, r...)
		}

		for _, l := range sp.funcLinks {
			links = append(links, l...)
		}
	} else {
		tp := newTagParser(body, string(pr.URI))
		tp.srcOffset = start
		tp.resolvers = pr.Resolvers
		tp.resolverSet = pr.resolverSet
		tp.extractLinks = true
		tp.argumentTypes = pr.argumentTypeLookup()
		tp.parse()
		tp.componentRefs, tp.funcRefs = pr.typeBodyPendingCalls(tp.componentRefs, tp.funcRefs, tp.pendingCalls)
		refs = tp.componentRefs

		links = tp.links
		for _, r := range tp.funcRefs {
			refs = append(refs, r...)
		}

		for _, l := range tp.funcLinks {
			links = append(links, l...)
		}
	}

	// Offset lines by funcStart for tag parser (script parser uses baseLine)
	if regionKind != RegionScript {
		for i := range refs {
			refs[i].Line += conv.Uint32(funcStart)
		}

		for i := range links {
			links[i].Line += conv.Uint32(funcStart)
		}
	}

	// Scan for function calls (still line-based as it's only for exportDeps)
	if len(pr.findCalls) > 0 && !pr.extractCalls {
		lineNum := funcStart + 1

		// Only the scopes overlapping this function can hold its lines, in
		// the order callerAtLine scans them.
		var near []FuncScope

		for _, sc := range pr.Scopes {
			if sc.End > funcStart && sc.Start < funcEnd {
				near = append(near, sc)
			}
		}

		scopeAt := scopeIndex(near, false)

		scan := pr.Content[start:end]
		for scan != "" {
			nl := strings.IndexByte(scan, '\n')

			var line string
			if nl < 0 {
				line = scan
				scan = ""
			} else {
				line = scan[:nl]
				scan = scan[nl+1:]
			}

			caller := ""
			if k := scopeAt(lineNum); k >= 0 {
				caller = near[k].Name
			}

			pr.scanLineForCalls(line, lineNum, caller)
			lineNum++
		}
	}

	refs = pr.resolveMethodReturnRefs(funcStart, funcEnd, refs)

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

// resolveMethodReturnRefs scans a function body for x = y.method() assignments
// where y has a known component ref and method declares a return type.
func (pr *ParseResult) resolveMethodReturnRefs(funcStart, funcEnd int, existingRefs []ComponentRef) []ComponentRef {
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
		retComp := pr.lookupMethodReturn(baseComp, methodName)
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
func (pr *ParseResult) lookupMethodReturn(component, methodName string) string {
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

func isIdentifier(s string) bool {
	for i, c := range s {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c == '$' {
			continue
		}

		if i > 0 && c >= '0' && c <= '9' {
			continue
		}

		return false
	}

	return s != ""
}

// isValidVarChain returns true if s looks like a valid CFML variable chain
// (e.g. "VARIABLES.prs", "obj", "result[1].data"). Rejects strings containing
// operators, quotes, hash signs, or whitespace.
func isValidVarChain(s string) bool {
	if s == "" {
		return false
	}

	for _, c := range s {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c == '$' || c >= '0' && c <= '9' || c == '.' || c == '[' || c == ']' {
			continue
		}

		return false
	}

	return true
}
