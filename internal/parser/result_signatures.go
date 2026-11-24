package parser

import (
	"maps"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/log"
)

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
