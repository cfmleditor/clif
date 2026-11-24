package formatter

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// ─── statement formatters ─────────────────────────────────────────────────────

func (f *Formatter) scriptLineComment(n *sitter.Node) {
	f.iLine(f.text(n))
	f.scriptWrite("\n")
}

func (f *Formatter) scriptBlockComment(n *sitter.Node) {
	raw := f.text(n)

	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		if i == 0 {
			f.iLine(strings.TrimRight(line, " \t"))
		} else {
			f.scriptWrite("\n")
			f.writeIndent()
			f.scriptWrite(strings.TrimRight(line, " \t"))
		}
	}

	f.scriptWrite("\n")
}

// componentKeywords are the declaration keywords that may precede a component
// body. The grammar emits them as anonymous nodes, so they have to be
// recognised explicitly rather than picked up as named children.
var componentKeywords = map[string]bool{
	"component": true,
	"interface": true,
	"abstract":  true,
	"final":     true,
}

// scriptComponent renders a CFC component declaration:
//
//	component [extends="X"] [implements="Y"] { ... }
func (f *Formatter) scriptComponent(n *sitter.Node) {
	// Collect modifiers / attributes that come before the body
	var attrs []string

	var body *sitter.Node

	// Declaration keywords are anonymous nodes preceding the body. The header
	// used to be hardcoded to "component", which rewrote `interface {}` as
	// `component {}` and dropped `abstract`/`final` modifiers.
	var keywords []string

	// v0.26.34 parses `component (extends="A") {}` — the attribute list wrapped
	// in parens. The parens are anonymous children, so the walk below dropped
	// them and re-emitted the header unparenthesised. That is a non-whitespace
	// change, which the whitespaceOnly guard rejects, so the file became
	// unformattable rather than merely reformatted.
	parenthesised := false

	// The parenthesised form separates its attributes with commas
	// (`component( output=false, javasettings={...} )`) while the bare form
	// separates them with spaces. The commas are anonymous children, so joining
	// everything with a space dropped them — a non-whitespace change that made
	// the file unformattable. Which attribute carried one is recorded rather
	// than inferred from the form, so neither is imposed on the other.
	var commaAfter []bool

	for i := range n.ChildCount() {
		c := n.Child(i)
		switch c.Kind() {
		case "statement_block", "block", "class_body", "component_body":
			body = c
		case "identifier":
			// 'component' keyword itself — skip
		case "(":
			parenthesised = true
		case ")":
			// closing half of the same pair; reproduced with the opening one
		case ",":
			if len(commaAfter) > 0 {
				commaAfter[len(commaAfter)-1] = true
			}
		default:
			switch {
			case c.IsNamed():
				attrs = append(attrs, f.text(c))
				commaAfter = append(commaAfter, false)
			case componentKeywords[strings.ToLower(c.Kind())]:
				keywords = append(keywords, f.text(c))
			}
		}
	}

	header := strings.Join(keywords, " ")
	if header == "" {
		header = "component"
	}

	if len(attrs) > 0 {
		var list strings.Builder

		for i, a := range attrs {
			if i > 0 {
				list.WriteString(" ")
			}

			list.WriteString(a)

			if commaAfter[i] {
				list.WriteString(",")
			}
		}

		if parenthesised {
			header += " (" + list.String() + ")"
		} else {
			header += " " + list.String()
		}
	}

	f.iLine(header)

	if body != nil {
		f.scriptBlock(body)
	}

	f.scriptWrite("\n")
}

// scriptFunction renders a function definition:
//
//	[access] [returnType] function name([params]) { body }
func (f *Formatter) scriptFunction(n *sitter.Node) {
	// Gather prefix tokens (access modifier, return type) and the name.
	var prefix []string

	name := n.ChildByFieldName("name")
	params := n.ChildByFieldName("parameters")
	body := n.ChildByFieldName("body")
	retType := n.ChildByFieldName("return_type")

	// Attributes written after the parameter list — `function f() localmode="true" {}`
	// — are siblings of the parameters, not of the modifiers. Hoisting one into
	// the prefix emits `localmode="true" function f() {}`, which does not compile,
	// so anything starting past the parameter list is kept as a suffix.
	var attrs []string

	attrsFrom := uint(0)
	haveParams := params != nil

	if haveParams {
		attrsFrom = params.EndByte()
	}

	sawFuncKeyword := false

	// Walk children to pick up access modifiers that have no field name.
	for i := range n.ChildCount() {
		c := n.Child(i)

		fieldName := n.FieldNameForChild(uint32(i))
		switch fieldName {
		case "name", "parameters", "body", "return_type":
			continue
		}

		// Drop the `function` keyword itself, which the signature re-emits.
		// Only the first one: `function function f()` declares a function
		// returning a function, and the second token is the keyword.
		if !c.IsNamed() && c.Kind() == "function" && !sawFuncKeyword {
			sawFuncKeyword = true

			continue
		}

		if haveParams && c.StartByte() >= attrsFrom {
			attrs = append(attrs, f.text(c))

			continue
		}

		// `User[] function getUsers()` — the suffix is its own node
		// sitting beside the return type, not part of it.
		if c.Kind() == "array_return_suffix" {
			prefix = appendTypeSuffix(prefix, f.text(c))

			continue
		}

		// Anonymous children reach here too. The grammar tokenises some type
		// and modifier keywords (`query`, `abstract`, `final`) as anonymous
		// nodes rather than named identifiers; gating on IsNamed dropped them
		// from the signature entirely.
		prefix = append(prefix, f.text(c))
	}

	var sig strings.Builder
	if len(prefix) > 0 {
		sig.WriteString(strings.Join(prefix, " "))
		sig.WriteString(" ")
	}

	if retType != nil {
		sig.WriteString(f.text(retType))
		sig.WriteString(" ")
	}

	sig.WriteString("function ")

	if name != nil {
		sig.WriteString(f.text(name))
	}

	paramStr := f.exprFuncDefParams(params, len(f.opts.indent(f.level))+sig.Len())
	sig.WriteString(paramStr)

	braceLead := " "

	if len(attrs) > 0 {
		sig.WriteString(" ")
		sig.WriteString(joinSignatureAttrs(attrs, f.opts.indent(f.level+1)))

		// The body's opening brace is written straight after the signature. A
		// trailing `//` annotation comment would swallow it, so it starts a
		// line of its own instead.
		if isLineCommentText(attrs[len(attrs)-1]) {
			sig.WriteString("\n")
			sig.WriteString(f.opts.indent(f.level))

			braceLead = ""
		}
	}

	f.iLine(sig.String())

	if body != nil {
		f.scriptBlockWith(body, braceLead)
	}

	f.scriptWrite("\n")
}

// scriptProperty renders a CFC property declaration:
//
//	property [type] name [= default];
func (f *Formatter) scriptProperty(n *sitter.Node) {
	f.iLine(trimSpace(f.text(n)))
	f.scriptWrite("\n")
}

// declKeyword returns the leading keyword(s) of a variable_declaration node.
// The grammar's variable_declaration accepts "var", "final", or the combined
// "final var" / "var final" (cfscript/grammar.js), each spelled as its own
// anonymous child rather than one token — a loop that stops at the first
// "var" it sees, or that only recognises "var", silently drops "final" and
// renders it as an ordinary local, discarding the immutability it declares.
// Every keyword actually present is kept, in the order it was written;
// "var" is the fallback only when the loop finds none, which should not
// happen for a real variable_declaration node.
func declKeyword(n *sitter.Node) string {
	var kws []string

	for i := range n.ChildCount() {
		c := n.Child(i)
		if c.IsNamed() {
			continue
		}

		switch c.Kind() {
		case "var", "local", "final":
			kws = append(kws, c.Kind())
		}
	}

	if len(kws) == 0 {
		return "var"
	}

	return strings.Join(kws, " ")
}

// declarator renders one variable_declarator: its name, then its initializer
// if it has one. A compound initializer (`var hqlOrder &= " ORDER BY"`) puts
// its operator in the `operator` field, and a plain `=` has none. Both
// declaration renderers wrote ` = ` whatever the field held, so `&=` and `+=`
// came out as `=`, and the guard refused the file.
func (f *Formatter) declarator(d *sitter.Node) string {
	s := f.expr(d.ChildByFieldName("name"))

	if v := d.ChildByFieldName("value"); v != nil {
		op := "="
		if o := d.ChildByFieldName("operator"); o != nil {
			op = f.text(o)
		}

		s += " " + op + " " + f.expr(v)
	}

	return s
}

// scriptVarDecl renders: var/local/final name [= expr][, name [= expr]];.
func (f *Formatter) scriptVarDecl(n *sitter.Node) {
	keyword := declKeyword(n)

	var decls []string

	// A comment among the declarators is not one of them. Left in the list it
	// was comma-joined like a declaration — `var x = [1], // note` — and the
	// terminating semicolon then went after it, where the comment swallowed it.
	// Comments are carried across separately and placed after the semicolon,
	// which is where a trailing one was written in the first place.
	var comments []string

	for i := range n.NamedChildCount() {
		d := n.NamedChild(i)

		if isCommentKind(d.Kind()) {
			comments = append(comments, trimSpace(f.text(d)))

			continue
		}

		switch d.Kind() {
		case "variable_declarator":
			decls = append(decls, f.declarator(d))
		default:
			decls = append(decls, f.expr(d))
		}
	}

	f.iLine(fmt.Sprintf("%s %s;", keyword, strings.Join(decls, ", ")))
	f.scriptWrite("\n")

	// Each comment goes on a line of its own rather than trailing the
	// semicolon. Once the semicolon is emitted the comment is no longer part of
	// the declaration — a second pass parses it as a statement-level comment
	// and renders it on its own line — so trailing it here would make the
	// formatter's output differ from its own output on that output.
	for _, c := range comments {
		f.iLine(c)
		f.scriptWrite("\n")
	}
}

// statementComments splits a statement's named children into the one that is
// not a comment and the comments beside it. A comment can sit inside an
// expression statement or a return, next to the expression rather than in it:
// `x = 45 // note` with no semicolon, or `return a /* + b */;`. Renderers that
// took NamedChild(0) and wrote the `;` dropped it, and the guard refused the
// file.
func (f *Formatter) statementComments(n *sitter.Node) (*sitter.Node, []string) {
	var (
		inner    *sitter.Node
		comments []string
	)

	for i := range n.NamedChildCount() {
		c := n.NamedChild(i)

		switch {
		case isCommentKind(c.Kind()):
			comments = append(comments, trimSpace(f.text(c)))
		case inner == nil:
			inner = c
		}
	}

	return inner, comments
}

// scriptComments writes each comment on a line of its own after the statement,
// as scriptVarDecl does and for the same reason: once the semicolon is written
// the comment is statement-level, and a second pass renders it on its own line.
func (f *Formatter) scriptComments(comments []string) {
	for _, c := range comments {
		f.iLine(c)
		f.scriptWrite("\n")
	}
}

func (f *Formatter) scriptExprStmt(n *sitter.Node) {
	// The named child is the expression; anonymous child is ";"
	inner, comments := f.statementComments(n)
	if inner == nil {
		f.scriptComments(comments)

		return
	}

	f.iLine(f.expr(inner) + ";")
	f.scriptWrite("\n")
	f.scriptComments(comments)
}

func (f *Formatter) scriptReturn(n *sitter.Node) {
	val, comments := f.statementComments(n)
	if val == nil {
		f.iLine("return;")
	} else {
		f.iLine("return " + f.expr(val) + ";")
	}

	f.scriptWrite("\n")
	f.scriptComments(comments)
}

func (f *Formatter) scriptThrow(n *sitter.Node) {
	val := n.NamedChild(0)

	// throw message="Access Denied" type="MyCustomError"; — the tag form
	// written in script, whose attributes are space-separated rather than a
	// comma-separated argument list. The grammar reports each as its own
	// parameter_attribute child, and every path below reads NamedChild(0)
	// alone, so a throw with more than one attribute silently lost all but the
	// first — deleting the `type` a catch block dispatches on. The attributes
	// are emitted as written: the shape inside one is the tag spelling
	// (`name="value"`), not the `name = value` this formatter gives a call's
	// named arguments, and rewriting it here would be a change no config asked
	// for. Parsed only since grammar v0.26.35; before that the file was refused.
	if val != nil && val.Kind() == "parameter_attribute" {
		var attrs []string

		for i := range n.NamedChildCount() {
			c := n.NamedChild(i)
			if c.Kind() == "parameter_attribute" {
				attrs = append(attrs, trimSpace(f.text(c)))
			}
		}

		f.iLine("throw " + strings.Join(attrs, " ") + ";")
		f.scriptWrite("\n")

		return
	}

	// throw(type = "x", message = "y"): the grammar gives this the same
	// `arguments` node a call expression gets, so render it through the same
	// helper rather than re-implementing argument layout here. Falling through
	// to the generic path below emitted the argument list verbatim, which left
	// throw as the only call in the language whose named arguments were not
	// spaced like every other call's — `throw (type="x")` beside
	// `writeLog(text = "x")`. exprArgs also handles splitting long lists, which
	// the bespoke code below did separately.
	if val != nil && val.Kind() == "arguments" {
		f.iLine("throw" + f.exprArgs(val) + ";")
		f.scriptWrite("\n")

		return
	}

	// Older grammars modelled the same call as a parenthesized sequence.
	// For throw(arg, arg, ...) — split args onto new lines if too long.
	if val != nil && val.Kind() == "parenthesized_expression" {
		inner := val.NamedChild(0)
		if inner != nil && inner.Kind() == "sequence_expression" {
			var parts []string
			for i := range inner.NamedChildCount() {
				parts = append(parts, f.expr(inner.NamedChild(i)))
			}

			inline := "throw (" + strings.Join(parts, ", ") + ");"
			indentLen := len(f.opts.indent(f.level))

			if indentLen+len(inline) > f.opts.LineWidth {
				indent := f.opts.indent(f.level + 1)
				outerIndent := f.opts.indent(f.level)
				f.iLine("throw (\n" + indent + strings.Join(parts, ",\n"+indent) + "\n" + outerIndent + ");")
				f.scriptWrite("\n")

				return
			}
		}
	}

	f.iLine("throw " + f.expr(val) + ";")
	f.scriptWrite("\n")
}

func (f *Formatter) scriptBreak(n *sitter.Node) {
	label := n.NamedChild(0)
	if label != nil {
		f.iLine("break " + f.text(label) + ";")
	} else {
		f.iLine("break;")
	}

	f.scriptWrite("\n")
}

func (f *Formatter) scriptContinue(n *sitter.Node) {
	label := n.NamedChild(0)
	if label != nil {
		f.iLine("continue " + f.text(label) + ";")
	} else {
		f.iLine("continue;")
	}

	f.scriptWrite("\n")
}

// scriptRaw re-emits an unknown node line-by-line with indentation applied.
func (f *Formatter) scriptRaw(n *sitter.Node) {
	raw := trimSpace(f.text(n))
	if raw == "" {
		return
	}

	lines := strings.Split(raw, "\n")
	prevBlank := false

	for _, line := range lines {
		trimmed := trimSpace(line)
		if trimmed == "" {
			if !prevBlank {
				f.scriptWrite("\n")
			}

			prevBlank = true

			continue
		}

		prevBlank = false

		f.writeIndent()
		f.scriptWrite(trimmed + "\n")
	}
}
