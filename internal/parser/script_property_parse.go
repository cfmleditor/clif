package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// parsePropertyWord handles the word `property`. A declaration is followed by a
// name or a type; `for( property in x )`, `property == y` and `f( property )` are
// a variable, and reading them as a declaration swallowed everything up to the
// next semicolon.
func (p *scriptParser) parsePropertyWord(tok Token) {
	switch next := p.sc.PeekSkipComments(); {
	case startsPropertyDecl(next):
		p.parseProperty(tok)
	case next.Kind == TokLParen:
		p.recordBareCallAndChain(tok)
	default:
		p.checkAssignRef(tok)
	}
}

// parseProperty handles script-style property declarations:
//
//	property name="person" type="models.Person";
//	property string name;
//	property name;
func (p *scriptParser) parseProperty(startTok Token) {
	line := conv.Uint32(p.baseLine + int(startTok.Line))

	var name, typeName string

	attrs := make(map[string]string)

	// Collect tokens until semicolon or EOF
	var tokens []Token

	endTok := startTok
	inHash := false // between the hashes of an unquoted `default=#f()#`

	for {
		tok := p.sc.NextSkipComments()
		if tok.Kind == TokEOF {
			break
		}

		endTok = tok
		if tok.Kind == TokSemicolon {
			break
		}

		// `default="#createUUID()#"`: a property's attribute is an expression,
		// and a call written in it is a call.
		if tok.Kind == TokString {
			p.scanInterpolation(tok)
		}

		// The same written without quotes, `default=#createUUID()#`, is the
		// hashes as tokens. A name met between them is the start of an
		// expression, which the argument scan reads whole.
		if tok.Kind == TokHash {
			inHash = !inHash
		} else if inHash && tok.Kind == TokIdent {
			p.scanNestedCall(tok)

			continue
		}

		tokens = append(tokens, tok)
	}

	// Check for attribute-style: property name="x" type="y" inject="z";
	// A flag WireBox reads by its presence (`inject delegate`) is recorded
	// with an empty value, and taken out of the positional form below.
	flags := 0

	var (
		nameTok    Token
		hasNameTok bool
	)

	for i, tok := range tokens {
		if tok.Kind != TokIdent {
			continue
		}

		if i+1 < len(tokens) && tokens[i+1].Kind == TokEquals {
			if i+2 < len(tokens) && (tokens[i+2].Kind == TokString || tokens[i+2].Kind == TokIdent || tokens[i+2].Kind == TokNumber) {
				val := propertyAttributeValue(tokens, i+2)
				key := strings.ToLower(tok.Value)
				attrs[key] = val

				if key == "name" {
					nameTok, hasNameTok = tokens[i+2], true
				}
			}

			continue
		}

		if key := strings.ToLower(tok.Value); propertyFlags[key] && (i > 0 || len(tokens) > 1) {
			if _, set := attrs[key]; !set {
				attrs[key] = ""
			}

			flags++
		}
	}

	name = attrs["name"]
	typeName = attrs["type"]

	// If no name= attribute, try positional: property [type] name [attrs...];
	if name == "" {
		var posType string

		posType, name, nameTok, hasNameTok = positionalPropertyName(tokens, flags)
		if posType != "" {
			typeName = posType
		}
	}

	if name == "" {
		return
	}

	if typeName != "" {
		attrs["type"] = typeName
	}

	p.properties = append(p.properties, p.buildPropertyDef(name, typeName, line, startTok, endTok, nameTok, hasNameTok, attrs))
}

func positionalPropertyName(tokens []Token, flags int) (typeName, name string, nameTok Token, hasNameTok bool) {
	idents := make([]Token, 0, 2)

	for i, tok := range tokens {
		if tok.Kind != TokIdent {
			break
		}

		if i+1 < len(tokens) && tokens[i+1].Kind == TokEquals {
			break
		}

		if flags > 0 && propertyFlags[strings.ToLower(tok.Value)] {
			break
		}

		idents = append(idents, tok)
	}

	switch len(idents) {
	case 1:
		return "", idents[0].Value, idents[0], true
	case 2:
		return idents[0].Value, idents[1].Value, idents[1], true
	default:
		return "", "", Token{}, false
	}
}

func (p *scriptParser) buildPropertyDef(name, typeName string, line uint32, startTok, endTok, nameTok Token, hasNameTok bool, attrs map[string]string) propertyDef {
	col := p.ensureExtras().cols.prefix(p.sc.src, startTok.Offset)
	endLine := conv.Uint32(p.baseLine + int(endTok.Line))
	endCol := p.ensureExtras().cols.prefix(p.sc.src, endTok.Offset+len(endTok.Value))

	var nameLine, nameCol, nameEndCol uint32

	if hasNameTok {
		off := nameTok.Offset
		if nameTok.Kind == TokString && len(nameTok.Value) >= 2 && (nameTok.Value[0] == '"' || nameTok.Value[0] == '\'') {
			off++
		}

		nameLine, nameCol, nameEndCol = p.ensureExtras().cols.nameRange(p.sc.src, off, p.baseLine+int(nameTok.Line), name)
	}

	return propertyDef{
		name:       name,
		typeName:   typeName,
		line:       line,
		col:        col,
		endLine:    endLine,
		endCol:     endCol,
		nameLine:   nameLine,
		nameCol:    nameCol,
		nameEndCol: nameEndCol,
		attrs:      attrs,
	}
}
