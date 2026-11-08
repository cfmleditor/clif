// The variable-only scan of a script: it skips every function body, which is what makes VariablesVars and ThisVars answerable per keystroke.

package parser

import (
	"github.com/cfmleditor/clif/internal/conv"
)

// globalScriptParser extracts only variable declarations outside function
// bodies. It is a second, much smaller scan than scriptParser's, which is what
// makes VariablesVars/ThisVars cheap enough to answer per keystroke — it skips
// every function body outright rather than parsing it.
//
// Being a second implementation, it is also a second place every rule about
// what does *not* declare a variable has to hold: named arguments, struct
// literal keys and script-tag attributes all fooled both.
// TestBothVariableScansAgree pins them together.
type globalScriptParser struct {
	sc       *Scanner
	vars     []VarDef
	baseLine int
	scopes   []FuncScope
	// scopeAt is findFuncScope's answer by index, through scopeIndex: asked
	// once per function, a scan of every scope was quadratic in a component
	// of thousands of them.
	scopeAt func(line int) int
}

// newGlobalScriptParser is only ever handed a RegionScript's text, so its
// scanner reads interpolated strings the way scriptParser's does — otherwise
// the two variable scans would tokenise the same file differently.
func newGlobalScriptParser(src string, baseLine int, scopes []FuncScope) *globalScriptParser {
	sc := NewScanner(blankNonScript(src))
	sc.interpStrings = true

	return &globalScriptParser{
		sc:       sc,
		baseLine: baseLine,
		scopes:   scopes,
		scopeAt:  scopeIndex(scopes, true),
	}
}

// funcScopeAt is findFuncScope(line, p.scopes).
func (p *globalScriptParser) funcScopeAt(line int) FuncScope {
	if i := p.scopeAt(line); i >= 0 {
		return p.scopes[i]
	}

	return FuncScope{Start: -1, End: -1}
}

func (p *globalScriptParser) parse() {
	afterLT := false

	for {
		tok := p.sc.NextSkipComments()
		if tok.Kind == TokEOF {
			return
		}

		wasLT := afterLT
		afterLT = tok.Kind == TokLT

		if tok.Kind != TokIdent {
			continue
		}

		line := p.baseLine + int(tok.Line)
		// Skip if inside a function
		if p.funcScopeAt(line).Start != -1 {
			// Skip to end of function body
			p.skipPastScope(line)

			continue
		}

		var buf foldScratch
		switch string(buf.lowerFold(tok.Value)) {
		case "var":
			p.parseVar(tok)
		case "local":
			p.parseDot(tok, ScopeVariables) // local outside func → variables
		case "this":
			p.parseDot(tok, ScopeThis)
		case "variables":
			p.parseDot(tok, ScopeVariables)
		case "component", "interface":
			// Their attribute lists are not assignments. parsePlain's keyword
			// guard runs before its tag-attribute check and both names are
			// keywords, so without this arm `component extends="models.Base"
			// accessors="true"` put `extends` and `accessors` into
			// VariablesVars — which is what completion offers and what the
			// index stores.
			p.skipTagAttrs()
		case "function", "public", "private", "remote", "package":
			// skip function declarations
		case "property":
			// The same test as scriptParser.parse's, which the two scans must
			// share: only a declaration is skipped.
			if !startsPropertyDecl(p.sc.PeekSkipComments()) {
				p.parsePlain(tok, wasLT)

				break
			}

			// skip property declaration tokens until semicolon
			for {
				t := p.sc.NextSkipComments()
				if t.Kind == TokEOF || t.Kind == TokSemicolon {
					break
				}
			}
		default:
			p.parsePlain(tok, wasLT)
		}
	}
}

func (p *globalScriptParser) skipPastScope(line int) {
	fs := p.funcScopeAt(line)
	if fs.Start == -1 {
		return
	}
	// Skip tokens until we're past the function's end line
	for {
		tok := p.sc.NextSkipComments()
		if tok.Kind == TokEOF {
			return
		}

		if p.baseLine+int(tok.Line) > fs.End {
			return
		}
	}
}

func (p *globalScriptParser) parseVar(tok Token) {
	name := p.sc.NextSkipComments()
	if name.Kind != TokIdent {
		return
	}

	eq := p.sc.PeekSkipComments()
	if eq.Kind != TokEquals {
		return
	}

	p.vars = append(p.vars, VarDef{
		Name: name.Value, Scope: ScopeVariables,
		Line: conv.Uint32(p.baseLine + int(tok.Line)),
	})

	p.consumeAssignment()
}

func (p *globalScriptParser) parseDot(tok Token, scope Scope) {
	dot := p.sc.PeekSkipComments()
	if dot.Kind != TokDot {
		return
	}

	p.sc.NextSkipComments()

	name := p.sc.NextSkipComments()
	if name.Kind != TokIdent {
		return
	}

	eq := p.sc.PeekSkipComments()
	if eq.Kind != TokEquals {
		return
	}

	p.vars = append(p.vars, VarDef{
		Name: name.Value, Scope: scope,
		Line: conv.Uint32(p.baseLine + int(tok.Line)),
	})

	p.consumeAssignment()
}

func (p *globalScriptParser) parsePlain(tok Token, afterLT bool) {
	if isKeyword(tok.Value) {
		return
	}

	eq := p.sc.PeekSkipComments()

	switch {
	case eq.Kind == TokLParen:
		// A call. Its argument list is not statements, and a named argument
		// written `f(force = true)` declares nothing.
		p.skipGroup()

		return
	case !afterLT && eq.Kind == TokIdent && looksLikeTagAttrs(p.sc):
		// A script-syntax CF tag. See scriptParser.parseScriptTagAttrs.
		p.skipTagAttrs()

		return
	case eq.Kind != TokEquals:
		return
	}

	p.vars = append(p.vars, VarDef{
		Name: tok.Value, Scope: ScopeVariables,
		Line: conv.Uint32(p.baseLine + int(tok.Line)),
	})

	p.consumeAssignment()
}

// consumeAssignment takes the "=" its three callers have only peeked at, and
// with it a struct or array literal standing as the right-hand side.
//
// This scan reads a bare `ident =` as a declaration, so anything it walks into
// that spells one declares a variable — and `{force = true}` spells one.
func (p *globalScriptParser) consumeAssignment() {
	p.sc.NextSkipComments() // consume =

	switch p.sc.PeekSkipComments().Kind {
	case TokLBrace, TokLBracket:
		p.skipGroup()
	default:
	}
}

// skipGroup consumes a balanced (...), {...} or [...] group that the scanner is
// positioned on. Returns false if it is on none of them, or the group is
// unclosed.
func (p *globalScriptParser) skipGroup() bool {
	var open, closing TokenKind

	switch p.sc.PeekSkipComments().Kind {
	case TokLParen:
		open, closing = TokLParen, TokRParen
	case TokLBrace:
		open, closing = TokLBrace, TokRBrace
	case TokLBracket:
		open, closing = TokLBracket, TokRBracket
	default:
		return false
	}

	p.sc.NextSkipComments() // consume the opener

	depth := 1

	for depth > 0 {
		switch tok := p.sc.NextSkipComments(); tok.Kind {
		case TokEOF:
			return false
		case open:
			depth++
		case closing:
			depth--
		default:
			// Any other token is passed over.
		}
	}

	return true
}

// skipTagAttrs consumes a script-syntax CF tag's attribute list, stopping
// before the body or the terminating semicolon.
func (p *globalScriptParser) skipTagAttrs() {
	for looksLikeTagAttrs(p.sc) {
		p.sc.NextSkipComments() // attribute name
		p.sc.NextSkipComments() // =

		if !p.skipTagAttrValue() {
			return
		}
	}
}

// skipTagAttrValue consumes one attribute value. See the scriptParser method of
// the same name for why it stops where it does.
func (p *globalScriptParser) skipTagAttrValue() bool {
	for {
		switch p.sc.PeekSkipComments().Kind {
		case TokEOF, TokSemicolon, TokLBrace, TokRBrace, TokLT, TokGT:
			return false
		case TokHash:
			p.sc.NextSkipComments() // consume the opening #

			for {
				tok := p.sc.NextSkipComments()
				if tok.Kind == TokEOF || tok.Kind == TokLBrace ||
					tok.Kind == TokLT || tok.Kind == TokGT {
					return false
				}

				if tok.Kind == TokHash {
					break
				}
			}
		case TokLParen, TokLBracket:
			if !p.skipGroup() {
				return false
			}
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
