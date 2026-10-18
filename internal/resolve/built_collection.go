package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// builtCollectionElement is the element of the collection a function builds:
// the body starts a local empty — arrayNew(1) or an empty [ ] — appends
// values to it with arrayAppend, and every return at the function's own
// level names it, or a field assigned from it. The appended values' common
// component is the array's element. Any return of something else, or an
// append whose value nobody proves, is an open world: the collection could
// hold anything, and the answer is withheld.
//
//	<cffunction name="getExtendSets" returntype="array">
//		<cfset var extendArray=arrayNew(1)>
//		<cfloop ...>
//			<cfset extendSetBean=getExtendSetBean()>
//			<cfset arrayAppend(extendArray, extendSetBean)>
//		</cfloop>
//		<cfreturn extendArray>
//	</cffunction>
//
// The factory's own return — createObject("component", literal).init(...)
// through a local — is the element, and the consumer's keyed read
// (extendSets[s]) types against it.
func (r *Resolver) builtCollectionElement(fd *parser.FunctionDef) string {
	pr := r.handlerParse(fd.URI.Path())
	if pr == nil {
		return ""
	}

	scope := pr.FuncScopeAt(int(fd.Line))
	if scope.Start == -1 || !strings.EqualFold(scope.Name, fd.Name) {
		return ""
	}

	tokens, ok := r.fnScopeTokens(fd)
	if !ok {
		return ""
	}

	// The span's tokens count lines from the function's first line; the
	// values' lookups need the file's own.
	lineOf := func(rel uint32) uint32 {
		return conv.Uint32(scope.Start + int(rel) - 1)
	}

	built, ok := builtArrayLocal(tokens)
	if !ok {
		return ""
	}

	values, ok := builtAppends(tokens, built)
	if !ok || len(values) == 0 {
		return ""
	}

	if !returnsBuiltCollection(tokens, built) {
		return ""
	}

	dir := filepath.Dir(fd.URI.Path())

	var elements []string

	for _, v := range values {
		comp := r.argumentExprComponent(v.tokens, lineOf(v.line), fd.Name, pr, dir, lookupCtx{})
		if comp == "" {
			return ""
		}

		for alt := range strings.SplitSeq(comp, "|") {
			if !slices.ContainsFunc(elements, func(held string) bool { return cfpath.SamePath(held, alt) }) {
				elements = append(elements, alt)
			}
		}
	}

	slices.Sort(elements)

	return strings.Join(elements, "|")
}

// builtArrayLocal is the name the body assigns an empty array — the
// collection the appends fill — when exactly one local is.
func builtArrayLocal(tokens []parser.Token) (string, bool) {
	name := ""

	for i := 1; i+2 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokEquals || tokens[i+1].Kind != parser.TokIdent ||
			!strings.EqualFold(tokens[i+1].Value, "arrayNew") || tokens[i-1].Kind != parser.TokIdent {
			continue
		}

		if name != "" && !strings.EqualFold(name, tokens[i-1].Value) {
			return "", false
		}

		name = tokens[i-1].Value
	}

	return name, name != ""
}

// appendValue is one arrayAppend's value: the tokens and the line they start.
type appendValue struct {
	tokens []parser.Token
	line   uint32
}

// builtAppends is every arrayAppend's value whose target is the built
// local, or ok=false when an append targets something else — a value the
// body ships elsewhere is not this collection's element.
func builtAppends(tokens []parser.Token, built string) ([]appendValue, bool) {
	var values []appendValue

	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokIdent || !strings.EqualFold(tokens[i].Value, "arrayAppend") ||
			tokens[i+1].Kind != parser.TokLParen {
			continue
		}

		args, ok := callArgs(tokens, i+1)
		if !ok || len(args) != 2 {
			return nil, false
		}

		if target := localNameOf(args[0]); target == "" || !strings.EqualFold(target, built) {
			return nil, false
		}

		if len(args[1]) == 0 {
			return nil, false
		}

		values = append(values, appendValue{tokens: args[1], line: conv.Uint32(args[1][0].Line)})
	}

	return values, len(values) > 0
}

// callArgs is the top-level comma arguments of the call whose ( is at open,
// or ok=false when the parens do not close.
func callArgs(tokens []parser.Token, open int) ([][]parser.Token, bool) {
	last := matchingParen(tokens, open)
	if last < 0 {
		return nil, false
	}

	var (
		args      [][]parser.Token
		itemStart = open + 1
		depth     = 0
	)

	for i := open + 1; i < last; i++ {
		switch tokens[i].Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			depth--
		case parser.TokComma:
			if depth != 0 {
				continue
			}

			args = append(args, tokens[itemStart:i])
			itemStart = i + 1
		default:
		}
	}

	args = append(args, tokens[itemStart:last])

	return args, true
}

// returnsBuiltCollection reports whether every return at the function's own
// level names the built local, or a field the body assigns it to.
func returnsBuiltCollection(tokens []parser.Token, built string) bool {
	returned := false

	for i := 0; i < len(tokens); i++ {
		switch {
		case tokens[i].Kind == parser.TokLBrace:
			if closureBody(tokens, i) {
				if e := matchingBrace(tokens, i); e > 0 {
					i = e
				}
			}
		case tokens[i].Kind == parser.TokIdent && (strings.EqualFold(tokens[i].Value, "return") || strings.EqualFold(tokens[i].Value, "cfreturn")):
			if !isBuiltOrAlias(tokens, localNameOf(returnExpr(tokens, i+1, len(tokens))), built) {
				return false
			}

			returned = true
		default:
		}
	}

	return returned
}

// isBuiltOrAlias reports whether name is the built local, or a field the
// body's assignments fill from it: the cached copy the getter returns is
// the collection it built.
func isBuiltOrAlias(tokens []parser.Token, name, built string) bool {
	if strings.EqualFold(name, built) {
		return true
	}

	if name == "" {
		return false
	}

	for i := 1; i+1 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokEquals || tokens[i-1].Kind != parser.TokIdent ||
			!strings.EqualFold(tokens[i-1].Value, name) {
			continue
		}

		end := i + 1
		for end < len(tokens) && tokens[end].Kind != parser.TokGT && tokens[end].Kind != parser.TokLT && tokens[end].Kind != parser.TokSemicolon {
			end++
		}

		if assignsExactly(tokens[i+1:end], built) {
			return true
		}
	}

	return false
}

// assignsExactly reports whether the value's tokens are exactly the local:
// the name alone, or variables.name.
func assignsExactly(tokens []parser.Token, name string) bool {
	if len(tokens) == 1 {
		return tokens[0].Kind == parser.TokIdent && strings.EqualFold(tokens[0].Value, name)
	}

	if len(tokens) == 3 {
		return tokens[0].Kind == parser.TokIdent && strings.EqualFold(tokens[0].Value, "variables") &&
			tokens[1].Kind == parser.TokDot && tokens[2].Kind == parser.TokIdent &&
			strings.EqualFold(tokens[2].Value, name)
	}

	return false
}
