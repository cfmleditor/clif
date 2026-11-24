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
