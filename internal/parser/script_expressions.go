// Reading the component an expression names: new, createObject, imports, entityNew and the instantiations found.

package parser

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// readNewComponent reads the component path after "new" keyword, and
// records the instantiation.
func (p *scriptParser) readNewComponent() string {
	at := p.sc.PeekSkipComments()
	comp := p.readNewComponentPath()
	p.addInstantiation(comp, at)

	return comp
}

// addInstantiation records that comp is built by the construction starting
// at tok. Gated as a call is (addCall): only a parse that extracts every
// call wants it, not one recording a few names' (call_names.go). A look ahead that rewinds and walks the same text again records
// the same token, which is kept once. The token's offset is the file's where
// the parse knows its origin: an interpolation's scanner, and every <cfset>
// a tag file hands over, count from zero, and two constructions on one line
// were the same entry.
func (p *scriptParser) addInstantiation(comp string, tok Token) {
	if !p.extractCalls || p.callNames() != nil || comp == "" || strings.HasPrefix(comp, "$") {
		return
	}

	at := tok.Offset
	if p.baseOffset >= 0 {
		at += p.baseOffset
	}

	in := Instantiation{Component: comp, Line: conv.Uint32(p.baseLine + int(tok.Line)), at: at}
	if p.extras == nil {
		p.extras = &scriptExtras{}
	}

	if !slices.Contains(p.extras.instantiations, in) {
		p.extras.instantiations = append(p.extras.instantiations, in)
	}
}

// instantiationsFound is what addInstantiation recorded.
func (p *scriptParser) instantiationsFound() []Instantiation {
	if p.extras == nil {
		return nil
	}

	return p.extras.instantiations
}

// readNewComponentPath reads the component path after "new" keyword.
func (p *scriptParser) readNewComponentPath() string {
	tok := p.sc.NextSkipComments()
	if tok.Kind == TokString {
		return unquote(tok.Value)
	}

	if tok.Kind != TokIdent {
		return ""
	}

	// Lucee's type prefix: `new java:java.io.File(…)`, `new cfml:models.Base(…)`.
	// The prefix is not part of the path — reading it as one yields a component
	// literally named "java", which then fails every method check against it.
	if (identEq(tok.Value, "java") || identEq(tok.Value, "cfml")) &&
		p.sc.PeekSkipComments().Kind == TokColon {
		p.sc.NextSkipComments() // consume the ':'

		path := p.readDottedPath(p.sc.NextSkipComments())

		if identEq(tok.Value, "cfml") {
			return p.applyImport(path)
		}

		// `java:` names a Java class, so it resolves through the same
		// javaStubsPath machinery as createObject("java", …).
		if path == "" {
			return ""
		}

		return p.resolveCall("createObject(\"java\",\"" + path + "\")")
	}

	// An inline component, `new component { … }` (Lucee), has no file to
	// name: its type is the body that follows. It was read as a path, a
	// component literally called "component", which then failed every
	// method check against it.
	//
	// It may carry attributes before the body (`new component accessors=true
	// { … }`, `javaSettings='…'`), which are consumed here: left to the
	// statement scan, `accessors=true` declared a variable called accessors.
	// A path would continue with a dot or a parenthesis, never a name.
	if identEq(tok.Value, "component") {
		switch p.sc.PeekSkipComments().Kind {
		case TokLBrace:
			return "$any"
		case TokIdent:
			if p.skipInlineComponentAttrs() {
				return "$any"
			}
		default:
		}
	}

	return p.applyImport(p.readDottedPath(tok))
}

// skipInlineComponentAttrs consumes the attributes between `new component`
// and its body, `name` or `name=value` pairs, and reports whether the body's
// `{` follows them. The scanner is restored when it does not.
func (p *scriptParser) skipInlineComponentAttrs() bool {
	saved := p.sc.Save()

	for p.sc.PeekSkipComments().Kind == TokIdent {
		p.sc.NextSkipComments()

		if p.sc.PeekSkipComments().Kind != TokEquals {
			continue
		}

		p.sc.NextSkipComments()

		switch p.sc.NextSkipComments().Kind {
		case TokString, TokIdent, TokNumber:
		default:
			p.sc.Restore(saved)

			return false
		}
	}

	if p.sc.PeekSkipComments().Kind == TokLBrace {
		return true
	}

	p.sc.Restore(saved)

	return false
}

// parseImport records `import models.User;`, so a later `new User()` resolves
// to models.User rather than to a component literally called User.
//
// Only the explicit form is recorded. `import models.*;` names a directory, and
// which component a bare name then refers to is a question about what is on
// disk — the parser has no filesystem and guessing would produce a confident
// wrong path, so a wildcard import is left to resolve as it did.
func (p *scriptParser) parseImport() {
	tok := p.sc.NextSkipComments()
	if tok.Kind == TokString {
		p.parseQuotedImport(unquote(tok.Value))

		return
	}

	path := p.readDottedPath(tok)
	if path == "" {
		return
	}

	_, last, ok := strings.CutLast(path, ".")
	if !ok {
		return
	}

	if last == "" || last == "*" {
		return
	}

	if p.imports == nil {
		p.imports = make(map[string]string, 4)
	}

	p.imports[strings.ToLower(last)] = path
}

// parseQuotedImport records `import "com.x.Class";`. The quoted form is how
// Lucee brings in a Java class (the component's javaSettings supply its jar),
// which the parser cannot see, so the bare name is a value of unknown shape.
func (p *scriptParser) parseQuotedImport(path string) {
	_, last, ok := strings.CutLast(path, ".")
	if !ok || last == "" || last == "*" {
		return
	}

	if p.imports == nil {
		p.imports = make(map[string]string, 4)
	}

	p.imports[strings.ToLower(last)] = "$any"
}

// applyImport rewrites a bare component name that an import brought into scope.
// A dotted path is already qualified and is left alone.
func (p *scriptParser) applyImport(comp string) string {
	if comp == "" || len(p.imports) == 0 || strings.ContainsRune(comp, '.') {
		return comp
	}

	var buf foldScratch
	if full, ok := p.imports[string(buf.lowerFold(comp))]; ok {
		return full
	}

	return comp
}

// readDottedPath continues a dotted path from an identifier already read,
// returning "" if that token was not an identifier.
func (p *scriptParser) readDottedPath(tok Token) string {
	if tok.Kind != TokIdent {
		return ""
	}

	var comp chainBuilder

	comp.reset(tok.Value)

	for {
		if p.sc.PeekSkipComments().Kind == TokDot {
			p.sc.NextSkipComments()

			next := p.sc.NextSkipComments()
			if next.Kind == TokIdent {
				comp.writeDot()
				comp.writeString(next.Value)
			}
		} else {
			break
		}
	}

	return comp.String()
}

// readCreateObjectComponent reads the component path from createObject("component","path")
// or resolves createObject("java","class") via configured resolvers. Scanner is left at the
// closing ) so the caller can consume it and scan for chained calls.
func (p *scriptParser) readCreateObjectComponent() string {
	at := p.sc.PeekSkipComments()
	comp := p.readCreateObjectPath()
	p.addInstantiation(comp, at)

	return comp
}

// readCreateObjectPath reads what createObject's arguments name.
func (p *scriptParser) readCreateObjectPath() string {
	if p.sc.NextSkipComments().Kind != TokLParen {
		return ""
	}

	arg1 := p.sc.NextSkipComments()
	if arg1.Kind != TokString {
		return ""
	}

	arg1Val := unquote(arg1.Value)

	if p.sc.NextSkipComments().Kind != TokComma {
		return ""
	}

	arg2 := p.sc.NextSkipComments()
	if arg2.Kind == TokRParen || arg2.Kind == TokEOF {
		return ""
	}

	// A path computed at run time — createObject( "component",
	// drivernames[ type ] ), "pkg." & name — names whichever component the
	// program picks, as an unmapped #...# in a literal path does, and is
	// dynamic for the same reason. Nothing was recorded for it, so every call
	// on the variable was "no component ref", and the scan stopped inside the
	// argument.
	if arg2.Kind != TokString || !p.atArgEnd() {
		p.skipComputedArg(arg2)

		return "$any"
	}

	if identEq(arg1Val, "component") {
		return unquote(arg2.Value)
	}

	// Non-component createObject (e.g. java) — try resolvers
	expr := "createObject(\"" + arg1Val + "\",\"" + unquote(arg2.Value) + "\")"

	return p.resolveCall(expr)
}

// atArgEnd reports whether the next token ends an argument.
func (p *scriptParser) atArgEnd() bool {
	k := p.sc.PeekSkipComments().Kind

	return k == TokRParen || k == TokComma
}

// skipComputedArg consumes the rest of an argument whose first token, first,
// is already consumed, recording the calls it makes, and stops before the ,
// or ) that ends it.
func (p *scriptParser) skipComputedArg(first Token) {
	depth := 0

	for tok := first; ; tok = p.sc.NextSkipComments() {
		switch tok.Kind {
		case TokLParen, TokLBracket, TokLBrace:
			depth++
		case TokRParen, TokRBracket, TokRBrace:
			depth--

			p.handleLiteralToken(tok)
		case TokIdent:
			p.scanNestedCall(tok)
		case TokString:
			p.handleLiteralToken(tok)
		case TokEOF:
			return
		default:
		}

		if next := p.sc.PeekSkipComments().Kind; next == TokEOF || (depth <= 0 && (next == TokRParen || next == TokComma)) {
			return
		}
	}
}

// readEntityNewComponent reads the entity name from entityNew("Name").
func (p *scriptParser) readEntityNewComponent() string {
	if p.sc.NextSkipComments().Kind != TokLParen {
		return ""
	}

	arg := p.sc.NextSkipComments()
	if arg.Kind != TokString {
		return ""
	}

	return p.entityArg(arg)
}

// entityArg is the entity the string arg names, or $any when the name is
// computed: entityNew( "Comp" & nbr ) is no entity called Comp.
func (p *scriptParser) entityArg(arg Token) string {
	if next := p.sc.PeekSkipComments().Kind; next != TokRParen && next != TokComma {
		return "$any"
	}

	return entityRef(unquote(arg.Value))
}

// EntityPrefix marks a component named as an ORM entity, by entityNew() or
// entityLoad(): the name is an entityname, which the resolver looks up as
// one before reading it as a path. Read as a path, entityNew( "question" ) in
// FW/1's services/question.cfc named that service, beside it.
const EntityPrefix = "entity:"

// entityRef is name marked as an entity name, or "" for none.
func entityRef(name string) string {
	if name == "" || name == "$any" {
		return name
	}

	return EntityPrefix + name
}
