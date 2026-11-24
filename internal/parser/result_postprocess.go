package parser

import (
	"cmp"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// applyServiceProperties scans the whole file for "@serviceproperty" annotations and
// registers a ComponentRef for the annotated variable at the annotation's line, resolved
// via the per-kind dot-path template in pr.ServicePropertyResolvers (e.g.
// "service" -> "tassweb.packages.${name}.service" turns "@serviceproperty objGLJournal
// service|gljournal" into a ref pointing at "tassweb.packages.gljournal.service"). A no-op
// when ServicePropertyResolvers isn't configured, so projects not using the convention pay
// no parsing cost. Runs after pr.Scopes is finalized so each ref can be routed to the
// enclosing function's scope, same as any other ref — this is what makes the annotation
// work like a real (if file-scanned rather than scanner-tracked) assignment: a later
// re-annotation of the same variable name overrides an earlier one, via the existing
// nearest-preceding-ComponentRef resolution in resolve.CanResolveCall.
func (pr *ParseResult) applyServiceProperties() {
	if len(pr.ServicePropertyResolvers) == 0 {
		return
	}

	// The matches come in order, so the line count is carried between them.
	lines := lineCounter{text: pr.Content}

	for _, m := range servicePropertyAnnotations(pr.Content) {
		varName := m.name
		kind := strings.ToLower(m.kind)
		name := m.target

		tmpl, ok := pr.ServicePropertyResolvers[kind]
		if !ok {
			continue
		}

		component := strings.ReplaceAll(tmpl, "${name}", name)
		line := conv.Uint32(lines.before(m.start))

		// resolve.CanResolveCall strips a receiver down to the text after its last
		// top-level "." before comparing against ComponentRef.Variable (e.g.
		// "VARIABLES.donorObj" is looked up as "donorObj"; StripReceiverScope skips any
		// "." inside a "[...]" subscript, so "linkMap[arguments.startSource]" is left
		// unchanged rather than being cut to "startSource]"). Apply the same stripping
		// here so an annotation written against the natural, full receiver text at the
		// call site (e.g. "result.data[i]", matching what a reader actually sees in the
		// source) is stored under the same key CanResolveCall will look it up with.
		lookupVar := StripReceiverScope(varName)

		ref := ComponentRef{
			Variable: lookupVar, Component: component,
			URI: pr.URI, Line: line,
		}

		if fs := findFuncScope(int(line), pr.Scopes); fs.Start != -1 {
			if pr.funcRefsMap == nil {
				pr.funcRefsMap = make(map[fnKey][]ComponentRef)
			}

			key := funcKey(fs.Start, fs.End)
			pr.funcRefsMap[key] = append(pr.funcRefsMap[key], ref)
		} else {
			pr.ComponentRefs = append(pr.ComponentRefs, ref)
		}
	}
}

// applyChainedReturnLookup re-checks component refs created from a receiver.method()
// call (see ComponentRef.ChainBase/ChainMethod) against FuncLookup, overriding the
// componentResolver's guess on the call-site text with the callee's own declared
// return type when FuncLookup can verify it (e.g. a Java stub's getInstance()
// modeling its real return type). Runs last so a verified answer always wins.
//
// It then walks ChainRest, the calls chained after the one that typed the ref,
// so "chart = b.width(1).height(2).build()" types chart by build rather than
// by width. A call it cannot type makes the variable dynamic: keeping the type
// of an earlier call is the wrong answer this exists to stop, and it reported
// every method of the real value as missing from the builder.
func (pr *ParseResult) applyChainedReturnLookup() {
	if pr.FuncLookup == nil {
		dropChainRest(pr, dynamicIfTyped)

		return
	}

	lookupBase := func(baseVar string, scopedRefs []ComponentRef) string {
		for i := range scopedRefs {
			ref := &scopedRefs[i]

			if strings.EqualFold(ref.Variable, baseVar) {
				return ref.Component
			}
		}

		for i := range pr.ComponentRefs {
			ref := &pr.ComponentRefs[i]

			if strings.EqualFold(ref.Variable, baseVar) {
				return ref.Component
			}
		}

		return ""
	}

	override := func(refs []ComponentRef, scoped []ComponentRef) {
		for i := range refs {
			if refs[i].ChainBase == "" {
				continue
			}

			baseComp := lookupBase(refs[i].ChainBase, scoped)
			if baseComp == "" {
				continue
			}

			if ret := pr.FuncLookup(baseComp, refs[i].ChainMethod); ret != "" {
				refs[i].Component = ret
			}
		}
	}

	override(pr.ComponentRefs, nil)

	for k, refs := range pr.funcRefsMap {
		override(refs, refs)
		pr.funcRefsMap[k] = refs
	}

	dropChainRest(pr, pr.walkChainRest)
}

// walkChainRest follows comp through the calls in rest by their declared
// return types. init() is taken to return the object it is called on, which is
// the convention for a CFC constructor and what a java stub's constructor
// models.
func (pr *ParseResult) walkChainRest(comp string, rest []string) string {
	for _, hop := range rest {
		name := callHopName(hop)
		switch {
		case comp == "" || comp == "$any":
			return comp

		case strings.EqualFold(name, "init"), IsMockDecoration(name):
			// A MockBox decoration returns the mock it is called on.
			continue
		}

		ret := pr.FuncLookup(comp, hop)
		if ret == "" && name != hop {
			ret = pr.FuncLookup(comp, name)
		}

		if ret == "" {
			return "$any"
		}

		comp = ret
	}

	return comp
}

// dynamicIfTyped is the answer without FuncLookup: a chain whose rest holds
// any call but init() ends somewhere unknown.
func dynamicIfTyped(comp string, rest []string) string {
	if restTypes(rest) {
		return "$any"
	}

	return comp
}

// restTypes reports whether a chain's rest can change the type it started
// with: anything but init() calls and MockBox decorations, which return what
// they are called on.
func restTypes(rest []string) bool {
	return slices.ContainsFunc(rest, func(h string) bool {
		return !strings.EqualFold(callHopName(h), "init") && !IsMockDecoration(callHopName(h))
	})
}

// settledComponent is ref's Component with any pending chain walked, for a
// reader during the parse that has FuncLookup to hand — resolvePendingCalls,
// typing a variable from the one it was called on — so it reads what the
// finished ref will hold rather than the first call's type.
func (pr *ParseResult) settledComponent(ref *ComponentRef) string {
	if !chainPending(ref) {
		return ref.Component
	}

	if pr.FuncLookup == nil {
		return strictChainComponent(ref, dynamicIfTyped(ref.Component, ref.ChainRest))
	}

	return strictChainComponent(ref, pr.walkChainRest(ref.Component, ref.ChainRest))
}

func strictChainComponent(ref *ComponentRef, component string) string {
	if ref.strictChain && strings.HasPrefix(component, "$") {
		return ""
	}

	return component
}

// chainPending reports whether ref's Component is not yet its answer: it was
// typed by the first call of a longer chain that applyChainedReturnLookup has
// still to walk. A lookup made during the parse must not read it — it would
// hand on the first call's type, which is exactly the wrong answer the walk
// exists to replace — and gets nothing instead, which the resolve step, run
// after the walk, answers from the finished ref.
func chainPending(ref *ComponentRef) bool {
	return restTypes(ref.ChainRest)
}

// dropChainRest applies type to every ref carrying a ChainRest, then clears it.
func dropChainRest(pr *ParseResult, typ func(comp string, rest []string) string) {
	apply := func(refs []ComponentRef) {
		for i := range refs {
			if len(refs[i].ChainRest) == 0 {
				continue
			}

			refs[i].Component = strictChainComponent(&refs[i], typ(refs[i].Component, refs[i].ChainRest))
			refs[i].ChainRest = nil
		}
	}

	apply(pr.ComponentRefs)

	for _, refs := range pr.funcRefsMap {
		apply(refs)
	}
}

// applyExpressionMappings replaces runtime expressions in component paths with static
// values, then treats anything still containing a "#...#" CFML expression as genuinely
// dynamic (e.g. CreateObject("component", "tools.templates.#ARGUMENTS.template#.generator") —
// a runtime-computed path with no configured mapping) and collapses it to "$any" so
// CanResolveCall accepts calls through it instead of surfacing the raw "#...#" text as a
// bogus "not found in #...#" component name. Runs even with no expressionMappings
// configured, since the "$any" fallback is unconditional.
func (pr *ParseResult) applyExpressionMappings() {
	for i := range pr.ComponentRefs {
		pr.ComponentRefs[i].Component = pr.replaceExpressions(pr.ComponentRefs[i].Component)
	}

	for k, refs := range pr.funcRefsMap {
		for i := range refs {
			refs[i].Component = pr.replaceExpressions(refs[i].Component)
		}

		pr.funcRefsMap[k] = refs
	}

	// CallSite.Component can also be baked in directly at parse time (e.g. a bare
	// "x.method()" call whose receiver was resolved inline via lookupComponentRef,
	// or a chained new/createObject) — fix those up too, not just ComponentRefs.
	for i := range pr.Calls {
		pr.Calls[i].Component = pr.replaceExpressions(pr.Calls[i].Component)
	}

	for k, calls := range pr.funcCallsMap {
		for i := range calls {
			calls[i].Component = pr.replaceExpressions(calls[i].Component)
		}

		pr.funcCallsMap[k] = calls
	}

	// A path built at run time builds whatever the program picks; the map
	// draws no edge to it.
	kept := pr.Instantiations[:0]

	for _, in := range pr.Instantiations {
		if in.Component = pr.replaceExpressions(in.Component); in.Component != "$any" {
			kept = append(kept, in)
		}
	}

	pr.Instantiations = kept
}

// ExpressionMappingOrder returns the keys of an expressionMappings block in the
// order they are applied: longest first, and by key among equals. Applying
// them in map order let two overlapping keys — `#A#` and `#A#.b` — replace in
// either order from run to run, so one component path named two different
// components, and a code-map edge moved between them. The longer, more
// specific expression goes first, as route aliases do.
func ExpressionMappingOrder(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}

	keys := slices.Collect(maps.Keys(m))
	slices.SortFunc(keys, func(a, b string) int {
		return cmp.Or(cmp.Compare(len(b), len(a)), strings.Compare(a, b))
	})

	return keys
}

func (pr *ParseResult) replaceExpressions(comp string) string {
	if !strings.Contains(comp, "#") {
		return comp
	}

	for _, key := range pr.expressionKeys {
		value := pr.expressionMappings[key]

		for expr := range strings.SplitSeq(key, "|") {
			if expr != "" && strings.Contains(comp, expr) {
				comp = strings.ReplaceAll(comp, expr, value)
			}
		}
	}

	if strings.Contains(comp, "#") {
		return "$any"
	}

	return comp
}

// appendResolverRefs keeps in Calls the calls of the names the parse was
// asked to find (findCalls), from every scope, in the order they are written.
func (pr *ParseResult) appendResolverRefs() {
	if len(pr.findCalls) == 0 {
		return
	}

	pr.filterCallsByName()

	if pr.extractCalls {
		return
	}

	// Only these were recorded, and Calls now holds every one: FuncCalls of
	// a parse that does not extract calls parses a body for its own.
	pr.funcCallsMap = nil
	pr.dropCallsIndex()

	sort.SliceStable(pr.Calls, func(i, j int) bool { return callWrittenBefore(&pr.Calls[i], &pr.Calls[j]) })
}

// callWrittenBefore reports whether a is written before b: by line, then by
// where its span starts.
func callWrittenBefore(a, b *CallSite) bool {
	if a.Line != b.Line {
		return a.Line < b.Line
	}

	return a.StartByte < b.StartByte
}

// filterCallsByName populates pr.Calls with calls from funcCallsMap and global
// calls that match the pr.findCalls target names.
func (pr *ParseResult) filterCallsByName() {
	targets := make(map[string]bool, len(pr.findCalls))
	for _, t := range pr.findCalls {
		targets[strings.ToLower(t)] = true
	}

	// Gather all calls (global + per-function)
	var allCalls []CallSite

	allCalls = append(allCalls, pr.Calls...)

	for _, calls := range pr.funcCallsMap {
		allCalls = append(allCalls, calls...)
	}

	// Filter by target names and resolve components
	pr.Calls = nil

	scopeAt := scopeIndex(pr.Scopes, false)

	for i := range allCalls {
		call := allCalls[i] // a copy: edited below, and kept

		if !targets[strings.ToLower(call.FuncName)] {
			continue
		}

		// Resolve component from variable name
		if call.Variable != "" && call.Component == "" {
			// Strip scope prefix (VARIABLES.service → service)
			resolveVar := call.Variable
			if dotIdx := strings.LastIndexByte(resolveVar, '.'); dotIdx >= 0 {
				resolveVar = resolveVar[dotIdx+1:]
			}

			call.Component = pr.resolveVarComponent(resolveVar)
			call.Resolved = call.Component != ""
		}

		// Determine caller from line
		if call.Caller == "" {
			if k := scopeAt(int(call.Line)); k >= 0 {
				call.Caller = pr.Scopes[k].Name
			}
		}

		pr.Calls = append(pr.Calls, call)
	}
}

func sortScopes(scopes []FuncScope) {
	for i := 1; i < len(scopes); i++ {
		for j := i; j > 0 && scopes[j].Start < scopes[j-1].Start; j-- {
			scopes[j], scopes[j-1] = scopes[j-1], scopes[j]
		}
	}
}
