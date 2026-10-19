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
// the body starts a local empty with arrayNew(1), appends values to it with
// arrayAppend, and every return at the function's own level names it, or a
// field assigned from it. The appended values' common component is the
// array's element. Any return of something else, an append whose value
// nobody proves, or another write to the local (built[2]=x, built.append(x),
// arrayPrepend, a reassignment) is an open world: the collection could hold
// anything, and the answer is withheld.
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

	if !writesAccounted(tokens, built, "") {
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
			expr := returnExpr(tokens, i+1, len(tokens))

			// A tag cfreturn's self-close — variables.extendsetArray /> —
			// the slash and the angle are the tag's, not the value's.
			for len(expr) > 0 && (expr[len(expr)-1].Kind == parser.TokSlash || expr[len(expr)-1].Kind == parser.TokGT) {
				expr = expr[:len(expr)-1]
			}

			if !isBuiltOrAlias(tokens, localNameOf(expr), built) {
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

// builtFieldElement is the element of the collection a getter's field holds:
// every return at the getter's own level names one field, and the file's own
// assignments fill it — a local assigned an empty array, appended with
// values whose common component is the element. The field may also be
// appended through the getter itself (arrayAppend(getAttributes(), a)). Any
// other write to the field, or through the getter — a setter's assignment,
// a member append, an indexed write — is an open world, and the answer is
// withheld.
//
//	<cffunction name="getAttributes">
//		<cfif not isArray(variables.instance.attributes)>
//			<cfset loadAttributes()>
//		</cfif>
//		<cfreturn variables.instance.attributes>
//	</cffunction>
//
//	<cffunction name="loadAttributes">
//		<cfset variables.instance.attributes=arrayNew(1)>
//		<cfloop ...>
//			<cfset attribute=createObject("component","mura.extend.extendAttribute").init(...)>
//			<cfset arrayAppend(variables.instance.attributes, attribute)>
//		</cfloop>
//	</cffunction>
func (r *Resolver) builtFieldElement(fd *parser.FunctionDef) string {
	pr := r.handlerParse(fd.URI.Path())
	if pr == nil {
		return ""
	}

	tokens, ok := r.fnScopeTokens(fd)
	if !ok {
		return ""
	}

	field := getterField(tokens)
	if field == "" {
		return ""
	}

	values := builtFieldAppends(pr, field, fd.Name)
	if len(values) == 0 {
		return ""
	}

	dir := filepath.Dir(fd.URI.Path())

	var elements []string

	for _, v := range values {
		comp := r.argumentExprComponent(v.tokens, v.line, fd.Name, pr, dir, lookupCtx{})
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

// getterField is the one field every return at the getter's own level names,
// as written (`variables.instance.attributes`), or "".
func getterField(tokens []parser.Token) string {
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
			expr := returnExpr(tokens, i+1, len(tokens))

			// A tag cfreturn's self-close — attributes /> — the slash and
			// the angle are the tag's, not the value's.
			for len(expr) > 0 && (expr[len(expr)-1].Kind == parser.TokSlash || expr[len(expr)-1].Kind == parser.TokGT) {
				expr = expr[:len(expr)-1]
			}

			name := dottedLocalName(expr)
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

// dottedLocalName is a plain identifier or a variables-led dotted field the
// tokens name, and nothing else.
func dottedLocalName(tokens []parser.Token) string {
	if len(tokens) == 0 || tokens[0].Kind != parser.TokIdent {
		return ""
	}

	name := tokens[0].Value

	var b strings.Builder

	b.WriteString(name)

	i := 1
	for i+1 < len(tokens) && tokens[i].Kind == parser.TokDot && tokens[i+1].Kind == parser.TokIdent {
		b.WriteByte('.')
		b.WriteString(tokens[i+1].Value)
		i += 2
	}

	if i != len(tokens) {
		return ""
	}

	name = b.String()

	if !strings.Contains(name, ".") {
		return name
	}

	if !strings.EqualFold(name, "variables."+strings.TrimPrefix(name, "variables.")) {
		return ""
	}

	return name
}

// dottedNameBefore is the dotted name the tokens ending at an equals spell:
// ident (. ident)*, read backwards from the equals at i.
func dottedNameBefore(tokens []parser.Token, i int) string {
	if i == 0 || tokens[i-1].Kind != parser.TokIdent {
		return ""
	}

	name := tokens[i-1].Value
	j := i - 2

	for j-1 >= 0 && tokens[j].Kind == parser.TokDot && tokens[j-1].Kind == parser.TokIdent {
		name = tokens[j-1].Value + "." + name
		j -= 2
	}

	return name
}

// builtFieldAppends is fieldAppends' values for the field the getter
// returns, when the file builds the field and accounts for every write to
// it, or nil. It reads the parse alone and is asked for every keyed read of
// the getter's collection, so it is found once per parse.
func builtFieldAppends(pr *parser.ParseResult, field, getter string) []appendValue {
	values, _ := pr.Memo("builtField\x00"+strings.ToLower(field)+"\x00"+strings.ToLower(getter), func() any {
		all := fileTokens(pr)

		if !fieldBuiltByFile(all, field) || !writesAccounted(all, field, getter) {
			return []appendValue(nil)
		}

		values, _ := fieldAppends(all, field, getter)

		return values
	}).([]appendValue)

	return values
}

// fileTokens is significantTokens of the whole file, once per parse.
func fileTokens(pr *parser.ParseResult) []parser.Token {
	tokens, _ := pr.Memo("significantTokens", func() any {
		return significantTokens(pr.Content)
	}).([]parser.Token)

	return tokens
}

// fieldBuiltByFile reports whether the file assigns the field an empty
// array — arrayNew(1) or an empty [ ] — the collection the appends fill.
func fieldBuiltByFile(tokens []parser.Token, field string) bool {
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokEquals {
			continue
		}

		if name := dottedNameBefore(tokens, i); name == "" || !strings.EqualFold(name, field) {
			continue
		}

		if tokens[i+1].Kind == parser.TokIdent && strings.EqualFold(tokens[i+1].Value, "arrayNew") {
			return true
		}

		if tokens[i+1].Kind == parser.TokLBracket {
			if last := matchingBracket(tokens, i+1); last > 0 && last == i+2 {
				return true
			}
		}
	}

	return false
}

// fieldAppends is every arrayAppend's value whose target is the field —
// written directly, or through the getter whose return the field is.
func fieldAppends(tokens []parser.Token, field, getter string) ([]appendValue, bool) {
	var values []appendValue

	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokIdent || !strings.EqualFold(tokens[i].Value, "arrayAppend") ||
			tokens[i+1].Kind != parser.TokLParen {
			continue
		}

		args, ok := callArgs(tokens, i+1)
		if !ok {
			return nil, false
		}

		if !strings.EqualFold(dottedLocalName(args[0]), field) && (len(args[0]) != 3 || !isGetterCall(args[0], 0, getter)) {
			continue
		}

		// An append writesAccounted lets through must be one counted here.
		if len(args) != 2 || len(args[1]) == 0 {
			return nil, false
		}

		values = append(values, appendValue{tokens: args[1], line: conv.Uint32(args[1][0].Line)})
	}

	return values, len(values) > 0
}

// isGetterCall reports whether the tokens at i call the getter with no
// arguments: getWidgets().
func isGetterCall(tokens []parser.Token, i int, getter string) bool {
	return getter != "" && i+2 < len(tokens) && tokens[i].Kind == parser.TokIdent &&
		strings.EqualFold(tokens[i].Value, getter) && tokens[i+1].Kind == parser.TokLParen &&
		tokens[i+2].Kind == parser.TokRParen
}

// writesAccounted reports whether every reference to the collection — the
// name, a dotted field with or without its variables scope, or a call of the
// getter that returns it — reads it, appends to it with arrayAppend, or
// assigns it an empty array or a string reset. Any other write (an
// assignment of another value, an indexed write, a member call that may
// mutate it, passing it to a function that may) is a value nobody proved.
func writesAccounted(tokens []parser.Token, name, getter string) bool {
	for i := range tokens {
		if tokens[i].Kind != parser.TokIdent || i > 0 && tokens[i-1].Kind == parser.TokDot {
			continue
		}

		viaGetter := isGetterCall(tokens, i, getter)

		end := i + 3
		if !viaGetter {
			end = referenceEnd(tokens, i, name)
		}

		if end < 0 {
			continue
		}

		if !referenceAccounted(tokens, i, end, viaGetter) {
			return false
		}
	}

	return true
}

// referenceEnd is the index past the dotted name at i when it spells name,
// a leading variables scope ignored on both, or -1.
func referenceEnd(tokens []parser.Token, i int, name string) int {
	want := strings.TrimPrefix(strings.ToLower(name), "variables.")

	var b strings.Builder

	for j := i; j < len(tokens) && tokens[j].Kind == parser.TokIdent; j += 2 {
		if j > i {
			b.WriteByte('.')
		}

		b.WriteString(strings.ToLower(tokens[j].Value))

		if strings.TrimPrefix(b.String(), "variables.") == want && b.String() != "variables" {
			return j + 1
		}

		if j+1 >= len(tokens) || tokens[j+1].Kind != parser.TokDot {
			break
		}
	}

	return -1
}

// referenceAccounted reports whether the reference spanning tokens[start:end]
// is a read, an arrayAppend's target, or an accounted assignment.
func referenceAccounted(tokens []parser.Token, start, end int, viaGetter bool) bool {
	if end >= len(tokens) {
		return true
	}

	assigns := func(at int) bool {
		return at < len(tokens) && tokens[at].Kind == parser.TokEquals &&
			(at+1 >= len(tokens) || tokens[at+1].Kind != parser.TokEquals) &&
			(start == 0 || tokens[start-1].Kind != parser.TokEquals)
	}

	switch next := tokens[end]; {
	case assigns(end):
		return !viaGetter && emptyOrReset(tokens[end+1:])
	case next.Kind == parser.TokLBracket:
		last := matchingBracket(tokens, end)

		return last > 0 && !assigns(last+1)
	case next.Kind == parser.TokDot:
		if end+2 < len(tokens) && tokens[end+1].Kind == parser.TokIdent && tokens[end+2].Kind == parser.TokLParen {
			return readOnlyMembers[strings.ToLower(tokens[end+1].Value)]
		}

		return true
	}

	if start == 0 || tokens[start-1].Kind != parser.TokLParen && tokens[start-1].Kind != parser.TokComma ||
		tokens[end].Kind != parser.TokRParen && tokens[end].Kind != parser.TokComma {
		return true
	}

	// An argument: arrayAppend's target is the counted append, and a
	// function that only reads its argument leaves the collection alone.
	call := enclosingCall(tokens, start)

	return strings.EqualFold(call, "arrayAppend") || readOnlyFunctions[strings.ToLower(call)]
}

// emptyOrReset reports whether an assignment's value is an empty array —
// arrayNew(…) or [ ] — or a string literal, which holds no element.
func emptyOrReset(rhs []parser.Token) bool {
	if len(rhs) >= 2 && rhs[0].Kind == parser.TokIdent && strings.EqualFold(rhs[0].Value, "arrayNew") && rhs[1].Kind == parser.TokLParen {
		return true
	}

	if len(rhs) >= 2 && rhs[0].Kind == parser.TokLBracket && rhs[1].Kind == parser.TokRBracket {
		return true
	}

	return len(rhs) >= 1 && rhs[0].Kind == parser.TokString &&
		(len(rhs) == 1 || rhs[1].Kind == parser.TokSemicolon || rhs[1].Kind == parser.TokSlash || rhs[1].Kind == parser.TokGT)
}

// enclosingCall is the name of the call whose argument list holds the token
// at i, or "".
func enclosingCall(tokens []parser.Token, i int) string {
	depth := 0

	for j := i - 1; j >= 0; j-- {
		switch tokens[j].Kind {
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			depth++
		case parser.TokLBracket, parser.TokLBrace:
			if depth == 0 {
				return ""
			}

			depth--
		case parser.TokLParen:
			if depth == 0 {
				if j > 0 && tokens[j-1].Kind == parser.TokIdent {
					return tokens[j-1].Value
				}

				return ""
			}

			depth--
		default:
		}
	}

	return ""
}

// readOnlyFunctions are the global functions that read an array argument
// and leave it as it was.
var readOnlyFunctions = map[string]bool{
	"isarray": true, "arraylen": true, "len": true, "arrayisempty": true, "arrayisdefined": true,
	"arraycontains": true, "arraycontainsnocase": true, "arrayfind": true, "arrayfindnocase": true,
	"arrayfindall": true, "arraytolist": true, "isnull": true, "isempty": true, "serializejson": true,
	"duplicate": true, "arrayslice": true,
}

// readOnlyMembers are the array member functions that leave it as it was.
var readOnlyMembers = map[string]bool{
	"len": true, "isempty": true, "isdefined": true, "tolist": true, "find": true, "findnocase": true,
	"findall": true, "contains": true, "containsnocase": true, "slice": true, "each": true, "map": true,
	"filter": true, "reduce": true, "some": true, "every": true, "first": true, "last": true,
}
