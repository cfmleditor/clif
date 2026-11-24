package parser

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// renameScope is a function's own scope: its parameter list and its body.
type renameScope struct {
	params              string
	paramsAt, paramsEnd int
	body, bodyEnd       int
}

// errUndeclared is planScope's answer for a scope that does not declare the
// name: the next scope out may.
var errUndeclared = refuse("not declared")

func planScope(content string, spans []FunctionSpan, sc renameScope, at int) (*LocalRename, error) {
	if at >= sc.paramsAt && at <= sc.paramsEnd && sc.paramsAt >= 0 {
		return nil, refuse("an argument is renamed by its callers too, and a variable is renamed in a function's body")
	}

	if at < sc.body || at > sc.bodyEnd {
		return nil, errUndeclared
	}

	name := wordAt(content, at)
	if name == "" {
		return nil, NotAVariable("there is no name under the cursor")
	}

	body := content[sc.body:sc.bodyEnd]

	if strings.Contains(body, "```") || strings.Contains(strings.ToLower(body), "<cf") {
		return nil, refuse("the function holds CF tags")
	}

	if slices.Contains(paramNames(sc.params), strings.ToLower(name)) {
		return nil, refuse("'" + name + "' is an argument, which callers pass by name")
	}

	w := &renameWalk{name: name, taken: map[string]string{}, nested: nestedRanges(spans, sc.body, sc.bodyEnd)}
	w.params(sc.params, sc.paramsAt)
	w.walk(body, sc.body, false)

	switch {
	case w.err != nil:
		return nil, w.err
	case !w.declared:
		return nil, errUndeclared
	case w.called && !w.assignedFn:
		return nil, refuse("'" + name + "' is also called, and a call to an unscoped name may be a method of the component, which a rename would leave behind")
	case len(w.ranges) == 0:
		return nil, refuse("no use of '" + name + "' found")
	}

	slices.SortFunc(w.ranges, func(a, b [2]int) int { return a[0] - b[0] })

	return &LocalRename{Name: name, Ranges: w.ranges, Taken: w.taken, StatementStart: w.statementStart}, nil
}

// closuresAround are the anonymous functions written in owner's body whose
// parameters or body hold at, innermost first.
func closuresAround(content string, owner *FunctionSpan, at int) []renameScope {
	var found []renameScope

	body := content[owner.Body:owner.BodyEnd]
	toks := scriptTokens(body)

	for i, t := range toks {
		if t.Kind != TokIdent || !strings.EqualFold(t.Value, "function") || i > 0 && toks[i-1].Kind == TokDot {
			continue
		}

		// An anonymous function: ( follows `function` directly.
		if i+1 >= len(toks) || toks[i+1].Kind != TokLParen {
			continue
		}

		rparen := matching(toks, i+1, TokLParen, TokRParen)
		if rparen < 0 {
			continue
		}

		open := rparen + 1
		for open < len(toks) && toks[open].Kind != TokLBrace && toks[open].Kind != TokSemicolon && toks[open].Kind != TokRParen {
			open++
		}

		if open >= len(toks) || toks[open].Kind != TokLBrace {
			continue
		}

		end := matching(toks, open, TokLBrace, TokRBrace)
		if end < 0 {
			continue
		}

		scope := renameScope{
			paramsAt: owner.Body + toks[i+1].Offset + 1, paramsEnd: owner.Body + toks[rparen].Offset,
			body: owner.Body + toks[open].Offset + 1, bodyEnd: owner.Body + toks[end].Offset,
		}
		scope.params = content[scope.paramsAt:scope.paramsEnd]

		if at >= scope.paramsAt && at <= scope.bodyEnd {
			found = append(found, scope)
		}
	}

	// An arrow function with a braced body, `( a, b ) => { ... }` or
	// `a => { ... }`, is a scope of its own too.
	for i := 1; i+2 < len(toks); i++ {
		if toks[i].Kind != TokEquals || toks[i+1].Kind != TokGT || toks[i+1].Offset != toks[i].Offset+1 || toks[i+2].Kind != TokLBrace {
			continue
		}

		end := matching(toks, i+2, TokLBrace, TokRBrace)
		if end < 0 {
			continue
		}

		var from, to int // the parameters, inside their parentheses when there are any

		switch toks[i-1].Kind {
		case TokRParen:
			lo := matchingBack(toks, i-1, TokLParen, TokRParen)
			if lo < 0 {
				continue
			}

			from, to = toks[lo].Offset+1, toks[i-1].Offset
		case TokIdent:
			from, to = toks[i-1].Offset, toks[i-1].Offset+len(toks[i-1].Value)
		default:
			continue
		}

		scope := renameScope{
			paramsAt: owner.Body + from, paramsEnd: owner.Body + to,
			body: owner.Body + toks[i+2].Offset + 1, bodyEnd: owner.Body + toks[end].Offset,
		}
		scope.params = content[scope.paramsAt:scope.paramsEnd]

		if at >= scope.paramsAt && at <= scope.bodyEnd {
			found = append(found, scope)
		}
	}

	// Innermost first: the smallest holds the others' left over.
	slices.SortFunc(found, func(a, b renameScope) int { return (a.bodyEnd - a.paramsAt) - (b.bodyEnd - b.paramsAt) })

	return found
}

// matching is the index of the token that closes the bracket at toks[i], or -1.
func matching(toks []Token, i int, open, shut TokenKind) int {
	depth := 0

	for j := i; j < len(toks); j++ {
		switch toks[j].Kind {
		case open:
			depth++
		case shut:
			depth--

			if depth == 0 {
				return j
			}
		default:
		}
	}

	return -1
}

// innermostScriptSpan is the smallest script function whose declaration holds
// at, or nil.
func innermostScriptSpan(spans []FunctionSpan, at int) *FunctionSpan {
	var best *FunctionSpan

	for i := range spans {
		s := &spans[i]
		if s.Kind == FunctionTag || s.Body < 0 || at < s.Start || at > s.End {
			continue
		}

		if best == nil || s.End-s.Start < best.End-best.Start {
			best = s
		}
	}

	return best
}

// nestedRanges are the spans of the named functions inside a body, which are
// scopes of their own.
func nestedRanges(spans []FunctionSpan, body, bodyEnd int) [][2]int {
	var out [][2]int

	for i := range spans {
		if s := &spans[i]; s.Start >= body && s.End <= bodyEnd && (s.Start != body || s.End != bodyEnd) {
			out = append(out, [2]int{s.Start, s.End})
		}
	}

	return out
}

// wordAt is the identifier that holds the byte offset at, or "".
func wordAt(content string, at int) string {
	if at < 0 || at > len(content) {
		return ""
	}

	start := at
	for start > 0 && lex.IsIdent(content[start-1]) {
		start--
	}

	end := at
	for end < len(content) && lex.IsIdent(content[end]) {
		end++
	}

	if start == end || content[start] >= '0' && content[start] <= '9' {
		return ""
	}

	return content[start:end]
}

// param is one parameter of a list: its name, and where its default
// expression stands in the list's text, or -1.
type param struct {
	name             string
	nameAt           int // where the name stands in the list's text
	defStart, defEnd int
}

// paramAttributes are the attributes a script parameter takes after its name.
// `inject` is WireBox's. A word that is none of these before an `=` is the
// name, with a type before it and a default after: `string c = "x"`, which
// reads the same as `c inject="x"` but for the word.
var paramAttributes = map[string]bool{
	"default": true, "hint": true, "type": true, "displayname": true, "passby": true, "inject": true, "required": true,
}

// paramTypes are the built-in types a parameter is declared with. A word after
// one is the parameter's name, whatever it is spelled: `string type = "a"` has a
// parameter called type, where `a type = "string"` has an attribute.
var paramTypes = map[string]bool{
	"any": true, "array": true, "binary": true, "boolean": true, "component": true, "date": true, "function": true,
	"guid": true, "numeric": true, "query": true, "string": true, "struct": true, "uuid": true, "variablename": true,
	"xml": true, "void": true, "required": true,
}

// parseParams reads a parameter list, inside its parentheses. Each parameter's
// name is its last word before a default: what comes before it is its type and
// modifiers.
func parseParams(text string) []param {
	toks := scriptTokens(text)

	var (
		out    []param
		depth  int
		cur    = param{defStart: -1, defEnd: -1}
		inDef  bool
		inAttr bool // after the name: `default="y" hint="…"`
	)

	flush := func(end int) {
		if cur.name != "" {
			if inDef {
				cur.defEnd = end
			}

			out = append(out, cur)
		}

		cur, inDef, inAttr = param{defStart: -1, defEnd: -1}, false, false
	}

	for i, t := range toks {
		switch t.Kind {
		case TokLParen, TokLBracket, TokLBrace:
			depth++
		case TokRParen, TokRBracket, TokRBrace:
			depth--
		case TokComma:
			if depth == 0 {
				flush(t.Offset)
			}
		case TokEquals:
			if depth == 0 && !inDef && !inAttr && !equalsPair(toks, i) && i+1 < len(toks) {
				inDef, cur.defStart = true, toks[i+1].Offset
			}
		case TokIdent:
			if depth != 0 || inDef || inAttr {
				break
			}

			// `x default="y"`: a word after the name and before an `=` is an
			// attribute of the parameter, not its name.
			if cur.name != "" && !paramTypes[strings.ToLower(cur.name)] && i+1 < len(toks) && toks[i+1].Kind == TokEquals && paramAttributes[strings.ToLower(t.Value)] {
				inAttr = true

				break
			}

			cur.name, cur.nameAt = t.Value, t.Offset
		default:
		}
	}

	flush(len(text))

	return out
}

// paramNames are the lowercased argument names of a parameter list.
func paramNames(params string) []string {
	parsed := parseParams(params)
	names := make([]string, 0, len(parsed))

	for _, p := range parsed {
		names = append(names, strings.ToLower(p.name))
	}

	return names
}
