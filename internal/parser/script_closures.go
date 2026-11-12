// Nested named functions and the bodies of closures.

package parser

import (
	"maps"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// skipNestedFunction handles a function met inside another function's body or
// inside a group. An anonymous one is a value and only its body is scanned; a
// named one is declared, for the reason the named branch gives.
func (p *scriptParser) skipNestedFunction(tok Token, _ int) {
	// Handle access modifier before function keyword
	if !identEq(tok.Value, "function") {
		next := p.sc.PeekSkipComments()
		if next.Kind != TokIdent {
			return
		}

		if identEq(next.Value, "function") {
			p.sc.NextSkipComments()
		} else {
			// access returnType function
			p.sc.NextSkipComments()

			next2 := p.sc.PeekSkipComments()
			if next2.Kind == TokIdent && identEq(next2.Value, "function") {
				p.sc.NextSkipComments()
			} else {
				return
			}
		}
	}
	// Skip name (or handle anonymous function)
	nameTok := p.sc.NextSkipComments()
	if nameTok.Kind == TokLParen {
		// Anonymous function: function() { ... }. The '(' is already consumed.
		// parseArgList records the calls an argument default holds, and the
		// names are the closure's parameters.
		args := p.parseArgList()
		p.scanNestedFunctionBody(argNames(args))

		return
	}

	if nameTok.Kind != TokIdent {
		return
	}

	lp := p.sc.NextSkipComments()
	if lp.Kind != TokLParen {
		return
	}

	// A *named* nested function is a declaration, not a value. CFML hoists it
	// into the enclosing component's variables scope, which is what lets
	// `function run(){ setup(); function setup(){…} }` call it before the line
	// it is written on — and the tag parser has always recorded it, so the two
	// syntaxes disagreed about the same code. Without it there is no index
	// entry, no completion and nowhere for go-to-definition to land.
	//
	// The argument list goes through parseArgList rather than scanParenBody so
	// the signature reaches signature help; both record the calls an argument
	// default holds, since parseArgList routes one through skipDefault.
	args, closing, defaults := p.parseArgs()
	funcLine := p.baseLine + int(tok.Line)

	p.funcs = append(p.funcs, FunctionDef{
		Name:      nameTok.Value,
		URI:       uriFromString(p.fileURI),
		Line:      conv.Uint32(funcLine),
		Arguments: args,
	})

	access := ""
	if isAccessModifier(tok.Value) {
		access = tok.Value
	}

	if attr := p.skipFunctionAttrs(); attr != "" {
		access = attr
	}

	// The calls written in its body are its own: a receiver among its
	// arguments is typed from its signature, which is read by caller.
	prevBodyFunc := p.bodyFunc
	p.bodyFunc = len(p.funcs)
	endLine, braces := p.scanNestedFunctionBody(argNames(args))
	p.bodyFunc = prevBodyFunc

	p.scopes = append(p.scopes, FuncScope{
		Name: nameTok.Value, Start: funcLine, End: p.baseLine + endLine,
		span: p.functionSpan(FunctionNested, nameTok.Value, access, tok, [2]int{lp.Offset, closing}, braces, defaults),
	})
}

// isAccessModifier reports whether word is one of CFScript's access
// modifiers.
func isAccessModifier(word string) bool {
	return identEq(word, "public") || identEq(word, "private") || identEq(word, "package") || identEq(word, "remote")
}

// scanNestedFunctionBody consumes a nested function's `{ ... }` or `;`,
// recording the calls written inside it.
//
// It used to discard the body, so an immediately-invoked function lost
// everything it called: `return (function(){ return svc.x(); })();` recorded
// nothing at all, while the same closure passed as an argument recorded
// svc.x — because that path counts parentheses rather than skipping the
// function.
//
// The calls are attributed to the *enclosing* function, which is where the
// closure's code runs from and matches what an argument-position closure
// already did. An anonymous closure declares no scope of its own — it is a
// value, for the reason parseFunctionValue gives — so the line this returns is
// read only by the named branch, which does, with the region offsets of the
// body's braces ({-1, -1} when it has none).
func (p *scriptParser) scanNestedFunctionBody(params []string) (int, [2]int) {
	tok := p.sc.PeekSkipComments()
	if tok.Kind == TokSemicolon {
		p.sc.NextSkipComments()

		return int(tok.Line), [2]int{-1, tok.Offset}
	}

	if tok.Kind != TokLBrace {
		return int(tok.Line), [2]int{-1, -1}
	}

	p.sc.NextSkipComments()

	line, closing := p.scanClosureBody(int(tok.Line), params)
	if closing < 0 {
		return line, [2]int{-1, -1}
	}

	return line, [2]int{tok.Offset, closing}
}

// argNames is the names of args, a closure's parameters.
func argNames(args []Argument) []string {
	names := make([]string, 0, len(args))
	for i := range args {
		names = append(names, args[i].Name)
	}

	return names
}

// declareClosureParams declares a closure's parameters, on the line it
// starts, in the enclosing function. Outside a function there is nowhere to
// put them, as scanClosureBody says of its locals.
func (p *scriptParser) declareClosureParams(params []string, line int) {
	if !p.inFunc.in {
		return
	}

	for _, name := range params {
		p.vars = append(p.vars, VarDef{Name: name, Scope: ScopeArguments, Line: conv.Uint32(p.baseLine + line)})
	}
}

// scopeToClosure gives the function refs and pending calls recorded since
// refsBefore and pendingBefore the closure's lines, from and to, as their
// visibility. One a nested closure already scoped keeps its narrower range.
func (p *scriptParser) scopeToClosure(refsBefore, pendingBefore, from, to, byteFrom, byteTo int) {
	refs := p.funcRefs[p.inFunc]
	for i := refsBefore; i < len(refs); i++ {
		if refs[i].VisibleTo == 0 && !refs[i].HasClosureSpan {
			refs[i].VisibleFrom, refs[i].VisibleTo = conv.Uint32(from), conv.Uint32(to)
			if p.baseOffset >= 0 {
				refs[i].ClosureFrom, refs[i].ClosureTo = conv.Uint32(p.baseOffset+byteFrom), conv.Uint32(p.baseOffset+byteTo)
				refs[i].HasClosureSpan = true
			}
		}
	}

	for i := pendingBefore; i < len(p.pendingCalls); i++ {
		if c := &p.pendingCalls[i]; c.visibleTo == 0 && !c.hasClosureSpan && !c.global {
			c.visibleFrom, c.visibleTo = conv.Uint32(from), conv.Uint32(to)
			if p.baseOffset >= 0 {
				c.closureFrom, c.closureTo = conv.Uint32(p.baseOffset+byteFrom), conv.Uint32(p.baseOffset+byteTo)
				c.hasClosureSpan = true
			}
		}
	}
}

// scanClosureBody consumes the body of a closure, an anonymous or arrow
// function, whose `{` has been consumed, and returns the line of its `}` and
// that brace's region offset (-1 when the text ends first).
//
// Inside a function it is read as statements, the way the function's own body
// is: `it( "x", () => { var t = prepareMock( … ); t.go(); } )` declares `t`
// and records what it holds. The body used to be scanned for calls alone, so
// every `var` and every assignment in a closure declared nothing and typed
// nothing — which is most of a TestBox spec, where each test is a closure,
// and go-to-definition, completion and `unresolved` all read `t` as unknown.
// What a closure declares is attributed to the enclosing function, as its
// calls always were: that is the scope the parse has, and a local of one
// closure is visible to the function's other closures only by name.
//
// Top-level closure locals use the empty-name function-ref bucket. Their
// closure byte spans keep them out of the file's globals and sibling callbacks.
func (p *scriptParser) scanClosureBody(line int, params []string) (last, closing int) {
	byteFrom := p.sc.pos - 1
	statements := p.argNesting < maxArgNesting
	prevInClosure, prevLocals := p.inClosure, p.localVarSet
	p.inClosure = statements || p.inClosure

	// What the closure declares is its own: its `var t` must not make a
	// later unscoped `t = …` in the enclosing function a local.
	if statements {
		p.localVarSet = maps.Clone(prevLocals)
		if p.localVarSet == nil {
			p.localVarSet = make(map[string]bool)
		}
	}

	refsBefore, pendingBefore := len(p.funcRefs[p.inFunc]), len(p.pendingCalls)

	// The parameters are declared where the closure starts: `( table ) => {`
	// and `function( ctx ) {` bind names the body reads.
	if statements {
		for _, name := range params {
			p.localVarSet[strings.ToLower(name)] = true
		}

		p.declareClosureParams(params, line)
	}

	p.argNesting++

	defer func() {
		p.inClosure, p.localVarSet = prevInClosure, prevLocals
		p.argNesting--
	}()

	depth := 1
	last, closing = line, -1

	defer func() {
		if statements {
			p.scopeToClosure(refsBefore, pendingBefore, p.baseLine+line, p.baseLine+last, byteFrom, p.sc.pos)
		}
	}()

	for depth > 0 {
		t := p.sc.NextSkipComments()
		last = int(t.Line)

		afterLT := p.afterLT
		p.afterLT = t.Kind == TokLT

		switch t.Kind {
		case TokEOF:
			return last, -1
		case TokLBrace:
			depth++
		case TokRBrace:
			depth--
			if depth == 0 {
				closing = t.Offset
			}
		case TokIdent:
			if statements {
				p.handleBodyToken(t, depth, afterLT)
			} else {
				p.scanNestedCall(t)
			}
		case TokString, TokRBracket:
			p.handleLiteralToken(t)
		default:
			// Any other token is passed over.
		}
	}

	return last, closing
}
