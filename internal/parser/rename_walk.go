package parser

import (
	"strings"
)

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
