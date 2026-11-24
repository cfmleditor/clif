package formatter

import (
	"fmt"
	"slices"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// exprAssignment formats a assignment_expression.
func (f *Formatter) exprAssignment(n *sitter.Node) string {
	left := n.ChildByFieldName("left")
	right := n.ChildByFieldName("right")
	op := f.assignmentOperator(n)

	for i := range n.ChildCount() {
		c := n.Child(i)
		if !c.IsNamed() && c.Kind() != "=" {
			t := c.Kind()
			if strings.HasSuffix(t, "=") {
				op = t
			}
		}
	}

	leftStr := f.expr(left)
	rightStr := f.expr(right)
	cmts := f.delimitedComments(n)

	// A colon binds to the name it follows, the way a struct literal's
	// `key: value` does; only "=" takes a space on both sides.
	sep := " " + op
	if op == ":" {
		sep = op
	}

	result := fmt.Sprintf("%s%s%s %s", leftStr, sep, cmts, rightStr)

	if len(result) > f.opts.LineWidth && !strings.Contains(rightStr, "\n") {
		f.level++
		rightStr = f.expr(right)
		f.level--

		if strings.Contains(rightStr, "\n") {
			indent := f.opts.indent(f.level + 1)

			return fmt.Sprintf("%s%s%s\n%s%s", leftStr, sep, cmts, indent, rightStr)
		}
	}

	return result
}

// exprBinary formats a binary_expression.
func (f *Formatter) exprBinary(n *sitter.Node) string {
	left := n.ChildByFieldName("left")
	right := n.ChildByFieldName("right")
	op := f.operatorToken(n)
	cmts := f.delimitedComments(n)

	// A word operator — `AND`, `EQ`, `IS NOT`, `DOES NOT CONTAIN` — is
	// lifted from the source. Until tree-sitter-cfml v0.26.36 the grammar
	// hid those tokens, so op was empty and this was the only path they
	// took; later releases expose them, and following the symbolic
	// operators' path instead would move a block comment written before
	// the operator to after it, which the guard rejects, and reproduce a
	// condition with a line comment in it verbatim rather than formatted.
	if op == "" || isWordOperator(op) {
		// gapOperator lifts the raw source between the operands, so
		// anything sitting in that gap — comments included — is already
		// carried across. Adding them again would emit them twice.
		op = f.gapOperator(n, left, right)
		cmts = ""
	}

	joined := fmt.Sprintf("%s %s%s %s", f.expr(left), op, cmts, f.expr(right))

	// A `//` comment between the operands is neither the left nor the
	// right field, and delimitedComments deliberately refuses to re-emit
	// one inline: it runs to end of line, so putting it back between the
	// operator and the right operand comments the rest of the expression
	// out. Dropping it was left to the guard — which is what rejected
	// ColdBox's Router.cfc, where `) & // multi-host` lost its comment
	// outright. Reproduce the expression as written instead, the same
	// fallback ternary_expression takes for a comment parked before its
	// `:`.
	//
	// A gap lifted by gapOperator needs the check too. Nothing in it was
	// dropped, but it was trimmed and the right operand joined on after it,
	// so a `//` comment written on its own line between `OR` and the next
	// clause — how WireBox's Binder.cfc annotates each clause of a filter —
	// came out as `OR // why ( clause )`, with the clause inside the
	// comment. Its only occurrence in the corpus is in a closure body, which
	// was copied verbatim until closures were formatted.
	if !f.keptLineComments(n, joined) {
		return f.text(n)
	}

	return joined

	// not_expression is `!` and `NOT` in tree-sitter-cfml after v0.26.36, which
	// split them out of unary_expression to give them CFML's precedence, looser
	// than the comparisons. Its operator is a not_operator node rather than a
	// token, but its text is the same.
}

// exprUnary formats a unary_expression.
func (f *Formatter) exprUnary(n *sitter.Node) string {
	op := n.ChildByFieldName("operator")
	arg := n.ChildByFieldName("argument")

	if op == nil {
		op = n.Child(0)
	}

	opStr := f.text(op)
	// word operators need a space: typeof, void, delete, not
	if isWordOp(opStr) {
		return fmt.Sprintf("%s %s", opStr, f.expr(arg))
	}

	return fmt.Sprintf("%s%s", opStr, f.expr(arg))
}

// exprTernary formats a ternary_expression.
func (f *Formatter) exprTernary(n *sitter.Node) string {
	cond := n.ChildByFieldName("condition")
	cons := n.ChildByFieldName("consequence")
	alt := n.ChildByFieldName("alternative")
	condStr := f.expr(cond)
	consStr := f.expr(cons)
	altStr := f.expr(alt)

	inline := fmt.Sprintf("%s ? %s : %s", condStr, consStr, altStr)

	// A `//` comment parked before the `:` — a common way to say why the
	// alternative is what it is — belongs to none of the three fields above
	// and was dropped outright. Reproduce the expression as written rather
	// than lose it; a line comment could not be re-inlined into either
	// branch anyway, since it runs to end of line.
	if !f.keptLineComments(n, inline) {
		return f.text(n)
	}

	if len(inline) > f.opts.LineWidth {
		indent := f.opts.indent(f.level + 1)

		return condStr + "\n" + indent + "? " + consStr + "\n" + indent + ": " + altStr
	}

	return inline
}

// exprCall formats a call_expression.
func (f *Formatter) exprCall(n *sitter.Node) string {
	fn := n.ChildByFieldName("function")
	args := n.ChildByFieldName("arguments")
	fnStr := f.expr(fn)
	// If fn has a chain break, evaluate args at deeper level.
	chainBroken := endsInChainBreak(fnStr)
	if chainBroken {
		f.level++
	}

	argsStr := f.exprArgs(args)
	result := fnStr + argsStr
	// If the full call exceeds line width and args are inline, split args.
	if !strings.Contains(argsStr, "\n") && len(result) > f.opts.LineWidth && args != nil && args.NamedChildCount() > 0 {
		argsStr = f.brokenCallArgs(args)
	}

	if chainBroken {
		f.level--
	}

	return fnStr + argsStr
}

// brokenCallArgs writes a call's arguments one per line, for a call too long
// to fit, with commas where CommaPosition puts them and none after a comment.
func (f *Formatter) brokenCallArgs(args *sitter.Node) string {
	f.level++

	count := args.NamedChildCount()
	parts := make([]string, 0, count)
	isComment := make([]bool, 0, count)

	for i := range count {
		c := args.NamedChild(i)
		parts = append(parts, f.expr(c))
		isComment = append(isComment, c.Kind() == "cf_comment")
	}

	indent := f.opts.indent(f.level)
	f.level--
	outerIndent := f.opts.indent(f.level)

	var sb strings.Builder

	sb.WriteString("(\n")

	leading := f.opts.CommaPosition == "before"

	for i, p := range parts {
		sb.WriteString(indent)

		switch {
		case isComment[i]:
			sb.WriteString(p)
		case leading:
			if anyArgument(isComment[:i]) {
				sb.WriteString(", ")
			}

			sb.WriteString(p)
		default:
			sb.WriteString(p)

			if anyArgument(isComment[i+1:]) {
				sb.WriteString(",")
			}
		}

		sb.WriteString("\n")
	}

	sb.WriteString(outerIndent)
	sb.WriteByte(')')

	return sb.String()
}

// anyArgument reports whether any entry is an argument rather than a comment.
func anyArgument(isComment []bool) bool {
	return slices.Contains(isComment, false)
}
