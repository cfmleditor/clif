package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// parseBodyVarDecl handles: var name = expr inside a function body.
func (p *scriptParser) parseBodyVarDecl(varTok Token) {
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
	p.noteAssign(nameTok.Value, int(varTok.Line))

	p.localVarSet.add(nameTok.Value)
	p.vars = append(p.vars, VarDef{
		Name: nameTok.Value, Scope: ScopeLocal,
		Line: conv.Uint32(p.baseLine + int(varTok.Line)),
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
		p.parseNewRef(nameTok.Value, int(varTok.Line))
	case "createobject":
		p.sc.NextSkipComments()
		p.parseCreateObjectRef(nameTok.Value, int(varTok.Line))
	case "entitynew":
		p.sc.NextSkipComments()
		p.parseEntityNewRef(nameTok.Value, int(varTok.Line))
	case "entityload", "entityloadbypk":
		p.sc.NextSkipComments()
		p.parseEntityNewRef(nameTok.Value, int(varTok.Line))
	default:
		// The same right-hand side an assignment outside a function has.
		p.checkVarRHS(nameTok.Value, int(varTok.Line))
	}
}

// parseBodyScopedVar handles: scope.name = expr inside a function body.
func (p *scriptParser) parseBodyScopedVar(scopeTok Token, scope Scope) {
	dot := p.sc.PeekSkipComments()
	if dot.Kind != TokDot {
		// Not "scope.name" at all — e.g. REQUEST[key].method(), indexing the
		// whole scope struct with a dynamic key rather than a dotted name.
		// checkBareCall knows how to skip the "[...]" group and poison the
		// receiver; without this, the scanner is left stuck at "[" and the
		// trailing ".method()" gets rediscovered as an orphaned bare call.
		if dot.Kind == TokLBracket {
			p.checkBareCall(scopeTok)
		}

		return
	}

	p.sc.NextSkipComments() // consume .

	nameTok := p.sc.NextSkipComments()
	if nameTok.Kind != TokIdent {
		return
	}

	if p.sc.PeekSkipComments().Kind != TokEquals {
		p.scopedCall(scopeTok, nameTok)

		return
	}

	p.sc.NextSkipComments() // consume =
	p.noteAssign(nameTok.Value, int(scopeTok.Line))

	p.vars = append(p.vars, VarDef{
		Name: nameTok.Value, Scope: scope,
		Line: conv.Uint32(p.baseLine + int(scopeTok.Line)),
	})

	isLocal := scope == ScopeLocal || scope == ScopeArguments
	if isLocal {
		p.localVarSet.add(nameTok.Value)
	} else {
		p.forceGlobal = true
	}

	p.refThis = scope == ScopeThis
	defer func() { p.refThis = false }()

	if p.chainedAssign(nameTok.Value) {
		p.forceGlobal = false

		return
	}

	// Check RHS for component refs
	rhs := p.rhsAfterLiteral()
	if rhs.Kind == TokIdent {
		var buf foldScratch
		switch string(buf.lowerFold(rhs.Value)) {
		case "new":
			p.sc.NextSkipComments()
			p.parseNewRef(nameTok.Value, int(scopeTok.Line))
		case "createobject":
			p.sc.NextSkipComments()
			p.parseCreateObjectRef(nameTok.Value, int(scopeTok.Line))
		case "entitynew":
			p.sc.NextSkipComments()
			p.parseEntityNewRef(nameTok.Value, int(scopeTok.Line))
		case "entityload", "entityloadbypk":
			p.sc.NextSkipComments()
			p.parseEntityNewRef(nameTok.Value, int(scopeTok.Line))
		case "this":
			p.sc.NextSkipComments()

			// Only treat `scope.x = this` (bare) as a self-ref.
			// `scope.x = this.prop = ...` is a chained assignment — the real
			// component comes from the continuation, so skip the ref here.
			if p.sc.PeekSkipComments().Kind != TokDot {
				if selfPath := strings.TrimPrefix(p.fileURI, "file://"); selfPath != "" {
					p.addRef(&ComponentRef{
						Variable: nameTok.Value, Component: selfPath,
						URI: uriFromString(p.fileURI), Line: conv.Uint32(p.baseLine + int(scopeTok.Line)),
					})
				}
			}
		default:
			if !isKeyword(rhs.Value) {
				p.sc.NextSkipComments()

				var fullChain chainBuilder
				fullChain.reset(rhs.Value)

				prevIdent, lastIdent, lastLine, ok := p.walkChain(&fullChain, rhs, scopeTok)
				if !ok {
					return
				}

				p.assignFromChain(nameTok.Value, &fullChain, prevIdent, lastIdent, int(scopeTok.Line), int(lastLine))
			}
		}
	}

	p.forceGlobal = false
}

func (p *scriptParser) checkAssignRef(tok Token) {
	if isKeyword(tok.Value) && !operatorWordIsName(tok, p.sc.PeekSkipComments()) {
		return
	}

	peek := p.sc.PeekSkipComments()
	if peek.Kind != TokEquals {
		// `for (row in qry)` without `var`. Unscoped, so it is a variables-scope
		// binding, which is CFML's rule for any unscoped assignment and the
		// reason the `var` form is the one to write.
		if peek.Kind == TokIdent && identEq(peek.Value, "in") {
			p.declareVar(tok, ScopeVariables)

			return
		}

		// Check for bare dotted call: obj.method() — also routes a bracket-indexed
		// receiver (obj[key].method()) through checkBareCall, whose own chain-walk
		// knows how to skip the "[...]" group; without TokLBracket here, a bare
		// statement starting with obj[key]... never reaches checkBareCall at all
		// and the scanner is left stuck at "[".
		if peek.Kind == TokDot || peek.Kind == TokLBracket {
			p.checkBareCall(tok)
		}

		return
	}

	// A name = inside a call's arguments is a named argument, and inside a
	// struct literal a key: neither assigns. Judged against the tree-sitter
	// grammar over the corpus, reading them as assignments declared 769
	// variables that were none (.ai/docs/SCANNER-CONSOLIDATION.md). The loop
	// reads on through them as through any expression.
	if !p.atStatementLevel() {
		return
	}

	p.sc.NextSkipComments() // consume =
	p.noteAssign(tok.Value, int(tok.Line))

	// Record the variable
	p.vars = append(p.vars, VarDef{
		Name: tok.Value, Scope: ScopeVariables,
		Line: conv.Uint32(p.baseLine + int(tok.Line)),
	})

	// If inside a function and variable is NOT declared local, route to componentRefs
	if p.inFunc.in && !p.isVarDeclaredLocal(tok.Value) {
		p.forceGlobal = true
	}

	if p.chainedAssign(tok.Value) {
		p.forceGlobal = false

		return
	}

	if p.parseFunctionValue(tok, tok) {
		p.forceGlobal = false

		return
	}

	rhs := p.rhsAfterLiteral()
	if rhs.Kind != TokIdent {
		p.forceGlobal = false

		return
	}

	var buf foldScratch
	switch string(buf.lowerFold(rhs.Value)) {
	case "new":
		p.sc.NextSkipComments()
		p.parseNewRef(tok.Value, int(tok.Line))
	case "createobject":
		p.sc.NextSkipComments()
		p.parseCreateObjectRef(tok.Value, int(tok.Line))
	case "entitynew":
		p.sc.NextSkipComments()
		p.parseEntityNewRef(tok.Value, int(tok.Line))
	case "entityload", "entityloadbypk":
		p.sc.NextSkipComments()
		p.parseEntityNewRef(tok.Value, int(tok.Line))
	default:
		// Check if RHS is a function call: funcName( or someVar.method(
		if !isKeyword(rhs.Value) {
			p.sc.NextSkipComments() // consume first ident

			var fullChain chainBuilder
			fullChain.reset(rhs.Value)

			prevIdent, lastIdent, lastLine, ok := p.walkChain(&fullChain, rhs, tok)
			if !ok {
				return
			}

			p.assignFromChain(tok.Value, &fullChain, prevIdent, lastIdent, int(tok.Line), int(lastLine))
		}
	}

	p.forceGlobal = false
}

// chainedAssign handles a right-hand side that is itself an assignment:
// `outer = b = rhs`, `outer = this.b = rhs` or `outer = variables.b = rhs`.
// TestBox keeps `$assert` in both scopes that way. The inner assignment is
// parsed as the statement it is, and every ref and pending call it makes is
// made again for outer under the state the outer assignment set up, so
// `var a = b = new X()` makes a local a and a variables-scope b. Reports
// whether the right-hand side was one; the scanner has not moved if not.
func (p *scriptParser) chainedAssign(outer string) bool {
	if p.chainDepth >= maxChainedAssignments {
		return false
	}

	// Decided from the peek and the bytes after it, which leave the peek
	// cached for the caller: saving and restoring on every assignment was
	// 4.5% of a script parse, because a restore drops it.
	peek := p.sc.PeekSkipComments()
	if peek.Kind != TokIdent {
		return false
	}

	switch first, second := p.sc.bytesAfterPeek(); {
	case first == '=' && second != '=':
		if isKeyword(peek.Value) {
			return false
		}
	case first == '.' && (identEq(peek.Value, "this") || identEq(peek.Value, "variables")):
	default:
		return false
	}

	st := p.sc.Save()
	tok := p.sc.NextSkipComments()

	scope, ok := p.assignmentAhead(tok)
	if !ok {
		p.sc.Restore(st)

		return false
	}

	forceGlobal, refThis := p.forceGlobal, p.refThis
	nGlobal, nLocal, nPending := len(p.componentRefs), len(p.funcRefs[p.inFunc]), len(p.pendingCalls)

	p.chainDepth++

	switch {
	case scope == ScopeLocal:
		p.checkAssignRef(tok)
	case p.inFunc.in:
		p.parseBodyScopedVar(tok, scope)
	default:
		p.parseScopedVar(tok, scope)
	}

	p.chainDepth--

	made := refsMadeFor(p.componentRefs, nGlobal, p.funcRefs[p.inFunc], nLocal, outer)
	p.forceGlobal, p.refThis = forceGlobal, refThis

	for i := range made {
		p.addRef(&made[i])
	}

	p.pendingCalls = appendPendingFor(p.pendingCalls, nPending, outer, refThis)

	return true
}

// assignmentAhead reports whether tok, just read, starts an assignment:
// `tok =`, or `this.name =` / `variables.name =`, and not `==`, which is two
// `=` tokens. An unscoped target is reported as ScopeLocal, meaning "no
// scope written"; the scanner does not move.
func (p *scriptParser) assignmentAhead(tok Token) (Scope, bool) {
	st := p.sc.Save()
	defer p.sc.Restore(st)

	scope := ScopeLocal

	switch {
	case identEq(tok.Value, "this"):
		scope = ScopeThis
	case identEq(tok.Value, "variables"):
		scope = ScopeVariables
	}

	if scope != ScopeLocal {
		if p.sc.NextSkipComments().Kind != TokDot || p.sc.NextSkipComments().Kind != TokIdent {
			return 0, false
		}
	}

	if p.sc.NextSkipComments().Kind != TokEquals || p.sc.PeekSkipComments().Kind == TokEquals {
		return 0, false
	}

	return scope, true
}

// addPendingCall records `varName = …prevIdent.lastIdent(…)` for
// resolvePendingCalls, which types varName once every function's return type
// is known, carrying on along whatever the chain calls after lastIdent.
func (p *scriptParser) addPendingCall(varName, prevIdent, lastIdent, chain string, line int) {
	// A receiver ending in an index is an element of prevIdent, not
	// prevIdent: `scopes[ k ].get()` says nothing about what scopes holds.
	recv := receiverOf(chain)
	if strings.HasSuffix(recv, "[]") {
		prevIdent = ""
	}

	p.pendingCalls = append(p.pendingCalls, pendingCall{
		varName:    varName,
		funcName:   lastIdent,
		expression: callExpressionAt(p.sc, chain),
		baseVar:    prevIdent,
		line:       conv.Uint32(p.baseLine + line),
		funcKey:    p.inFunc,
		refThis:    p.refThis,
		global:     p.forceGlobal,
		baseScope:  ReceiverRefScope(recv),
		baseLocal:  readThroughLocal(recv) || recv == prevIdent && p.inFunc.in && p.isVarDeclaredLocal(prevIdent) || recv == "" && p.isVarDeclaredLocal(lastIdent),
		rebinds:    rebinds(varName, recv),
	})
	p.pendingCalls[len(p.pendingCalls)-1].rest = p.continueChainCalls(receiverOf(chain), lastIdent, line)

	// An index on the end of the chain reads an element of what the chain
	// returns, which is not what it returns: the assignment is untyped.
	if p.sc.PeekSkipComments().Kind == TokLBracket {
		last := &p.pendingCalls[len(p.pendingCalls)-1]
		last.rest = append(last.rest, ElementHop)
	}
}

// scopedChainCall records `scope.name.a.b(…)`, a call on a scoped variable
// that is a statement rather than an assignment.
// scopedCall records the call a `scope.name` begins when it is not an
// assignment: `scope.name(…)` directly on the scope, or a chain on a value it
// holds, `scope.name.method(…)`. The scanner must sit just past the name.
func (p *scriptParser) scopedCall(scopeTok, nameTok Token) {
	switch p.sc.PeekSkipComments().Kind {
	case TokLParen:
		p.recordScopedMemberCall(scopeTok, nameTok)
	case TokDot:
		p.scopedChainCall(scopeTok, nameTok)
	default:
	}
}

func (p *scriptParser) scopedChainCall(scopeTok, nameTok Token) {
	var fullChain chainBuilder
	fullChain.reset(scopeTok.Value)
	fullChain.writeDot()
	fullChain.writeString(nameTok.Value)

	// The call is on the line of the name it is made on, the last one walked:
	// a chain written a name to a line is not on its first line.
	callLine := nameTok.Line

	for {
		switch p.sc.PeekSkipComments().Kind {
		case TokLBracket:
			// See walkChain for why.
			if !p.skipBracketIndex() {
				return
			}

			fullChain.writeString("[]")

			continue
		case TokDot:
			p.sc.NextSkipComments()

			next := p.sc.PeekSkipComments()
			if next.Kind == TokIdent {
				p.sc.NextSkipComments()

				fullChain.writeDot()
				fullChain.writeString(next.Value)

				callLine = next.Line

				continue
			}
		default:
		}

		break
	}

	switch p.sc.PeekSkipComments().Kind {
	case TokLParen:
		p.recordChainFromScope(fullChain.String(), int(callLine))
	case TokEquals:
		if chain := strings.Split(fullChain.String(), "."); len(chain) > 2 {
			p.checkMemberSet(chain, int(scopeTok.Line))
		}
	default:
	}
}

// assignFromChain types varName from the chain on the right of its `=`: a
// call's resolved component, else a pending call for resolvePendingCalls; a
// whole argument value or configured expression when it is not a call.
func (p *scriptParser) assignFromChain(varName string, c *chainBuilder, prevIdent, lastIdent string, line, callLine int) {
	if p.sc.PeekSkipComments().Kind != TokLParen {
		p.assignNonCall(varName, c.String(), line)

		return
	}

	chainStr := c.String()
	p.recordCallFromChain(chainStr, callLine)

	if comp := p.tryResolveCall(chainStr); comp != "" {
		recv := receiverOf(chainStr)
		rest := p.recordChainContinuation(recv, lastIdent, comp, line)
		p.addRef(&ComponentRef{
			Variable: varName, Component: comp,
			ChainBase: prevIdent, ChainMethod: lastIdent, ChainRest: rest,
			URI: uriFromString(p.fileURI), Line: conv.Uint32(p.baseLine + line),
			Rebinds: rebinds(varName, recv),
		})

		return
	}

	p.addPendingCall(varName, prevIdent, lastIdent, chainStr, line)
}

// argumentComponent is the declared type of a whole arguments.name value.
// Copying it into variables scope preserves the constructor's dependency;
// typing the parameter itself at component level instead leaked to siblings.
func (p *scriptParser) argumentComponent(chain string) string {
	return wholeArgumentComponent(chain, p.inFunc, p.funcs)
}

// assignNonCall handles a whole argument value or a configured non-call RHS.
func (p *scriptParser) assignNonCall(varName, chain string, line int) {
	comp := ""
	if p.atStatementEnd() {
		comp = p.argumentComponent(chain)
	}

	if comp == "" && len(p.resolvers) > 0 {
		comp = p.resolveCall(chain)
	}

	if comp != "" {
		p.addRef(&ComponentRef{
			Variable: varName, Component: comp,
			URI: uriFromString(p.fileURI), Line: conv.Uint32(p.baseLine + line),
		})
	}
}

// walkChain walks the `.name` hops after the identifier first, already
// consumed and already in c, writing each into c. A bracket index is skipped
// and marked "[]": a dynamic key cannot be resolved statically, and the marker
// can never match a real resolver or ComponentRef, so resolution fails rather
// than being misattributed to whatever the bare base identifier resolves to
// elsewhere. A `::` makes what was walked a component, not a receiver: the
// static call is recorded against staticTok (see parseStaticCall) and the
// walk reports the statement consumed, as does a bracket index that does not
// close. Otherwise it reports the last two identifiers walked and the line the
// last one is on, which is the line of a call made on it: a chain written a
// name to a line is not on its first line.
func (p *scriptParser) walkChain(c *chainBuilder, first Token, staticTok Token) (prevIdent, lastIdent string, lastLine int32, ok bool) {
	lastIdent = first.Value
	lastLine = first.Line

	for {
		switch p.sc.PeekSkipComments().Kind {
		case TokLBracket:
			if !p.skipBracketIndex() {
				return prevIdent, lastIdent, lastLine, false
			}

			c.writeString("[]")
		case TokDoubleColon:
			p.recordStaticCall(c.String(), staticTok)

			return prevIdent, lastIdent, lastLine, false
		case TokDot:
			p.sc.NextSkipComments() // consume .

			next := p.sc.PeekSkipComments()
			if next.Kind != TokIdent {
				return prevIdent, lastIdent, lastLine, true
			}

			p.sc.NextSkipComments()

			prevIdent = lastIdent
			lastIdent = next.Value
			lastLine = next.Line

			c.writeDot()
			c.writeString(next.Value)
		default:
			return prevIdent, lastIdent, lastLine, true
		}
	}
}

// checkMemberSet records `a.m = …`, a value stored on a member of a variable,
// when the scanner is at the `=`. fw1's tests give a bean a method at run
// time — `a.getVariables = getVariables;` — and then call it (see
// ParseResult.AssignsMember). `a.m == x` is a comparison, and
// `a[ k ].m = …` is on an element, not on a.
//
// It travels as a pendingCall marked memberSet, which is already keyed by
// function, rekeyed and merged per region, and resolvePendingCalls files it
// in ParseResult.memberSets. A slice of its own put scriptParser in a larger
// size class, and a ref for `a.m` reached every other file through the index.
func (p *scriptParser) checkMemberSet(identChain []string, line int) {
	if len(identChain) < 2 || strings.HasSuffix(identChain[len(identChain)-2], "[]") ||
		p.sc.PeekSkipComments().Kind != TokEquals {
		return
	}

	if first, _ := p.sc.bytesAfterPeek(); first == '=' {
		return
	}

	p.pendingCalls = append(p.pendingCalls, pendingCall{
		varName:   strings.Join(identChain[:len(identChain)-1], "."),
		funcName:  identChain[len(identChain)-1],
		line:      conv.Uint32(p.baseLine + line),
		funcKey:   p.inFunc,
		memberSet: true,
	})
}

// checkBareCall handles obj.method() calls (not in assignment context),
// including further .method() calls chained off the return value of a previous
// call in the same chain (e.g. "kpg.generateKeyPair().getPublic().getParams()").
// Records one CallSite per hop when extractCalls is enabled. The scanner
// position is on the first ident (the variable/scope prefix); peek is a dot.
func (p *scriptParser) checkBareCall(tok Token) {
	// Walk the leading dot chain of plain identifiers: a.b.c
	identChain := []string{tok.Value}
	callLine := tok.Line

chainWalk:
	for {
		switch p.sc.PeekSkipComments().Kind {
		case TokLBracket:
			// Dynamic key (e.g. REQUEST['a' & b & 'c'].method()) — skip it and
			// poison the receiver so it can't be misattributed to whatever
			// the bare base identifier resolves to elsewhere. See
			// checkVarRHS's identical case for the full rationale.
			if !p.skipBracketIndex() {
				return
			}

			identChain[len(identChain)-1] += "[]"
		case TokDot:
			p.sc.NextSkipComments() // consume .

			next := p.sc.PeekSkipComments()
			if next.Kind != TokIdent {
				break chainWalk
			}

			p.sc.NextSkipComments()

			identChain = append(identChain, next.Value)
			callLine = next.Line
		case TokDoubleColon:
			// `models.Foo::bar()`. Everything walked so far is the component,
			// not a receiver — see parseStaticCall.
			p.recordStaticCall(strings.Join(identChain, "."), tok)

			return
		default:
			break chainWalk
		}
	}

	// Must end with ( to be a call
	if p.sc.PeekSkipComments().Kind != TokLParen {
		p.checkMemberSet(identChain, int(tok.Line))

		return
	}

	// The last identifier is the method; everything before it is the receiver variable.
	funcName := identChain[len(identChain)-1]
	varName := strings.Join(identChain[:len(identChain)-1], ".")

	caller := ""
	if p.inFunc.in && len(p.funcs) > 0 {
		caller = p.currentFunc().Name
	}

	call := CallSite{
		FuncName: funcName,
		Variable: varName,
		Line:     conv.Uint32(p.baseLine + int(callLine)),
		Caller:   caller,
	}

	p.setCallSpan(&call)

	comp, hop, ok := p.skipParensResolving(strings.Join(identChain, "."))
	if !ok {
		return
	}

	// A call made directly on a scope, reached here from an argument list —
	// `f( server.getTestService() )` — is the call the statement
	// `server.getTestService()` is.
	call.onScope(varName)

	p.addCall(&call)

	// Continue walking further .method() hops chained off this call's return
	// value. Each hop's CallSite keeps Variable pointing at the original
	// receiver and accumulates the intermediate method names in Chain, so
	// CanResolveCall can walk the receiver's type through each call.
	p.recordChainContinuation(varName, hop, comp, int(tok.Line))
}

func (p *scriptParser) parseNewRef(varName string, line int) {
	component := p.readNewComponent()

	var hops []string

	// Consume constructor args and handle any chained .method() calls. The
	// args are consumed even when nothing resolved (an unmapped `new java:…`
	// with no javaStubsPath, say), or the scanner would be left sitting on
	// the "(" and the rest of the statement would be read as its own.
	if p.sc.PeekSkipComments().Kind == TokLParen {
		p.skipParens()

		if component != "" {
			hops = p.scanChainedCalls(component)
		}
	}

	if component != "" {
		p.addRef(&ComponentRef{
			Variable: varName, Component: component, ChainRest: hops,
			URI: uriFromString(p.fileURI), Line: conv.Uint32(p.baseLine + line),
		})
	}
}

func (p *scriptParser) parseCreateObjectRef(varName string, line int) {
	comp := p.readCreateObjectComponent()

	// Consume closing ) and handle any chained .method() calls
	if p.sc.PeekSkipComments().Kind == TokRParen {
		p.sc.NextSkipComments()
	}

	if comp != "" {
		hops := p.scanChainedCalls(comp)
		p.addRef(&ComponentRef{
			Variable: varName, Component: comp, ChainRest: hops,
			URI: uriFromString(p.fileURI), Line: conv.Uint32(p.baseLine + line),
		})
	}
}

// scanChainedCalls records resolved CallSites for any .method() chains following
// a constructor or factory call whose component is already known, and returns
// the method names in order, for an assignment to type its variable by.
//
// Each hop carries the hops before it in Chain. It used to give every one the
// constructed component and nothing else, so in
// createObject("java", "FirebaseOptions").builder().setCredentials(c) the
// setCredentials call was checked against FirebaseOptions rather than against
// what builder() returns. A hop is on the line its name is written on, not the
// line the constructor began on.
func (p *scriptParser) scanChainedCalls(component string) []string {
	caller := ""
	if p.inFunc.in && len(p.funcs) > 0 {
		caller = p.currentFunc().Name
	}

	// hops is both what the caller's ref walks and the chain carried by each
	// CallSite (each call with its arguments).
	var hops []string

	for p.sc.PeekSkipComments().Kind == TokDot {
		p.sc.NextSkipComments() // consume .

		methTok := p.sc.PeekSkipComments()
		if methTok.Kind != TokIdent {
			break
		}

		p.sc.NextSkipComments()

		if p.sc.PeekSkipComments().Kind != TokLParen {
			hops = append(hops, PropertyHop(methTok.Value))

			continue
		}

		p.addCall(&CallSite{
			FuncName:  methTok.Value,
			Component: component,
			Chain:     chainSoFar(hops),
			Line:      conv.Uint32(p.baseLine + int(methTok.Line)),
			Caller:    caller,
			Resolved:  true,
		})

		hops = append(hops, callHopAt(p.sc, methTok.Value))

		p.skipParens()
	}

	return hops
}

func (p *scriptParser) parseEntityNewRef(varName string, line int) {
	lp := p.sc.NextSkipComments()
	if lp.Kind != TokLParen {
		return
	}

	arg := p.sc.NextSkipComments()
	if arg.Kind != TokString {
		return
	}

	comp := p.entityArg(arg)
	if comp != "" {
		p.addRef(&ComponentRef{
			Variable: varName, Component: comp,
			URI: uriFromString(p.fileURI), Line: conv.Uint32(p.baseLine + line),
		})
	}
}
