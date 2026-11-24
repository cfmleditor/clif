package parser

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

func (p *scriptParser) addRef(ref *ComponentRef) {
	ref.This = p.refThis

	if p.flow != nil && p.inFunc.in {
		p.flow.note(ref.Variable, ref.Line, p.flow.innermost())
	}

	if !p.inFunc.in && !p.inClosure || p.forceGlobal {
		p.componentRefs = append(p.componentRefs, *ref)
	} else {
		if p.funcRefs == nil {
			p.funcRefs = make(map[fnKey][]ComponentRef)
		}

		p.funcRefs[p.inFunc] = append(p.funcRefs[p.inFunc], *ref)
	}
}

// noteAssign records, for the flow analysis, an assignment to name in a
// function body of a value no component can be (untypedValueText), read from
// the token after the `=` (flowBlocks.untypedReaches).
func (p *scriptParser) noteAssign(name string, line int) {
	if p.flow == nil || !p.inFunc.in || p.inClosure {
		return
	}

	switch rhs := p.sc.PeekSkipComments(); {
	case rhs.Kind == TokString || rhs.Kind == TokNumber,
		rhs.Kind == TokIdent && (identEq(rhs.Value, "true") || identEq(rhs.Value, "false") || identEq(rhs.Value, "arguments")):
		p.flow.noteAssign(name, conv.Uint32(p.baseLine+line))
	}
}

func (p *scriptParser) addCall(call *CallSite) {
	// The single gate on recording a call. Consuming one is not gated: the
	// dispatch that walks a call and its argument list runs in every mode,
	// because a (...) group left unconsumed is not skipped — it is scanned as
	// if it were statements, and `svc.save(force = true)` then declares a
	// variable called `force`. See parseScriptTagAttrs for the other half of
	// that class of bug.
	if !p.recordsCall(call.FuncName) {
		return
	}

	p.setCallSpan(call)

	if !p.inFunc.in {
		p.calls = append(p.calls, *call)
	} else {
		if p.funcCalls == nil {
			p.funcCalls = make(map[fnKey][]CallSite)
		}

		p.funcCalls[p.inFunc] = append(p.funcCalls[p.inFunc], *call)
	}
}

// asCFScript marks the parser's text as genuine CFScript rather than a tag
// function's raw body, which turns on interpolation-aware string scanning.
//
// It is opt-in so that a caller which has not thought about it gets the safe
// scan: applied to markup the rule pairs the hashes of two `href="#…"`
// fragments and swallows what is between them. See Scanner.scanString.
func (p *scriptParser) asCFScript() *scriptParser {
	p.sc.interpStrings = true

	return p
}

// recordBareCallAndChain handles funcName(...) optionally followed by .method(...) chains.
// When the call stands alone — no hop or index follows it — alone is the
// expression the componentResolvers are offered for it, so a `return` can be
// typed as an assignment's right-hand side is; it is "" otherwise.
func (p *scriptParser) recordBareCallAndChain(tok Token) (alone string) {
	caller := ""
	if p.inFunc.in && len(p.funcs) > 0 {
		caller = p.currentFunc().Name
	}

	p.addCall(&CallSite{
		FuncName: tok.Value,
		Line:     conv.Uint32(p.baseLine + int(tok.Line)),
		Caller:   caller,
	})

	// Consume the argument list, capturing the first string arg for the
	// resolver and recording any calls written inside it. This used to be a
	// second copy of scanParenBody's loop, which is why a call nested in a
	// *bare* call's arguments stayed invisible after the dotted path learned to
	// see one: writeOutput(svc.getName()) recorded only writeOutput.
	argStart := p.sc.PeekSkipComments().Offset
	p.sc.NextSkipComments() // consume (

	firstArg, positional, ok := p.scanParenArgs()
	if !ok {
		return ""
	}

	callExpr := resolverCallExpr(tok.Value, firstArg, positional)

	// createObject( "component", "x" ) outside an assignment builds x as
	// much as one inside does (readCreateObjectComponent).
	if len(positional) == 2 && identEq(tok.Value, "createObject") && identEq(positional[0], "component") {
		p.addInstantiation(positional[1], tok)
	}

	// hop is funcName as a Chain entry: the call with its arguments, which
	// resolution reads an argument-sensitive return from.
	hop := p.hopSince(tok.Value, argStart)

	if next := p.sc.PeekSkipComments().Kind; next != TokDot && next != TokLBracket {
		alone = callExpr
	}

	// comp is this bare call's resolved return component (if any); it's the
	// base receiver for every subsequent chained hop below. Hops beyond the
	// first accumulate in chainHops so CanResolveCall can walk comp's type
	// forward through each intermediate call before checking the current one.
	//
	// The argument list is already consumed, so this matches callExpr as
	// built above rather than going through tryResolveCall, which expects to
	// read the arguments itself from an unconsumed "(". A call with no string
	// argument is offered as name() — getObjInit().getMailServer() — the same
	// form tryResolveCall offers an assignment's right-hand side. It is only
	// asked when a hop follows, since nothing else reads comp.
	comp := ""

	if len(p.resolvers) > 0 && p.sc.PeekSkipComments().Kind == TokDot {
		comp = p.resolveCall(callExpr)
	}

	if !p.walkBareCallHops(caller, comp, hop) {
		return ""
	}

	return alone
}

// walkBareCallHops follows the .method( hops written after a bare call whose
// own call is already recorded, recording each one with the calls before it as
// its Chain. comp is the bare call's resolved component, which stands as the
// first hop's receiver in place of the call; hop is the call written as a
// Chain entry. It is false when an argument list or an index does not close.
func (p *scriptParser) walkBareCallHops(caller, comp, hop string) bool {
	var chainHops []string

	first := true

	// Check for .method( chain
	for {
		if p.sc.PeekSkipComments().Kind == TokLBracket {
			// Dynamic index between hops (e.g. someFunc()[key].method()) —
			// skip it and keep walking the chain; no receiver identifier is
			// at risk of misattribution here (the base is a call return, not
			// a bare variable), so no poisoning is needed, just continuity.
			if !p.skipBracketIndex() {
				return false
			}

			continue
		}

		if p.sc.PeekSkipComments().Kind != TokDot {
			break
		}

		p.sc.NextSkipComments() // consume .

		methTok := p.sc.PeekSkipComments()
		if methTok.Kind != TokIdent {
			break
		}

		p.sc.NextSkipComments() // consume method name

		if p.sc.PeekSkipComments().Kind == TokLParen {
			// The bare call is the first hop's receiver. When a resolver
			// named its component the hop carries that instead; otherwise
			// the call goes at the front of the chain, so resolution walks
			// its declared return type. Leaving it out made `f().g()` record
			// a `g` indistinguishable from a bare call to a function named
			// g, as recordChainContinuationFrom already avoids.
			if comp == "" || !first {
				chainHops = append(chainHops, hop)
			}

			first = false

			hops := chainSoFar(chainHops)
			if hops == nil {
				hops = []string{} // as the copy this replaced made it
			}

			p.addCall(&CallSite{
				FuncName:  methTok.Value,
				Component: comp,
				Chain:     hops,
				Line:      conv.Uint32(p.baseLine + int(methTok.Line)),
				Caller:    caller,
				Resolved:  comp != "",
			})

			// Consume this hop's arguments too, so the chain walk can
			// continue — through the same scan, so a call written inside
			// them is found: this was the third copy of the paren loop and
			// the last one still losing `a().b(svc.c())`.
			argStart := p.sc.PeekSkipComments().Offset
			p.sc.NextSkipComments() // consume (

			if _, ok := p.scanParenBody(); !ok {
				return false
			}

			hop = p.hopSince(methTok.Value, argStart)
		} else {
			break
		}
	}

	return true
}

// scopeReceiver says how a call made directly on a scope is recorded, for the
// scopes a function body dispatches as call receivers. `this.` and
// `variables.` name a member of the component being parsed, so the call is
// recorded unqualified and resolves against the file's own functions. Every
// other scope holds a value put there at runtime, so the receiver is kept and
// the component is `$any`. ok is false for a word that is not one of them.
//
// Every path that records such a call goes through here — a statement, a
// `return` and an assignment's right-hand side. The last two used to record
// the scope itself as the receiver, so `x = variables.f()` reported
// "variable 'variables' has no component ref" where the statement
// `variables.f()` resolved.
func scopeReceiver(word string) (variable, component string, ok bool) {
	var buf foldScratch

	switch string(buf.lowerFold(word)) {
	case "this", "variables":
		return "", "", true
	case "local", "arguments", "request", "session", "application", "server":
		return word, "$any", true
	default:
		return "", "", false
	}
}

// onScope gives c the receiver scopeReceiver says a call made directly on
// the scope word has, when word is one, and records whether it was `this.`:
// the parser records `this.f()` and `f()` alike, and only the first reaches
// onMissingMethod.
func (c *CallSite) onScope(word string) {
	v, comp, ok := scopeReceiver(word)
	if !ok {
		return
	}

	c.Variable, c.Component, c.Resolved = v, comp, comp != ""
	c.This = identEq(word, "this")
}

// recordCallFromChain records a call site when a dot chain ending in ( is detected.
// fullChain is e.g. "VARIABLES.service.GetData" or just "GetData", line is the source line.
func (p *scriptParser) recordCallFromChain(fullChain string, line int) {
	caller := ""
	if p.inFunc.in && len(p.funcs) > 0 {
		caller = p.currentFunc().Name
	}

	recv, method, ok := strings.CutLast(fullChain, ".")
	if !ok {
		// Bare function call (no dot)
		p.addCall(&CallSite{
			FuncName: fullChain,
			Line:     conv.Uint32(p.baseLine + line),
			Caller:   caller,
		})

		return
	}

	call := CallSite{
		FuncName: method,
		Variable: recv,
		Line:     conv.Uint32(p.baseLine + line),
		Caller:   caller,
	}

	call.onScope(recv)

	p.addCall(&call)
}

// continueChainCalls consumes the rest of a chained call expression whose
// first hop's resolver lookup failed (tryResolveCall/tryExtendChain both
// returned "", falling through to a pendingCall) — e.g.
// "document.getJavaUtils().getRGBColor(r=16,g=58,b=59)" where
// "document.getJavaUtils" matched no resolver. tryResolveCall/tryExtendChain
// restore the scanner to the unconsumed "(" on failure, so without this the
// caller's next token read rediscovers ".getRGBColor(...)" as an orphaned,
// unqualified bare call ("no qualifier, not in file") instead of a call
// chained off baseVar. Must be called with the scanner positioned at that
// first "(". baseVar is the chain's receiver (may be "" for a bare call
// chain) and funcName is the first hop's own name; further hops get
// CallSites with Variable=baseVar and an accumulating Chain, matching
// checkBareCall's shape so CanResolveCall can walk them.
func (p *scriptParser) continueChainCalls(baseVar, funcName string, line int) []string {
	callExpr := funcName
	if baseVar != "" {
		callExpr = baseVar + "." + funcName
	}

	comp, hop, ok := p.skipParensResolving(callExpr)
	if !ok {
		return nil
	}

	return p.recordChainContinuation(baseVar, hop, comp, line)
}

// skipParensResolving is skipParens for the first hop of a chain: it also
// matches componentResolvers against callExpr with the group's string
// arguments (see resolverCallExpr), so a hop that names its component in an argument —
// getService("company") — hands the next hop that component. A Chain entry
// is only a method name, and resolving the hop again from it later sees
// getService() and can only reach a resolver that ignores which service was
// asked for. Unlike tryResolveCall this reads the argument wherever the scan
// finds it, so the named form getService(service="company") resolves too.
// comp is "" when no resolver matches.
//
// hop is the call as a chain entry for the hops after it: with its arguments
// when a hop follows, so resolution can read an argument-sensitive return
// from them, and the bare method name otherwise.
func (p *scriptParser) skipParensResolving(callExpr string) (comp, hop string, ok bool) {
	_, hop, _ = strings.CutLast("."+callExpr, ".")

	if p.sc.PeekSkipComments().Kind != TokLParen {
		return "", hop, false
	}

	start := p.sc.PeekSkipComments().Offset
	p.sc.NextSkipComments() // consume (

	firstArg, positional, ok := p.scanParenArgs()
	if !ok || p.sc.PeekSkipComments().Kind != TokDot {
		return "", hop, ok
	}

	hop = p.hopSince(hop, start)
	if firstArg == "" {
		return "", hop, true
	}

	return p.resolveCall(resolverCallExpr(callExpr, firstArg, positional)), hop, true
}

// recordChainContinuation is continueChainCalls' shared core, factored out so
// it can also run after the first hop's resolver lookup SUCCEEDED — e.g.
// "document.getJavaUtils()" matching a "getJavaUtils" resolver even though
// the real expression continues ".getRGBColor(r=16,g=58,b=59)". In that case
// tryResolveCall has already consumed the first hop's own "(...)" (it must,
// to know where the match ends), so — unlike continueChainCalls — the
// caller here must already have the scanner positioned right after that
// closing ')', with no extra skipParens() call needed for the first hop.
//
// baseComp is the component the first hop was already resolved to, when the
// caller knows it — tryResolveCall matching "getService(\"company\")" against
// its string arguments. The hops then start from that component rather than
// re-deriving the first hop from its bare name: a chain entry carries no
// arguments, so the resolve step would see "getService()" and could only reach
// a resolver that ignores which service was asked for.
//
// It returns the names of the hops it consumed after funcName, in order, which
// an assignment keeps on its ref (ComponentRef.ChainRest) so the variable is
// typed by the last call rather than by funcName.
//
// funcName is the first hop as a chain entry: a bare name, or a CallHop when
// the caller read its arguments before consuming them.
func (p *scriptParser) recordChainContinuation(baseVar, funcName, baseComp string, line int) (consumed []string) {
	// A chain on `this.f()` is on a call onMissingMethod can answer, which
	// a chain on a bare f() is not: each hop says so (CallSite.This).
	viaThis := identEq(baseVar, "this") && baseComp == ""

	// A chain on a call made directly on a scope carries the receiver
	// scopeReceiver gives that call: none for this component's own scopes,
	// and a dynamic one, all the way down, for the rest.
	if v, comp, ok := scopeReceiver(baseVar); ok {
		baseVar = v

		if comp != "" {
			baseComp = comp
		}
	}

	return p.recordChainContinuationFrom(baseVar, nil, funcName, baseComp, line, viaThis)
}

// recordChainContinuationFrom is recordChainContinuation for a chain that
// already has hops before funcName: prior is carried at the front of every
// hop's Chain.
func (p *scriptParser) recordChainContinuationFrom(baseVar string, prior []string, funcName, baseComp string, _ int, viaThis bool) (consumed []string) {
	caller := ""
	if p.inFunc.in && len(p.funcs) > 0 {
		caller = p.currentFunc().Name
	}

	chainHops := slices.Clone(prior)
	_, scopeComp, directScope := scopeReceiver(baseVar)
	keepScopeOrigin := directScope && scopeComp == "$any"

	// hop is funcName as a Chain entry. The first is whatever the caller
	// passed, which carries its arguments when the caller read them; every
	// later one carries them.
	hop := funcName
	first := true

	for p.sc.PeekSkipComments().Kind == TokDot {
		p.sc.NextSkipComments() // consume .

		methTok := p.sc.PeekSkipComments()
		if methTok.Kind != TokIdent {
			break
		}

		p.sc.NextSkipComments() // consume method name

		if p.sc.PeekSkipComments().Kind != TokLParen {
			if baseComp == "" || keepScopeOrigin || !first {
				chainHops = append(chainHops, hop)
			}

			first = false
			funcName = PropertyHop(methTok.Value)
			hop = funcName
			consumed = append(consumed, funcName)

			continue
		}

		if baseComp == "" || keepScopeOrigin || !first {
			chainHops = append(chainHops, hop)
		}

		first = false
		funcName = methTok.Value
		hop = callHopAt(p.sc, funcName)
		consumed = append(consumed, CallHop(callExpressionAt(p.sc, funcName)))

		var call CallSite
		if p.extractCalls {
			call = CallSite{
				FuncName: funcName, Variable: baseVar, Component: baseComp,
				Chain: chainSoFar(chainHops), Line: conv.Uint32(p.baseLine + int(methTok.Line)),
				Caller: caller, Resolved: baseComp != "", This: viaThis,
			}
			// Each hop has its own lexical origin. Capture it while the
			// scanner still points at this hop's argument list, then retain
			// the existing call-recording order after nested arguments.
			p.setCallSpan(&call)
		}

		if !p.skipParens() {
			return consumed
		}

		if p.extractCalls {
			p.addCall(&call)
		}
	}

	return consumed
}

// recordChainFromScope records `scope.name.method(...)` and then any further
// hops chained onto it.
//
// Both scoped-var handlers recorded the first call and then merely skipped its
// argument list, so the `.c()` in `variables.a.b().c()` was left for the outer
// loop to rediscover as an orphaned *bare* call. That is a wrong answer rather
// than a missing one: in a component that declares a `c`, it is an edge the
// call never takes. `continueChainCalls` exists to stop exactly this on the
// unscoped path; this routes the scoped path through the same helper.
func (p *scriptParser) recordChainFromScope(fullChain string, line int) {
	p.recordCallFromChain(fullChain, line)

	comp, hop, ok := p.skipParensResolving(fullChain)
	if !ok {
		return
	}

	base, _, _ := strings.CutLast(fullChain, ".")

	p.recordChainContinuation(base, hop, comp, line)
}

func (p *scriptParser) isVarDeclaredLocal(name string) bool {
	return p.localVarSet.has(name)
}

// fileLinks files links, the region's, under the function each is in, or
// in p.links when it is in none.
func (p *scriptParser) fileLinks(links []DocumentLink) {
	p.funcLinks = extractLinksByScope(links, p.scopes, p.funcLinks, &p.links)
}

// parseVarDecl handles: var name = expr.
func (p *scriptParser) parseVarDecl(tok Token) {
	nameTok := p.sc.NextSkipComments()
	if nameTok.Kind != TokIdent {
		return
	}

	peek := p.sc.PeekSkipComments()
	if peek.Kind != TokEquals {
		p.declareLoopVar(nameTok, peek)

		return
	}

	p.sc.NextSkipComments() // consume =

	p.localVarSet.add(nameTok.Value)

	p.vars = append(p.vars, VarDef{
		Name: nameTok.Value, Scope: ScopeLocal,
		Line: conv.Uint32(p.baseLine + int(tok.Line)),
	})

	if p.chainedAssign(nameTok.Value) {
		return
	}

	// Check RHS for component refs
	rhs := p.rhsAfterLiteral()
	if rhs.Kind != TokIdent {
		return
	}

	var buf foldScratch
	switch string(buf.lowerFold(rhs.Value)) {
	case "new":
		p.sc.NextSkipComments()
		p.parseNewRef(nameTok.Value, int(tok.Line))
	case "createobject":
		p.sc.NextSkipComments()
		p.parseCreateObjectRef(nameTok.Value, int(tok.Line))
	case "entitynew":
		p.sc.NextSkipComments()
		p.parseEntityNewRef(nameTok.Value, int(tok.Line))
	case "entityload", "entityloadbypk":
		p.sc.NextSkipComments()
		p.parseEntityNewRef(nameTok.Value, int(tok.Line))
	default:
		p.checkVarRHS(nameTok.Value, int(tok.Line))
	}
}

// checkVarRHS handles unrecognized RHS for var declarations: walks dot chain,
// tries resolver match or records pending call.
func (p *scriptParser) checkVarRHS(varName string, line int) {
	rhs := p.sc.PeekSkipComments()

	// `var node = this` holds this component, as `variables.x = this` does
	// in parseBodyScopedVar; `this` is a keyword, so the check below would
	// otherwise drop it. `x = this.y` reads a member and is not this.
	if rhs.Kind == TokIdent && identEq(rhs.Value, "this") {
		st := p.sc.Save()
		p.sc.NextSkipComments()

		if p.sc.PeekSkipComments().Kind != TokDot {
			if selfPath := strings.TrimPrefix(p.fileURI, "file://"); selfPath != "" {
				p.addRef(&ComponentRef{
					Variable: varName, Component: selfPath,
					URI: uriFromString(p.fileURI), Line: conv.Uint32(p.baseLine + line),
				})
			}

			return
		}

		p.sc.Restore(st)
	}

	if rhs.Kind != TokIdent || isKeyword(rhs.Value) {
		return
	}

	p.sc.NextSkipComments() // consume first ident

	var fullChain chainBuilder
	fullChain.reset(rhs.Value)

	prevIdent, lastIdent, lastLine, ok := p.walkChain(&fullChain, rhs, Token{Line: conv.Int32(line)})
	if !ok {
		return
	}

	if p.sc.PeekSkipComments().Kind == TokLParen {
		chainStr := fullChain.String()
		p.recordCallFromChain(chainStr, int(lastLine))

		if comp := p.tryResolveCall(chainStr); comp != "" {
			recv := receiverOf(chainStr)
			rest := p.recordChainContinuation(recv, lastIdent, comp, line)
			p.addRef(&ComponentRef{
				Variable: varName, Component: comp,
				ChainBase: prevIdent, ChainMethod: lastIdent, ChainRest: rest,
				URI: uriFromString(p.fileURI), Line: conv.Uint32(p.baseLine + line),
				Rebinds: rebinds(varName, recv),
			})
		} else if comp, ext := p.tryExtendChain(chainStr); comp != "" {
			rest := p.continueExtendedChain(receiverOf(chainStr), lastIdent, ext, line)
			p.addRef(&ComponentRef{
				Variable: varName, Component: comp, ChainRest: rest,
				URI: uriFromString(p.fileURI), Line: conv.Uint32(p.baseLine + line),
			})
		} else if p.builtinReturnLookup != nil {
			p.recordBuiltinAssignment(varName, prevIdent, lastIdent, chainStr, line)
		} else {
			p.addPendingCall(varName, prevIdent, lastIdent, chainStr, line)
		}
	} else {
		p.assignNonCall(varName, fullChain.String(), line)
	}
}

// parseScopedVar handles: scope.name = expr (local., arguments., this., variables.)
// recordScopedMemberCall records `scope.name(...)` — a call whose whole
// receiver is a scope.
//
// Both scoped-var handlers read `scope` `.` `name` and then looked only for
// `=` (an assignment) or `.` (a longer chain), so the shape where the statement
// *is* the call recorded nothing: `this.init()`, `variables.buildCache()`,
// `request.getRemoteClients()`. `x = request.getRemote()` and
// `request.a.getRemote()` both worked, which is why this survived — it is only
// the bare statement form that was lost, and that is how a component calls its
// own method with an explicit scope.
//
// `this.` and `variables.` name a member of the component being parsed, so the
// call is recorded unqualified and resolves against the file's own functions —
// including the ones `parseFunctionValue` files from `this.helper = function(){}`.
// Every other scope holds a value put there at runtime, so the receiver is
// `$any`: the call site is recorded, and the method-exists check is skipped
// rather than answered wrongly against a same-named function elsewhere.
func (p *scriptParser) recordScopedMemberCall(scopeTok, nameTok Token) {
	caller := ""
	if p.inFunc.in && len(p.funcs) > 0 {
		caller = p.currentFunc().Name
	}

	call := CallSite{
		FuncName: nameTok.Value,
		Line:     conv.Uint32(p.baseLine + int(nameTok.Line)),
		Caller:   caller,
	}

	// The scope is read from the token rather than from the Scope value the
	// dispatch passed: what matters is whether the scope is this component's
	// own (this., variables.) or not, which the token says directly. Testing
	// the enum once recorded request., session. and application. calls as calls
	// to functions of that name in this file, when they were dispatched as
	// ScopeVariables.
	call.onScope(scopeTok.Value)

	p.addCall(&call)

	comp, hop, ok := p.skipParensResolving(nameTok.Value)
	if !ok {
		return
	}

	// A hop chained onto it is a call on what the first one returned, and
	// leaving it to the outer loop makes it an orphaned bare call — the same
	// wrong answer recordChainFromScope exists to stop. A member of this
	// component walks the chain through its declared return type; a dynamic
	// receiver stays dynamic all the way down, as a literal receiver's chain
	// already does.
	if call.Component == "" {
		p.recordChainContinuation(scopeTok.Value, hop, comp, int(scopeTok.Line))

		return
	}

	p.recordChainContinuation(call.Variable, hop, call.Component, int(scopeTok.Line))
}

// recordDynamicChain records the hops chained onto a call whose receiver is
// dynamic, keeping every one of them dynamic. The scanner must sit just past
// the first call's ')'.
// Each hop is on the line its own name is written on.
func (p *scriptParser) recordDynamicChain(recv, caller string) {
	for p.sc.PeekSkipComments().Kind == TokDot {
		p.sc.NextSkipComments() // consume .

		methTok := p.sc.PeekSkipComments()
		if methTok.Kind != TokIdent {
			return
		}

		p.sc.NextSkipComments()

		if p.sc.PeekSkipComments().Kind != TokLParen {
			return
		}

		p.addCall(&CallSite{
			FuncName:  methTok.Value,
			Variable:  recv,
			Component: "$any",
			Line:      conv.Uint32(p.baseLine + int(methTok.Line)),
			Caller:    caller,
			Resolved:  true,
		})

		if !p.skipParens() {
			return
		}
	}
}
