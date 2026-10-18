package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// The registry consumers loop over the array a family member's getter
// returns, and their pages read an element's accessors:
//
//	<cfloop array="#driver.getCustomFields()#" index="field">
//		<cfoutput>#field.getName()#</cfoutput>
//
// The sources say what the elements are. The family's base declares
// field()/group() returning createObject("component","Field"/"Group").init(...)
// and getCustomFields() returning `fields`; each member's own file assigns
// that variable an array literal of those calls — Classic's fields=array(...),
// the AI subclasses' variables.fields = [ ... ] with named arguments. The
// loop element is thus the union of the components the literals name —
// all of them, over the members the form may pick — and it is only an answer
// when every member's file proves a non-empty set: any other
// shape of assignment — an append, an unassigned getter, an item whose
// function returns anything else — leaves the element untyped. On the union,
// a method any alternative declares resolves, and the pages'
// isInstanceOf(field,"Group") guards still split the branches.
//
// The union only answers an element read; the array itself has no component.

// familyLoopElement is the registry family's array getter's element, or ""
// when the family does not prove one. Every member's own files must give a
// non-empty literal element set — the page serves whichever member the form
// chose, and the element is the union of what members' literals name.
func (r *Resolver) familyLoopElement(members []string, method string) string {
	var elements []string

	for _, member := range members {
		set := r.memberArrayElements(member, method)
		if set == "" {
			return ""
		}

		for comp := range strings.SplitSeq(set, "|") {
			if !slices.Contains(elements, comp) {
				elements = append(elements, comp)
			}
		}
	}

	if len(elements) == 0 {
		return ""
	}

	slices.Sort(elements)

	return strings.Join(elements, "|")
}

// memberArrayElements is one member's element union. Every own-level return
// of the method resolves to one identifier, every assignment to it in the
// member's own file is an array or list literal, and every literal item is a
// call whose function returns, everywhere, one component by a literal path.
func (r *Resolver) memberArrayElements(member, method string) string {
	fd := r.LookupFuncWithExtends(member, method)
	if fd == nil {
		return ""
	}

	name := r.fdReturnsLocal(fd)
	if name == "" {
		return ""
	}

	assignments, ok := r.memberAssignmentLiterals(name, member)
	if !ok {
		return ""
	}

	var elements []string

	for _, rhs := range assignments {
		items, ok := literalItems(rhs)
		if !ok {
			return ""
		}

		for _, item := range items {
			comp, ok := r.constructorItemComponent(member, item)
			if !ok || comp == "" {
				return ""
			}

			if !slices.Contains(elements, comp) {
				elements = append(elements, comp)
			}
		}
	}

	if len(elements) == 0 {
		return ""
	}

	slices.Sort(elements)

	return strings.Join(elements, "|")
}

// fnScopeTokens is fd's function's span as tokens — the script body's or the
// tag cffunction's — or ok=false. Unlike fnBody it does not need a brace to
// open what a tag function's body has none of.
func (r *Resolver) fnScopeTokens(fd *parser.FunctionDef) ([]parser.Token, bool) {
	if fd == nil || !fd.URI.IsFile() {
		return nil, false
	}

	pr := r.handlerParse(fd.URI.Path())
	if pr == nil {
		return nil, false
	}

	scope := pr.FuncScopeAt(int(fd.Line))
	if scope.Start == -1 || !strings.EqualFold(scope.Name, fd.Name) {
		return nil, false
	}

	if _, ok := lineOffset(pr.Content, scope.End); !ok {
		return nil, false
	}

	return significantTokens(sourceText(pr.Content, scope.Start, scope.End+1)), true
}

// fdReturnsLocal is the one name every return at fd's own level gives —
// `fields` or `variables.fields` — or "". Script `return` and tag
// `cfreturn` read the same here.
func (r *Resolver) fdReturnsLocal(fd *parser.FunctionDef) string {
	tokens, ok := r.fnScopeTokens(fd)
	if !ok {
		return ""
	}

	held := ""

	for i := 0; i < len(tokens); i++ {
		switch {
		case tokens[i].Kind == parser.TokLBrace:
			if closureBody(tokens, i) {
				if e := matchingBrace(tokens, i); e > 0 {
					i = e
				}
			}
		case tokens[i].Kind == parser.TokIdent && (strings.EqualFold(tokens[i].Value, "return") || strings.EqualFold(tokens[i].Value, "cfreturn")):
			name := localNameOf(returnExpr(tokens, i+1, len(tokens)))
			if name == "" {
				return ""
			}

			if held != "" && !strings.EqualFold(held, name) {
				return ""
			}

			held = name
		default:
		}
	}

	return held
}

// returnExpr is the tokens a return at from names: script `return` runs to
// its `;`, a tag `cfreturn` runs to the tag that follows it; either ends at
// the body's end.
func returnExpr(tokens []parser.Token, from, end int) []parser.Token {
	for at := from; at < end; at++ {
		switch tokens[at].Kind {
		case parser.TokLT, parser.TokGT, parser.TokSemicolon:
			return tokens[from:at]
		default:
		}
	}

	return tokens[from:end]
}

// localNameOf is a plain or variables-prefixed identifier: an expression the
// getter returns and the member's own environments assign.
func localNameOf(tokens []parser.Token) string {
	switch len(tokens) {
	case 1:
		if tokens[0].Kind == parser.TokIdent {
			return tokens[0].Value
		}
	case 3:
		if tokens[0].Kind == parser.TokIdent && strings.EqualFold(tokens[0].Value, "variables") &&
			tokens[1].Kind == parser.TokDot && tokens[2].Kind == parser.TokIdent {
			return tokens[2].Value
		}
	}

	return ""
}

// memberAssignmentLiterals is every assignment of name in the member's own
// file, or ok=false when the file assigns it none. Whether each is a literal
// is the caller's demand.
func (r *Resolver) memberAssignmentLiterals(name, member string) ([][]parser.Token, bool) {
	pr := r.handlerParse(member)
	if pr == nil {
		return nil, false
	}

	tokens := significantTokens(pr.Content)

	var found [][]parser.Token

	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokIdent || tokens[i+1].Kind != parser.TokEquals ||
			!strings.EqualFold(tokens[i].Value, name) {
			continue
		}

		stop := assignmentEnd(tokens, i+2)
		if stop < len(tokens) {
			stop++
		}

		if stop > len(tokens) {
			return nil, false
		}

		found = append(found, tokens[i+2:stop])
		i = stop - 1
	}

	if len(found) == 0 {
		return nil, false
	}

	return found, true
}

// assignmentEnd is the index of the token that closes the grouping value an
// assignment names — array( ... ) or [ ... ] — or, for another kind of value,
// the token its statement ends at.
func assignmentEnd(tokens []parser.Token, from int) int {
	if from < len(tokens) {
		switch tokens[from].Kind {
		case parser.TokLParen:
			if at := matchingParen(tokens, from); at > 0 {
				return at
			}
		case parser.TokLBracket:
			if at := matchingBracket(tokens, from); at > 0 {
				return at
			}
		default:
			if from+2 < len(tokens) && tokens[from].Kind == parser.TokIdent &&
				strings.EqualFold(tokens[from].Value, "array") && tokens[from+1].Kind == parser.TokLParen {
				if at := matchingParen(tokens, from+1); at > 0 {
					return at
				}
			}
		}
	}

	return statementEnd(tokens, from, len(tokens))
}

// matchingParen is the index of the ) closing the ( at open.
func matchingParen(tokens []parser.Token, open int) int {
	depth := 0

	for i := open; i < len(tokens); i++ {
		switch tokens[i].Kind {
		case parser.TokLParen:
			depth++
		case parser.TokRParen:
			depth--

			if depth == 0 {
				return i
			}
		default:
		}
	}

	return -1
}

// matchingBracket is the index of the ] closing the [ at open.
func matchingBracket(tokens []parser.Token, open int) int {
	depth := 0

	for i := open; i < len(tokens); i++ {
		switch tokens[i].Kind {
		case parser.TokLBracket:
			depth++
		case parser.TokRBracket:
			depth--

			if depth == 0 {
				return i
			}
		default:
		}
	}

	return -1
}

// literalItems is the top-level comma items of the array( ... ) or [ ... ]
// literal rhs, or ok=false when it is another kind of value. The literal's
// own closing token may be in the rhs or not: the items stop at the outer
// balance either way.
// literalStartOf is where a literal's items begin: after the array( or the
// opening [.
func literalStartOf(rhs []parser.Token) (int, bool) {
	switch {
	case len(rhs) > 2 && rhs[0].Kind == parser.TokIdent && strings.EqualFold(rhs[0].Value, "array") &&
		rhs[1].Kind == parser.TokLParen:
		return 2, true
	case len(rhs) > 1 && rhs[0].Kind == parser.TokLBracket:
		return 1, true
	default:
		return 0, false
	}
}

func literalItems(rhs []parser.Token) ([][]parser.Token, bool) {
	if len(rhs) < 3 {
		return nil, false
	}

	start, ok := literalStartOf(rhs)
	if !ok {
		return nil, false
	}

	var (
		items     [][]parser.Token
		itemStart = start
		depth     = 1
		end       = len(rhs)
	)

	for i := start; i < end; i++ {
		switch t := rhs[i]; t.Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			depth--

			if depth < 0 {
				return nil, false
			}

			if depth == 0 {
				end = i
			}
		case parser.TokComma:
			if depth != 1 {
				continue
			}

			if i == itemStart {
				return nil, false
			}

			items = append(items, rhs[itemStart:i])
			itemStart = i + 1
		default:
		}
	}

	if itemStart >= end {
		return nil, false
	}

	items = append(items, rhs[itemStart:end])
	if len(items) == 0 {
		return nil, false
	}

	return items, true
}

// constructorItemComponent is the component a literal item names: the item is
// a bare call whose function — the member's own chain, usually an extends
// base's field()/group() — returns, everywhere, the construction
// createObject("component", literal), and the path resolves to one file
// beside that function's own.
func (r *Resolver) constructorItemComponent(member string, item []parser.Token) (string, bool) {
	if len(item) < 2 || item[0].Kind != parser.TokIdent ||
		item[1].Kind != parser.TokLParen || item[len(item)-1].Kind != parser.TokRParen {
		return "", false
	}

	fd := r.LookupFuncWithExtends(member, item[0].Value)
	if fd == nil {
		return "", false
	}

	dir := filepath.Dir(fd.URI.Path())

	tokens, ok := r.fnScopeTokens(fd)
	if !ok {
		return "", false
	}

	first := ""

	for i := 0; i < len(tokens); i++ {
		switch {
		case tokens[i].Kind == parser.TokLBrace:
			if closureBody(tokens, i) {
				if e := matchingBrace(tokens, i); e > 0 {
					i = e
				}
			}
		case tokens[i].Kind == parser.TokIdent && (strings.EqualFold(tokens[i].Value, "return") || strings.EqualFold(tokens[i].Value, "cfreturn")):
			lit := producedComponent(producerText(returnExpr(tokens, i+1, len(tokens))))
			if lit == "" {
				return "", false
			}

			comp := r.componentFromConstruction(lit, dir)
			if comp == "" {
				return "", false
			}

			// The construction's own directory proved the component: keep its
			// file, not the spelling it went by. Two families' helpers are
			// spelled the same — an appender's Field and a layout's Field —
			// and the element union must not collapse them into one name.
			if p := r.ComponentPath(comp, dir); p != "" {
				comp = p
			}

			if first == "" {
				first = comp
			} else if !cfpath.SamePath(first, comp) {
				return "", false
			}
		default:
		}
	}

	if first == "" {
		return "", false
	}

	return first, true
}

// producedComponent is the component path the text's
// createObject("component", literal) names, or "".
func producedComponent(text string) string {
	if !strings.HasPrefix(strings.TrimSpace(text), "createObject") &&
		!strings.HasPrefix(strings.ToLower(strings.TrimSpace(text)), "createobject") {
		return ""
	}

	m := createdComponent.FindStringSubmatch(text)
	if m == nil {
		return ""
	}

	return m[1]
}
