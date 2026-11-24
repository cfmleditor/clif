package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
)

// propertyDef holds parsed property metadata.
type propertyDef struct {
	name       string
	typeName   string
	line       uint32
	col        uint32
	endLine    uint32
	endCol     uint32
	nameLine   uint32
	nameCol    uint32
	nameEndCol uint32
	attrs      map[string]string // all attribute key=value pairs (lowercase keys)
}

// PropertySymbol describes a declared component property for document outlines,
// with UTF-16 columns for both the full declaration and its identifier name.
type PropertySymbol struct {
	Name       string
	Type       string
	Line       uint32
	Col        uint32
	EndLine    uint32
	EndCol     uint32
	NameLine   uint32
	NameCol    uint32
	NameEndCol uint32
}

// PropertySymbols returns the component's declared properties with their ranges.
func (pr *ParseResult) PropertySymbols() []PropertySymbol {
	if pr == nil || len(pr.Properties) == 0 {
		return nil
	}

	out := make([]PropertySymbol, len(pr.Properties))
	for i := range pr.Properties {
		p := &pr.Properties[i]
		out[i] = PropertySymbol{
			Name:       p.name,
			Type:       p.typeName,
			Line:       p.line,
			Col:        p.col,
			EndLine:    p.endLine,
			EndCol:     p.endCol,
			NameLine:   p.nameLine,
			NameCol:    p.nameCol,
			NameEndCol: p.nameEndCol,
		}
	}

	return out
}

// scriptParser extracts function signatures, component refs, and variable
// declarations from CFScript source in a single pass.
type scriptParser struct {
	sc                  *Scanner
	funcs               []FunctionDef
	vars                []VarDef
	componentRefs       []ComponentRef
	funcRefs            map[fnKey][]ComponentRef // keyed by function (fnKey)
	funcLinks           map[fnKey][]DocumentLink // keyed by function (fnKey)
	funcCalls           map[fnKey][]CallSite     // keyed by function (fnKey)
	links               []DocumentLink           // global scope links
	calls               []CallSite               // global scope call sites
	scopes              []FuncScope
	properties          []propertyDef
	extends             string
	fileURI             string
	baseLine            int
	baseOffset          int // -1 for synthetic expressions without a source origin
	bodyFunc            int // 1 + the index in funcs of the function whose body is parsed; 0 for none
	resolvers           []Resolver
	resolverSet         *ResolverSet
	imports             map[string]string // last segment (lowercased) → full dot-path, from `import`
	builtinReturnLookup func(string) string
	argumentTypes       func(string, string, []Argument)
	inFunc              fnKey         // current function scope key, noFn if global
	localVarSet         *localNames   // var'd/local. names in current function
	returnVar           string        // last "return varName" seen in current function
	pendingCalls        []pendingCall // unresolved varName = funcCall(...) assignments
	flow                *flowBlocks   // the blocks a function body opens; nil outside a full parse

	// extras holds the instantiations and the call-name filter, made on the
	// first: a slice header here moved the struct up a size class, for
	// records most parses never fill (call_names.go).
	extras *scriptExtras

	// The flags sit together: spread between the wider fields above, each
	// was padded to eight bytes, and the struct fell into a larger size class.
	keepVars     bool // whether this walk retains declaration positions for AllVars
	accessors    bool
	persistent   bool
	extractCalls bool  // whether to extract all call sites
	afterLT      bool  // previous token was '<' — see looksLikeTagAttrs
	forceGlobal  bool  // when true, addRef routes to componentRefs
	inClosure    bool  // scanning a closure's body as statements — see scanClosureBody
	refThis      bool  // the assignment being parsed is through `this.` — see ComponentRef.This
	chainDepth   uint8 // chained assignments being parsed; see maxChainedAssignments
	// openDepth is how many brackets openKinds holds, past 16 counted as
	// unknown; afterFor is whether the last token a loop read was `for`.
	openDepth uint8
	afterFor  bool

	// argNesting fits beside the flags and returnLine after it, in the eight
	// bytes after the pointers.
	argNesting int16 // recursion depth of skipParenBody's scan, bounded by maxArgNesting
	// openKinds is the brackets the parse loops have seen open and not
	// closed, two bits each, innermost lowest (see trackOpen): a bare
	// `name =` declares only at statement level.
	openKinds  uint32
	returnLine uint32 // the line of the return returnVar came from
}

// pendingCall records an unresolved assignment from a function call.
type pendingCall struct {
	varName    string
	funcName   string
	expression string // argument-dependent root call, resolved after parsing
	baseVar    string // for x = baseVar.method() — resolve x to same component as baseVar
	line       uint32

	// refThis is carried to the ref as ComponentRef.This; baseScope says which
	// scope's refs baseVar may be read from. global files the ref at
	// component level, as addRef does under forceGlobal: an unscoped
	// assignment in a function is a variables-scope one, and the component
	// the call returns is what every function — and every sibling closure —
	// reads. funcKey is kept, since baseVar may still be a local. Beside line,
	// in its padding.
	refThis    bool
	global     bool
	returnExpr bool   // a return expression, grouped by function for factory-chain inference
	memberSet  bool   // not a call: `varName.funcName = …`; see checkMemberSet
	alias      bool   // not a call: `varName = baseVar`; see aliasComponent
	baseLocal  bool   // receiver or bare callable is locally owned; see baseVarComponent
	rebinds    bool   // the call is made on varName itself; see ComponentRef.Rebinds
	baseGuess  bool   // typed as baseVar itself, the method unlooked-up; see ComponentRef.BaseGuess
	block      uint32 // the block the assignment was made in; see flowBlocks
	baseScope  RefScope

	funcKey fnKey    // scope key, noFn if global
	rest    []string // calls chained after funcName, carried to the ref as ChainRest

	// The closure the assignment was made in, carried to the ref; see
	// ComponentRef.VisibleFrom.
	visibleFrom, visibleTo uint32
	closureFrom, closureTo uint32
	hasClosureSpan         bool
}

func newScriptParser(src, fileURI string, baseLine int, resolvers []Resolver) *scriptParser {
	return &scriptParser{
		sc:         NewScanner(blankNonScript(src)),
		fileURI:    fileURI,
		baseLine:   baseLine,
		baseOffset: -1,
		resolvers:  resolvers,
	}
}

// reset makes p the parser newScriptParser(src, fileURI, baseLine, resolvers)
// returns, for one that has finished with an earlier source and whose results
// have been read. It keeps the storage that parse filled (the per-function
// maps and the lists of calls, pending calls, instantiations, functions, scopes,
// variables and references, emptied) and the
// scanner, and nothing else: p is zeroed first, so a field added to the struct
// and left out here is cleared rather than carried into the next source.
// A tag file reads each expression it holds with one of these; building a
// parser, a scanner and their maps for every <cfset> was a third of the
// allocations of a tag parse.
func (p *scriptParser) reset(src, fileURI string, baseLine int, resolvers []Resolver) {
	sc, funcRefs, funcLinks, funcCalls := p.sc, p.funcRefs, p.funcLinks, p.funcCalls
	calls, pending, extras := p.calls[:0], p.pendingCalls[:0], p.extras
	funcs, scopes, vars, refs := p.funcs[:0], p.scopes[:0], p.vars[:0], p.componentRefs[:0]

	*p = scriptParser{fileURI: fileURI, baseLine: baseLine, baseOffset: -1, resolvers: resolvers}

	sc.reset(src)

	clear(funcRefs)
	clear(funcLinks)
	clear(funcCalls)

	if extras != nil {
		extras.instantiations = extras.instantiations[:0]
		extras.callNames = nil
	}

	p.sc, p.funcRefs, p.funcLinks, p.funcCalls = sc, funcRefs, funcLinks, funcCalls
	p.calls, p.pendingCalls, p.extras = calls, pending, extras
	p.funcs, p.scopes, p.vars, p.componentRefs = funcs, scopes, vars, refs
}

func (p *scriptParser) parseScopedVar(tok Token, scope Scope) {
	dot := p.sc.PeekSkipComments()
	if dot.Kind != TokDot {
		// See parseBodyScopedVar's identical case for why.
		if dot.Kind == TokLBracket {
			p.checkBareCall(tok)
		}

		return
	}

	p.sc.NextSkipComments() // consume .

	nameTok := p.sc.NextSkipComments()
	if nameTok.Kind != TokIdent {
		return
	}

	eq := p.sc.PeekSkipComments()
	if eq.Kind != TokEquals {
		if eq.Kind == TokLParen {
			p.recordScopedMemberCall(tok, nameTok)

			return
		}

		// Not an assignment — check for method call chain: scope.name.method(...)
		if eq.Kind == TokDot {
			var fullChain chainBuilder
			fullChain.reset(tok.Value)
			fullChain.writeDot()
			fullChain.writeString(nameTok.Value)

			callLine := nameTok.Line

			for p.sc.PeekSkipComments().Kind == TokDot {
				p.sc.NextSkipComments()

				next := p.sc.PeekSkipComments()
				if next.Kind == TokIdent {
					p.sc.NextSkipComments()

					fullChain.writeDot()
					fullChain.writeString(next.Value)

					callLine = next.Line
				} else {
					break
				}
			}

			if p.sc.PeekSkipComments().Kind == TokLParen {
				p.recordChainFromScope(fullChain.String(), int(callLine))
			}
		}

		return
	}

	p.sc.NextSkipComments() // consume =

	p.vars = append(p.vars, VarDef{
		Name: nameTok.Value, Scope: scope,
		Line: conv.Uint32(p.baseLine + int(tok.Line)),
	})

	isLocal := scope == ScopeLocal || scope == ScopeArguments
	if isLocal {
		p.localVarSet.add(nameTok.Value)
	}

	if !isLocal {
		p.forceGlobal = true
	}

	p.refThis = scope == ScopeThis
	defer func() { p.refThis = false }()

	if p.chainedAssign(nameTok.Value) {
		p.forceGlobal = false

		return
	}

	// Check RHS for component refs
	if (scope == ScopeThis || scope == ScopeVariables) &&
		p.parseFunctionValue(nameTok, tok) {
		return
	}

	rhs := p.rhsAfterLiteral()
	if rhs.Kind == TokIdent {
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
		case "this":
			p.sc.NextSkipComments()

			// Only treat `scope.x = this` (bare) as a self-ref.
			// `scope.x = this.prop = ...` is a chained assignment — the real
			// component comes from the continuation, so skip the ref here.
			if p.sc.PeekSkipComments().Kind != TokDot {
				if selfPath := strings.TrimPrefix(p.fileURI, "file://"); selfPath != "" {
					p.addRef(&ComponentRef{
						Variable: nameTok.Value, Component: selfPath,
						URI: uriFromString(p.fileURI), Line: conv.Uint32(p.baseLine + int(tok.Line)),
					})
				}
			}
		default:
			p.checkVarRHS(nameTok.Value, int(tok.Line))
		}
	}

	p.forceGlobal = false
}

// parseDottedStatement reads a statement that starts with a name and a dot,
// with the name already taken: a dotted return type before `function`, or a
// call on a dotted receiver, whose later hops are walked on. It is false when
// the call's argument list never closes, which ends the parse.
func (p *scriptParser) parseDottedStatement(tok Token) bool {
	peek := p.sc.PeekSkipComments()

	// Walk the dot chain — could be dotted return type or bare call
	named := !isKeyword(tok.Value) || operatorWordIsName(tok, peek)

	var retVal chainBuilder
	retVal.reset(tok.Value)

	lastIdent := tok.Value
	lastLine := tok.Line
	prevIdent := ""

	for p.sc.PeekSkipComments().Kind == TokDot {
		p.sc.NextSkipComments()

		seg := p.sc.PeekSkipComments()
		if seg.Kind == TokIdent {
			p.sc.NextSkipComments()

			prevIdent = lastIdent
			lastIdent = seg.Value
			lastLine = seg.Line

			retVal.writeDot()
			retVal.writeString(seg.Value)
		} else {
			break
		}
	}

	next := p.sc.PeekSkipComments()
	if next.Kind == TokIdent && identEq(next.Value, "function") {
		p.sc.NextSkipComments()
		p.parseFunction(tok, "", retVal.String())
	} else if next.Kind == TokLParen && named {
		_ = prevIdent

		varName := ""
		chain := retVal.String()

		if recv, _, ok := strings.CutLast(chain, "."); ok {
			varName = recv
		}

		funcName := lastIdent
		line := int(tok.Line)

		call := CallSite{
			FuncName: funcName,
			Variable: varName,
			Line:     conv.Uint32(p.baseLine + int(lastLine)),
		}
		// This dispatch also handles script islands seeded with a tag
		// function scope. Capture identity while the opening parenthesis
		// is still ahead of the scanner, before nested argument calls.
		p.setCallSpan(&call)

		comp, hop, ok := p.skipParensResolving(chain)
		if !ok {
			return false
		}

		p.addCall(&call)

		// Continue walking further .method() hops chained off this
		// call's return value (e.g. "table.getTable().setSkipFirstHeader(...)")
		// — without this, the scanner resumes right after this hop's
		// "(" and the next ".method(" is rediscovered as an orphaned,
		// unqualified bare call.
		p.recordChainContinuation(varName, hop, comp, line)
	}

	return true
}

func (p *scriptParser) parse() {
	for {
		tok := p.sc.NextSkipComments()
		if tok.Kind == TokEOF {
			break
		}

		p.trackOpen(tok)

		afterLT := p.afterLT
		p.afterLT = tok.Kind == TokLT

		if tok.Kind != TokIdent {
			p.handleLiteralToken(tok)

			continue
		}

		var buf foldScratch
		switch string(buf.lowerFold(tok.Value)) {
		case "function":
			p.parseFunction(tok, "", "")
		case "public", "private", "remote", "package":
			p.parseAccessModified(tok, tok.Value)
		case "component", "interface":
			// `component function init()` returns a component: the word is a
			// return type, not the component's header, which read the whole
			// function as attributes and lost it.
			if next := p.sc.PeekSkipComments(); next.Kind == TokIdent && identEq(next.Value, "function") {
				p.sc.NextSkipComments()
				p.parseFunction(tok, "", tok.Value)

				continue
			}

			// An interface's attribute list is a component's — `interface
			// extends="IBase"` left `extends` to be read as an assignment, and
			// declared a variable called `extends`.
			p.parseComponentAttrs()
		case "property":
			p.parsePropertyWord(tok)
		case "var":
			p.parseVarDecl(tok)
		case "local":
			p.parseScopedVar(tok, ScopeLocal)
		case "arguments":
			p.parseScopedVar(tok, ScopeArguments)
		case "this":
			p.parseScopedVar(tok, ScopeThis)
		case "variables":
			p.parseScopedVar(tok, ScopeVariables)
		case "request", "session", "application", "server":
			// See the matching case in handleBodyToken for why this is needed:
			// without it, a top-level (outside any function) "REQUEST.x = ..."
			// assignment falls through to the bare-call path and its RHS
			// component type is silently dropped.
			p.parseScopedVar(tok, sharedScopeOf(tok))
		case "return":
			// A script island can be inside a tag function whose scope was seeded.
			if p.inFunc.in {
				p.checkReturnComponent()
			}
		case "import":
			p.parseImport()
		case "new":
			p.parseStandaloneNew()
		case "catch":
			p.parseCatchVar()
		case "throw":
			p.consumeThrowArgs()
		default:
			// Check for returnType function pattern (e.g. "string function getName()")
			peek := p.sc.PeekSkipComments()
			switch {
			case peek.Kind == TokIdent && identEq(peek.Value, "function"):
				p.sc.NextSkipComments()
				p.parseFunction(tok, "", tok.Value)
			case peek.Kind == TokIdent && isFunctionModifier(tok.Value) && startsDeclaration(p.sc):
				// `static numeric function f()`: the modifier starts the
				// declaration, as an access word does.
				p.parseAccessModified(tok, "")
			case peek.Kind == TokDot:
				if !p.parseDottedStatement(tok) {
					return
				}
			case peek.Kind == TokDoubleColon:
				p.parseStaticCall(tok)
			case peek.Kind == TokLParen && !isKeyword(tok.Value):
				p.recordBareCallAndChain(tok)
			case !afterLT && peek.Kind == TokIdent && !isKeyword(tok.Value) && looksLikeTagAttrs(p.sc):
				p.parseScriptTagAttrs()
			default:
				p.checkAssignRef(tok)
			}
		}
	}
}

func (p *scriptParser) parseComponentAttrs() {
	for {
		tok := p.sc.PeekSkipComments()
		if tok.Kind == TokLBrace || tok.Kind == TokEOF {
			return
		}

		p.sc.NextSkipComments()

		// `component hint="prefix-#now()#"`: the value of an attribute this
		// does not read is still an expression.
		if tok.Kind == TokString {
			p.scanInterpolation(tok)
		}

		if tok.Kind != TokIdent {
			continue
		}

		switch {
		case strings.EqualFold(tok.Value, "extends"):
			if val, ok := p.attrValue(); ok {
				switch val.Kind {
				case TokString:
					p.extends = unquote(val.Value)
				case TokIdent:
					// CFML lets an attribute value go unquoted:
					// `component extends=testbox.system.BaseSpec {`.
					p.extends = dottedRest(p.sc, val.Value)
				default:
				}
			}
		case strings.EqualFold(tok.Value, "implements"):
			if val, ok := p.attrValue(); ok {
				switch val.Kind {
				case TokString:
					p.ensureExtras().implements = unquote(val.Value)
				case TokIdent:
					p.ensureExtras().implements = dottedRest(p.sc, val.Value)
				default:
				}
			}
		case strings.EqualFold(tok.Value, "accessors"):
			if val, ok := p.attrValue(); ok {
				p.accessors = isTruthy(unquote(val.Value))
			}
		case strings.EqualFold(tok.Value, "persistent"):
			if val, ok := p.attrValue(); ok &&
				((val.Kind == TokString && isTruthy(unquote(val.Value))) || (val.Kind == TokIdent && isTruthy(val.Value))) {
				p.persistent = true
			}
		}
	}
}

// dottedRest is first followed by every `.name` the scanner holds next: an
// unquoted dotted attribute value.
func dottedRest(sc *Scanner, first string) string {
	var b chainBuilder
	b.reset(first)

	for sc.PeekSkipComments().Kind == TokDot {
		sc.NextSkipComments()

		next := sc.NextSkipComments()
		if next.Kind != TokIdent {
			break
		}

		b.writeDot()
		b.writeString(next.Value)
	}

	return b.String()
}

// attrValue reads `= value` after an attribute name, returning the value
// token, or false when no `=` follows.
func (p *scriptParser) attrValue() (Token, bool) {
	if p.sc.PeekSkipComments().Kind != TokEquals {
		return Token{}, false
	}

	p.sc.NextSkipComments()

	val := p.sc.NextSkipComments()
	if val.Kind == TokString {
		p.scanInterpolation(val)
	}

	return val, true
}

// parseStaticCall records `Foo::bar()`, Lucee's and ACF's static member access.
//
// The `::` was two unrecognised tokens, so `Foo` was dropped and the call was
// recorded as a bare `bar()` — reported as `bar (no qualifier, not in file)`,
// which names the wrong problem: the call *is* qualified, and by a component
// rather than a variable. The CallSite therefore carries Component, not
// Variable, so resolution looks for the method in `Foo` instead of among the
// file's own functions.
//
// Only the statement form is recognised. A static call in an expression
// (`x = Foo::bar()`) is left where it was, since the chain walks that would
// have to learn about it each hold a receiver *name* and this has none.
func (p *scriptParser) parseStaticCall(qualifier Token) {
	p.recordStaticCall(qualifier.Value, qualifier)
}

// recordStaticCall consumes `::method(...)` from the scanner and files the call
// against component, which the caller has already read.
func (p *scriptParser) recordStaticCall(component string, startTok Token) {
	p.sc.NextSkipComments() // consume ::

	methTok := p.sc.PeekSkipComments()
	if methTok.Kind != TokIdent {
		return
	}

	p.sc.NextSkipComments()

	caller := ""
	if p.inFunc.in && len(p.funcs) > 0 {
		caller = p.currentFunc().Name
	}

	qualifier := component
	component = p.staticQualifier(component, startTok)

	// A static field, `Color::RED.getRed()`: a value of a type nothing here
	// can say, so what is chained on it is dynamic, not a bare call.
	if p.sc.PeekSkipComments().Kind != TokLParen {
		p.recordDynamicChain(qualifier, caller)

		return
	}

	call := CallSite{
		FuncName:  methTok.Value,
		Component: component,
		Line:      conv.Uint32(p.baseLine + int(startTok.Line)),
		Caller:    caller,
		Resolved:  true,
	}

	// `super::method()` is the parent's method, the call `super.method()` is,
	// not a static call on a component named super.
	if identEq(component, "super") {
		call.Variable, call.Component, call.Resolved = component, "", false
	}

	p.addCall(&call)

	if p.skipParens() {
		// What a static method returns is not typed here either.
		p.recordDynamicChain(qualifier, caller)
	}
}

// staticQualifier is the component a static call's qualifier names, as `new`
// reads its path: a name an import brought in is what the import says, and a
// Java class (`java:Double`, or a quoted import) is what createObject("java")
// would make of it. Anything else is the path as written.
func (p *scriptParser) staticQualifier(component string, startTok Token) string {
	if p.afterJavaPrefix(startTok) {
		if comp := p.resolveCall("createObject(\"java\",\"" + component + "\")"); comp != "" {
			return comp
		}

		return "$any"
	}

	return p.applyImport(component)
}

// afterJavaPrefix reports whether the token is written `java:Name`, Lucee's
// prefix for a Java class.
func (p *scriptParser) afterJavaPrefix(tok Token) bool {
	const prefix = "java:"

	off := tok.Offset
	if off < len(prefix) || off > len(p.sc.src) {
		return false
	}

	return strings.EqualFold(p.sc.src[off-len(prefix):off], prefix)
}

// consumeThrowArgs consumes `throw(...)`'s argument list.
//
// `throw` is the one CFScript keyword invoked like a function, and being a
// keyword it never reached the dispatch that consumes an argument list — so
// `throw(type = "x", message = "y")` declared variables called `type` and
// `message`, and `throw(object = new errors.Bad())` recorded a component ref
// for a variable called `object`. The other keywords that take parentheses
// (`if`, `while`, `switch`) hold an expression rather than named arguments, and
// the statement scan reads those correctly as it is.
//
// `throw "message";` and `throw new Foo();` take no parentheses, and are left
// for the loop to carry on with.
func (p *scriptParser) consumeThrowArgs() {
	if p.sc.PeekSkipComments().Kind == TokLParen {
		p.skipParens()
	}
}

// handleLiteralToken is what every token loop does with a string or a closing
// bracket: scan a string's #...# interpolations for calls, and record a member
// function called on the literal.
//
// It is one function because there are six loops that walk tokens — parse,
// parseBody, scanParenBody, scanNestedFunctionBody, skipLiteralGroup and
// skipDefault — and a literal reaches all of them.
// TestEveryTokenLoopHandlesLiterals fails if one stops routing through here.
func (p *scriptParser) handleLiteralToken(tok Token) {
	if tok.Kind == TokString {
		p.scanInterpolation(tok)
	}

	if isLiteralReceiver(tok.Kind) && p.sc.PeekSkipComments().Kind == TokDot {
		p.recordLiteralMemberCall()
	}
}

// isLiteralReceiver reports whether a token can close a literal that a member
// function is then called on.
func isLiteralReceiver(k TokenKind) bool {
	return k == TokString || k == TokRBracket
}

// recordLiteralMemberCall records `"abc".ucase()` and `[1,2].map(f)`.
//
// The receiver is a value rather than a variable, so the scan walked past it
// and recorded a *bare* `ucase()` — indistinguishable from an unqualified call
// to a function of that name, which in a component that happens to declare one
// is an edge to it that does not exist. The component is `$any`, the existing
// spelling for "genuinely dynamic", so the method-exists check is skipped the
// way it is for any other value whose type is not static.
func (p *scriptParser) recordLiteralMemberCall() {
	caller := ""
	if p.inFunc.in && len(p.funcs) > 0 {
		caller = p.currentFunc().Name
	}

	for p.sc.PeekSkipComments().Kind == TokDot {
		p.sc.NextSkipComments() // consume .

		methTok := p.sc.PeekSkipComments()
		if methTok.Kind != TokIdent {
			return
		}

		p.sc.NextSkipComments()

		if p.sc.PeekSkipComments().Kind != TokLParen {
			continue // a property read, e.g. `"a,b".listLen` — nothing to record
		}

		p.addCall(&CallSite{
			FuncName:  methTok.Value,
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

func isTruthy(s string) bool {
	return strings.EqualFold(s, "true") || strings.EqualFold(s, "yes")
}

// looksLikeCFCType returns true if a property type looks like a CFC reference
// (dotted path or non-primitive name).
func looksLikeCFCType(t string) bool {
	if strings.Contains(t, ".") {
		return true
	}

	var buf foldScratch
	switch string(buf.lowerFold(t)) {
	case "string", "numeric", "boolean", "date", "struct", "array", "query",
		"binary", "guid", "uuid", "void", "any", "xml", "function":
		return false
	}

	return true
}

// tryResolveCall attempts to resolve a function call via resolvers.
// Called when scanner is positioned at '(' (peeked, not consumed).
// Reconstructs the call expression (e.g. getService("foo")) and tries resolvers.
func (p *scriptParser) tryResolveCall(callExpr string) string {
	if comp := p.tryConfiguredResolvers(callExpr); comp != "" {
		return comp
	}

	// No resolver typed it; a call whose value no component describes is
	// dynamic (dynamicCall). Its arguments are consumed as a match's are.
	if dynamicCall(callExpr+"()") == "" || p.sc.PeekSkipComments().Kind != TokLParen {
		return ""
	}

	p.sc.NextSkipComments() // consume (

	// createMock( "models.User" ) and createEmptyMock( className = … ) name
	// the class; the argument list is read to its end either way.
	class := ""
	if name := callExpr[strings.LastIndexByte(callExpr, '.')+1:]; strings.EqualFold(name, "createMock") || strings.EqualFold(name, "createEmptyMock") {
		class = p.peekMockClass()
	}

	p.skipParenBody()

	if class != "" {
		return mockOf(class)
	}

	return "$any"
}

// peekMockClass is the class the argument list the scanner is inside names,
// without moving it: a leading string, or className = "…".
func (p *scriptParser) peekMockClass() string {
	saved := p.sc.Save()
	defer p.sc.Restore(saved)

	tok := p.sc.NextSkipComments()
	if tok.Kind == TokString {
		return unquote(tok.Value)
	}

	for depth := 0; tok.Kind != TokEOF; tok = p.sc.NextSkipComments() {
		switch tok.Kind {
		case TokLParen:
			depth++
		case TokRParen:
			if depth == 0 {
				return ""
			}

			depth--
		case TokIdent:
			if depth == 0 && identEq(tok.Value, "className") {
				if eq := p.sc.NextSkipComments(); eq.Kind == TokEquals || eq.Kind == TokColon {
					if v := p.sc.NextSkipComments(); v.Kind == TokString {
						return unquote(v.Value)
					}
				}
			}
		default:
		}
	}

	return ""
}

// tryConfiguredResolvers is tryResolveCall against the componentResolvers.
func (p *scriptParser) tryConfiguredResolvers(callExpr string) string {
	if len(p.resolvers) == 0 {
		return ""
	}

	// Quick prefix check
	hasPrefix := false

	for i := range p.resolvers {
		if p.resolvers[i].Prefix == "" || prefixContainsFold(callExpr, p.resolvers[i].Prefix) {
			hasPrefix = true

			break
		}
	}

	if !hasPrefix {
		return ""
	}

	// Save position — peek ahead to build the expression
	saved := p.sc.Save()
	p.sc.NextSkipComments() // consume (

	arg := p.sc.PeekSkipComments()

	// A single named argument, getService(service="company"), is read as the
	// positional one: falling through to the no-arg match below instead
	// offers the resolvers getService(), which only a catch-all can answer —
	// with a component that ignores which service was named.
	if arg.Kind == TokIdent {
		named := p.sc.Save()
		p.sc.NextSkipComments() // name

		if kind := p.sc.PeekSkipComments().Kind; kind == TokEquals || kind == TokColon {
			p.sc.NextSkipComments() // = or :

			if p.sc.PeekSkipComments().Kind == TokString {
				arg = p.sc.PeekSkipComments()
			} else {
				p.sc.Restore(named)
			}
		} else {
			p.sc.Restore(named)
		}
	}

	if arg.Kind == TokString {
		p.sc.NextSkipComments()

		if kind := p.sc.PeekSkipComments().Kind; kind != TokComma && kind != TokRParen {
			p.sc.Restore(saved)

			return ""
		}

		// Build arg list: read comma-separated string args
		var args strings.Builder
		args.WriteByte('"')
		args.WriteString(unquote(arg.Value))
		args.WriteByte('"')

		for {
			next := p.sc.PeekSkipComments()
			if next.Kind != TokComma {
				break
			}

			p.sc.NextSkipComments() // consume ,

			nextArg := p.sc.PeekSkipComments()
			if nextArg.Kind != TokString {
				break
			}

			p.sc.NextSkipComments()

			args.WriteString(`, "`)
			args.WriteString(unquote(nextArg.Value))
			args.WriteByte('"')
		}

		expr := callExpr + "(" + args.String() + ")"
		if comp := p.resolveCall(expr); comp != "" {
			// The matched resolver only had to account for the string args we
			// read above — consume whatever real tokens remain through the
			// matching ')' (there may be more args, or none) so a further
			// ".method(...)" chained after this call isn't left dangling for
			// the caller to rediscover as an orphaned bare call.
			p.skipParenBody()

			return comp
		}
	} else {
		// Try no-arg match: callExpr()
		expr := callExpr + "()"
		if comp := p.resolveCall(expr); comp != "" {
			p.skipParenBody()

			return comp
		}
	}

	// No match — try bare name for exact-match resolvers (e.g. "getFile" matches getFile(...))
	// Only use exact match to avoid substring conflicts (e.g. "_objInit" matching "objInit")
	for i := range p.resolvers {
		r := &p.resolvers[i]
		if r.Prefix != "" && prefixEqualFold(callExpr, r.Prefix) {
			if comp := matchResolverWithCache(callExpr, r); comp != "" {
				p.resolverSet.noteSoft(r, comp)

				p.skipParenBody()

				return comp
			}
		}
	}

	// No match — restore scanner
	p.sc.Restore(saved)

	return ""
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') {
		return s[1 : len(s)-1]
	}

	return s
}

// unquoteDoubled is a string token's contents with CFML's escape, a doubled
// quote, read as the one quote it stands for.
func unquoteDoubled(s string) string {
	if len(s) < 2 || s[0] != '"' && s[0] != '\'' {
		return s
	}

	q := s[:1]

	return strings.ReplaceAll(s[1:len(s)-1], q+q, q)
}

// parseStandaloneNew handles `new com.path(args)` expressions that are not
// assignment RHS (those are handled by parseNewRef). It consumes the component
// dot-path and constructor args so the scanner doesn't mis-parse the path as a
// variable.method() call site. Any chained `.method()` calls after the constructor
// are recorded as resolved call sites against the instantiated component.
func (p *scriptParser) parseStandaloneNew() {
	if p.sc.PeekSkipComments().Kind != TokIdent {
		return
	}

	component := p.readNewComponent()

	if p.sc.PeekSkipComments().Kind != TokLParen {
		return
	}

	p.skipParens()

	if component == "" {
		return
	}

	p.scanChainedCalls(component)
}

// operatorWordIsName reports whether tok, a word operator, is a variable's
// name: CFML lets `mod` and the comparison words name a variable, and
// cfwheels' specs hold their module in one, `mod.generate( … )`. Read as the
// operator, the receiver was dropped and the call recorded as a bare one. A
// dot after the word is what decides it: an operand never starts with one
// but a number, `x mod .5`, and a chain needs a name after the dot.
//
// `default` is the same case with a different keyword: the label of a switch is
// followed by a colon, so a dot after it is a variable, which ColdBox's CacheBox
// report page names its cache (`default.get( "Test" )`).
func operatorWordIsName(tok, next Token) bool {
	if next.Kind != TokDot {
		return false
	}

	var buf foldScratch
	switch string(buf.lowerFold(tok.Value)) {
	case "and", "or", "not", "eq", "neq", "lt", "gt", "lte", "gte", "mod", "default":
		return true
	}

	return false
}

// startsPropertyDecl reports whether the token after a `property` keyword makes
// it a declaration: a name, or a type before one. Anything else — an operator
// word such as `in`, a bracket, `=`, `.` — leaves `property` a variable.
func startsPropertyDecl(next Token) bool {
	if next.Kind != TokIdent {
		return false
	}

	var buf foldScratch
	switch string(buf.lowerFold(next.Value)) {
	case "in", "and", "or", "not", "eq", "neq", "lt", "gt", "lte", "gte", "mod", "is", "contains", "xor", "eqv", "imp":
		return false
	}

	return true
}

// operatorWordIsNameStr is operatorWordIsName for text: s begins with the
// word name, and a dot follows it directly.
func operatorWordIsNameStr(s, name string) bool {
	if len(s) <= len(name) || s[len(name)] != '.' {
		return false
	}

	return operatorWordIsName(Token{Kind: TokIdent, Value: name}, Token{Kind: TokDot})
}

func isKeyword(s string) bool {
	var buf foldScratch
	switch string(buf.lowerFold(s)) {
	case "var", "local", "if", "else", "elseif", "for", "while", "do", "switch", "case",
		"try", "catch", "finally", "return", "break", "continue", "function",
		"component", "interface", "new", "throw", "import", "true", "false",
		"and", "or", "not", "eq", "neq", "lt", "gt", "lte", "gte", "mod",
		"in", "default", "null":
		return true
	}

	return false
}

// applyJSDocParams parses @param {type} name annotations from a JSDoc comment
// and sets the Type on matching arguments (only if their Type is empty or "any").
func applyJSDocParams(comment string, args []Argument) {
	for comment != "" {
		idx := strings.Index(comment, "@param")
		if idx < 0 {
			break
		}

		comment = comment[idx+6:]

		// Skip whitespace
		i := 0
		for i < len(comment) && (comment[i] == ' ' || comment[i] == '\t') {
			i++
		}

		if i >= len(comment) || comment[i] != '{' {
			continue
		}

		// Extract type inside braces
		i++ // skip {

		end := strings.IndexByte(comment[i:], '}')
		if end < 0 {
			break
		}

		typeName := strings.TrimSpace(comment[i : i+end])
		comment = comment[i+end+1:]

		if typeName == "" {
			continue
		}

		// Skip whitespace to get param name
		i = 0
		for i < len(comment) && (comment[i] == ' ' || comment[i] == '\t') {
			i++
		}

		// Read param name
		nameStart := i
		for i < len(comment) && comment[i] != ' ' && comment[i] != '\t' && comment[i] != '\n' && comment[i] != '\r' && comment[i] != '*' {
			i++
		}

		paramName := comment[nameStart:i]
		if paramName == "" {
			continue
		}

		// Match to argument and override type if it's generic
		for j := range args {
			if strings.EqualFold(args[j].Name, paramName) {
				if args[j].Type == "" || strings.EqualFold(args[j].Type, "any") || strings.EqualFold(args[j].Type, "struct") {
					args[j].Type = typeName
				}

				break
			}
		}
	}
}

// Unquoted attribute literals may be dotted component names. Keep the complete
// path rather than interpreting its first segment as the property's type.
func propertyAttributeValue(tokens []Token, start int) string {
	if tokens[start].Kind != TokIdent {
		return unquote(tokens[start].Value)
	}

	var value strings.Builder
	value.WriteString(tokens[start].Value)

	for i := start + 1; i+1 < len(tokens) && tokens[i].Kind == TokDot && tokens[i+1].Kind == TokIdent; i += 2 {
		value.WriteByte('.')
		value.WriteString(tokens[i+1].Value)
	}

	return value.String()
}

func (p *scriptParser) recordBuiltinAssignment(variable, receiver, method, expression string, line int) {
	comp := p.builtinReturnLookup(method)
	// A call the chain goes on from is typed by walking the chain, which a
	// pending call does: the ref below would leave `.getResponse()` in
	// `getPageContext().getResponse()` a call nothing is known to be made on.
	if comp == "" || receiver != "" || strings.HasPrefix(comp, NativeTypePrefix) || p.chainFollowsCall() {
		p.addPendingCall(variable, receiver, method, expression, line)

		return
	}

	p.addRef(&ComponentRef{Variable: variable, Component: comp, URI: uriFromString(p.fileURI), Line: conv.Uint32(p.baseLine + line)})
}

// chainFollowsCall reports whether the call whose "(" the scanner is about to
// read is followed by `.`, `?.` or `[`, without moving the scanner.
func (p *scriptParser) chainFollowsCall() bool {
	open := p.sc.PeekSkipComments()
	if open.Kind != TokLParen {
		return false
	}

	end := skipCallGroup(p.sc.src, open.Offset)
	if end < 0 {
		return false
	}

	end = lex.SkipSpace(p.sc.src, end)

	return end < len(p.sc.src) && (p.sc.src[end] == '.' || p.sc.src[end] == '[' || strings.HasPrefix(p.sc.src[end:], "?."))
}

// ElementHop is the chain entry for an index read on what the chain before it
// returns. No component declares a method of that name, so a hop walk that
// reaches it from a typed component finds nothing, and from a dynamic value
// keeps the value dynamic.
const ElementHop = "[]"

// The kinds of bracket trackOpen keeps.
const (
	openValue = 1 // a call's or a group's (, a [, a struct literal's {
	openBlock = 2 // a block's {
	openFor   = 3 // a for header's (: its init and step are statements

	openBits     = 2
	maxOpenDepth = 32 / openBits
)

// trackOpen keeps openKinds for a token a parse loop read. A handler that
// reads a statement on its own reads its brackets in pairs, so the loops see
// only the ones nothing consumed: the arguments of a call no handler took, a
// struct or array literal, a block.
func (p *scriptParser) trackOpen(tok Token) {
	afterFor := p.afterFor
	p.afterFor = tok.Kind == TokIdent && identEq(tok.Value, "for")

	switch tok.Kind {
	case TokLParen:
		if afterFor {
			p.pushOpen(openFor)
		} else {
			p.pushOpen(openValue)
		}
	case TokLBracket:
		p.pushOpen(openValue)
	case TokLBrace:
		if structLiteralAt(p.sc.src, tok.Offset) {
			p.pushOpen(openValue)
		} else {
			p.pushOpen(openBlock)
		}
	case TokRParen, TokRBracket, TokRBrace:
		if p.openDepth > 0 {
			if p.openDepth <= maxOpenDepth {
				p.openKinds >>= openBits
			}

			p.openDepth--
		}
	default:
	}
}

func (p *scriptParser) pushOpen(kind uint32) {
	if p.openDepth < maxOpenDepth {
		p.openKinds = p.openKinds<<openBits | kind
	}

	if p.openDepth < 255 {
		p.openDepth++
	}
}

// atStatementLevel reports whether the loop stands where a statement can
// begin: in no bracket, a block, or a for header.
func (p *scriptParser) atStatementLevel() bool {
	if p.openDepth == 0 {
		return true
	}

	if p.openDepth > maxOpenDepth {
		return false
	}

	kind := p.openKinds & (1<<openBits - 1)

	return kind == openBlock || kind == openFor
}

// structLiteralAt reports whether the { at offset opens a struct literal
// rather than a block: it follows what only an expression follows, an
// operator that takes a value or the word return.
func structLiteralAt(src string, offset int) bool {
	i := offset - 1
	for i >= 0 && lex.IsRegexpSpace(src[i]) {
		i--
	}

	if i < 0 {
		return false
	}

	// An arrow function's => ends in >, so its body reads as the block it is.
	switch src[i] {
	case '=', '(', '[', ',', ':', '?':
		return true
	}

	return i >= 5 && strings.EqualFold(src[i-5:i+1], "return") && (i < 6 || !lex.IsWord(src[i-6]))
}
