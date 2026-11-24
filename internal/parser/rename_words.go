package parser

import (
	"strings"
)

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
