package formatter

import (
	"fmt"
	"slices"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// scriptBlockOf renders the named child at field `field` as a block.
// Falls back to scriptBlock on the node directly if field lookup fails.
// func (f *Formatter) scriptBlockOf(n *sitter.Node, field string) {
// 	body := n.ChildByFieldName(field)
// 	if body == nil {
// 		// try last child heuristic
// 		body = n.Child(n.ChildCount() - 1)
// 	}
// 	if body.Kind() == "statement_block" || body.Kind() == "block" {
// 		f.scriptBlock(body)
// 	} else {
// 		// single-statement body — still wrap in braces for canonical form
// 		f.scriptWrite(" {")
// 		f.scriptWrite("\n")
// 		f.level++
// 		f.formatScriptNode(body)
// 		f.scriptWrite("\n")
// 		f.level--
// 		f.writeIndent()
// 		f.scriptWrite("}")
// 	}
// }

// expr renders an expression node inline and returns the string.
// Expressions are never written directly; they are embedded in statements.
func (f *Formatter) expr(n *sitter.Node) string {
	if n == nil {
		return ""
	}

	if f.overBudget() {
		return f.text(n)
	}

	if n.ChildCount() == 0 || len(f.src) == 0 {
		return f.exprNode(n)
	}

	key := exprKey{src: &f.src[0], start: n.StartByte(), end: n.EndByte(), kind: n.KindId(), level: f.level, col: f.lineLen}
	if e, ok := f.exprs[key]; ok {
		if f.parseErr == nil {
			f.parseErr = e.err
		}

		return e.text
	}

	before := f.parseErr
	text := f.exprNode(n)

	var err error
	if before == nil {
		err = f.parseErr
	}

	if f.exprs == nil {
		f.exprs = map[exprKey]exprResult{}
	}

	f.exprs[key] = exprResult{text: text, err: err}

	return text
}

// exprKey identifies one rendering of an expression. A rendering is a function
// of the node, the indent level and the column the line had reached (which a
// literal reads to decide whether it fits): expressions return text rather than
// write it, so nothing else they read changes while one is being built. src
// tells a re-parsed region's source apart from the document's, and kind a node
// from a child spanning the same bytes.
//
// The cache is what keeps nesting polynomial. An argument list, a struct or an
// array that goes multi-line renders its entries twice (once to measure the
// inline form, once a level deeper to lay them out) and a pair renders its
// value twice, so each level of nesting doubled the work: a 160-byte file of
// calls nested 52 deep took 38 seconds, and format-on-save started another
// such job on every save. closureKey was the same fix for closure bodies alone.
type exprKey struct {
	src        *byte
	start, end uint
	kind       uint16
	level, col int
}

// exprResult is a cached rendering and the parse error it recorded, which a
// later use of it records in turn, as rendering the node again would.
type exprResult struct {
	text string
	err  error
}

func (f *Formatter) exprNode(n *sitter.Node) string {
	switch n.Kind() {
	// ── literals ──────────────────────────────────────────────────────────
	case "identifier", "property_identifier", "shorthand_property_identifier",
		"number", "true", "false", "null", "undefined", "this", "super":
		text := f.text(n)
		if n.Kind() == "identifier" || n.Kind() == "property_identifier" {
			text = f.applyScopeCase(text)
		}

		return text

	case "string", "template_string":
		return f.exprString(n)

	// ── operators / compound expressions ──────────────────────────────────
	case "assignment_expression":
		return f.exprAssignment(n)

	case "augmented_assignment_expression":
		left := n.ChildByFieldName("left")
		right := n.ChildByFieldName("right")
		op := f.operatorToken(n)

		return fmt.Sprintf("%s %s%s %s", f.expr(left), op, f.delimitedComments(n), f.expr(right))

	case "binary_expression":
		return f.exprBinary(n)
	case "unary_expression", "not_expression":
		return f.exprUnary(n)

	case "update_expression":
		arg := n.ChildByFieldName("argument")
		op := f.operatorToken(n)
		// prefix vs postfix
		if n.Child(0).Kind() == op || n.Child(0).Kind() == "++" || n.Child(0).Kind() == "--" {
			return fmt.Sprintf("%s%s", op, f.expr(arg))
		}

		return fmt.Sprintf("%s%s", f.expr(arg), op)

	case "ternary_expression":
		return f.exprTernary(n)

	case "query_expression":
		return f.exprQuery(n)

	case "call_expression":
		return f.exprCall(n)

	case "new_expression":
		return f.exprNew(n)

	case "member_expression":
		return f.exprMember(n)

	case "subscript_expression":
		return f.exprSubscript(n)

	case "parenthesized_expression":
		return f.exprParenthesized(n)

	case "sequence_expression":
		// comma-separated list
		var parts []string
		for i := range n.NamedChildCount() {
			parts = append(parts, f.expr(n.NamedChild(i)))
		}

		return strings.Join(parts, ", ")

	// ── spread / rest ─────────────────────────────────────────────────────
	case "spread_element":
		return "..." + f.expr(n.NamedChild(0))

	// ── array / object literals ───────────────────────────────────────────
	case "array":
		return f.exprArray(n)

	case "object":
		return f.exprObject(n)

	case "object_pattern":
		return f.exprObjectPattern(n)

	case "pair":
		return f.exprPair(f.expr(n.ChildByFieldName("key")), ":", n.ChildByFieldName("value"))

	case "cf_pair":
		return f.exprPair(f.structKey(n.ChildByFieldName("key")), " =", n.ChildByFieldName("value"))

	case "object_assignment_pattern":
		return f.exprObjectAssignmentPattern(n)

	// ── functions ─────────────────────────────────────────────────────────
	case "arrow_function":
		return f.exprArrow(n)

	case "function_expression":
		return f.exprFunctionExpr(n)

	// ── type cast (CF-specific syntax) ────────────────────────────────────
	case "type_cast_expression":
		t := n.ChildByFieldName("type")
		val := n.ChildByFieldName("value")

		return fmt.Sprintf("(%s) %s", f.text(t), f.expr(val))

	// ── fallback ─────────────────────────────────────────────────────────
	default:
		return f.text(n)
	}
}

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
