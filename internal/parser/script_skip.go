package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// skipParens consumes a balanced (...) group from the current scanner position.
// Returns false if the scanner is not positioned at '(' or the group is unclosed.
func (p *scriptParser) skipParens() bool {
	if p.sc.PeekSkipComments().Kind != TokLParen {
		return false
	}

	p.sc.NextSkipComments() // consume (

	return p.skipParenBody()
}

// maxArgNesting bounds how deep the scan below will recurse into nested
// argument lists.
//
// The recursion is driven by the source, and Go cannot recover from stack
// exhaustion — it is a fatal runtime error, not a panic, so the recover() that
// guards every parse entry point would not catch it. Real CFML does not nest
// calls anywhere near this deep; a file that does gets its innermost calls
// skipped, which is what this did everywhere before.
const maxArgNesting = 64

// skipParenBody consumes the rest of a (...) group whose opening '(' has
// already been consumed (depth starts at 1), recording any calls written
// inside it. Returns false if the group is unclosed (EOF reached first).
//
// It used to discard the group a token at a time, which made every call in an
// argument list invisible: `writeOutput(svc.getName())` recorded only
// writeOutput, and `arrayAppend(rows, dao.load(id))` only arrayAppend. Nothing
// else lost calls this way — a condition, a return, an assignment's RHS, a
// string concatenation, a struct or array literal and a ternary all extract
// correctly — so an argument list was the one place a call could hide.
//
// What it cost was not completeness for its own sake. A method called only
// from inside an argument list had no edge into it, so internal/codemap read
// it as unreachable and the `unresolved` scan never checked it; a broken call
// there was reported nowhere.
//
// Nested calls are dispatched back through the same two helpers the statement
// path uses, which consume the call and its own argument list whole. That is
// what makes the depth counter here stay correct through a recursive call, and
// it is also what makes a closure passed as an argument work without a case of
// its own: its body is just more tokens to scan.
func (p *scriptParser) skipParenBody() bool {
	_, ok := p.scanParenBody()

	return ok
}

// scanParenBody is skipParenBody's body, and also reports the first string
// argument at the top level of the group — which the bare-call path resolves
// componentResolvers against, and which is the only reason it used to keep a
// loop of its own.
func (p *scriptParser) scanParenBody() (firstArg string, ok bool) {
	firstArg, _, ok = p.scanParenArgs()

	return firstArg, ok
}

// scanParenArgs is scanParenBody that also reports the leading run of
// arguments that are each a string literal and nothing else, stopping at the
// first that is not — the arguments tryResolveCall reads for an assignment.
// A resolver can need more than one: createObject("java", "java.io.File")
// names its component in the second, so offering it only the first left
// createObject("java", "x").m() unresolved as a bare call to m.
func (p *scriptParser) scanParenArgs() (firstArg string, positional []string, ok bool) {
	depth := 1
	// argStart is true while the scan sits at the start of a top-level
	// argument; pending holds a string that began one, until what follows it
	// shows whether it was the whole argument.
	argStart, collecting := true, true
	pending, havePending := "", false
	wholeCandidate := ""
	assign, namesArgument := false, false

	var arrow arrowParams

	for depth > 0 {
		tok := p.sc.NextSkipComments()
		if tok.Kind == TokEOF {
			return "", nil, false
		}

		// An arrow function's block body: `=>` is `=` then `>`.
		if tok.Kind == TokGT && arrow.prev == TokEquals && p.sc.PeekSkipComments().Kind == TokLBrace {
			p.sc.NextSkipComments() // consume {
			p.scanClosureBody(int(tok.Line), arrow.params(), true)

			arrow = arrowParams{prev: TokRBrace}
			argStart = false

			continue
		}

		// An expression-bodied arrow, `( v ) => v.isActive()`: its body is
		// scanned here as it always was, and only its parameters need
		// declaring.
		if tok.Kind == TokGT && arrow.prev == TokEquals {
			p.declareClosureParams(arrow.params(), int(tok.Line))
		}

		arrow.see(tok, depth)

		// A string offered as the first argument is the whole of its
		// argument: `getBean( "x" )` or `getBean( name = "x" )`. One that is a
		// piece of an expression, `getBean( url.p & "Provider" )`, names
		// nothing, and read as the argument it typed the call as a component
		// called "Provider". The string is held until the token after it
		// shows which it was.
		if wholeCandidate != "" {
			if tok.Kind == TokComma || tok.Kind == TokRParen {
				firstArg = wholeCandidate
			}

			wholeCandidate = ""
		}

		if tok.Kind == TokString && firstArg == "" && depth == 1 && (argStart || assign) {
			wholeCandidate = unquote(tok.Value)
		}

		// An `=` or `:` straight after the first word of an argument names it,
		// `f( name = "x" )`; after anything else it is half of `==`, `!=`, `<=`
		// or a ternary's `:`, and the string after it is a piece of an
		// expression.
		assign = (tok.Kind == TokEquals || tok.Kind == TokColon) && namesArgument
		namesArgument = (tok.Kind == TokIdent || tok.Kind == TokString) && argStart && depth == 1

		if depth == 1 && collecting {
			switch {
			case havePending && (tok.Kind == TokComma || tok.Kind == TokRParen):
				positional = append(positional, pending)
				havePending = false
			case havePending:
				collecting, havePending = false, false
			case argStart && tok.Kind == TokString:
				pending, havePending = unquote(tok.Value), true
			case argStart && tok.Kind != TokRParen:
				collecting = false
			}
		}

		argStart = depth == 1 && tok.Kind == TokComma

		switch tok.Kind {
		case TokLParen:
			depth++
		case TokRParen:
			depth--
		case TokIdent:
			p.scanNestedCall(tok)
		case TokString, TokRBracket:
			p.handleLiteralToken(tok)
		default:
			// Any other token is passed over.
		}
	}

	return firstArg, positional, true
}

// arrowParams follows an argument list's tokens closely enough to name an
// arrow function's parameters once its `=>` arrives, since by then they have
// been scanned: the names in the last (...) group, one per slot (the last
// identifier before each `,` or `)`, so `(any a)` names `a`), or the single
// identifier before `=>` in `x => {…}`.
type arrowParams struct {
	slot       string
	group      []string
	last       []string
	lastIdent  string
	groupDepth int
	prev       TokenKind
	prevprev   TokenKind
}

func (a *arrowParams) see(tok Token, depth int) {
	switch tok.Kind {
	case TokLParen:
		// depth has not counted this `(` yet.
		a.groupDepth, a.group, a.slot = depth+1, nil, ""
	case TokIdent:
		if depth == a.groupDepth {
			a.slot = tok.Value
		}

		a.lastIdent = tok.Value
	case TokComma:
		if depth == a.groupDepth && a.slot != "" {
			a.group, a.slot = append(a.group, a.slot), ""
		}
	case TokRParen:
		if depth == a.groupDepth {
			if a.slot != "" {
				a.group = append(a.group, a.slot)
			}

			a.last, a.group, a.slot, a.groupDepth = a.group, nil, "", 0
		}
	default:
	}

	a.prevprev, a.prev = a.prev, tok.Kind
}

// params is the parameters of the arrow whose `=` was the last token seen.
func (a *arrowParams) params() []string {
	switch a.prevprev {
	case TokRParen:
		return a.last
	case TokIdent:
		return []string{a.lastIdent}
	default:
		return nil
	}
}

// resolverCallExpr builds the expression a first hop is offered to the
// resolvers as: every leading string argument when there are any, as
// tryResolveCall builds it, else the first string argument found anywhere —
// the named getService(service="x") — else name().
func resolverCallExpr(name, firstArg string, positional []string) string {
	switch {
	case len(positional) > 0:
		return name + `("` + strings.Join(positional, `", "`) + `")`
	case firstArg != "":
		return name + `("` + firstArg + `")`
	default:
		return name + "()"
	}
}

// scanNestedCall dispatches an identifier met inside an argument list to the
// call-recording helpers, when it begins one.
func (p *scriptParser) scanNestedCall(tok Token) {
	if p.argNesting >= maxArgNesting {
		return
	}

	p.argNesting++
	defer func() { p.argNesting-- }()

	// `new a.b.C()` is an instantiation, and its path is not a receiver and a
	// method. Skipping the `new` keyword and letting the scan meet `a.b.C(`
	// on its own invents a call to C on a — which the test for this found
	// before the change was committed, and which is the same class of made-up
	// answer this scan exists to stop losing.
	if identEq(tok.Value, "new") {
		p.skipInstantiation()

		return
	}

	// A named function met inside a group is a declaration, and CFML hoists it
	// into the enclosing component's variables scope however deeply it is
	// nested — `it( function(){ function helper(){…} })` declares `helper`.
	// Reaching skipNestedFunction from here is what makes that true in every
	// group, since this is the one place all six token loops dispatch an
	// identifier through.
	if identEq(tok.Value, "function") {
		p.skipNestedFunction(tok, 0)

		return
	}

	// Any other keyword is handled by what follows it rather than by being
	// read as a receiver — bar a scope, which is one: `f( local.g() )` left
	// `.g()` to the next loop, which recorded a bare call to g.
	if _, _, scope := scopeReceiver(tok.Value); !scope && isKeyword(tok.Value) && !operatorWordIsName(tok, p.sc.PeekSkipComments()) {
		return
	}

	switch p.sc.PeekSkipComments().Kind {
	case TokLParen:
		p.recordBareCallAndChain(tok)
	case TokDoubleColon:
		// `f( Foo::bar() )`: a static call, as it is standing alone.
		p.parseStaticCall(tok)
	case TokDot, TokLBracket:
		// A dotted or bracket-indexed receiver. checkBareCall walks the chain
		// and records only when it ends at a '(' — a bare `a.b` property read
		// is consumed and dropped, which is the same thing the old scan did to
		// it.
		p.checkBareCall(tok)
	default:
		// Anything else after the name: it is not a call.
	}
}

// skipInstantiation consumes the component path of a `new` expression and its
// constructor arguments, whose own contents are still scanned for calls.
func (p *scriptParser) skipInstantiation() {
	// The path is read as every other `new` reads it, which records the
	// instantiation: one in an argument list or a literal was the one kind
	// the code map never drew.
	component := p.readNewComponent()

	if p.sc.PeekSkipComments().Kind != TokLParen {
		return
	}

	p.skipParens()

	// Hops chained onto it are calls on the object it made, as they are for a
	// `new` standing alone (parseStandaloneNew). Left unread, `expect( new
	// a.B().make().sound() )` met `make` as a bare call to a function of this
	// file, and a method the component lacks was never checked.
	if component != "" {
		p.scanChainedCalls(component)
	}
}

// skipBracketIndex consumes a balanced [...] group from the current scanner
// position, e.g. the dynamic key expression in REQUEST['a' & b & 'c'] or
// arr[i]. Mirrors skipParens() for (...). Returns false if the scanner isn't
// at a '[' or the group is unclosed (EOF reached first) — callers should
// treat false as "nothing to skip" / "malformed, bail" respectively (the
// only way to tell them apart is checking the token before calling).
// skipLiteralGroup consumes a struct or array literal standing as an
// assignment's right-hand side, recording any calls written inside it.
//
// A struct literal's keys may be spelled `{force = true}` as well as
// `{force: true}`, and nothing else consumes the group — `{` is not an
// identifier, so the statement scan walked straight through it and read the
// first spelling as an assignment, declaring a variable called `force`. Only
// the `=` spelling was ever affected, which is why the two look like different
// constructs in a `.cfc` and are the same one.
// rhsAfterLiteral steps over a struct or array literal a right-hand side
// starts with (skipLiteralGroup) and is the token that then begins the value
// to read for a type. After a literal there is none: the value was the
// literal, and an identifier after it begins the next statement, as one does
// where Lucee lets a line end a statement without a semicolon. Read as the
// value, `x = {}` then `y = 2` on the next line took y for x's type and never
// declared it.
func (p *scriptParser) rhsAfterLiteral() Token {
	if p.skipLiteralGroup() {
		return Token{Kind: TokSemicolon}
	}

	return p.sc.PeekSkipComments()
}

// skipLiteralGroup steps over the struct or array literal the scanner is at,
// and reports whether there was one.
func (p *scriptParser) skipLiteralGroup() bool {
	switch p.sc.PeekSkipComments().Kind {
	case TokLBrace, TokLBracket:
	default:
		return false
	}

	p.sc.NextSkipComments() // consume { or [

	depth := 1

	for depth > 0 {
		tok := p.sc.NextSkipComments()

		switch tok.Kind {
		case TokEOF:
			return true
		case TokLBrace, TokLBracket:
			depth++
		case TokRBrace, TokRBracket:
			depth--
		case TokLParen:
			// Hand a nested (...) to the scan that knows it, so a call written
			// inside a literal's value is still found.
			if !p.skipParenBody() {
				return true
			}
		case TokIdent:
			p.scanNestedCall(tok)
		case TokString:
			p.handleLiteralToken(tok)
		default:
			// Any other token is passed over.
		}
	}

	return true
}

// looksLikeTagAttrs reports whether the scanner sits on `ident =`, without
// moving it.
//
// That is two tokens of lookahead and the scanner offers one, so this saves and
// restores rather than peeking twice. It takes a scanner rather than a parser,
// so any reader of CFScript can ask it.
func looksLikeTagAttrs(sc *Scanner) bool {
	saved := sc.Save()
	defer sc.Restore(saved)

	if sc.NextSkipComments().Kind != TokIdent {
		return false
	}

	return sc.PeekSkipComments().Kind == TokEquals
}

// parseScriptTagAttrs consumes the attribute list of a script-syntax CF tag —
// `query name="q" datasource="ds" { … }`, `savecontent variable="out" { … }`,
// `lock name="l" timeout="5" { … }`, `param name="form.id" default="0";` — and
// stops before the body or the terminating semicolon, which the caller's loop
// goes on to read as usual.
//
// The tag name is an ordinary identifier to the scanner and the parser holds no
// list of tag names, so without this the attributes were read as statements and
// every one of them declared a variable: go-to-definition on `name` anywhere in
// a file containing a `<cfquery>` written in script jumped to that tag's
// attribute. A list of tag names would be the other way to recognise this, and
// is worse: `internal/docs` is generated and would have to be imported into a
// package kept deliberately dependency-light, and a tag it does not list would
// silently go back to declaring variables.
//
// `ident ident =` is what identifies the shape, and it spells nothing else in
// CFScript — a typed declaration is `var x = …`, and a return-typed function is
// caught by the `function` case that runs before this one.
func (p *scriptParser) parseScriptTagAttrs() {
	for looksLikeTagAttrs(p.sc) {
		name := p.sc.NextSkipComments() // attribute name
		p.sc.NextSkipComments()         // =

		// `invoke component=… method=… returnVariable="local.result"` assigns
		// what the method returns, which nothing here describes.
		if value := p.sc.PeekSkipComments(); value.Kind == TokString && identEq(name.Value, "returnVariable") {
			if variable := strings.TrimPrefix(unquote(value.Value), "local."); isDottedIdentText(variable) {
				p.addRef(&ComponentRef{
					Variable: variable, Component: "$any",
					URI: uriFromString(p.fileURI), Line: conv.Uint32(p.baseLine + int(name.Line)),
				})
			}
		}

		if !p.skipTagAttrValue() {
			return
		}
	}
}

// skipTagAttrValue consumes one attribute value and reports whether the scan
// can continue.
//
// It takes one token, one #...# interpolation or one balanced group, and then
// keeps going only through the operators that genuinely extend a value — a
// call, an index, a dot or an `&` concatenation. Consuming "everything up to
// the body" instead is what an earlier version did, and on the tag text that
// parseFuncBody hands this parser it swallowed a whole `<cffunction>` body
// along with the `<cfset var x = 1>` inside it.
func (p *scriptParser) skipTagAttrValue() bool {
	for {
		switch p.sc.PeekSkipComments().Kind {
		case TokEOF, TokSemicolon, TokLBrace, TokRBrace, TokLT, TokGT:
			return false
		case TokHash:
			if !p.skipHashExpr() {
				return false
			}
		case TokLParen:
			if !p.skipParens() {
				return false
			}
		case TokLBracket:
			if !p.skipBracketIndex() {
				return false
			}
		case TokIdent:
			// An attribute value can be a call: `array=structKeyArray(rows)`,
			// `result=serializeJson(body.data)`. Consuming the identifier
			// without dispatching it recorded the calls *inside* its argument
			// list — skipParens scans those — and never the call itself.
			tok := p.sc.NextSkipComments()
			p.scanNestedCall(tok)
		case TokString:
			tok := p.sc.NextSkipComments()
			p.handleLiteralToken(tok)
		default:
			p.sc.NextSkipComments()
		}

		switch p.sc.PeekSkipComments().Kind {
		case TokLParen, TokLBracket, TokDot, TokAmpersand, TokHash:
		default:
			return true
		}
	}
}

// skipHashExpr consumes a #...# interpolation, which an attribute value may be
// written as: `directory="#root#"` unquoted is `directory=#root#`.
func (p *scriptParser) skipHashExpr() bool {
	p.sc.NextSkipComments() // consume the opening #

	for {
		switch tok := p.sc.NextSkipComments(); tok.Kind {
		case TokEOF, TokSemicolon, TokLBrace, TokLT, TokGT:
			return false
		case TokHash:
			return true
		default:
			// Any other token is part of the span.
		}
	}
}

// skipBracketIndex consumes a balanced [...] group, recording any calls written
// inside it.
//
// It mirrored skipParens — the *old* skipParens, which discarded its group a
// token at a time. An index is an expression like any other, so
// `sorted[ sorted.len() ]`, `arr[ f() ]` and `a.b[ f() ]` recorded nothing at
// all, and `g( arr[ f() ] )` recorded only `g`. That is the same defect
// skipParenBody exists to have fixed, left in the one group the consolidation
// did not reach.
//
// The chain text the callers build is unaffected: a bracket still poisons it
// with the literal "[]" marker, so `REQUEST[key].method()` still falls through
// to an honest "no component ref" rather than resolving as `REQUEST.method()`.
// What changes is only that the key expression is read rather than thrown away.
func (p *scriptParser) skipBracketIndex() bool {
	if p.sc.PeekSkipComments().Kind != TokLBracket {
		return false
	}

	p.sc.NextSkipComments() // consume [

	depth := 1

	for depth > 0 {
		tok := p.sc.NextSkipComments()
		if tok.Kind == TokEOF {
			return false
		}

		switch tok.Kind {
		case TokLBracket:
			depth++
		case TokRBracket:
			depth--
		case TokIdent:
			p.scanNestedCall(tok)
		case TokString:
			p.handleLiteralToken(tok)
		default:
			// Any other token is passed over.
		}
	}

	return true
}

// tryExtendChain is called after tryResolveCall has failed and the scanner
// sits at the '(' that tryResolveCall restored. It skips the argument list,
// checks for a following '.method(' continuation, and tries tryResolveCall
// on the extended chain. On success the scanner is advanced past the new
// call's ')'. On failure the scanner position is unchanged.
//
// It also returns the names it read past the first call's ')', in order: the
// method it resolved on, preceded by any property names between the two
// (`a.b(x).c.d(y)` reads c and d). The caller records that call and carries on
// from it, since the scanner now sits beyond both.
func (p *scriptParser) tryExtendChain(chain string) (string, []string) {
	if len(p.resolvers) == 0 {
		return "", nil
	}

	saved, recorded := p.sc.Save(), p.mark()

	// The look ahead walks the argument list, which records what it holds;
	// rewinding the scanner without rewinding those left them recorded, and
	// the walk that follows recorded them again — every call in the
	// arguments of a top-level scoped assignment twice, wherever a
	// componentResolver was configured.
	rewind := func() {
		p.sc.Restore(saved)
		p.rewind(&recorded)
	}

	if !p.skipParens() {
		rewind()

		return "", nil
	}

	if p.sc.PeekSkipComments().Kind != TokDot {
		rewind()

		return "", nil
	}

	p.sc.NextSkipComments() // consume .

	next := p.sc.PeekSkipComments()
	if next.Kind != TokIdent {
		rewind()

		return "", nil
	}

	p.sc.NextSkipComments() // consume method name

	var extChain chainBuilder
	extChain.reset(chain)
	extChain.writeDot()
	extChain.writeString(next.Value)

	names := []string{next.Value}

	for p.sc.PeekSkipComments().Kind == TokDot {
		p.sc.NextSkipComments()

		n2 := p.sc.PeekSkipComments()
		if n2.Kind != TokIdent {
			break
		}

		p.sc.NextSkipComments()

		extChain.writeDot()
		extChain.writeString(n2.Value)

		names = append(names, n2.Value)
	}

	if p.sc.PeekSkipComments().Kind != TokLParen {
		rewind()

		return "", nil
	}

	if comp := p.tryResolveCall(extChain.String()); comp != "" {
		return comp, names
	}

	rewind()

	return "", nil
}

// parseMark is how much a scriptParser has recorded, taken before a look
// ahead so that a look ahead which is rewound can be un-recorded too.
type parseMark struct {
	calls, funcCalls, pending, vars, refs, funcRefs, funcs, scopes, links, funcLinks int
	hadFuncCalls, hadFuncRefs, hadFuncLinks                                          bool
}

func (p *scriptParser) mark() parseMark {
	fc, hadFC := p.funcCalls[p.inFunc]
	fr, hadFR := p.funcRefs[p.inFunc]
	fl, hadFL := p.funcLinks[p.inFunc]

	return parseMark{
		calls: len(p.calls), funcCalls: len(fc), pending: len(p.pendingCalls), vars: len(p.vars),
		refs: len(p.componentRefs), funcRefs: len(fr), funcs: len(p.funcs), scopes: len(p.scopes),
		links: len(p.links), funcLinks: len(fl),
		hadFuncCalls: hadFC, hadFuncRefs: hadFR, hadFuncLinks: hadFL,
	}
}

// rewind drops everything recorded since m. p.inFunc must be what it was.
func (p *scriptParser) rewind(m *parseMark) {
	p.calls = p.calls[:m.calls]
	p.pendingCalls = p.pendingCalls[:m.pending]
	p.vars = p.vars[:m.vars]
	p.componentRefs = p.componentRefs[:m.refs]
	p.funcs = p.funcs[:m.funcs]
	p.scopes = p.scopes[:m.scopes]
	p.links = p.links[:m.links]

	rewindEntry(p.funcCalls, p.inFunc, m.funcCalls, m.hadFuncCalls)
	rewindEntry(p.funcRefs, p.inFunc, m.funcRefs, m.hadFuncRefs)
	rewindEntry(p.funcLinks, p.inFunc, m.funcLinks, m.hadFuncLinks)
}

// rewindEntry cuts m[key] back to n, removing it if it was not there.
func rewindEntry[T any](m map[fnKey][]T, key fnKey, n int, had bool) {
	if !had {
		delete(m, key)

		return
	}

	if xs, ok := m[key]; ok {
		m[key] = xs[:n]
	}
}

// continueExtendedChain records the call tryExtendChain resolved on and every
// hop chained after it, from recv, the receiver the first call was made on.
//
// tryExtendChain reads `first(...).ext(...)` whole to match a resolver against
// it, and its callers used to take the component and stop: ext was never
// recorded, and a hop after it was left for the outer loop to rediscover as a
// bare call. In `VARIABLES.x = REQUEST.kernel.getSandBox("f").getEntityObj()
// .getSelected()`, getEntityObj went missing and getSelected was reported "no
// qualifier, not in file". The hops are walked from recv with no component
// assumed: the component tryExtendChain returns is a resolver's guess at the
// whole text, and the walk through the declared return types is the better
// answer. It returns the hops after ext, which the caller's ref walks.
//
// A property access between the two calls (`a.b(x).c.d(y)`) breaks the walk:
// the value of .c is not a return type. The call and what follows are then
// recorded against a dynamic receiver, as a literal receiver's chain is.
func (p *scriptParser) continueExtendedChain(recv, first string, ext []string, line int) []string {
	caller := ""
	if p.inFunc.in && len(p.funcs) > 0 {
		caller = p.currentFunc().Name
	}

	if len(ext) == 0 {
		return nil
	}

	method := ext[len(ext)-1]

	if len(ext) > 1 {
		p.addCall(&CallSite{
			FuncName: method, Variable: recv, Component: "$any", Resolved: true,
			Line: conv.Uint32(p.baseLine + line), Caller: caller,
		})
		p.recordDynamicChain(recv, caller)

		return []string{method, "?"} // the rest is dynamic: see dynamicIfTyped
	}

	p.addCall(&CallSite{
		FuncName: method, Variable: recv, Chain: []string{first},
		Line: conv.Uint32(p.baseLine + line), Caller: caller,
	})

	return p.recordChainContinuationFrom(recv, []string{first}, method, "", line, false)
}

// receiverOf is the receiver of the last call in a dotted chain: "a.b" for
// "a.b.c", "" for a bare "c". A chain continued from an assignment is walked
// from it, not from the last name before the call — which in
// "x = REQUEST.kernel.a().b()" was "kernel".
func receiverOf(chain string) string {
	if recv, _, ok := strings.CutLast(chain, "."); ok {
		return recv
	}

	return ""
}
