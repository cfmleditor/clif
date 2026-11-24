package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

func (p *scriptParser) parseArgList() []Argument {
	args, _, _ := p.parseArgs(false)

	return args
}

// parseArgs is parseArgList with what a FunctionSpan keeps of the list: the
// offset of its closing parenthesis in the region (-1 when it runs to the
// end), and its literal defaults (FunctionSpan.Defaults).
func (p *scriptParser) parseArgs(recordDeclarations bool) ([]Argument, int, map[string]string) {
	var localArgs [8]Argument

	args := localArgs[:0]
	closing := -1

	var defaults map[string]string

	for {
		tok := p.sc.NextSkipComments()
		if tok.Kind == TokRParen {
			closing = tok.Offset

			break
		}

		if tok.Kind == TokEOF {
			break
		}

		if tok.Kind == TokComma {
			continue
		}

		if tok.Kind != TokIdent {
			continue
		}

		var required bool

		var typeName, name string

		// Stack-backed array of 3 because the loop below stops there: an argument
		// is at most `required type name`.
		var identsBuf [3]string

		identsBuf[0] = tok.Value
		idents := identsBuf[:1]
		nameTok := tok

	loop:
		for {
			peek := p.sc.PeekSkipComments()
			switch peek.Kind {
			case TokDot:
				p.sc.NextSkipComments() // consume dot

				next := p.sc.PeekSkipComments()
				if next.Kind == TokIdent {
					p.sc.NextSkipComments()

					idents[len(idents)-1] += "." + next.Value
				}
			case TokIdent:
				p.sc.NextSkipComments()

				idents = append(idents, peek.Value)
				nameTok = peek

				if len(idents) == 3 {
					break loop
				}
			default:
				break loop
			}
		}

		switch len(idents) {
		case 1:
			name = idents[0]
		case 2:
			if identEq(idents[0], "required") {
				required = true
				name = idents[1]
			} else {
				typeName = idents[0]
				name = idents[1]
			}
		case 3:
			required = identEq(idents[0], "required")
			typeName = idents[1]
			name = idents[2]
		}

		peek := p.sc.PeekSkipComments()
		if peek.Kind == TokEquals {
			p.sc.NextSkipComments()

			if literal, ok := p.skipDefault(); ok {
				if defaults == nil {
					defaults = make(map[string]string, 1)
				}

				defaults[strings.ToLower(name)] = literal
			}
		}

		args = append(args, Argument{Name: name, Type: typeName, Required: required})
		if recordDeclarations {
			decl := VarDef{Name: name, Scope: ScopeArguments, Line: conv.Uint32(p.baseLine + int(nameTok.Line))}
			if p.baseOffset >= 0 {
				decl.NameStart = p.baseOffset + nameTok.Offset
			}

			p.vars = append(p.vars, decl)
		}
	}

	return p.allocArgs(args), closing, defaults
}

// skipDefault consumes an argument's default value, recording any calls written
// in it: `function load(id, dao = newDao())` calls newDao on every invocation
// that omits it, and the scan walked past it a token at a time.
//
// It returns the default when it is a literal string, one string token: its
// contents, with a doubled quote read as one.
func (p *scriptParser) skipDefault() (string, bool) {
	depth := 0

	var first Token

	n := 0

	for {
		peek := p.sc.PeekSkipComments()
		if peek.Kind == TokEOF {
			return "", false
		}

		if depth == 0 && (peek.Kind == TokComma || peek.Kind == TokRParen) {
			if n != 1 || first.Kind != TokString {
				return "", false
			}

			return unquoteDoubled(first.Value), true
		}

		if n == 0 {
			first = peek
		}

		n++

		p.sc.NextSkipComments()

		switch peek.Kind {
		case TokLParen, TokLBrace, TokLBracket:
			depth++
		case TokRParen, TokRBrace, TokRBracket:
			depth--
		case TokIdent:
			p.scanNestedCall(peek)
		case TokString:
			p.handleLiteralToken(peek)
		default:
			// Any other token is passed over.
		}
	}
}
