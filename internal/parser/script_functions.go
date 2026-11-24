package parser

import (
	"github.com/cfmleditor/clif/internal/conv"
)

// parseAccessModified reads a declaration whose first word, accessTok, is its
// access (access) or a modifier (static, final, abstract: access ""), and the
// return type that may follow.
func (p *scriptParser) parseAccessModified(accessTok Token, access string) {
	next := p.sc.PeekSkipComments()
	if next.Kind != TokIdent {
		return
	}

	if identEq(next.Value, "function") {
		p.sc.NextSkipComments()
		p.parseFunction(accessTok, access, "")

		return
	}

	retType := p.sc.NextSkipComments()

	// A modifier between the access and the return type (`package final any
	// function`), or an access after a modifier (`final package function`),
	// is not the return type; the word after it is.
	for isFunctionModifier(retType.Value) || access == "" && isAccessModifier(retType.Value) {
		after := p.sc.PeekSkipComments()
		if after.Kind != TokIdent {
			break
		}

		if isAccessModifier(retType.Value) {
			access = retType.Value
		}

		if identEq(after.Value, "function") {
			p.sc.NextSkipComments()
			p.parseFunction(accessTok, access, "")

			return
		}

		retType = p.sc.NextSkipComments()
	}

	var retVal chainBuilder
	retVal.reset(retType.Value)

	// Handle dotted return types (e.g. models.User)
	for p.sc.PeekSkipComments().Kind == TokDot {
		p.sc.NextSkipComments()

		seg := p.sc.NextSkipComments()
		if seg.Kind == TokIdent {
			retVal.writeDot()
			retVal.writeString(seg.Value)
		}
	}

	// An array of a type, `private string[] function`, returns an array.
	for p.sc.PeekSkipComments().Kind == TokLBracket {
		saved := p.sc.Save()
		p.sc.NextSkipComments()

		if p.sc.NextSkipComments().Kind != TokRBracket {
			p.sc.Restore(saved)

			break
		}

		retVal.reset("array")
	}

	next2 := p.sc.PeekSkipComments()
	if next2.Kind == TokIdent && identEq(next2.Value, "function") {
		p.sc.NextSkipComments()
		p.parseFunction(accessTok, access, retVal.String())
	}
}

// startsDeclaration reports whether the scanner, just past a modifier, holds
// the rest of a function's header: words, dots and `[]` up to `function`.
// It looks without moving.
func startsDeclaration(sc *Scanner) bool {
	saved := sc.Save()
	defer sc.Restore(saved)

	for range 8 {
		switch t := sc.NextSkipComments(); {
		case t.Kind == TokIdent && identEq(t.Value, "function"):
			return true
		case t.Kind == TokIdent, t.Kind == TokDot, t.Kind == TokLBracket, t.Kind == TokRBracket:
		default:
			return false
		}
	}

	return false
}

// isFunctionModifier reports whether word is a modifier a script function's
// header may carry besides its access.
func isFunctionModifier(word string) bool {
	return identEq(word, "static") || identEq(word, "final") || identEq(word, "abstract")
}

// skipFunctionAttrs steps over a declaration's attributes (the shared
// skipFunctionAttrs), reading each value as the expression it is: a string's
// #...# spans and a bare call are recorded like any other. It returns the
// value of an access= attribute, "" when there is none.
func (p *scriptParser) skipFunctionAttrs() string {
	access := ""

	skipFunctionAttrs(p.sc, func(name, v Token) {
		if identEq(name.Value, "access") && (v.Kind == TokString || v.Kind == TokIdent) {
			access = unquote(v.Value)
		}

		switch {
		case v.Kind == TokString:
			p.scanInterpolation(v)
		case v.Kind == TokIdent && p.sc.PeekSkipComments().Kind == TokLParen:
			p.recordBareCallAndChain(v)
		default:
		}
	})

	return access
}

// functionSpan is the FunctionSpan of a function the walk has just read,
// nil where the text has no place in a file (baseOffset < 0). The offsets
// are the region's, from start (the declaration's first token), the
// parameter list's parentheses and the body's braces.
func (p *scriptParser) functionSpan(kind FunctionKind, name, access string, start Token, parens, braces [2]int, defaults map[string]string) *FunctionSpan {
	if p.baseOffset < 0 {
		return nil
	}

	base := p.baseOffset
	s := p.allocSpan()
	*s = FunctionSpan{
		Name: name, Access: access, Defaults: defaults, Kind: kind,
		Line:   p.baseLine + int(start.Line),
		Start:  base + start.Offset,
		Params: -1, ParamsEnd: -1, Body: -1, BodyEnd: -1,
	}

	s.End = s.Start

	if parens[0] >= 0 && parens[1] >= parens[0] {
		s.Params, s.ParamsEnd = base+parens[0]+1, base+parens[1]
		s.End = s.ParamsEnd + 1
	}

	switch {
	case braces[0] >= 0 && braces[1] >= braces[0]:
		s.Body, s.BodyEnd = base+braces[0]+1, base+braces[1]
		s.End = s.BodyEnd + 1
	case braces[0] < 0 && braces[1] >= 0:
		s.End = base + braces[1] + 1 // past the `;` of a declaration without a body
	default:
	}

	return s
}

func (p *scriptParser) parseFunction(startTok Token, access string, returnType string) {
	// Capture JSDoc comment that preceded this function
	docComment := p.sc.LastBlockComment
	p.sc.LastBlockComment = ""

	kind := FunctionDeclared
	if p.inFunc.in {
		kind = FunctionNested
	}

	nameTok := p.sc.NextSkipComments()
	if nameTok.Kind != TokIdent {
		return
	}

	// `public function function startWorker()` returns a function: the
	// keyword read first was its return type.
	if identEq(nameTok.Value, "function") && p.sc.PeekSkipComments().Kind == TokIdent {
		returnType = nameTok.Value
		nameTok = p.sc.NextSkipComments()
	}

	lp := p.sc.NextSkipComments()
	if lp.Kind != TokLParen {
		return
	}

	args, closing, defaults := p.parseArgs(p.keepVars)

	// Apply JSDoc @param {type} annotations to arguments
	if docComment != "" {
		applyParameterDocs(docComment, args)
	}

	if !p.inFunc.in {
		if p.argumentTypes != nil {
			p.argumentTypes(nameTok.Value, access, args)
		}
	}

	funcLine := p.baseLine + int(startTok.Line)
	nameLine, nameCol, nameEndCol := p.tokenNameRange(nameTok)

	p.funcs = append(p.funcs, FunctionDef{
		Name:       nameTok.Value,
		URI:        uriFromString(p.fileURI),
		Line:       conv.Uint32(funcLine),
		NameLine:   nameLine,
		NameCol:    nameCol,
		NameEndCol: nameEndCol,
		Arguments:  args,
		ReturnType: returnType,
		DocReturn:  docReturn(docComment),
	})

	// Process body: set inFunc scope, parse assignments, then clear
	stated := access
	if attr := p.skipFunctionAttrs(); attr != "" {
		stated = attr
	}

	endLine, braces := p.parseBody(funcLine, args)
	p.scopes = append(p.scopes, FuncScope{
		Name: nameTok.Value, Access: access, ReturnType: returnType, Start: funcLine, End: p.baseLine + endLine,
		span: p.functionSpan(kind, nameTok.Value, stated, startTok, [2]int{lp.Offset, closing}, braces, defaults),
	})
}

// parseFunctionValue declares a method for a function assigned to a name, and
// reports whether it consumed one.
//
// `this.helper = function(a) { … }` and `variables.helper = (a) => a` are how a
// component exposes a method it builds rather than declares, and the parser
// recorded only a variable: no FunctionDef, so the component had none of those
// methods in Funcs — no completion, no signature help, no go-to-definition and
// nothing in the index.
//
// It fires only outside a function body, and only for `this.`, `variables.` and
// an unscoped name. A `var`- or `local.`-scoped closure is a local value, not a
// method of the component, and declaring one as a method would put a helper
// private to one function into every caller's completion list.
//
// The paren-less single-argument arrow (`this.x = a => a * 2`) is not
// recognised: telling it from an ordinary `this.x = a` needs three tokens of
// lookahead on every assignment whose right-hand side is a bare identifier,
// which is most of them.
func (p *scriptParser) parseFunctionValue(nameTok, startTok Token) bool {
	if p.inFunc.in {
		return false
	}

	// The docblock sits before the assignment, not before the `function`
	// keyword, so it is read here rather than where parseFunction reads it.
	docComment := p.sc.LastBlockComment

	rhs := p.sc.PeekSkipComments()

	switch {
	case rhs.Kind == TokIdent && identEq(rhs.Value, "function"):
		p.sc.NextSkipComments() // consume `function`

		// A named function expression: `this.x = function x() { … }`.
		if p.sc.PeekSkipComments().Kind == TokIdent {
			p.sc.NextSkipComments()
		}

		if p.sc.PeekSkipComments().Kind != TokLParen {
			return false
		}

	case rhs.Kind == TokLParen && p.arrowFollowsParens():

	default:
		return false
	}

	lp := p.sc.NextSkipComments() // consume (

	p.sc.LastBlockComment = ""

	args, closing, defaults := p.parseArgs(p.keepVars)
	if docComment != "" {
		applyParameterDocs(docComment, args)
	}

	if rhs.Kind == TokLParen {
		// The `=>` the lookahead found, now that the argument list is consumed.
		p.sc.NextSkipComments()
		p.sc.NextSkipComments()
	}

	p.recordFunctionValue(nameTok, startTok, args, functionParams{parens: [2]int{lp.Offset, closing}, defaults: defaults})

	return true
}

// functionParams is where a function value's parameter list was and the
// literal defaults it held, for its FunctionSpan.
type functionParams struct {
	parens   [2]int
	defaults map[string]string
}

// arrowFollowsParens reports whether the (...) the scanner is positioned on is
// an arrow function's argument list, without moving it.
//
// `=>` is two tokens to this scanner, and the parentheses have to be walked to
// reach them, so this is the one place a rewound scan is worth its cost: an
// assignment whose right-hand side opens with a paren is uncommon, and telling
// `(a) => a` from `(a + b) * c` has no cheaper test.
func (p *scriptParser) arrowFollowsParens() bool {
	saved := p.sc.Save()
	defer p.sc.Restore(saved)

	if !p.skipParensQuiet() {
		return false
	}

	if p.sc.PeekSkipComments().Kind != TokEquals {
		return false
	}

	p.sc.NextSkipComments()

	return p.sc.PeekSkipComments().Kind == TokGT
}

// skipParensQuiet consumes a balanced (...) group without recording anything,
// for a lookahead that will be rewound — scanParenBody would record every call
// inside the group twice over.
func (p *scriptParser) skipParensQuiet() bool {
	if p.sc.PeekSkipComments().Kind != TokLParen {
		return false
	}

	p.sc.NextSkipComments() // consume (

	depth := 1

	for depth > 0 {
		switch tok := p.sc.NextSkipComments(); tok.Kind {
		case TokEOF:
			return false
		case TokLParen:
			depth++
		case TokRParen:
			depth--
		default:
			// Any other token is passed over.
		}
	}

	return true
}

// recordFunctionValue files the declaration and walks the body, as
// parseFunction does for a declared method.
func (p *scriptParser) recordFunctionValue(nameTok, startTok Token, args []Argument, params functionParams) {
	funcLine := p.baseLine + int(startTok.Line)
	nameLine, nameCol, nameEndCol := p.tokenNameRange(nameTok)

	p.funcs = append(p.funcs, FunctionDef{
		Name:       nameTok.Value,
		URI:        uriFromString(p.fileURI),
		Line:       conv.Uint32(funcLine),
		NameLine:   nameLine,
		NameCol:    nameCol,
		NameEndCol: nameEndCol,
		Arguments:  args,
	})

	endLine, braces := p.parseBody(funcLine, args)
	p.scopes = append(p.scopes, FuncScope{
		Name: nameTok.Value, Start: funcLine, End: p.baseLine + endLine,
		span: p.functionSpan(FunctionValue, nameTok.Value, "", startTok, params.parens, braces, params.defaults),
	})
}

func (p *scriptParser) tokenNameRange(tok Token) (line, col, endCol uint32) {
	return nameRangeAt(p.sc.src, tok.Offset, p.baseLine+int(tok.Line), tok.Value)
}

// declareLoopVar declares the loop variable of `for (var row in qry)`.
//
// The two var-decl parsers only ever looked for `=`, so a `var` that binds
// through `in` declared nothing and go-to-definition on the loop variable
// failed — in every for-in loop, which is how CFML iterates a query, an array
// and a struct.
func (p *scriptParser) declareLoopVar(nameTok, next Token) {
	if next.Kind != TokIdent || !identEq(next.Value, "in") {
		return
	}

	p.declareVar(nameTok, ScopeLocal)
}

// parseCatchVar declares a catch block's exception variable.
//
// `catch (any e)` and `catch (e)` bind `e` for the block, and neither was
// declared: go-to-definition on the exception variable failed in every catch
// block there is.
func (p *scriptParser) parseCatchVar() {
	if p.sc.PeekSkipComments().Kind != TokLParen {
		return
	}

	p.sc.NextSkipComments() // consume (

	// The **last** identifier before the closing paren is the variable, and
	// whatever precedes it is the type — which is how one rule reads `catch (e)`,
	// `catch (any e)` and a dotted `catch (org.Foo e)` alike.
	var last Token

	for {
		tok := p.sc.NextSkipComments()

		switch tok.Kind {
		case TokIdent:
			last = tok
		case TokDot:
		case TokRParen, TokEOF:
			if last.Kind == TokIdent {
				p.declareVar(last, ScopeLocal)
			}

			return
		default:
			return
		}
	}
}

// declareVar records a binding that is neither an assignment nor an argument —
// a loop variable or a caught exception. Outside a function body there is no
// local scope to put it in, so it lands where an unscoped assignment would.
func (p *scriptParser) declareVar(nameTok Token, scope Scope) {
	if !p.inFunc.in {
		scope = ScopeVariables
	} else if scope == ScopeLocal {
		p.localVarSet.add(nameTok.Value)
	}

	p.vars = append(p.vars, VarDef{
		Name: nameTok.Value, Scope: scope,
		Line: conv.Uint32(p.baseLine + int(nameTok.Line)),
	})
}
