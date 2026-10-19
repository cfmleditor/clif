package resolve

import (
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
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
// when the family does not prove one. Every member's own files must prove a
// set — an empty one is a set too; the page serves whichever member the form
// chose, and the element is the union of what members' literals name.
func (r *Resolver) familyLoopElement(members []string, method string) string {
	var elements []string

	for _, member := range members {
		set, proved := r.memberArrayElements(member, method)
		if !proved {
			return ""
		}

		for comp := range strings.SplitSeq(set, "|") {
			if comp != "" && !slices.Contains(elements, comp) {
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

// memberArrayElements is one member's element union and whether the member's
// own files prove a set of it. Every own-level return of the method resolves
// to one identifier, every assignment to it in the member's own file is an
// array or list literal — save the copy a getter's own body makes of the
// member's own fields, whose source's literal is the set the copy proves —
// and every literal item is a call whose function returns, everywhere, one
// component by a literal path.
func (r *Resolver) memberArrayElements(member, method string) (string, bool) {
	fd := r.LookupFuncWithExtends(member, method)
	if fd == nil {
		return "", false
	}

	name := r.fdReturnsLocal(fd)
	if name == "" {
		return "", false
	}

	assignments, ok := r.memberAssignments(name, member)
	if !ok {
		return "", false
	}

	bodyStart, bodyEnd, spanned := r.memberGetterSpan(member, fd)

	var elements []string

	for _, as := range assignments {
		items, ok := literalItems(as.rhs)

		if !ok {
			// An empty literal is a set too: a member whose own file assigns
			// `fields=array()` proves it holds none, and contributes nothing
			// to the union.
			if emptyLiteral(as.rhs) {
				continue
			}

			// A member's own getter that copies its own fields walks the
			// copy's source — `var fields=duplicate(variables.fields)` —
			// and the copy proves the same set the member's own fields
			// literal holds. A non-literal anywhere else, or a copy of
			// something the member's own file does not assign, leaves the
			// element untyped.
			src, dup := as.copyOf, as.copies
			if !dup || !spanned || as.at < bodyStart || as.at >= bodyEnd ||
				!strings.EqualFold(src, name) {
				return "", false
			}

			items, ok = r.memberFieldItems(name, member, bodyStart, bodyEnd)
			if !ok {
				return "", false
			}
		}

		for _, item := range items {
			comp, ok := r.constructorItemComponent(member, item)
			if !ok || comp == "" {
				return "", false
			}

			if !slices.Contains(elements, comp) {
				elements = append(elements, comp)
			}
		}
	}

	slices.Sort(elements)

	return strings.Join(elements, "|"), true
}

// memberGetterSpan is the member's own file's byte range of the function
// fd's scope, or ok=false when the file holds no scope it names.
func (r *Resolver) memberGetterSpan(member string, fd *parser.FunctionDef) (start, end int, ok bool) {
	pr := r.handlerParse(member)
	if pr == nil || fd == nil {
		return 0, 0, false
	}

	scope := pr.FuncScopeAt(int(fd.Line))
	if scope.Start == -1 || !strings.EqualFold(scope.Name, fd.Name) {
		return 0, 0, false
	}

	start, ok = lineOffset(pr.Content, scope.Start)
	if !ok {
		return 0, 0, false
	}

	end, ok = lineOffset(pr.Content, scope.End+1)
	if !ok {
		end = len(pr.Content)
	}

	return start, end, true
}

// memberFieldItems is the literal items the member's own file proves for
// name outside the getter's body — the top-level fields literal a getter's
// own duplicate copy walks. An assignment there that is not a literal
// leaves the set unproven.
func (r *Resolver) memberFieldItems(name, member string, bodyStart, bodyEnd int) ([][]parser.Token, bool) {
	assignments, ok := r.memberAssignments(name, member)
	if !ok {
		return nil, false
	}

	var items [][]parser.Token

	for _, as := range assignments {
		if as.at >= bodyStart && as.at < bodyEnd { // the getter's own copy
			continue
		}

		lits, ok := literalItems(as.rhs)
		if !ok {
			return nil, false
		}

		items = append(items, lits...)
	}

	if len(items) == 0 {
		return nil, false
	}

	return items, true
}

// emptyLiteral is whether the value's tokens are an empty array or list
// literal — `array()`, `[ ]` — a set of none.
func emptyLiteral(rhs []parser.Token) bool {
	switch {
	case len(rhs) == 3 && rhs[0].Kind == parser.TokIdent && strings.EqualFold(rhs[0].Value, "array") &&
		rhs[1].Kind == parser.TokLParen && rhs[2].Kind == parser.TokRParen:
		return true
	case len(rhs) == 2 && rhs[0].Kind == parser.TokLBracket && rhs[1].Kind == parser.TokRBracket:
		return true
	}

	return false
}

// duplicatePlain is the plain name a `duplicate(X)` text copies, its
// variables. spelling trimmed, or ok=false. The text is normalized the way
// an element read's expression is, and the tail a tag assignment's
// statement runs past — the `>` and the body after it — does not read.
var duplicatePlainRe = regexp.MustCompile(`(?i)^duplicate\s*\(\s*(?:var\s+)?(?:variables\.)?([A-Za-z_]\w*)\s*\)`)

func duplicatePlain(text string) (string, bool) {
	text = strings.ReplaceAll(text, " . ", ".")

	m := duplicatePlainRe.FindStringSubmatch(strings.TrimSpace(text))
	if m == nil {
		return "", false
	}

	return m[1], true
}

// fnScopeTokens is fd's function's span as tokens — the script body's or the
// tag cffunction's — or ok=false. Unlike fnBody it does not need a brace to
// open what a tag function's body has none of.
func (r *Resolver) fnScopeTokens(fd *parser.FunctionDef) ([]parser.Token, bool) {
	text, ok := r.fnScopeText(fd)
	if !ok {
		return nil, false
	}

	return significantTokens(text), true
}

// fnScopeText is the source text of fd's own scope, which fnScopeTokens
// tokenises: a caller that needs a word in it can look before paying for that.
func (r *Resolver) fnScopeText(fd *parser.FunctionDef) (string, bool) {
	if fd == nil || !fd.URI.IsFile() {
		return "", false
	}

	pr := r.handlerParse(fd.URI.Path())
	if pr == nil {
		return "", false
	}

	return funcScopeText(pr, fd)
}

// funcScopeText is fd's own scope's text in pr, the parse that declares it.
func funcScopeText(pr *parser.ParseResult, fd *parser.FunctionDef) (string, bool) {
	scope := pr.FuncScopeAt(int(fd.Line))
	if scope.Start == -1 || !strings.EqualFold(scope.Name, fd.Name) {
		return "", false
	}

	if _, ok := lineOffset(pr.Content, scope.End); !ok {
		return "", false
	}

	return sourceText(pr.Content, scope.Start, scope.End+1), true
}

// fdReturnsLocal is the one name every return at fd's own level gives —
// `fields` or `variables.fields` — or "". Script `return` and tag
// `cfreturn` read the same here.
func (r *Resolver) fdReturnsLocal(fd *parser.FunctionDef) string {
	if fd == nil || !fd.URI.IsFile() {
		return ""
	}

	pr := r.handlerParse(fd.URI.Path())
	if pr == nil {
		return ""
	}

	// Asked for every element read a member's getter types, so each
	// function's answer is found once per parse.
	held, _ := pr.Memo("returnsLocal\x00"+strconv.Itoa(int(fd.Line))+"\x00"+strings.ToLower(fd.Name), func() any {
		return r.returnsLocalOf(fd)
	}).(string)

	return held
}

// returnsLocalOf is fdReturnsLocal's answer, read from fd's own scope.
func (r *Resolver) returnsLocalOf(fd *parser.FunctionDef) string {
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

// memberAssignment is one assignment of a name in a member's own file: the
// file offset the assigned name's token sits at, and the value's tokens.
// copyOf is the name a duplicate() value copies, when copies says it is one.
type memberAssignment struct {
	at     int
	rhs    []parser.Token
	copyOf string
	copies bool
}

// memberAssignments is every assignment of name in the member's own file,
// with the offset the assigned name's token holds in it, or ok=false when
// the file assigns it none. Whether each is a literal is the caller's
// demand.
func (r *Resolver) memberAssignments(name, member string) ([]memberAssignment, bool) {
	pr := r.handlerParse(member)
	if pr == nil {
		return nil, false
	}

	// Asked for every element read the member's getter types, so each
	// name's assignments are found once per parse of the member.
	found, _ := pr.Memo("memberAssignments\x00"+strings.ToLower(name), func() any {
		return assignmentsOf(fileTokens(pr), name)
	}).([]memberAssignment)

	return found, len(found) > 0
}

// assignmentsOf is every assignment of name in the tokens, or nil when one
// does not close.
func assignmentsOf(tokens []parser.Token, name string) []memberAssignment {
	var found []memberAssignment

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
			return nil
		}

		as := memberAssignment{at: tokens[i].Offset, rhs: tokens[i+2 : stop]}

		// A tag assignment's value runs past its tag, so its text is read
		// here, once, rather than for every element read that asks.
		if len(as.rhs) > 0 && as.rhs[0].Kind == parser.TokIdent && strings.EqualFold(as.rhs[0].Value, "duplicate") {
			as.copyOf, as.copies = duplicatePlain(producerText(as.rhs))
		}

		found = append(found, as)
		i = stop - 1
	}

	return found
}

// assignmentEnd is the index of the token that closes the grouping value an
// assignment names — array( ... ) or [ ... ] — or, for another kind of value
// or a group the expression goes on past ([ a, b ].map(f)), the token its
// statement ends at.
func assignmentEnd(tokens []parser.Token, from int) int {
	if at := groupEnd(tokens, from); at > 0 && !continuesPast(tokens, at) {
		return at
	}

	return statementEnd(tokens, from, len(tokens))
}

// groupEnd is the index of the token closing the array( ... ), ( ... ) or
// [ ... ] at from, or -1.
func groupEnd(tokens []parser.Token, from int) int {
	if from >= len(tokens) {
		return -1
	}

	switch tokens[from].Kind {
	case parser.TokLParen:
		return matchingParen(tokens, from)
	case parser.TokLBracket:
		return matchingBracket(tokens, from)
	default:
		if from+2 < len(tokens) && tokens[from].Kind == parser.TokIdent &&
			strings.EqualFold(tokens[from].Value, "array") && tokens[from+1].Kind == parser.TokLParen {
			return matchingParen(tokens, from+1)
		}
	}

	return -1
}

// continuesPast is whether the expression goes on after the token at: a
// member, a call, an index or an operator applied to the group.
func continuesPast(tokens []parser.Token, at int) bool {
	if at+1 >= len(tokens) {
		return false
	}

	switch tokens[at+1].Kind {
	case parser.TokDot, parser.TokDoubleColon, parser.TokLParen, parser.TokLBracket,
		parser.TokAmpersand, parser.TokPlus, parser.TokMinus, parser.TokStar,
		parser.TokPercent, parser.TokCaret, parser.TokQuestion, parser.TokColon:
		return true
	default:
		return false
	}
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

			// The literal is the whole value: [ a, b ].map(f) is what map
			// returns. A statement's own ; may follow it.
			if depth == 0 {
				if i+1 < len(rhs) && (i+2 < len(rhs) || rhs[i+1].Kind != parser.TokSemicolon) {
					return nil, false
				}

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
	// The call must be the whole item: field("a").setX(true) is what setX
	// returns.
	if len(item) < 2 || item[0].Kind != parser.TokIdent ||
		item[1].Kind != parser.TokLParen || matchingParen(item, 1) != len(item)-1 {
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
