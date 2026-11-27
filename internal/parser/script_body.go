package parser

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// parseBody processes { ... } or ; for a function body, extracting refs.
// Sets inFunc/localVarSet on entry, clears on exit. Returns the line of the
// closing token, and the region offsets of the body's braces ({-1, -1} when
// it has no closed brace body; {-1, offset} for a declaration ended by the
// `;` at offset).
func (p *scriptParser) parseBody(funcLine int, args []Argument) (int, [2]int) {
	tok := p.sc.PeekSkipComments()
	if tok.Kind == TokSemicolon {
		t := p.sc.NextSkipComments()

		return int(t.Line), [2]int{-1, t.Offset}
	}

	if tok.Kind != TokLBrace {
		return int(tok.Line), [2]int{-1, -1}
	}

	p.sc.NextSkipComments() // consume {

	return p.parseBraceBody(funcLine, args, tok)
}

// parseBraceBody is parseBody's work on a body whose `{`, tok, has been
// consumed.
func (p *scriptParser) parseBraceBody(funcLine int, args []Argument, tok Token) (int, [2]int) {
	noBody := [2]int{-1, -1}
	braces := [2]int{tok.Offset, -1}

	// The body is a block: what it assigns runs before a return in it.
	if p.flow != nil {
		p.flow.open(tok.Offset, "")
		defer p.flow.close("")
	}

	// Enter function scope with a temporary key (will fix up after finding end)
	prevInFunc := p.inFunc
	prevLocalVarSet := p.localVarSet
	prevReturnVar, prevReturnLine := p.returnVar, p.returnLine
	prevBodyFunc := p.bodyFunc
	p.bodyFunc = len(p.funcs)

	// Use a placeholder funcKey; we'll remap after finding the real end
	tempKey := funcKey(funcLine, funcLine)
	p.inFunc = tempKey
	pendingFrom := len(p.pendingCalls)
	p.returnVar = ""

	if prevLocalVarSet == nil {
		root := &p.ensureExtras().rootLocals
		root.reset()
		p.localVarSet = root
	} else {
		p.localVarSet = &localNames{}
	}

	for _ai := range args {
		a := &args[_ai]
		p.localVarSet.add(a.Name)

		if comp := argumentComponentType(a); comp != "" {
			if p.funcRefs == nil {
				p.funcRefs = make(map[fnKey][]ComponentRef)
			}

			p.funcRefs[tempKey] = append(p.funcRefs[tempKey], ComponentRef{
				Variable: a.Name, Component: comp,
				URI: uriFromString(p.fileURI), Line: conv.Uint32(funcLine),
			})
		}
	}

	depth := 1

	var endLine int

	// A body's statements start at its own brace, whatever surrounds it.
	prevKinds, prevDepth, prevFor := p.openKinds, p.openDepth, p.afterFor
	p.openKinds, p.openDepth, p.afterFor = 0, 0, false

	defer func() { p.openKinds, p.openDepth, p.afterFor = prevKinds, prevDepth, prevFor }()

	for depth > 0 {
		t := p.sc.NextSkipComments()
		if t.Kind == TokEOF {
			p.inFunc = prevInFunc
			p.localVarSet = prevLocalVarSet
			p.returnVar, p.returnLine = prevReturnVar, prevReturnLine
			p.bodyFunc = prevBodyFunc

			if p.flow != nil {
				p.flow.dropBraceless()
			}

			return int(t.Line), noBody
		}

		if p.flow != nil && len(p.flow.ends) > 0 {
			p.flow.closeEnded(t.Offset)
		}

		afterLT := p.afterLT
		p.afterLT = t.Kind == TokLT

		p.trackOpen(t)

		switch t.Kind {
		case TokLBrace:
			depth++

			if p.flow != nil {
				p.flow.open(t.Offset, "")
			}
		case TokRBrace:
			depth--
			if depth == 0 {
				endLine = int(t.Line)
				braces[1] = t.Offset
			} else if p.flow != nil {
				p.flow.close("")
			}
		case TokIdent:
			if depth > 0 {
				if p.flow != nil {
					p.openBracelessBody(t)
				}

				mark := p.flowMark()
				p.handleBodyToken(t, depth, afterLT)
				p.stampFlow(mark)
			}
		case TokString, TokRBracket:
			if depth > 0 {
				mark := p.flowMark()
				p.handleLiteralToken(t)
				p.stampFlow(mark)
			}
		default:
			// Any other token is passed over.
		}
	}

	realKey := funcKey(funcLine, p.baseLine+endLine)
	p.rekeyFunc(tempKey, realKey, pendingFrom)

	p.inFunc = realKey

	// Resolve ReturnComponent on the current function
	if len(p.funcs) > 0 {
		p.settleReturnComponent(p.currentFunc())
	}

	// Exit function scope
	p.inFunc = prevInFunc
	p.localVarSet = prevLocalVarSet
	p.returnVar, p.returnLine = prevReturnVar, prevReturnLine
	p.bodyFunc = prevBodyFunc

	return endLine, braces
}

// openBracelessBody opens a block for the statement a braceless `if`, `else`,
// `for` or `while` governs, so an assignment in it is read as one that may not
// run. It acts only when the whole statement is one line ending in a
// semicolon with no brace in it: CFScript needs neither, and a statement it
// cannot bound is left as it was read before, as one that always runs. A
// lookahead on a saved scanner state; the tokens are read again.
func (p *scriptParser) openBracelessBody(t Token) {
	if len(t.Value) < 2 || len(t.Value) > 6 {
		return
	}

	var buf foldScratch

	withCondition := false

	switch string(buf.lowerFold(t.Value)) {
	case "if", "elseif", "for", "while":
		withCondition = true
	case "else":
	default:
		return
	}

	saved := p.sc.Save()
	defer p.sc.Restore(saved)

	if withCondition {
		if p.sc.NextSkipComments().Kind != TokLParen {
			return
		}

		for depth := 1; depth > 0; {
			switch p.sc.NextSkipComments().Kind {
			case TokLParen:
				depth++
			case TokRParen:
				depth--
			case TokEOF:
				return
			default:
			}
		}
	}

	first := p.sc.NextSkipComments()
	switch first.Kind {
	case TokLBrace, TokRBrace, TokSemicolon, TokEOF:
		return
	default:
	}

	if !withCondition && first.Kind == TokIdent && strings.EqualFold(first.Value, "if") {
		return // `else if` is the if's block
	}

	for tok, depth := first, 0; ; tok = p.sc.NextSkipComments() {
		if tok.Line != first.Line {
			return
		}

		switch tok.Kind {
		case TokLParen, TokLBracket:
			depth++
		case TokRParen, TokRBracket:
			depth--
		case TokLBrace, TokRBrace, TokEOF:
			if depth <= 0 {
				return
			}
		case TokSemicolon:
			if depth == 0 {
				p.flow.openBraceless(t.Offset, tok.Offset)

				return
			}
		default:
		}
	}
}

// flowMark is how many pending calls there were before a statement, so
// stampFlow can find the ones it made. A ref notes its block itself, in
// addRef; a pending call is made at a dozen sites and its ref only later.
func (p *scriptParser) flowMark() int { return len(p.pendingCalls) }

// stampFlow records the innermost open block as the one the pending calls
// made since mark were made in.
func (p *scriptParser) stampFlow(mark int) {
	stampPending(p.flow, p.pendingCalls[mark:])
}

// stampPending gives calls the innermost block open in fb.
func stampPending(fb *flowBlocks, calls []pendingCall) {
	if fb == nil || len(calls) == 0 {
		return
	}

	b := fb.innermost()
	for i := range calls {
		calls[i].block = b
	}
}

// sharedScopeOf is the scope a request., session., application. or server.
// keyword names. Only called from the dispatch arms for those four.
func sharedScopeOf(tok Token) Scope {
	scope, _ := ScopeForPrefix(tok.Value)

	return scope
}

// rekeyFunc moves what a function body recorded under the placeholder key it
// was parsed with to its real key, once its end is known.
//
// Only the pending calls recorded since the body began can carry tempKey, so
// the search starts at pendingFrom: scanning them all for every function was
// quadratic in a component of thousands of functions.
func (p *scriptParser) rekeyFunc(tempKey, realKey fnKey, pendingFrom int) {
	if refs, ok := p.funcRefs[tempKey]; ok {
		delete(p.funcRefs, tempKey)
		p.funcRefs[realKey] = refs
	}

	if calls, ok := p.funcCalls[tempKey]; ok {
		delete(p.funcCalls, tempKey)
		p.funcCalls[realKey] = calls
	}

	for i := pendingFrom; i < len(p.pendingCalls); i++ {
		if p.pendingCalls[i].funcKey == tempKey {
			p.pendingCalls[i].funcKey = realKey
		}
	}
}

// settleReturnComponent gives f the component its return variable holds,
// from a ref in its body or at component level, or records the variable for
// resolvePendingCalls to settle later.
func (p *scriptParser) settleReturnComponent(f *FunctionDef) {
	if f.ReturnComponent != "" || p.returnVar == "" {
		return
	}

	// Look up returnVar in this function's refs, then in componentRefs. A
	// name read through variables. or this. is the component's alone, and
	// only a ref made through that scope holds it. Of the refs for it, the
	// ones that may reach the return decide; a closure's local of the same
	// name is another variable.
	name, called, scope := returnedVar(p.returnVar)

	lookIn := [][]ComponentRef{p.funcRefs[p.inFunc], p.componentRefs}
	admit := func(ref *ComponentRef) bool { return ref.VisibleTo == 0 }

	if scope != RefAny {
		lookIn = lookIn[1:]
		admit = scope.Admits
	}

	for _, refs := range lookIn {
		reaching := p.flow.reaching(refs, name, p.returnLine, admit)
		if len(reaching) == 0 {
			continue
		}

		// A ref still to be typed, or an assignment among those reaching the
		// return that only resolvePendingCalls can type, is settled there.
		// Two branches assigning two components return either, and a
		// branch assigning something untyped makes the return dynamic.
		if !slices.ContainsFunc(reaching, chainPending) && !assignedBetween(p.pendingCalls, name, reaching[0].Line, p.returnLine) {
			comp := agreedComponent(reaching, func(ref *ComponentRef) string { return ref.Component })
			if comp != "" && p.flow.untypedReaches(reaching, name, f.Line, p.returnLine) {
				comp, f.ReturnsUntyped = "$any", true
			}

			f.ReturnComponent = returnedComponent(comp, called)
		}

		break
	}

	if f.ReturnComponent != "" {
		return
	}

	// If still unresolved, store for deferred resolution
	f.returnVar, f.returnLine = p.returnVar, p.returnLine
}

// handleBodyToken processes an identifier inside a function body.
func (p *scriptParser) handleBodyToken(tok Token, depth int, afterLT bool) {
	var buf foldScratch
	switch string(buf.lowerFold(tok.Value)) {
	case "var":
		p.parseBodyVarDecl(tok)
	case "static", "final":
		peek := p.sc.PeekSkipComments()
		switch {
		case peek.Kind == TokIdent && identEq(peek.Value, "var"):
			varTok := p.sc.NextSkipComments()
			p.parseBodyVarDecl(varTok)
		case peek.Kind == TokIdent:
			p.parseBodyVarDecl(tok)
		case peek.Kind == TokDoubleColon:
			p.parseStaticCall(tok)
		case peek.Kind == TokLParen:
			p.recordBareCallAndChain(tok)
		default:
			p.checkAssignRef(tok)
		}
	case "local":
		p.parseBodyScopedVar(tok, ScopeLocal)
	case "arguments":
		p.parseBodyScopedVar(tok, ScopeArguments)
	case "variables":
		p.parseBodyScopedVar(tok, ScopeVariables)
	case "this":
		p.parseBodyScopedVar(tok, ScopeThis)
	case "request", "session", "application", "server":
		// Request/session/application/server-scoped assignments (e.g. "REQUEST.generator =
		// document.createTable(1);") — checkAssignRef's default path only recognizes
		// a bare "x = ..." (identifier directly followed by "="); for a scope-prefixed
		// LHS the next token is "." not "=", so without this case it falls through to
		// checkBareCall and the assignment (and any component type it establishes) is
		// silently dropped. parseBodyScopedVar already handles the "scope.name = rhs"
		// vs "scope.name.method()" split correctly for variables./this./arguments.; the
		// same handling applies verbatim here. Treated as global (forceGlobal), same as
		// this./variables., since these scopes outlive the current function.
		//
		// The declaration keeps its own scope. It used to be recorded as
		// ScopeVariables, which put `application.cache` among the component's
		// variables and left go-to-definition on `application.cache` nothing to
		// find in a script-syntax Application.cfc. Only the scope is recorded
		// differently: the component ref, the forced-global filing and the call
		// extraction are the same for either value.
		p.parseBodyScopedVar(tok, sharedScopeOf(tok))
	case "return":
		if p.inClosure {
			// A closure's return is the closure's, not the enclosing
			// function's: its call is recorded, its type is not the
			// function's return type.
			if peek := p.sc.PeekSkipComments(); peek.Kind == TokIdent {
				_, _ = p.returnCall(peek)
			}

			return
		}

		p.checkReturnComponent()
	case "new":
		p.parseStandaloneNew()
	case "function", "public", "private", "remote", "package":
		// Nested function — skip its entire body
		p.skipNestedFunction(tok, depth)
	case "catch":
		p.parseCatchVar()
	case "throw":
		p.consumeThrowArgs()
	default:
		peek := p.sc.PeekSkipComments()
		if peek.Kind == TokDoubleColon {
			p.parseStaticCall(tok)

			return
		}

		if !isKeyword(tok.Value) && peek.Kind == TokLParen {
			p.recordBareCallAndChain(tok)

			return
		}

		if !afterLT && peek.Kind == TokIdent && !isKeyword(tok.Value) && looksLikeTagAttrs(p.sc) {
			p.parseScriptTagAttrs()

			return
		}

		p.checkAssignRef(tok)
	}
}

// checkReturnComponent checks if a return statement returns a component expression or variable.
func (p *scriptParser) checkReturnComponent() {
	expr := scriptReturnExpression(p.sc)
	if p.inFunc.in {
		p.pendingCalls = append(p.pendingCalls, pendingCall{
			varName: expr, funcKey: p.inFunc, returnExpr: true,
		})
	}

	// `return flag ? new A() : new B()` returns one or the other, not flag,
	// and so does `return ( flag ? … )`, which starts with no name.
	if comp := conditionalConstructions(expr, p.resolveCall, p.applyImport); comp != "" && len(p.funcs) > 0 {
		f := p.currentFunc()
		f.ReturnComponent = unionReturn(f.ReturnComponent, comp)
	}

	// A returned mock is the mock the same call assigned would be
	// (dynamicCall). MockBox declares no return for createMock, so a chain
	// ending in one, `return getMockBox().createMock( object=new X() )`,
	// said nothing, where `var x = …` of the same call is dynamic.
	if mock := MockCall(expr); mock != "" && len(p.funcs) > 0 {
		f := p.currentFunc()
		f.ReturnComponent = unionReturn(f.ReturnComponent, mock)
	}

	peek := p.sc.PeekSkipComments()
	if peek.Kind != TokIdent {
		return
	}

	var comp string

	var buf foldScratch
	switch string(buf.lowerFold(peek.Value)) {
	case "new":
		p.sc.NextSkipComments()

		comp = p.readNewComponent()
		if p.sc.PeekSkipComments().Kind == TokLParen {
			p.skipParens()
		}

		p.scanChainedCalls(comp)
	case "createobject":
		p.sc.NextSkipComments()

		comp = p.readCreateObjectComponent()
		if p.sc.PeekSkipComments().Kind == TokRParen {
			p.sc.NextSkipComments()
		}

		p.scanChainedCalls(comp)
	case "entitynew":
		p.sc.NextSkipComments()

		comp = p.readEntityNewComponent()
		if p.sc.PeekSkipComments().Kind == TokRParen {
			p.sc.NextSkipComments()
		}

		p.scanChainedCalls(comp)
	case "entityloadbypk":
		// The entity named by the first argument, or none; the primary key
		// after it is read as the argument list it is.
		p.sc.NextSkipComments()

		comp = p.readEntityNewComponent()
		if p.sc.PeekSkipComments().Kind == TokComma {
			p.skipParenBody()
		} else if p.sc.PeekSkipComments().Kind == TokRParen {
			p.sc.NextSkipComments()
		}

		p.scanChainedCalls(comp)
	default:
		bareThis := identEq(peek.Value, "this") && p.returnsBareThis()
		scopedVar, scope := p.returnedScopedVar()

		// return varName — track for resolution after body parse. Only the
		// name alone is what the variable holds: `return shell.pwd()`
		// returns what pwd() does, and read as `return shell` it declared the
		// function a Shell. A call on a dynamic value is dynamic, though,
		// which returnsCallOn keeps.
		p.returnVar = peek.Value
		p.returnLine = conv.Uint32(p.baseLine + int(peek.Line))

		if p.flow != nil {
			p.flow.noteReturn(p.returnLine)
		}

		bare, call := p.returnCall(peek)
		if !bare {
			p.returnVar = returnsCallOn + peek.Value
		}

		// A bare call standing alone is typed as it would be on an
		// assignment's right-hand side: `return getInstance( "X" )` returns
		// what a componentResolver says getInstance( "X" ) is.
		if call != "" && p.atStatementEnd() {
			if comp := p.resolveCall(call); comp != "" && len(p.funcs) > 0 {
				f := p.currentFunc()
				f.ReturnComponent = unionReturn(f.ReturnComponent, comp)
			}
		}

		switch {
		case bareThis:
			p.returnVar = returnsThis
		case scopedVar != "":
			p.returnVar = scopedReturnVar(scope, scopedVar)
		}

		return
	}

	if comp != "" && len(p.funcs) > 0 {
		f := p.currentFunc()
		f.ReturnComponent = unionReturn(f.ReturnComponent, comp)
	}
}

// currentFunc is the function whose body is being parsed. It is not always
// the last one declared: a named function nested in the body is filed after
// it. Without a body entered (a script island in a tag function) it is the
// last. Callers check p.funcs is not empty.
func (p *scriptParser) currentFunc() *FunctionDef {
	if p.bodyFunc > 0 && p.bodyFunc <= len(p.funcs) {
		return &p.funcs[p.bodyFunc-1]
	}

	return &p.funcs[len(p.funcs)-1]
}

// atStatementEnd reports whether the next token ends a statement.
func (p *scriptParser) atStatementEnd() bool {
	switch p.sc.PeekSkipComments().Kind {
	case TokSemicolon, TokRBrace, TokEOF:
		return true
	default:
		return false
	}
}

// returnedScopedVar is the name a return reads through variables. or this.
// when that is the whole returned expression — `return variables.print;` —
// and the scope it names, without moving the scanner. It is "" for anything
// else, `return variables.print.line()` and `return this;` included.
func (p *scriptParser) returnedScopedVar() (string, RefScope) {
	saved := p.sc.Save()
	defer p.sc.Restore(saved)

	scopeTok := p.sc.NextSkipComments()

	scope := RefVariables
	if identEq(scopeTok.Value, "this") {
		scope = RefThis
	} else if !identEq(scopeTok.Value, "variables") {
		return "", RefAny
	}

	if p.sc.NextSkipComments().Kind != TokDot {
		return "", RefAny
	}

	name := p.sc.NextSkipComments()
	if name.Kind != TokIdent {
		return "", RefAny
	}

	if !p.atStatementEnd() {
		return "", RefAny
	}

	return name.Value, scope
}

// returnsBareThis reports whether the `this` the scanner is on is the whole
// returned expression — `return this;` rather than `return this.x;` —
// without moving the scanner.
func (p *scriptParser) returnsBareThis() bool {
	saved := p.sc.Save()
	defer p.sc.Restore(saved)

	p.sc.NextSkipComments() // this

	return p.atStatementEnd()
}

// returnCall records the call a `return` statement makes, through the same
// walkers a statement uses: `return svc.a().b()` and `return f().g()` carry
// the receiver to every hop, and `return variables.f()` is recorded as the
// statement `variables.f()` is. It had a chain walk of its own that recorded
// the first call and left the rest to the outer loop, which met `.b()` with
// nothing before it and recorded a bare call to `b`.
//
// Only walkers that record calls are reached, never the statement dispatch
// itself: a return holds an expression, and `return x == 1` read as a
// statement is the assignment `x = …`. tok is the peeked first token after
// `return`; a keyword other than a scope is left for the caller's loop.
// It reports whether the name stood alone, the case in which the function
// returns what that variable holds, and for a bare call standing alone the
// expression the componentResolvers are offered for it.
func (p *scriptParser) returnCall(tok Token) (bare bool, call string) {
	_, _, isScope := scopeReceiver(tok.Value)
	if !isScope && isKeyword(tok.Value) {
		return true, ""
	}

	p.sc.NextSkipComments()

	next := p.sc.PeekSkipComments()

	switch {
	case isScope && next.Kind == TokDot:
		p.sc.NextSkipComments()

		nameTok := p.sc.PeekSkipComments()
		if nameTok.Kind != TokIdent {
			return false, ""
		}

		p.sc.NextSkipComments()
		p.scopedCall(tok, nameTok)
	case next.Kind == TokLParen:
		call = p.recordBareCallAndChain(tok)
	case next.Kind == TokDot || next.Kind == TokLBracket:
		p.checkBareCall(tok)
	case next.Kind == TokDoubleColon:
		p.parseStaticCall(tok)
	default:
		return true, ""
	}

	return false, call
}
