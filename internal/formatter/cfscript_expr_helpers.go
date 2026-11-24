package formatter

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// ─── helpers for expr ────────────────────────────────────────────────────────

// delimitedComments returns the text of any delimited comment sitting directly
// inside n, prefixed with a space, or "" when there is none. Rendering an
// expression from its named fields alone drops these, and commenting an operand
// out mid-expression is a common way to park it:
//
//	<cfset cols = "a," &
//	    "b," &
//	<!---    "c," & --->
//	    "d">
//
// Only delimited forms are re-emitted. A "//" comment cannot be moved onto the
// same line as the operand that follows it without swallowing it, so it is left
// for the whitespace-only guard to report.
func (f *Formatter) delimitedComments(n *sitter.Node) string {
	var sb strings.Builder

	for i := range n.ChildCount() {
		c := n.Child(i)

		switch c.Kind() {
		case "cf_comment", "block_comment":
			sb.WriteString(" ")
			sb.WriteString(trimSpace(f.text(c)))
		case "comment":
			// The document grammar gives a `/* … */` inside a <cfset> this
			// kind rather than block_comment, so matching on kind alone
			// dropped it: `<cfset x = /* why */ f()>` lost the comment while
			// the identical one written after the value survived, because that
			// one is a child of the tag rather than of the assignment. The
			// text is what distinguishes the two forms; a "//" still cannot be
			// re-emitted inline and is left to the guard.
			if text := trimSpace(f.text(c)); !strings.HasPrefix(text, "//") {
				sb.WriteString(" ")
				sb.WriteString(text)
			}
		}
	}

	return sb.String()
}

// isWordOp reports whether a prefix operator is a word, which needs a space
// before its operand. CFML is case-insensitive: matched case-sensitively, `Not x`
// was joined into `Notx`, a different identifier, and a whitespace-only change
// the guard cannot see.
func isWordOp(op string) bool {
	switch strings.ToLower(op) {
	case "typeof", "void", "delete", "not":
		return true
	}

	return false
}

// isWordOperator reports whether a binary operator is spelled as a word (`AND`,
// `eq`, `IS NOT`, `in`) rather than as symbols.
func isWordOperator(op string) bool {
	if op == "" {
		return false
	}

	c := op[0] | 0x20 // ASCII lowercase; every CFML operator word is ASCII

	return c >= 'a' && c <= 'z'
}

// operatorToken finds the operator in a binary/unary expression. The operator
// field can hold more than one token — CFML's `IS NOT` is two in
// tree-sitter-cfml after v0.26.36 — and ChildByFieldName returns only the first,
// so `a IS NOT b` read as `a IS b`. Every token in the field is joined.
func (f *Formatter) operatorToken(n *sitter.Node) string {
	var parts []string

	for i := range n.ChildCount() {
		if n.FieldNameForChild(uint32(i)) == "operator" {
			parts = append(parts, f.text(n.Child(i)))
		}
	}

	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	// Fallback: first anonymous child
	for i := range n.ChildCount() {
		c := n.Child(i)
		if !c.IsNamed() {
			return c.Kind()
		}
	}

	return ""
}

// gapOperator extracts a word operator (EQ, GT, AND, etc.) from the source
// text between left and right children of a binary_expression.
func (f *Formatter) gapOperator(_ *sitter.Node, left, right *sitter.Node) string {
	if left == nil || right == nil {
		return ""
	}

	gap := trimSpace(string(f.src[left.EndByte():right.StartByte()]))

	return gap
}

// hasChildOfKind reports whether n has a direct child of the given kind.
func hasChildOfKind(n *sitter.Node, kind string) bool {
	for i := range n.ChildCount() {
		if n.Child(i).Kind() == kind {
			return true
		}
	}

	return false
}

// mixedArgSeparators reports whether some arguments in args are separated by a
// comma and others only by whitespace. Comments are not arguments, so they
// neither need nor break a separator.
func mixedArgSeparators(args *sitter.Node) bool {
	withComma, without := false, false
	seenArg, comma := false, false

	for i := range args.ChildCount() {
		c := args.Child(i)

		switch {
		case c.Kind() == ",":
			comma = true
		case !c.IsNamed() || isCommentKind(c.Kind()):
			continue
		default:
			if seenArg {
				if comma {
					withComma = true
				} else {
					without = true
				}
			}

			seenArg, comma = true, false
		}
	}

	return withComma && without
}

// tagStyleArgs reports whether args is a script-syntax CF tag's attribute list —
// `cfdirectory(directory="x" action="create")` — rather than an ordinary argument
// list. The grammar models both as an `arguments` node holding assignment_expressions;
// what separates them is that the attribute form has no comma tokens at all. A single
// argument is ambiguous and treated as an ordinary call, since nothing is joined.
func tagStyleArgs(args *sitter.Node) bool {
	if args == nil || args.NamedChildCount() < 2 {
		return false
	}

	for i := range args.ChildCount() {
		if args.Child(i).Kind() == "," {
			return false
		}
	}

	return true
}

// childToken returns the first anonymous token child matching typ.
// assignmentOperator returns the token joining an assignment_expression's two
// sides. It is not always "=": a CF tag written in script may separate an
// attribute from its value with a colon — `cfparam (name:"local.d"
// default:"DDD")` — and the grammar gives both spellings the same node, with
// the operator as an anonymous child. The previous helper took the token it was
// asked for and returned it whether or not the node had one, so every colon
// came back as "=" and the file was refused.
func (f *Formatter) assignmentOperator(n *sitter.Node) string {
	for i := range n.ChildCount() {
		c := n.Child(i)
		if c.IsNamed() {
			continue
		}

		switch c.Kind() {
		case "=", ":":
			return c.Kind()
		}
	}

	return "="
}
