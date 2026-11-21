package parser

import (
	"errors"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// LocalRename is what renaming a local variable of a script function changes:
// every place the function writes the variable, and the names it already
// uses, which a new name must not collide with.
type LocalRename struct {
	// Name is the variable as the function declares it.
	Name string
	// Ranges are the byte ranges of the name in the text, in order: the
	// declaration, each use, `local.name` (the name alone), and a use inside a
	// `#...#` of a string.
	Ranges [][2]int
	// Taken are the lowercased names the whole function means something by
	// without a qualifier, closures included, each with what it is: "an
	// argument", "a variable", "a catch variable", or "a name it reads or
	// calls", which is a variable it reads from outside (`config`) or a
	// function it calls. A new name from this set would shadow what the
	// function already means by it. A struct key, a named argument's label,
	// another object's member and a type are not names it means anything by.
	Taken map[string]string
	// StatementStart is set when the variable is written at the start of a
	// statement somewhere, `count = count + 1;`, where a statement word
	// (`lock`, `param`, `thread`) would be read as that statement.
	StatementStart bool
}

// RangeAt is the range of the variable's name at the byte offset at, the end
// of the name included as an editor's cursor is, or false when at is not on one
// of the places the variable is written: it may be on a struct key or another
// object's member that is spelled the same. Two names are never side by side
// without a character between them, so at is on at most one range.
func (r *LocalRename) RangeAt(at int) (start, end int, ok bool) {
	for _, rg := range r.Ranges {
		if at >= rg[0] && at <= rg[1] {
			return rg[0], rg[1], true
		}
	}

	return 0, 0, false
}

// hardReserved are the words no variable may be called, and no name the
// function means something by: CFML's scopes, its literals and operators, and
// the words its control flow and declarations are written with.
var hardReserved = map[string]bool{
	"var": true, "function": true, "return": true, "if": true, "else": true, "for": true, "while": true,
	"do": true, "switch": true, "case": true, "default": true, "break": true, "continue": true, "try": true,
	"catch": true, "finally": true, "throw": true, "in": true, "new": true, "true": true, "false": true,
	"null": true, "local": true, "arguments": true, "variables": true, "this": true, "super": true,
	"application": true, "session": true, "request": true, "server": true, "url": true, "form": true,
	"cgi": true, "cookie": true, "client": true, "cfthread": true, "and": true, "or": true, "not": true,
	"xor": true, "eq": true, "neq": true, "is": true, "gt": true, "lt": true, "gte": true, "lte": true,
	"ge": true, "le": true, "mod": true, "contains": true, "eqv": true, "imp": true,
}

// statementWords are the words that begin a statement of their own in CFScript
// (`lock`, `param`, `thread`), or modify a declaration. They are fine variable
// names where an expression stands, `var lock = 1; return lock;`, and the
// engine reads them as the statement where one begins, `lock = lock + 1;`.
var statementWords = map[string]bool{
	"rethrow": true, "import": true, "include": true, "component": true, "interface": true,
	"property": true, "param": true, "lock": true, "transaction": true, "thread": true, "abort": true,
	"exit": true, "savecontent": true, "trace": true, "assert": true, "abstract": true, "final": true,
	"static": true, "public": true, "private": true, "package": true, "remote": true, "required": true,
}

// NameWord is what a word is as a variable's name.
type NameWord uint8

const (
	// NameFree is a word that may be a variable's name.
	NameFree NameWord = iota
	// NameReserved is a word no variable may be called.
	NameReserved
	// NameStatementWord is a word that begins a statement of its own: a fine
	// variable name, except where the variable is written as a statement's
	// first word (LocalRename.StatementStart).
	NameStatementWord
)

// operatorWords are the words of CFML's longhand operators that hardReserved
// does not hold, `a is greater than or equal to b`. They are not refused as the
// name a variable has now, `var to = 1`, which reads where an expression stands,
// but a new name must not be one: a function that spells the operator would read
// the variable as part of it.
var operatorWords = map[string]bool{
	"equal": true, "greater": true, "less": true, "than": true, "does": true, "contain": true, "to": true,
}

// ClassifyVariableName says whether name, in any case, may be a variable's.
func ClassifyVariableName(name string) NameWord {
	lower := strings.ToLower(name)

	switch {
	case hardReserved[lower], operatorWords[lower]:
		return NameReserved
	case statementWords[lower]:
		return NameStatementWord
	default:
		return NameFree
	}
}

// What a taken name is, strongest first: a name taken as an argument stays so
// when it is also read.
const (
	takenArgument = "an argument"
	takenLocal    = "a variable"
	takenCatch    = "a catch variable"
	// TakenReadOrCalled is what LocalRename.Taken says of a name the function
	// reads from outside or calls, as against one it declares.
	TakenReadOrCalled = "a name it reads or calls"
	takenOther        = TakenReadOrCalled
)

var takenRank = map[string]int{takenArgument: 3, takenLocal: 2, takenCatch: 1, takenOther: 0}

// ErrNotRenameable is wrapped by every refusal of PlanLocalRename. A rename of
// a variable is a change to code the server cannot run, so it makes only the
// ones it can show are the whole of the variable's uses, and says why when it
// cannot.
var ErrNotRenameable = errors.New("cannot rename")

func refuse(why string) error { return refusal{why} }

// refusal is a reason a variable cannot be renamed; it matches ErrNotRenameable.
type refusal struct{ why string }

func (r refusal) Error() string { return ErrNotRenameable.Error() + ": " + r.why }

func (r refusal) Is(target error) bool { return target == ErrNotRenameable }

// ErrNoVariable is what a refusal matches when the cursor is not on a
// variable at all, where the others name a variable that cannot be renamed.
// An editor asking whether to offer a rename is answered "no" to the first and
// given the reason for the second.
var ErrNoVariable = errors.New("no variable here")

type noVariableError struct{ why string }

func (e noVariableError) Error() string { return ErrNotRenameable.Error() + ": " + e.why }

func (e noVariableError) Is(target error) bool {
	return target == ErrNoVariable || target == ErrNotRenameable
}

// NotAVariable is a refusal that matches ErrNoVariable and ErrNotRenameable.
func NotAVariable(why string) error { return noVariableError{why} }

// PlanLocalRename finds the local variable of a script function under the byte
// offset at in content, given the spans of the file's functions
// (ParseResult.FunctionSpans). The variable is one the function declares,
// `var x` or `local.x = …`; an argument is not renamed, since its name is how a
// caller passes it. It is the function's own: a variable a closure declares
// for itself, or a name something nested shadows, is not renamed, and nor is
// one the function also names in a string or reads with `evaluate`, where the
// name is a value and no rename can follow it.
func PlanLocalRename(content string, spans []FunctionSpan, at int) (*LocalRename, error) {
	owner := innermostScriptSpan(spans, at)

	var (
		scopes    []renameScope
		ownerSpan FunctionSpan
	)

	if owner != nil {
		ownerSpan = *owner
		scopes = append(closuresAround(content, owner, at), renameScope{
			params: owner.ParamsText(content), body: owner.Body, bodyEnd: owner.BodyEnd,
			paramsAt: owner.Params, paramsEnd: owner.ParamsEnd,
		})
	} else {
		if !isScriptPage(content) {
			return nil, NotAVariable("the cursor is not in a script function's body")
		}

		ownerSpan = FunctionSpan{Start: 0, End: len(content), Params: -1, ParamsEnd: -1, Body: 0, BodyEnd: len(content)}
		scopes = append(closuresAround(content, &ownerSpan, at), renameScope{
			body: 0, bodyEnd: len(content), paramsAt: -1, paramsEnd: -1,
		})
	}

	for _, sc := range scopes {
		if plan, handled, err := planCatchScope(content, spans, &ownerSpan, sc, at); handled {
			return plan, err
		}

		plan, err := planScope(content, spans, sc, at)
		if errors.Is(err, errUndeclared) {
			continue
		}

		if err == nil && (sc.body != ownerSpan.Body || sc.bodyEnd != ownerSpan.BodyEnd) {
			// A closure's variable is renamed among the names of the whole
			// function around it, whose own variables it reads.
			for name, why := range mentionedNames(content, spans, &ownerSpan) {
				takeName(plan.Taken, name, why)
			}
		}

		return plan, err
	}

	return nil, NotAVariable("'" + wordAt(content, at) + "' is not declared in this function with var or local")
}

// isScriptPage reports whether content is a standalone CFScript page (no CF
// tags and not a component/interface wrapper).
func isScriptPage(content string) bool {
	if strings.TrimSpace(content) == "" || strings.Contains(content, "```") || IndexFold(content, "<cf") >= 0 {
		return false
	}

	if !isScriptFile(content) {
		return false
	}

	if _, isComp := ComponentOutline(content); isComp {
		return false
	}

	return true
}

type catchBinding struct {
	name      string
	nameStart int
	nameEnd   int
	bodyStart int
	bodyEnd   int
}

// planCatchScope plans a rename when the cursor is on a catch clause's bound
// variable or a use of it inside that catch block, provided the enclosing scope
// does not also declare the same variable.
func planCatchScope(content string, spans []FunctionSpan, owner *FunctionSpan, sc renameScope, at int) (*LocalRename, bool, error) {
	if at < sc.body || at > sc.bodyEnd {
		return nil, false, nil
	}

	name := wordAt(content, at)
	if name == "" {
		return nil, false, nil
	}

	cb, ok := innermostCatchBinding(content, sc, at, name)
	if !ok {
		return nil, false, nil
	}

	// If the enclosing scope also declares this name as an argument, local, or
	// var outside the catch, fall through to the existing shadowing safeguard.
	if slices.Contains(paramNames(sc.params), strings.ToLower(name)) {
		return nil, false, nil
	}

	if scopeDeclaresOutsideCatch(content[sc.body:sc.bodyEnd], sc.body, cb, name) {
		return nil, false, nil
	}

	body := content[cb.bodyStart:cb.bodyEnd]
	if strings.Contains(body, "```") || strings.Contains(strings.ToLower(body), "<cf") {
		return nil, true, refuse("the catch block holds CF tags")
	}

	w := &renameWalk{
		name:     cb.name,
		declared: true,
		taken:    map[string]string{},
		nested:   nestedRanges(spans, cb.bodyStart, cb.bodyEnd),
		ranges:   [][2]int{{cb.nameStart, cb.nameEnd}},
	}
	w.walk(body, cb.bodyStart, false)

	switch {
	case w.err != nil:
		return nil, true, w.err
	case w.called && !w.assignedFn:
		return nil, true, refuse("'" + cb.name + "' is also called, and a call to an unscoped name may be a method of the component, which a rename would leave behind")
	}

	for taken, why := range mentionedNames(content, spans, owner) {
		takeName(w.taken, taken, why)
	}

	slices.SortFunc(w.ranges, func(a, b [2]int) int { return a[0] - b[0] })

	return &LocalRename{Name: cb.name, Ranges: w.ranges, Taken: w.taken, StatementStart: w.statementStart}, true, nil
}

func innermostCatchBinding(content string, sc renameScope, at int, name string) (catchBinding, bool) {
	toks := scriptTokens(content[sc.body:sc.bodyEnd])

	var (
		best  catchBinding
		found bool
	)

	for i, t := range toks {
		if t.Kind != TokIdent || !strings.EqualFold(t.Value, "catch") || i > 0 && toks[i-1].Kind == TokDot {
			continue
		}

		if i+1 >= len(toks) || toks[i+1].Kind != TokLParen {
			continue
		}

		rparen := matching(toks, i+1, TokLParen, TokRParen)
		if rparen < 0 || rparen+1 >= len(toks) || toks[rparen+1].Kind != TokLBrace {
			continue
		}

		rbrace := matching(toks, rparen+1, TokLBrace, TokRBrace)
		if rbrace < 0 {
			continue
		}

		last := -1

		for j := i + 2; j < rparen; j++ {
			if toks[j].Kind == TokIdent {
				last = j
			}
		}

		if last < 0 || !strings.EqualFold(toks[last].Value, name) {
			continue
		}

		cb := catchBinding{
			name:      toks[last].Value,
			nameStart: sc.body + toks[last].Offset,
			nameEnd:   sc.body + toks[last].Offset + len(toks[last].Value),
			bodyStart: sc.body + toks[rparen+1].Offset + 1,
			bodyEnd:   sc.body + toks[rbrace].Offset,
		}

		onHeader := at >= cb.nameStart && at <= cb.nameEnd
		inBody := at >= cb.bodyStart && at <= cb.bodyEnd

		if !onHeader && !inBody {
			continue
		}

		if !found || (cb.bodyEnd-cb.nameStart) < (best.bodyEnd-best.nameStart) {
			best, found = cb, true
		}
	}

	return best, found
}

func scopeDeclaresOutsideCatch(body string, base int, cb catchBinding, name string) bool {
	toks := scriptTokens(body)

	for i, t := range toks {
		off := base + t.Offset
		if off >= cb.nameStart && off <= cb.bodyEnd {
			continue
		}

		if t.Kind == TokIdent && strings.EqualFold(t.Value, "var") && (i == 0 || toks[i-1].Kind != TokDot) {
			last := -1

			for j := i + 1; j < len(toks); j++ {
				if toks[j].Kind == TokDot {
					continue
				}

				if toks[j].Kind != TokIdent || strings.EqualFold(toks[j].Value, "in") {
					break
				}

				if last >= 0 && toks[j].Line != toks[last].Line {
					break
				}

				last = j
			}

			if last >= 0 && strings.EqualFold(toks[last].Value, name) {
				return true
			}
		}

		if t.Kind == TokIdent && strings.EqualFold(t.Value, name) &&
			i >= 2 && toks[i-1].Kind == TokDot && toks[i-2].Kind == TokIdent && strings.EqualFold(toks[i-2].Value, "local") &&
			(i < 3 || toks[i-3].Kind != TokDot) &&
			i+1 < len(toks) && toks[i+1].Kind == TokEquals && !equalsPair(toks, i+1) {
			return true
		}
	}

	return false
}

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

// frame kinds on renameWalk's bracket stack.
type renameBracket uint8

const (
	rnGroup  renameBracket = iota // ( for grouping or a condition
	rnCall                        // ( of a call: `f(x = 1)` names an argument
	rnParams                      // ( of a function's parameters
	rnBlock                       // { of a statement block
	rnStruct                      // { of a struct literal: `{ x = 1 }` names a key
	rnFunc                        // { of a nested function's body
	rnIndex                       // [
)

type rnFrame struct {
	kind      renameBracket
	ternary   int  // `?` seen at this level, not yet matched by its `:`
	inDefault bool // in a parameter list, after the `=` of a default
}

type renameWalk struct {
	name           string
	taken          map[string]string
	nested         [][2]int
	ranges         [][2]int
	declared       bool
	statementStart bool
	// called is set when the name is called, `foo( 2 )`, and assignedFn when
	// the variable is given a function, `foo = function(`: a call to an
	// unscoped name is the variable's only if the variable holds a function,
	// and else a method of the component, which keeps its name.
	called, assignedFn bool
	// mentionOnly collects the names the text means something by and renames
	// nothing, so it refuses nothing: it reads a function around the one being
	// renamed.
	mentionOnly bool
	err         error
}

// fail refuses the rename for why, unless the walk only collects names.
func (w *renameWalk) fail(why string) {
	if !w.mentionOnly && w.err == nil {
		w.err = refuse(why)
	}
}

// take records that the function means something by name, which is why.
func (w *renameWalk) take(name, why string) { takeName(w.taken, strings.ToLower(name), why) }

func takeName(taken map[string]string, name, why string) {
	if cur, ok := taken[name]; !ok || takenRank[why] > takenRank[cur] {
		taken[name] = why
	}
}

// notCalls are the words that may stand before a ( without calling anything.
var notCalls = map[string]bool{
	"if": true, "else": true, "while": true, "for": true, "switch": true, "catch": true,
	"return": true, "function": true, "in": true, "and": true, "or": true, "not": true, "eq": true,
	"neq": true, "is": true, "contains": true, "case": true, "throw": true, "try": true,
}

// structBefore are the tokens after which a `{` opens a struct literal.
func structBefore(t Token) bool {
	switch t.Kind {
	case TokEquals, TokLParen, TokComma, TokLBracket, TokColon, TokQuestion:
		return true
	case TokIdent:
		return strings.EqualFold(t.Value, "return")
	default:
		return false
	}
}

// bracketState is the brackets open at a point of a walk, and what the next
// ( or { is for.
type bracketState struct {
	stack     []rnFrame
	fnDepth   int  // closure bodies open
	pendFunc  bool // a `function` was read; its parameter list is next
	pendBrace bool // its parameters are closed; its body brace is next
	// expr is set for a fragment that is an expression, a `#...#` or a
	// parameter's default, where nothing is a statement.
	expr bool
	// types are the tokens that are a type, `string` in `var string s`, which
	// is no name the function uses.
	types map[int]bool
	// arrows are the token ranges of the braced bodies of arrow functions,
	// which are scopes of their own that are not pushed as frames.
	arrows [][2]int
}

// inArrow reports whether toks[i] stands in the braced body of an arrow
// function.
func (b *bracketState) inArrow(i int) bool {
	for _, r := range b.arrows {
		if i > r[0] && i < r[1] {
			return true
		}
	}

	return false
}

func (b *bracketState) top() *rnFrame {
	if len(b.stack) == 0 {
		return nil
	}

	return &b.stack[len(b.stack)-1]
}

// open files the bracket of toks[i].
func (b *bracketState) open(toks []Token, i int, hash bool) {
	var prev Token
	if i > 0 {
		prev = toks[i-1]
	}

	switch toks[i].Kind {
	case TokLParen:
		kind := rnGroup

		switch {
		case b.pendFunc:
			kind, b.pendFunc = rnParams, false
		case i > 0 && (prev.Kind == TokRParen || prev.Kind == TokRBracket):
			kind = rnCall
		case i > 0 && prev.Kind == TokIdent && !notCalls[strings.ToLower(prev.Value)]:
			kind = rnCall
		default:
		}

		b.stack = append(b.stack, rnFrame{kind: kind})
	case TokLBracket:
		b.stack = append(b.stack, rnFrame{kind: rnIndex})
	case TokLBrace:
		kind := rnBlock

		switch {
		case b.pendBrace:
			kind, b.pendBrace = rnFunc, false
			b.fnDepth++
		case i > 0 && prev.Kind == TokColon && afterCaseLabel(toks, i):
			// `case 1: { ... }` is a block, whose assignments are statements.
		case i == 0 && hash, i > 0 && structBefore(prev):
			kind = rnStruct
		default:
		}

		b.stack = append(b.stack, rnFrame{kind: kind})
	default:
	}
}

// shut files the closing bracket of kind.
func (b *bracketState) shut(kind TokenKind) {
	f := b.top()
	if f == nil {
		return
	}

	switch {
	case kind == TokRParen && (f.kind == rnGroup || f.kind == rnCall || f.kind == rnParams):
		b.pendBrace = b.pendBrace || f.kind == rnParams
	case kind == TokRBracket && f.kind == rnIndex:
	case kind == TokRBrace && (f.kind == rnBlock || f.kind == rnStruct || f.kind == rnFunc):
		if f.kind == rnFunc {
			b.fnDepth--
		}
	default:
		return
	}

	b.stack = b.stack[:len(b.stack)-1]
}

// walk reads text, which stands at base in the page, as the body of the
// function. hash is set for the expression of a `#...#`, which has no
// statements.
func (w *renameWalk) walk(text string, base int, hash bool) {
	toks := scriptTokens(text)

	st := bracketState{types: map[int]bool{}, expr: hash}

	for i, t := range toks {
		if w.err != nil {
			return
		}

		if w.inNested(base + t.Offset) {
			w.nestedRead(toks, i)

			continue
		}

		switch t.Kind {
		case TokLParen, TokLBracket, TokLBrace:
			st.open(toks, i, hash)
		case TokRParen, TokRBracket, TokRBrace:
			st.shut(t.Kind)
		case TokQuestion:
			if f := st.top(); f != nil {
				f.ternary++
			}
		case TokColon:
			if f := st.top(); f != nil && f.ternary > 0 {
				f.ternary--
			}
		case TokComma:
			if f := st.top(); f != nil {
				f.inDefault = false
			}
		case TokString:
			w.checkString(t, base)
		case TokIdent:
			w.word(toks, i, &st, base)
		case TokEquals:
			// `=>` makes an arrow function: the parameters before it are its own.
			if i+1 < len(toks) && toks[i+1].Kind == TokGT && toks[i+1].Offset == t.Offset+1 {
				w.arrowParams(toks, i)

				if i+2 < len(toks) && toks[i+2].Kind == TokLBrace {
					end := matching(toks, i+2, TokLBrace, TokRBrace)
					if end < 0 {
						end = len(toks)
					}

					st.arrows = append(st.arrows, [2]int{i + 2, end})
				}
			}

			if f := st.top(); f != nil && f.kind == rnParams && !equalsPair(toks, i) {
				f.inDefault = true
			}
		default:
		}
	}
}

// inNested reports whether the page offset off lies in a named function inside
// the body, which is a scope of its own.
func (w *renameWalk) inNested(off int) bool {
	for _, r := range w.nested {
		if off >= r[0] && off < r[1] {
			return true
		}
	}

	return false
}

// word reads the identifier toks[i].
func (w *renameWalk) word(toks []Token, i int, st *bracketState, base int) {
	t := toks[i]
	lower := strings.ToLower(t.Value)

	var prev, next Token
	if i > 0 {
		prev = toks[i-1]
	}

	if i+1 < len(toks) {
		next = toks[i+1]
	}

	switch lower {
	case "function":
		if prev.Kind != TokDot {
			st.pendFunc = true
		}
	case "evaluate", "precisionevaluate":
		w.fail("the function calls " + t.Value + "(), which reads variables by name")

		return
	case "include", "cfinclude":
		// `include "a.cfm";` and `cfinclude( template = ... )` run a template in
		// this function's scope, which reads its variables by name.
		if prev.Kind != TokDot && !st.types[i] && (lower == "include" && (next.Kind == TokString || next.Kind == TokIdent) || lower == "cfinclude" && next.Kind == TokLParen) {
			w.fail("the function includes a template, which reads variables by name")

			return
		}
	case "thread":
		// `thread name="t" { ... }`: the body has a scope of its own and is
		// given values by attributes.
		if prev.Kind != TokDot && !st.types[i] && (next.Kind == TokIdent || next.Kind == TokLBrace) {
			w.fail("the function starts a thread, whose body has a scope of its own")

			return
		}
	case "local":
		// `local.x` names one variable; `local` or `local[k]` alone is the
		// whole scope, whose keys are read by name. (`arguments` holds the
		// arguments only, which are not renamed here.)
		if prev.Kind != TokDot && next.Kind != TokDot {
			w.fail("the function uses the whole " + lower + " scope, which reads variables by name")

			return
		}
	case "var":
		w.declaration(toks, i, st)
	case "for":
		w.forInBinding(toks, i, st)
	case "catch":
		w.catchVariable(toks, i, st)
	}

	if st.types[i] {
		return
	}

	// The word between `function` and ( is the nested function's name.
	if prev.Kind == TokIdent && strings.EqualFold(prev.Value, "function") && next.Kind == TokLParen {
		return
	}

	if f := st.top(); f != nil && f.kind == rnParams && !f.inDefault {
		w.nestedParameter(toks, i)

		return
	}

	if strings.EqualFold(t.Value, w.name) && w.name != "" {
		switch {
		case next.Kind == TokLParen && prev.Kind != TokDot && !declaresOrCreates(prev):
			w.called = true
		case next.Kind == TokEquals && !equalsPair(toks, i+1) && i+2 < len(toks) && toks[i+2].Kind == TokIdent && strings.EqualFold(toks[i+2].Value, "function"):
			w.assignedFn = true
		}
	}

	// `{ x }` is the shorthand of `{ x: x }`: the key is the name, which a
	// rename would change.
	if strings.EqualFold(t.Value, w.name) && (prev.Kind == TokLBrace || prev.Kind == TokComma) && (next.Kind == TokRBrace || next.Kind == TokComma) {
		if f := st.top(); f != nil && f.kind == rnStruct {
			w.fail("'" + w.name + "' stands alone in a struct, where it is the key as well")

			return
		}
	}

	// `thread name="t" count=count`, `http url="a" result=result`: after a
	// statement's word an `=` names an attribute, which is not the variable.
	if strings.EqualFold(t.Value, w.name) && next.Kind == TokEquals && !equalsPair(toks, i+1) && labelsAnAttribute(toks, i, st) {
		w.fail("'" + w.name + "' is also an attribute of a statement, which no rename can follow")

		return
	}

	switch classifyWord(toks, i, st) {
	case wordOther:
		return
	case wordLocal:
		// `local.x` is the variable x, written or read: where this function's
		// own scope is; in a closure `local` is the closure's.
		if strings.EqualFold(t.Value, w.name) && (st.fnDepth > 0 || st.inArrow(i)) {
			w.fail("a function inside this one uses local." + w.name + ", which is that function's own variable")

			return
		}

		w.take(t.Value, takenLocal)
	default:
		// A word the language reserves is no name the function means anything by.
		if !hardReserved[lower] {
			w.take(t.Value, takenOther)
		}
	}

	if strings.EqualFold(t.Value, w.name) {
		w.occurrence(toks, i, st, base)
	}
}

// nestedParameter reads the word toks[i] of a parameter list of a function
// inside this one: a parameter's name stands before a comma, a default or the
// end; what stands before that is its type and modifiers.
func (w *renameWalk) nestedParameter(toks []Token, i int) {
	t := toks[i]

	if i+1 < len(toks) && (toks[i+1].Kind == TokIdent || toks[i+1].Kind == TokDot) {
		return // a type or a modifier
	}

	if strings.EqualFold(t.Value, w.name) {
		w.fail("a function inside this one takes '" + w.name + "' as its own argument")

		return
	}

	w.take(t.Value, takenArgument)
}

// wordKind is what an identifier is in the text around it.
type wordKind uint8

const (
	// wordPlain is a name the text means something by: a variable, or a
	// function it calls.
	wordPlain wordKind = iota
	// wordLocal is `local.name`: the local variable, written with its scope.
	wordLocal
	// wordOther is a word that is no variable of this function: another
	// object's member, a struct key, a named argument's label, or a type.
	wordOther
)

// classifyWord says what the identifier toks[i] is.
func classifyWord(toks []Token, i int, st *bracketState) wordKind {
	if i > 0 {
		switch prev := toks[i-1]; {
		case prev.Kind == TokDoubleColon:
			return wordOther
		case prev.Kind == TokDot:
			// `a.x` is another object's; `local.x` is the variable's.
			// (A `local` that is itself a member, `request.local.x`, is not the scope.)
			if i >= 2 && toks[i-2].Kind == TokIdent && strings.EqualFold(toks[i-2].Value, "local") && (i < 3 || toks[i-3].Kind != TokDot) {
				return wordLocal
			}

			return wordOther
		case prev.Kind == TokIdent && strings.EqualFold(prev.Value, "new"):
			return wordOther // `new Foo()` names a type
		}
	}

	// `User::find()`: the qualifier is a component's name.
	if i+1 < len(toks) && toks[i+1].Kind == TokDoubleColon {
		return wordOther
	}

	if i+1 < len(toks) {
		var f *rnFrame
		if len(st.stack) > 0 {
			f = &st.stack[len(st.stack)-1]
		}

		switch next := toks[i+1]; {
		case next.Kind == TokEquals && !equalsPair(toks, i+1) && f != nil &&
			(f.kind == rnCall || f.kind == rnStruct || f.kind == rnParams):
			// `f( x = 1 )` and `{ x = 1 }` name an argument and a key.
			return wordOther
		case next.Kind == TokColon && f != nil && f.ternary == 0 && (f.kind == rnStruct || f.kind == rnCall):
			return wordOther
		}
	}

	return wordPlain
}

// declaration reads the variable `var` at toks[i] declares and files it.
func (w *renameWalk) declaration(toks []Token, i int, st *bracketState) {
	if i > 0 && toks[i-1].Kind == TokDot {
		return
	}

	last := -1

	for j := i + 1; j < len(toks); j++ {
		t := toks[j]
		if t.Kind == TokDot {
			continue
		}

		if t.Kind != TokIdent || strings.EqualFold(t.Value, "in") {
			break
		}

		// Semicolons are optional: `var x` ends at its line, and the next
		// line's first word is another statement.
		if last >= 0 && t.Line != toks[last].Line {
			break
		}

		last = j
	}

	if last < 0 {
		return
	}

	// What stands before the name is the declaration's type.
	for j := i + 1; j < last; j++ {
		st.types[j] = true
	}

	name := toks[last].Value
	w.take(name, takenLocal)

	if strings.EqualFold(name, w.name) {
		if st.fnDepth > 0 || st.inArrow(i) {
			w.fail("a function inside this one declares '" + w.name + "' for itself")

			return
		}

		w.declared = true
	}
}

// forInBinding recognizes `for ( item in expr )` without `var`: it marks
// `item` as declared in this scope when renaming it (while leaving `for ( var
// item in expr )` to declaration and keeping `Taken` classification unchanged).
func (w *renameWalk) forInBinding(toks []Token, i int, st *bracketState) {
	if i > 0 && toks[i-1].Kind == TokDot {
		return
	}

	if i+3 >= len(toks) || toks[i+1].Kind != TokLParen || toks[i+2].Kind != TokIdent || toks[i+3].Kind != TokIdent {
		return
	}

	if strings.EqualFold(toks[i+2].Value, "var") || !strings.EqualFold(toks[i+3].Value, "in") {
		return
	}

	if strings.EqualFold(toks[i+2].Value, w.name) {
		if st.fnDepth > 0 || st.inArrow(i) {
			w.fail("a function inside this one declares '" + w.name + "' for itself")

			return
		}

		w.declared = true
	}
}

// catchVariable reads `catch ( any e )`: its last word is the variable it
// binds, which a rename of the same name is refused, and the rest is its type.
func (w *renameWalk) catchVariable(toks []Token, i int, st *bracketState) {
	last := -1

	for j := i + 1; j < len(toks) && toks[j].Kind != TokRParen; j++ {
		if toks[j].Kind == TokIdent {
			last = j
		}
	}

	if last < 0 {
		return
	}

	for j := i + 1; j < last; j++ {
		st.types[j] = true
	}

	w.take(toks[last].Value, takenCatch)

	if strings.EqualFold(toks[last].Value, w.name) {
		w.fail("'" + w.name + "' is also what a catch binds")
	}
}

// arrowParams refuses a name the parameters of an arrow function, which end
// at toks[i], bind for it alone.
func (w *renameWalk) arrowParams(toks []Token, i int) {
	if i == 0 {
		return
	}

	lo := i - 1
	if toks[lo].Kind == TokRParen {
		if open := matchingBack(toks, lo, TokLParen, TokRParen); open >= 0 {
			lo = open
		}
	}

	for _, t := range toks[lo:i] {
		if t.Kind == TokIdent && strings.EqualFold(t.Value, w.name) {
			w.fail("an arrow function inside this one takes '" + w.name + "' as its own argument")

			return
		}
	}
}

// checkString reads a string: one that is the variable's name is a use the
// rename cannot follow, and a `#...#` in one is code.
func (w *renameWalk) checkString(t Token, base int) {
	if len(t.Value) < 2 {
		return
	}

	inner := stringContent(t.Value)
	lower := strings.ToLower(strings.TrimSpace(inner))

	if w.name != "" && namesInPath(lower, strings.ToLower(w.name)) {
		w.fail("the function names '" + w.name + "' in a string, where no rename can follow it")

		return
	}

	for _, seg := range hashSegments(inner) {
		w.walk(inner[seg[0]:seg[1]], base+t.Offset+1+seg[0], true)
	}
}

// occurrence files toks[i], a word that is the variable's name and is the
// variable, not a key, a label or another object's member.
func (w *renameWalk) occurrence(toks []Token, i int, st *bracketState, base int) {
	t := toks[i]

	if i > 0 && toks[i-1].Kind == TokDot && i+1 < len(toks) && toks[i+1].Kind == TokEquals && !equalsPair(toks, i+1) && st.fnDepth == 0 && !st.inArrow(i) {
		w.declared = true // `local.x = …` declares it
	}

	if startsStatement(toks, i, st) {
		w.statementStart = true
	}

	w.ranges = append(w.ranges, [2]int{base + t.Offset, base + t.Offset + len(t.Value)})
}

// startsStatement reports whether the word toks[i] is the first of a
// statement: after a `;`, a block's `{` or a `}`, after the `)` of an if,
// while, for, switch or catch, after else, do, try or finally, after a case
// label's `:`, or at the start of a function's body. A fragment that is an
// expression has no statements.
func startsStatement(toks []Token, i int, st *bracketState) bool {
	if st.expr {
		return false
	}

	inStruct := len(st.stack) > 0 && st.stack[len(st.stack)-1].kind == rnStruct

	if i == 0 {
		return !inStruct
	}

	switch prev := toks[i-1]; prev.Kind {
	case TokSemicolon, TokRBrace:
		return true
	case TokLBrace:
		return !inStruct
	case TokRParen:
		return closesControlParen(toks, i-1)
	case TokColon:
		return afterCaseLabel(toks, i-1)
	case TokIdent:
		switch strings.ToLower(prev.Value) {
		case "else", "do", "try", "finally":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

// closesControlParen reports whether the `)` at toks[i] closes the condition of
// an if, while, for, switch or catch, whose statement follows it.
func closesControlParen(toks []Token, i int) bool {
	depth := 0

	for j := i; j >= 0; j-- {
		switch toks[j].Kind {
		case TokRParen:
			depth++
		case TokLParen:
			depth--

			if depth == 0 {
				if j == 0 || toks[j-1].Kind != TokIdent {
					return false
				}

				switch strings.ToLower(toks[j-1].Value) {
				case "if", "while", "for", "switch", "catch":
					return true
				default:
					return false
				}
			}
		default:
		}
	}

	return false
}

// afterCaseLabel reports whether the `:` at toks[i] ends a `case x:` or
// `default:` label, as against a ternary's.
func afterCaseLabel(toks []Token, i int) bool {
	for j := i - 1; j >= 0; j-- {
		switch toks[j].Kind {
		case TokSemicolon, TokLBrace, TokRBrace:
			return false
		case TokIdent:
			if v := strings.ToLower(toks[j].Value); v == "case" || v == "default" {
				return true
			}
		default:
		}
	}

	return false
}

// equalsPair reports whether the `=` at toks[i] is half of `==`.
func equalsPair(toks []Token, i int) bool {
	return i+1 < len(toks) && toks[i+1].Kind == TokEquals && toks[i+1].Offset == toks[i].Offset+1 ||
		i > 0 && toks[i-1].Kind == TokEquals && toks[i-1].Offset+1 == toks[i].Offset
}

// mentionedNames are the names the function means something by, anywhere in
// its parameters and body, each with what it is: found by the same walk as a
// rename's, so a key, a label, a member and a type are not among them.
func mentionedNames(content string, spans []FunctionSpan, owner *FunctionSpan) map[string]string {
	w := &renameWalk{mentionOnly: true, taken: map[string]string{}, nested: nestedRanges(spans, owner.Body, owner.BodyEnd)}

	w.params(owner.ParamsText(content), owner.Params)
	w.walk(content[owner.Body:owner.BodyEnd], owner.Body, false)

	return w.taken
}

// params files the parameters of a list: their names, and the names their
// defaults use.
func (w *renameWalk) params(text string, base int) {
	// A default is evaluated before the function's variables exist: it names
	// what is outside, so it is read for names and renames nothing.
	defaults := &renameWalk{mentionOnly: true, taken: w.taken, nested: w.nested}

	for _, p := range parseParams(text) {
		w.take(p.name, takenArgument)

		if p.defStart >= 0 && p.defEnd > p.defStart {
			defaults.walk(text[p.defStart:p.defEnd], base+p.defStart, true)
		}
	}
}

// stringContent is the text of a string token without its quotes. The string
// ends at the first quote that is not doubled, which is an escaped one; a string
// cut off by the end of the text, as one is while it is being typed, has no
// closing quote, and keeps its last character.
func stringContent(tok string) string {
	if tok == "" {
		return ""
	}

	q := tok[0]

	for i := 1; i < len(tok); i++ {
		if tok[i] != q {
			continue
		}

		if i+1 < len(tok) && tok[i+1] == q {
			i++

			continue
		}

		return tok[1:i]
	}

	return tok[1:]
}

// hashSegments are the [start, end) offsets in inner of the expressions
// between its `#` pairs; `##` is a hash character. A `#` with no partner
// opens none.
func hashSegments(inner string) [][2]int {
	var out [][2]int

	for from := 0; from < len(inner); {
		open := strings.IndexByte(inner[from:], '#')
		if open < 0 {
			return out
		}

		open += from

		if open+1 < len(inner) && inner[open+1] == '#' {
			from = open + 2

			continue
		}

		// The expression ends at the next hash outside a string of its own:
		// `#f( '#x#' )#` holds a string, which has hashes too.
		end, quote := -1, byte(0)

		for k := open + 1; k < len(inner) && end < 0; k++ {
			switch c := inner[k]; {
			case quote != 0:
				if c == quote {
					quote = 0
				}
			case c == '"' || c == '\'':
				quote = c
			case c == '#':
				end = k
			}
		}

		if end < 0 {
			return out
		}

		out = append(out, [2]int{open + 1, end})
		from = end + 1
	}

	return out
}

// labelsAnAttribute reports whether the word toks[i], which an `=` follows, is
// the label of a statement's attribute: it stands at statement level straight
// after a string or a word that is not a keyword or a type.
func labelsAnAttribute(toks []Token, i int, st *bracketState) bool {
	if f := st.top(); f != nil && f.kind != rnBlock {
		return false
	}

	if i == 0 {
		return false
	}

	switch prev := toks[i-1]; prev.Kind {
	case TokString:
		return true
	case TokIdent:
		if st.types[i-1] {
			return false
		}

		switch strings.ToLower(prev.Value) {
		case "var", "else", "do", "final", "static", "try", "finally", "default":
			return false
		default:
			return true
		}
	default:
		return false
	}
}

// namesInPath reports whether the string text, lowercased, is a name that
// reads the variable name, however it is reached: `x`, `local.x`, and a longer
// path that begins with it, `x.y` or `x[1]`.
func namesInPath(text, name string) bool {
	text = strings.TrimPrefix(strings.TrimPrefix(text, "local."), "arguments.")

	return text == name || strings.HasPrefix(text, name+".") || strings.HasPrefix(text, name+"[")
}

// matchingBack is the index of the token that opens the bracket closed at
// toks[i], or -1.
func matchingBack(toks []Token, i int, open, shut TokenKind) int {
	depth := 0

	for j := i; j >= 0; j-- {
		switch toks[j].Kind {
		case shut:
			depth++
		case open:
			depth--
			if depth == 0 {
				return j
			}
		default:
		}
	}

	return -1
}

// nestedRead refuses the rename when a named function inside the body reads
// the variable, toks[i] among its tokens: it is a scope of its own, which this
// walk does not follow, so the read could not be renamed with the rest. The
// function's own name is not a read.
func (w *renameWalk) nestedRead(toks []Token, i int) {
	t := toks[i]

	// A string that names the variable, or reads it in a `#...#`.
	if t.Kind == TokString && w.name != "" && len(t.Value) >= 2 {
		inner := stringContent(t.Value)

		mentioned := namesInPath(strings.ToLower(strings.TrimSpace(inner)), strings.ToLower(w.name))
		for _, seg := range hashSegments(inner) {
			mentioned = mentioned || hasWordFold(inner[seg[0]:seg[1]], w.name)
		}

		if mentioned {
			w.fail("a function inside this one names '" + w.name + "' in a string, where no rename can follow it")
		}

		return
	}

	if t.Kind != TokIdent || w.name == "" || !strings.EqualFold(t.Value, w.name) {
		return
	}

	if i > 0 && toks[i-1].Kind == TokDot {
		return
	}

	if i > 0 && toks[i-1].Kind == TokIdent && strings.EqualFold(toks[i-1].Value, "function") && i+1 < len(toks) && toks[i+1].Kind == TokLParen {
		return
	}

	w.fail("a function inside this one reads '" + w.name + "', which this rename cannot follow")
}

// scriptTokens are the tokens of src read as CFScript, without the EOF that
// tokenize ends with.
func scriptTokens(src string) []Token {
	toks := tokenize(src)

	return toks[:len(toks)-1]
}

// declaresOrCreates reports whether t is the word before a name that makes it a
// function's or a type's name, `function foo(` and `new Foo(`: not a call.
func declaresOrCreates(t Token) bool {
	return t.Kind == TokIdent && (strings.EqualFold(t.Value, "function") || strings.EqualFold(t.Value, "new"))
}
