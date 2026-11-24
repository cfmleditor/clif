package formatter

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// exprNew formats a new_expression.
func (f *Formatter) exprNew(n *sitter.Node) string {
	ctor := n.ChildByFieldName("constructor")
	args := n.ChildByFieldName("arguments")

	// An inline component literal — `new component { property name="x"; function
	// f() {} }` — has neither a constructor nor an argument list: the class is
	// its body. The field-based rendering below found nothing in either field
	// and emitted `new ()`, deleting the keyword and the whole body with it
	// (16 files in the corpus, all rejected by the whitespaceOnly guard, so in
	// the editor this reads as format-on-save doing nothing).
	//
	// Emitted verbatim, in the same spirit as a function_expression's body:
	// it's a declaration rather than an expression to re-space, and rendering
	// it properly needs the statement machinery, which does not return a string.
	if ctor == nil && hasChildOfKind(n, "component_body") {
		return f.text(n)
	}

	// `new java:java.io.File(p)` — the type prefix is a single token that
	// already carries its colon, and dropping it changes which object gets
	// constructed, so it has to be reproduced verbatim.
	prefix := ""

	if p := n.ChildByFieldName("prefix"); p != nil {
		prefix = f.text(p)
	}

	return fmt.Sprintf("new %s%s%s", prefix, f.expr(ctor), f.exprArgs(args))
}

// exprMember formats a member_expression.
func (f *Formatter) exprMember(n *sitter.Node) string {
	obj := n.ChildByFieldName("object")
	prop := n.ChildByFieldName("property")
	objStr := f.expr(obj)
	propStr := f.expr(prop)
	op := memberOperator(n)

	// A comment between a chained call and its next hop belongs to no
	// field, so joining object and property dropped it.
	if obj != nil && prop != nil {
		if comments := f.commentsBetween(n, obj.EndByte(), prop.StartByte()); len(comments) > 0 {
			indent := f.opts.indent(f.level + 1)

			var b strings.Builder

			b.WriteString(objStr)

			for _, c := range comments {
				b.WriteString("\n")
				b.WriteString(indent)
				b.WriteString(c)
			}

			b.WriteString("\n")
			b.WriteString(indent)
			b.WriteString(op)
			b.WriteString(propStr)

			return b.String()
		}
	}

	inline := objStr + op + propStr
	// Break if the object part is multi-line or the last line exceeds width.
	lastLine := inline
	if _, after, ok := strings.CutLast(inline, "\n"); ok {
		lastLine = after
	}

	if len(lastLine) > f.opts.LineWidth &&
		obj != nil && (obj.Kind() == "call_expression" || obj.Kind() == "member_expression") {
		indent := f.opts.indent(f.level + 1)

		return objStr + "\n" + indent + op + propStr
	}

	return inline
}

// exprSubscript formats a subscript_expression.
func (f *Formatter) exprSubscript(n *sitter.Node) string {
	obj := n.ChildByFieldName("object")
	idx := n.ChildByFieldName("index")

	// A subscript can be reached statically — `Test::["f"]()`, the
	// subscripted form of `Test::f()` — and the grammar reports the `::` as
	// a named static_chain field, exactly as it does on a member_expression
	// (see memberOperator). Rendering the node from object and index alone
	// dropped it and turned a static call into an instance call. The
	// grammar has only parsed this form since v0.26.35 (#79), so until then
	// the file was refused rather than mis-rendered.
	accessor := ""
	if sc := n.ChildByFieldName("static_chain"); sc != nil {
		accessor = "::"
	}

	return fmt.Sprintf("%s%s[%s]", f.expr(obj), accessor, f.expr(idx))
}

// exprParenthesized formats a parenthesized_expression.
func (f *Formatter) exprParenthesized(n *sitter.Node) string {
	// Every named child is rendered, not just the first. A comment inside
	// the parens — commonly a commented-out clause parked at the end of a
	// long condition — is a named child like any other, so taking child 0
	// dropped it, or worse rendered it as the expression itself.
	var sb strings.Builder

	sb.WriteString("(" + f.opts.condPad())

	for i := range n.NamedChildCount() {
		c := n.NamedChild(i)

		if i > 0 {
			sb.WriteString(" ")
		}

		switch c.Kind() {
		case "comment", "block_comment", "cf_comment":
			text := trimSpace(f.text(c))
			sb.WriteString(text)

			// A "//" comment runs to end of line, so without a break it
			// would swallow the rest of the condition and the ")".
			if strings.HasPrefix(text, "//") {
				sb.WriteString("\n")
			}
		default:
			sb.WriteString(f.expr(c))
		}
	}

	sb.WriteString(f.opts.condPad() + ")")

	return sb.String()
}

// exprObjectPattern formats a object_pattern.
func (f *Formatter) exprObjectPattern(n *sitter.Node) string {
	// A struct literal written with `=` (`{ a = 1, "b" = 2 }`) parses as an
	// object_pattern, because the grammar shares the rule with JavaScript
	// destructuring. It used to fall through to the verbatim default, so
	// the most common struct spelling in CFML was never formatted at all.
	// Anything that is not a plain key/value list keeps its source text.
	if !isStructPattern(n) {
		return f.text(n)
	}

	return f.exprObject(n)
}

// exprObjectAssignmentPattern formats a object_assignment_pattern.
func (f *Formatter) exprObjectAssignmentPattern(n *sitter.Node) string {
	// `{ a = 1 }`: the key is a bare name, so the grammar reads it as a
	// destructuring default rather than a cf_pair. Only that shape reaches
	// here; isStructPattern sends every other one to the verbatim default.
	left := n.ChildByFieldName("left")
	if left == nil {
		return f.text(n)
	}

	return f.exprPair(f.text(left), " =", n.ChildByFieldName("right"))
}
