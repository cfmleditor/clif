package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// A Matcher is a resolver's match written as code. The built-in framework
// presets used to spell theirs as patterns, `(?:^|\.)getBean\(\s*…\)$`, which
// every lookup compiled and ran; these recognise the same expressions by
// reading them. A configuration's own `match` is still a string, a plain one or
// a pattern, and takes the other route (Resolver.compiledRe).
//
// A matcher is handed the expression sliced from where the resolver's prefix
// was found, ignores case as the patterns did (they were compiled with (?i)),
// and answers whether the whole of it is the one it recognises and what it
// captured: the text a `$1` in the resolver's template stands for.
type Matcher interface {
	// Match reports whether expr is the expression the matcher recognises.
	Match(expr string) (captures []string, ok bool)
	// String says what it recognises, for the explain trace.
	String() string
}

// NameSet recognises a variable by name: one of names, alone, or with
// `variables.` or `arguments.` before it.
func NameSet(names ...string) Matcher { return &nameSetMatcher{names: names} }

type nameSetMatcher struct{ names []string }

func (n *nameSetMatcher) Match(expr string) ([]string, bool) {
	for _, scope := range []string{"", "variables.", "arguments."} {
		if !lex.HasPrefixFold(expr, scope) {
			continue
		}

		rest := expr[len(scope):]
		for _, name := range n.names {
			if strings.EqualFold(rest, name) {
				return nil, true
			}
		}
	}

	return nil, false
}

func (n *nameSetMatcher) String() string { return "variable " + strings.Join(n.names, ", ") }

// EmptyCall recognises a call to name with no arguments that ends the
// expression, as the whole of it or after a dot: `content()`, `x.content( )`,
// with space allowed before the name and inside the parentheses.
func EmptyCall(name string) Matcher { return &emptyCallMatcher{name: name} }

type emptyCallMatcher struct{ name string }

func (e *emptyCallMatcher) Match(expr string) ([]string, bool) {
	for _, start := range callStarts(expr) {
		i := lex.SkipRegexpSpace(expr, start)
		if !lex.HasPrefixFold(expr[i:], e.name) {
			continue
		}

		i = lex.SkipRegexpSpace(expr, i+len(e.name))
		if i >= len(expr) || expr[i] != '(' {
			continue
		}

		i = lex.SkipRegexpSpace(expr, i+1)
		if i < len(expr) && expr[i] == ')' && i+1 == len(expr) {
			return nil, true
		}
	}

	return nil, false
}

func (e *emptyCallMatcher) String() string { return "call " + e.name + "()" }

// callStarts are where a call may begin in expr: its start, and after each dot.
func callStarts(expr string) []int {
	starts := []int{0}

	for i := 0; i < len(expr); i++ {
		if expr[i] == '.' {
			starts = append(starts, i+1)
		}
	}

	return starts
}

// An IDForm is what a call's string argument may be, besides one given text.
type IDForm uint8

const (
	// IDLiteral is the one text LiteralCall.Literal.
	IDLiteral IDForm = iota
	// IDWord is a name: a letter or an underscore, then word characters.
	IDWord
	// IDWire is a WireBox or DI/1 id, `Name`, `pkg.Name` or `Name@module`, which
	// may carry an `id:` or `model:` tag in front of it and an `@` after it; it
	// captures what lies between.
	IDWire
)

// A NamedArg says whether the string may be given as a named argument,
// `name = "x"`, and by what name.
type NamedArg uint8

const (
	// NoNamedArg allows only the string, positionally.
	NoNamedArg NamedArg = iota
	// AnyNamedArg allows any word, joined to the string by `=` or `:`.
	AnyNamedArg
	// AnyNamedArgEquals is AnyNamedArg with `=` alone.
	AnyNamedArgEquals
	// ExactNamedArg allows LiteralCall.ArgName, joined by `=` or `:`.
	ExactNamedArg
)

// LiteralCall recognises a call whose argument is a string, `getBean( "x" )`,
// ending the expression: Func, a parenthesis, the string, and the parenthesis
// that closes it. With Rest further arguments may follow the string. It is a
// Matcher as a pointer: `&LiteralCall{Func: "getBean", …}`.
type LiteralCall struct {
	Func string
	// Bare says the call is the whole expression; otherwise it may follow a dot.
	Bare bool
	// Named says whether and how the string may be a named argument.
	Named NamedArg
	// ArgName is the name an ExactNamedArg takes.
	ArgName string
	// Form is what the string holds; for IDLiteral it is Literal, and nothing
	// is captured. Otherwise the id is captured.
	Form    IDForm
	Literal string
	// Rest allows `, more` after the string.
	Rest bool
}

func (c *LiteralCall) String() string {
	arg := `"…"`
	if c.Form == IDLiteral {
		arg = `"` + c.Literal + `"`
	}

	if c.Rest {
		arg += ", …"
	}

	return "call " + c.Func + "(" + arg + ")"
}

func (c *LiteralCall) Match(expr string) ([]string, bool) {
	starts := []int{0}
	if !c.Bare {
		starts = callStarts(expr)
	}

	for _, start := range starts {
		if id, ok := c.matchAt(expr, start); ok {
			if c.Form == IDLiteral {
				return nil, true
			}

			return []string{id}, true
		}
	}

	return nil, false
}

// matchAt reads the call at start, to the end of expr.
func (c *LiteralCall) matchAt(expr string, start int) (id string, ok bool) {
	if !lex.HasPrefixFold(expr[start:], c.Func) {
		return "", false
	}

	i := start + len(c.Func)
	if i >= len(expr) || expr[i] != '(' {
		return "", false
	}

	i = c.afterNamedArg(expr, lex.SkipRegexpSpace(expr, i+1))
	if i < 0 || i >= len(expr) || !lex.IsQuote(expr[i]) {
		return "", false
	}

	i++

	var end int

	switch c.Form {
	case IDLiteral:
		if !lex.HasPrefixFold(expr[i:], c.Literal) {
			return "", false
		}

		end = i + len(c.Literal)
	case IDWord:
		if i >= len(expr) || !lex.IsWordStart(expr[i]) {
			return "", false
		}

		end = i + 1
		for end < len(expr) && lex.IsWord(expr[end]) {
			end++
		}

		id = expr[i:end]
	case IDWire:
		from, to, after, found := wireID(expr, i)
		if !found {
			return "", false
		}

		id, end = expr[from:to], after
	}

	if end >= len(expr) || !lex.IsQuote(expr[end]) {
		return "", false
	}

	return id, c.closed(expr, lex.SkipRegexpSpace(expr, end+1))
}

// afterNamedArg is where the string begins once a named argument before it, if
// the call may have one, is read; -1 when there is a word there that is not one.
func (c *LiteralCall) afterNamedArg(expr string, i int) int {
	if c.Named == NoNamedArg || i >= len(expr) || lex.IsQuote(expr[i]) {
		return i
	}

	switch c.Named {
	case ExactNamedArg:
		if !lex.HasPrefixFold(expr[i:], c.ArgName) {
			return -1
		}

		i += len(c.ArgName)
	default:
		start := i
		for i < len(expr) && lex.IsWord(expr[i]) {
			i++
		}

		if i == start {
			return -1
		}
	}

	i = lex.SkipRegexpSpace(expr, i)
	if i >= len(expr) || expr[i] != '=' && (expr[i] != ':' || c.Named == AnyNamedArgEquals) {
		return -1
	}

	return lex.SkipRegexpSpace(expr, i+1)
}

// closed reports whether expr from i is the end of the call: its parenthesis,
// or with Rest a comma, arguments holding no parenthesis, and the parenthesis.
func (c *LiteralCall) closed(expr string, i int) bool {
	if c.Rest && i < len(expr) && expr[i] == ',' {
		i++
		for i < len(expr) && expr[i] != '(' && expr[i] != ')' {
			i++
		}
	}

	return i < len(expr) && expr[i] == ')' && i+1 == len(expr)
}

// wireID reads an id at i, as
//
//	(?:id:|model:)?([A-Za-z_][\w.]*(?:@[\w.-]+)?)@?
//
// did, with a quote after it: the id's bounds, and where the text after it
// (the quote) stands. A tag is tried first, and given up if what follows it
// does not make an id the string ends on; a module is tried before the lone
// `@` that may end the id.
func wireID(expr string, i int) (from, to, after int, ok bool) {
	for _, tag := range []string{"id:", "model:"} {
		if lex.HasPrefixFold(expr[i:], tag) {
			if from, to, after, ok = wireName(expr, i+len(tag)); ok {
				return from, to, after, true
			}
		}
	}

	return wireName(expr, i)
}

// wireName reads `[A-Za-z_][\w.]*(?:@[\w.-]+)?@?` at i and the quote after it.
func wireName(expr string, i int) (from, to, after int, ok bool) {
	if i >= len(expr) || !lex.IsWordStart(expr[i]) {
		return 0, 0, 0, false
	}

	name := i + 1
	for name < len(expr) && (lex.IsWord(expr[name]) || expr[name] == '.') {
		name++
	}

	quoteAt := func(p int) bool { return p < len(expr) && lex.IsQuote(expr[p]) }

	// A module, `@name`: the id runs through it.
	if name < len(expr) && expr[name] == '@' {
		module := name + 1
		for module < len(expr) && (lex.IsWord(expr[module]) || expr[module] == '.' || expr[module] == '-') {
			module++
		}

		if module > name+1 {
			if module < len(expr) && expr[module] == '@' && quoteAt(module+1) {
				return i, module, module + 1, true
			}

			if quoteAt(module) {
				return i, module, module, true
			}
		}
	}

	// No module: the id ends at the name, and a lone `@` may follow it.
	if name < len(expr) && expr[name] == '@' && quoteAt(name+1) {
		return i, name, name + 1, true
	}

	if quoteAt(name) {
		return i, name, name, true
	}

	return 0, 0, 0, false
}

// JavaClass recognises createObject( "java", "some.Class" ) anywhere in an
// expression, capturing the class: the shortest text before a quote that is
// followed by the parenthesis closing the call.
func JavaClass() Matcher { return javaClassMatcher{} }

type javaClassMatcher struct{}

func (javaClassMatcher) String() string { return `call createObject("java", "…")` }

func (javaClassMatcher) Match(expr string) ([]string, bool) {
	const fn = "createObject"

	for from := 0; from < len(expr); from++ {
		at := IndexFoldFrom(expr, fn, from)
		if at < 0 {
			return nil, false
		}

		if class, ok := javaClassAt(expr, at+len(fn)); ok {
			return []string{class}, true
		}

		from = at
	}

	return nil, false
}

// javaClassAt reads `\s*\(\s*['"]java['"]\s*,\s*['"](.+?)['"]\s*\)` at i.
func javaClassAt(expr string, i int) (string, bool) {
	i = lex.SkipRegexpSpace(expr, i)
	if i >= len(expr) || expr[i] != '(' {
		return "", false
	}

	i = lex.SkipRegexpSpace(expr, i+1)
	if i >= len(expr) || !lex.IsQuote(expr[i]) || !lex.HasPrefixFold(expr[i+1:], "java") {
		return "", false
	}

	i += 1 + len("java")
	if i >= len(expr) || !lex.IsQuote(expr[i]) {
		return "", false
	}

	i = lex.SkipRegexpSpace(expr, i+1)
	if i >= len(expr) || expr[i] != ',' {
		return "", false
	}

	i = lex.SkipRegexpSpace(expr, i+1)
	if i >= len(expr) || !lex.IsQuote(expr[i]) {
		return "", false
	}

	from := i + 1

	// The class is one character or more but a newline, the fewest that a
	// quote, space and the closing parenthesis follow.
	for end := from + 1; end <= len(expr); end++ {
		if expr[end-1] == '\n' {
			return "", false
		}

		if end < len(expr) && lex.IsQuote(expr[end]) {
			if close := lex.SkipRegexpSpace(expr, end+1); close < len(expr) && expr[close] == ')' {
				return expr[from:end], true
			}
		}
	}

	return "", false
}
