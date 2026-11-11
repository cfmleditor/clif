package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// The statements a registry recipe is read from (registry.go), found in the
// template's code: a listing, a store, a construction, an append, a factory
// call, a guard. Each was a regular expression over the template's text,
// which read a commented-out alternative as a statement.

// packageListing is `ComponentListPackageAsStruct( "lucee.admin.debug",
// driverNames )`: the package literal, the struct it merges into as written,
// and the offset just past the call.
type packageListing struct {
	pkg, target string
	end         int
}

// packageListings are the listings in content's code.
func packageListings(content string) []packageListing {
	var out []packageListing

	for _, call := range codeCalls(textCode(content), "ComponentListPackageAsStruct") {
		pieces := call.pieces()
		if len(pieces) != 2 {
			continue
		}

		pkg, ok := soleString(pieces[0])
		if !ok || strings.ContainsAny(pkg, `"'`) {
			continue
		}

		if target, ok := storeVar(pieces[1]); ok {
			out = append(out, packageListing{pkg: pkg, target: target, end: call.end()})
		}
	}

	return out
}

// storeVar is the variable piece is when it is a name or variables.name, as
// written.
func storeVar(piece []parser.Token) (string, bool) {
	switch {
	case len(piece) == 1 && piece[0].Kind == parser.TokIdent:
		return piece[0].Value, true
	case len(piece) == 3 && isWord(piece[0], "variables") && piece[1].Kind == parser.TokDot && piece[2].Kind == parser.TokIdent:
		return piece[0].Value + "." + piece[2].Value, true
	default:
		return "", false
	}
}

// nameBefore is the variable the identifier at tokens[i] names, with the
// variables. written before it, if one is: `variables.x` or `x`, and `x` of
// `a.x`.
func nameBefore(tokens []parser.Token, i int) string {
	if i >= 2 && tokens[i-1].Kind == parser.TokDot && isWord(tokens[i-2], "variables") {
		return tokens[i-2].Value + "." + tokens[i].Value
	}

	return tokens[i].Value
}

// registryStore is `R[key] = …`: R as nameBefore reads it, the identifier
// before the bracket, and the index of the value's first token.
type registryStore struct {
	registry, ident string
	value           int
}

// registryStores are the keyed stores in code, in order.
func registryStores(code []parser.Token) []registryStore {
	var out []registryStore

	for i := 1; i < len(code); i++ {
		if code[i].Kind != parser.TokLBracket || code[i-1].Kind != parser.TokIdent {
			continue
		}

		closing := producerGroupEnd(code, i, parser.TokLBracket, parser.TokRBracket)
		if closing < 0 || !isAssignAt(code, closing+1) || closing+2 >= len(code) {
			continue
		}

		out = append(out, registryStore{registry: nameBefore(code, i-1), ident: code[i-1].Value, value: closing + 2})
	}

	return out
}

// createdComponentArg reads `createObject( "component", V )` at tokens[i]:
// V, when it is a name.
func createdComponentArg(tokens []parser.Token, i int) (string, bool) {
	if i+1 >= len(tokens) || !isWord(tokens[i], "createObject") || tokens[i+1].Kind != parser.TokLParen {
		return "", false
	}

	closing := producerGroupEnd(tokens, i+1, parser.TokLParen, parser.TokRParen)
	if closing < 0 {
		return "", false
	}

	pieces := producerSplit(tokens[i+2 : closing])
	if len(pieces) != 2 {
		return "", false
	}

	if kind, ok := soleString(pieces[0]); !ok || !strings.EqualFold(strings.TrimSpace(kind), "component") {
		return "", false
	}

	path, next := dottedAt(pieces[1], 0)
	if path == "" || next != len(pieces[1]) {
		return "", false
	}

	return path, true
}

// appendArgs reads `arrayAppend( ARR, value … )`: the array, a name or
// variables.name, and the value's tokens. A call with more than two
// arguments has its value read, and n says how many it has.
func appendArgs(call codeCall) (arr string, value []parser.Token, n int) {
	pieces := call.pieces()
	if len(pieces) < 2 {
		return "", nil, 0
	}

	arr, ok := storeVar(pieces[0])
	if !ok || len(pieces[1]) == 0 {
		return "", nil, 0
	}

	return arr, pieces[1], len(pieces)
}

// factoryCall is `CONTAINER = FN( ARR … )`: the assignment whose right side
// is the call that turns an appended array into the struct the page reads
// back.
type factoryCall struct {
	container, fn, arg string
}

// factoryCalls are code's assignments of a call whose first argument is a
// name or variables.name, in order.
func factoryCalls(code []parser.Token) []factoryCall {
	var out []factoryCall

	for i := 1; i+3 < len(code); i++ {
		if !isAssignAt(code, i) || code[i-1].Kind != parser.TokIdent || code[i+1].Kind != parser.TokIdent || code[i+2].Kind != parser.TokLParen {
			continue
		}

		arg, next := code[i+3].Value, i+4
		if code[i+3].Kind != parser.TokIdent {
			continue
		}

		if next+1 < len(code) && isWord(code[i+3], "variables") && code[next].Kind == parser.TokDot && code[next+1].Kind == parser.TokIdent {
			arg, next = arg+"."+code[next+1].Value, next+2
		}

		if next < len(code) && (code[next].Kind == parser.TokComma || code[next].Kind == parser.TokRParen) {
			out = append(out, factoryCall{container: nameBefore(code, i-1), fn: code[i+1].Value, arg: arg})
		}
	}

	return out
}

// factoryWalkItem is the item a factory body's first walk binds, when it walks
// the array its parameter param received: `loop array=arguments.P
// item="X"`, or, failing that, `for ( X in arguments.P )`.
func factoryWalkItem(code []parser.Token, param string) (string, bool) {
	for i := 0; i+8 < len(code); i++ {
		if !isWord(code[i], "loop") || !isWord(code[i+1], "array") || !isAssignAt(code, i+2) || !argumentsName(code, i+3) ||
			!isWord(code[i+6], "item") || !isAssignAt(code, i+7) {
			continue
		}

		lit, ok := stringLiteral(code[i+8])
		if !ok {
			continue
		}

		if strings.EqualFold(code[i+5].Value, param) {
			return strings.TrimPrefix(strings.TrimPrefix(lit, "local."), "var "), true
		}

		break
	}

	for i := 0; i+5 < len(code); i++ {
		if !isWord(code[i], "for") || code[i+1].Kind != parser.TokLParen {
			continue
		}

		j := i + 2
		if isWord(code[j], "var") {
			j++
		}

		if j+4 >= len(code) || code[j].Kind != parser.TokIdent || !isWord(code[j+1], "in") || !argumentsName(code, j+2) {
			continue
		}

		if strings.EqualFold(code[j+4].Value, param) {
			return code[j].Value, true
		}

		return "", false
	}

	return "", false
}

// argumentsName reports whether code holds `arguments.NAME` at i.
func argumentsName(code []parser.Token, i int) bool {
	return i+2 < len(code) && isWord(code[i], "arguments") && code[i+1].Kind == parser.TokDot && code[i+2].Kind == parser.TokIdent
}

// nameAt reports whether the dotted name is written at tokens[i].
func nameAt(tokens []parser.Token, i int, name string) bool {
	for k, part := range strings.Split(name, ".") {
		j := i + 2*k
		if j >= len(tokens) || tokens[j].Kind != parser.TokIdent || !strings.EqualFold(tokens[j].Value, part) {
			return false
		}

		if k > 0 && tokens[j-1].Kind != parser.TokDot {
			return false
		}
	}

	return name != ""
}

// returnsName reports whether code returns name, or variables.name.
func returnsName(code []parser.Token, name string) bool {
	for i := 0; i+1 < len(code); i++ {
		if !isWord(code[i], "return") {
			continue
		}

		j := i + 1
		if j+2 < len(code) && isWord(code[j], "variables") && code[j+1].Kind == parser.TokDot {
			j += 2
		}

		if isWord(code[j], name) {
			return true
		}
	}

	return false
}

// keyCompare is a compare of a name with a literal, `key == "name"`: whether
// it tests equality, and the offsets of its start and of its end.
type keyCompare struct {
	key, name  string
	equal      bool
	start, end int
}

// keyCompares are code's compares of a name with a literal, by ==, ===,
// !=, !==, EQ or NEQ.
func keyCompares(code []parser.Token) []keyCompare {
	var out []keyCompare

	adjacent := func(i, n int, kinds ...parser.TokenKind) bool {
		for k := range n {
			if i+k >= len(code) || code[i+k].Kind != kinds[k] || k > 0 && code[i+k].Offset != code[i+k-1].Offset+1 {
				return false
			}
		}

		return i+n >= len(code) || code[i+n].Kind != parser.TokEquals || code[i+n].Offset != code[i+n-1].Offset+1
	}

	for i := 0; i+2 < len(code); i++ {
		if code[i].Kind != parser.TokIdent {
			continue
		}

		op := i + 1

		var (
			n     int
			equal bool
		)

		switch {
		case adjacent(op, 3, parser.TokEquals, parser.TokEquals, parser.TokEquals):
			n, equal = 3, true
		case adjacent(op, 2, parser.TokEquals, parser.TokEquals):
			n, equal = 2, true
		case adjacent(op, 3, parser.TokBang, parser.TokEquals, parser.TokEquals):
			n = 3
		case adjacent(op, 2, parser.TokBang, parser.TokEquals):
			n = 2
		case isWord(code[op], "EQ"):
			n, equal = 1, true
		case isWord(code[op], "NEQ"):
			n = 1
		default:
			continue
		}

		if op+n >= len(code) {
			continue
		}

		lit, ok := stringLiteral(code[op+n])
		if !ok || lit == "" || strings.ContainsAny(lit, `"'`) {
			continue
		}

		value := code[op+n]
		out = append(out, keyCompare{key: code[i].Value, name: lit, equal: equal, start: code[i].Offset, end: value.Offset + len(value.Value)})
	}

	return out
}
