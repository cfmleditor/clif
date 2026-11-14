package parser

import (
	"errors"
	"slices"
	"strings"
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
	// Taken are the lowercased names the function declares: its arguments and
	// its other locals.
	Taken map[string]bool
}

// ErrNotRenameable is wrapped by every refusal of PlanLocalRename. A rename of
// a variable is a change to code the server cannot run, so it makes only the
// ones it can show are the whole of the variable's uses, and says why when it
// cannot.
var ErrNotRenameable = errors.New("cannot rename")

func refuse(why string) error { return errors.New(ErrNotRenameable.Error() + ": " + why) }

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
	if owner == nil {
		return nil, refuse("the cursor is not in a script function's body")
	}

	// The scopes that hold the cursor, innermost first: the closures written
	// in the function around it, then the function. A variable belongs to the
	// nearest of them that declares it.
	scopes := append(closuresAround(content, owner, at), renameScope{
		params: owner.ParamsText(content), body: owner.Body, bodyEnd: owner.BodyEnd,
		paramsAt: owner.Params, paramsEnd: owner.ParamsEnd,
	})

	for _, sc := range scopes {
		plan, err := planScope(content, spans, sc, at)
		if errors.Is(err, errUndeclared) {
			continue
		}

		return plan, err
	}

	return nil, refuse("'" + wordAt(content, at) + "' is not declared in this function with var or local")
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
		return nil, refuse("there is no name under the cursor")
	}

	body := content[sc.body:sc.bodyEnd]

	if strings.Contains(body, "```") || strings.Contains(strings.ToLower(body), "<cf") {
		return nil, refuse("the function holds CF tags")
	}

	args := paramNames(sc.params)
	if slices.Contains(args, strings.ToLower(name)) {
		return nil, refuse("'" + name + "' is an argument, which callers pass by name")
	}

	w := &renameWalk{name: name, taken: map[string]bool{}, nested: nestedRanges(spans, sc.body, sc.bodyEnd)}
	for _, a := range args {
		w.taken[a] = true
	}

	w.walk(body, sc.body, false)

	switch {
	case w.err != nil:
		return nil, w.err
	case !w.declared:
		return nil, errUndeclared
	case len(w.ranges) == 0:
		return nil, refuse("no use of '" + name + "' found")
	}

	slices.SortFunc(w.ranges, func(a, b [2]int) int { return a[0] - b[0] })

	return &LocalRename{Name: name, Ranges: w.ranges, Taken: w.taken}, nil
}

// closuresAround are the anonymous functions written in owner's body whose
// parameters or body hold at, innermost first.
func closuresAround(content string, owner *FunctionSpan, at int) []renameScope {
	var (
		found []renameScope
		toks  []Token
	)

	body := content[owner.Body:owner.BodyEnd]

	sc := NewScriptScanner(body)
	for t := sc.NextSkipComments(); t.Kind != TokEOF; t = sc.NextSkipComments() {
		toks = append(toks, t)
	}

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
		if s := &spans[i]; s.Start > body && s.End <= bodyEnd {
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
	for start > 0 && isRenameIdentByte(content[start-1]) {
		start--
	}

	end := at
	for end < len(content) && isRenameIdentByte(content[end]) {
		end++
	}

	if start == end || content[start] >= '0' && content[start] <= '9' {
		return ""
	}

	return content[start:end]
}

func isRenameIdentByte(c byte) bool {
	return c == '_' || c == '$' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// paramNames are the lowercased argument names of a parameter list, inside
// its parentheses: for each one up to a default, its last word.
func paramNames(params string) []string {
	var (
		names []string
		last  string
		depth int
		inDef bool
	)

	flush := func() {
		if last != "" {
			names = append(names, strings.ToLower(last))
		}

		last, inDef = "", false
	}

	sc := NewScriptScanner(params)

	for t := sc.NextSkipComments(); t.Kind != TokEOF; t = sc.NextSkipComments() {
		switch t.Kind {
		case TokLParen, TokLBracket, TokLBrace:
			depth++
		case TokRParen, TokRBracket, TokRBrace:
			depth--
		case TokComma:
			if depth == 0 {
				flush()
			}
		case TokEquals:
			if depth == 0 {
				inDef = true
			}
		case TokIdent:
			if depth == 0 && !inDef {
				last = t.Value
			}
		default:
		}
	}

	flush()

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
	kind    renameBracket
	ternary int // `?` seen at this level, not yet matched by its `:`
}

type renameWalk struct {
	name     string
	taken    map[string]bool
	nested   [][2]int
	ranges   [][2]int
	declared bool
	err      error
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
	var toks []Token

	sc := NewScriptScanner(text)
	for t := sc.NextSkipComments(); t.Kind != TokEOF; t = sc.NextSkipComments() {
		toks = append(toks, t)
	}

	var st bracketState

	for i, t := range toks {
		if w.err != nil {
			return
		}

		if w.inNested(base + t.Offset) {
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
		case TokString:
			w.checkString(t, base)
		case TokIdent:
			w.word(toks, i, &st, base)
		case TokEquals:
			// `=>` makes an arrow function: the parameters before it are its own.
			if i+1 < len(toks) && toks[i+1].Kind == TokGT && toks[i+1].Offset == t.Offset+1 {
				w.arrowParams(toks, i)
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
		w.err = refuse("the function calls " + t.Value + "(), which reads variables by name")

		return
	case "var":
		w.declaration(toks, i, st.fnDepth)
	case "catch":
		w.catchVariable(toks, i)
	}

	// The word between `function` and ( is the nested function's name.
	if prev.Kind == TokIdent && strings.EqualFold(prev.Value, "function") && next.Kind == TokLParen {
		return
	}

	if f := st.top(); f != nil && f.kind == rnParams {
		// An argument of a function inside this one, or a default it gives.
		if strings.EqualFold(t.Value, w.name) && next.Kind != TokDot {
			w.err = refuse("a function inside this one takes '" + w.name + "' as its own argument")

			return
		}

		if next.Kind != TokEquals || equalsPair(toks, i+1) {
			w.taken[lower] = true
		}

		return
	}

	if strings.EqualFold(t.Value, w.name) {
		w.occurrence(toks, i, st.stack, st.fnDepth, base)
	}
}

// declaration reads the variable `var` at toks[i] declares and files it.
func (w *renameWalk) declaration(toks []Token, i, fnDepth int) {
	if i > 0 && toks[i-1].Kind == TokDot {
		return
	}

	var last Token

	for j := i + 1; j < len(toks); j++ {
		t := toks[j]
		if t.Kind == TokDot {
			continue
		}

		if t.Kind != TokIdent || strings.EqualFold(t.Value, "in") {
			break
		}

		last = t
	}

	if last.Value == "" {
		return
	}

	w.taken[strings.ToLower(last.Value)] = true

	if strings.EqualFold(last.Value, w.name) {
		if fnDepth > 0 {
			w.err = refuse("a function inside this one declares '" + w.name + "' for itself")

			return
		}

		w.declared = true
	}
}

// catchVariable refuses a name a catch binds: `catch ( any e )`.
func (w *renameWalk) catchVariable(toks []Token, i int) {
	for j := i + 1; j < len(toks) && toks[j].Kind != TokRParen; j++ {
		if toks[j].Kind == TokIdent && strings.EqualFold(toks[j].Value, w.name) {
			w.err = refuse("'" + w.name + "' is also what a catch binds")

			return
		}
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
		for lo > 0 && toks[lo].Kind != TokLParen {
			lo--
		}
	}

	for _, t := range toks[lo:i] {
		if t.Kind == TokIdent && strings.EqualFold(t.Value, w.name) {
			w.err = refuse("an arrow function inside this one takes '" + w.name + "' as its own argument")

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

	inner := t.Value[1 : len(t.Value)-1]
	lower := strings.ToLower(strings.TrimSpace(inner))

	switch lower {
	case strings.ToLower(w.name), "local." + strings.ToLower(w.name), "arguments." + strings.ToLower(w.name):
		w.err = refuse("the function names '" + w.name + "' in a string, where no rename can follow it")

		return
	default:
	}

	for from := 0; from < len(inner); {
		open := strings.IndexByte(inner[from:], '#')
		if open < 0 {
			return
		}

		open += from

		if open+1 < len(inner) && inner[open+1] == '#' {
			from = open + 2

			continue
		}

		end := strings.IndexByte(inner[open+1:], '#')
		if end < 0 {
			return
		}

		end += open + 1
		w.walk(inner[open+1:end], base+t.Offset+1+open+1, true)
		from = end + 1
	}
}

// occurrence files toks[i], a word that is the variable's name, when it is the
// variable and not a key or an argument's label.
func (w *renameWalk) occurrence(toks []Token, i int, stack []rnFrame, fnDepth, base int) {
	t := toks[i]

	if i > 0 && toks[i-1].Kind == TokDot {
		// `a.x` is another object's; `local.x` is the variable's.
		if i < 2 || toks[i-2].Kind != TokIdent || !strings.EqualFold(toks[i-2].Value, "local") {
			return
		}

		if i+1 < len(toks) && toks[i+1].Kind == TokEquals && !equalsPair(toks, i+1) && fnDepth == 0 {
			w.declared = true
		}
	} else if i > 0 && toks[i-1].Kind == TokDoubleColon {
		return
	}

	if i+1 < len(toks) {
		next := toks[i+1]

		var f *rnFrame
		if len(stack) > 0 {
			f = &stack[len(stack)-1]
		}

		switch {
		case next.Kind == TokEquals && !equalsPair(toks, i+1) && f != nil &&
			(f.kind == rnCall || f.kind == rnStruct || f.kind == rnParams):
			// `f( x = 1 )` and `{ x = 1 }` name an argument and a key.
			return
		case next.Kind == TokColon && f != nil && f.ternary == 0 && (f.kind == rnStruct || f.kind == rnCall):
			return
		}
	}

	w.ranges = append(w.ranges, [2]int{base + t.Offset, base + t.Offset + len(t.Value)})
}

// equalsPair reports whether the `=` at toks[i] is half of `==`.
func equalsPair(toks []Token, i int) bool {
	return i+1 < len(toks) && toks[i+1].Kind == TokEquals && toks[i+1].Offset == toks[i].Offset+1 ||
		i > 0 && toks[i-1].Kind == TokEquals && toks[i-1].Offset+1 == toks[i].Offset
}
