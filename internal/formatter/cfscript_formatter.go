package formatter

// cfscript_formatter.go — recursive pretty-printer for the cfscript sub-grammar.

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// ─── entry point ─────────────────────────────────────────────────────────────

// formatScriptNode dispatches a cfscript statement node and writes output.
// It is called for every direct child of a <cfscript> block or a block body.
func (f *Formatter) formatScriptNode(n *sitter.Node) {
	switch n.Kind() {
	// ── trivial whitespace tokens emitted by the grammar ──────────────────
	case "\n", "\r\n", "\r":
		// handled by caller's blank-line logic; skip
		return

	// ── comments ──────────────────────────────────────────────────────────
	case "comment":
		f.scriptLineComment(n)
	case "block_comment":
		f.scriptBlockComment(n)

	// ── declarations ──────────────────────────────────────────────────────
	case "component_declaration", "component":
		f.scriptComponent(n)
	case "function_definition", "function_declaration", "method_definition":
		f.scriptFunction(n)
	case "property_declaration":
		f.scriptProperty(n)
	case "variable_declaration":
		f.scriptVarDecl(n)

	// ── statements ────────────────────────────────────────────────────────
	case "expression_statement":
		f.scriptExprStmt(n)
	case "return_statement":
		f.scriptReturn(n)
	case "throw_statement":
		f.scriptThrow(n)
	case "break_statement":
		f.scriptBreak(n)
	case "continue_statement":
		f.scriptContinue(n)
	case "if_statement":
		f.scriptIf(n)
	case "switch_statement":
		f.scriptSwitch(n)
	case "while_statement":
		f.scriptWhile(n)
	case "do_statement":
		f.scriptDo(n)
	case "for_statement":
		f.scriptFor(n)
	case "for_in_statement", "for_of_statement":
		f.scriptForIn(n)
	case "try_statement":
		f.scriptTry(n)
	case "import_statement":
		f.scriptPassthru(n)
	case "tag_statement", "query_tag":
		f.scriptTagStatement(n)

	// ── block (anonymous body) ─────────────────────────────────────────────
	case "statement_block", "block":
		// A bare block `{ ... }` not attached to anything.
		f.scriptBareBlock(n)

	// ── fallback: unknown node → emit raw, re-indented ────────────────────
	default:
		f.scriptRaw(n)
	}
}

// ─── helpers shared by all script formatters ─────────────────────────────────

// endsInChainBreak reports whether a rendered callee ends on a line of its own
// that starts with the member accessor — `\n\t.catch` — which is what the
// member_expression renderer writes when it breaks a chain. The arguments of
// that call then belong one level deeper, under the accessor.
//
// Any newline used to count, but a closure earlier in the chain puts newlines
// in the callee without breaking it: in `p.then(function(r) { … }).catch(…)`
// the `.catch` sits on the `})` line, and its callback was indented a level
// deeper than the `then` one beside it.
func endsInChainBreak(callee string) bool {
	_, after, ok := strings.CutLast(callee, "\n")
	if !ok {
		return false
	}

	last := strings.TrimLeft(after, " \t")

	return strings.HasPrefix(last, ".") || strings.HasPrefix(last, "?.") || strings.HasPrefix(last, "::")
}

// memberOperator returns the accessor token joining a member_expression's
// object and property. It is not always ".": Lucee and BoxLang spell static
// access `Widget::getData()`, and rendering that as `Widget.getData()` turns a
// static call into an instance call.
func memberOperator(n *sitter.Node) string {
	// `::` is reported as a named `static_chain` node, not an anonymous token.
	if sc := n.ChildByFieldName("static_chain"); sc != nil {
		return "::"
	}

	// Likewise `?.` is a named `optional_chain` node wrapping the anonymous
	// token, not a bare "?." child — the loop below skips every named child,
	// so without this check it fell through to the default "." and silently
	// turned a null-safe chain into one that throws.
	if oc := n.ChildByFieldName("optional_chain"); oc != nil {
		return "?."
	}

	for i := range n.ChildCount() {
		c := n.Child(i)
		if c.IsNamed() {
			continue
		}

		switch c.Kind() {
		case ".", "::", "?.":
			return c.Kind()
		}
	}

	return "."
}

// writeInterveningComments emits any comment child of n lying strictly between
// byte offsets from and to, each on its own line. Comments in these positions —
// between a block and its `else`, or between `try` and its `catch` — belong to
// no field, so navigating to the continuation by field name skipped straight
// past them and deleted them. Reports whether anything was written, which tells
// the caller the continuation keyword can no longer sit on the closing brace.
func (f *Formatter) writeInterveningComments(n *sitter.Node, from, to uint) bool {
	wrote := false

	for i := range n.ChildCount() {
		c := n.Child(i)

		switch c.Kind() {
		case "comment", "block_comment":
		default:
			continue
		}

		if c.StartByte() < from || c.EndByte() > to {
			continue
		}

		f.scriptWrite("\n")
		f.writeIndent()
		f.scriptWrite(trimSpace(f.text(c)))

		wrote = true
	}

	return wrote
}

// deferBlockComments queues the comments of n lying strictly between from and
// to for emission inside the next block that opens. A comment between a
// condition and an Allman-style brace belongs to no field, so navigating from
// the condition straight to the body skipped past it and deleted it. It cannot
// be emitted where it sits either: a "//" comment printed before the "{" would
// swallow the brace, so it is carried into the body instead.
func (f *Formatter) deferBlockComments(n, from, to *sitter.Node) {
	if n == nil || from == nil || to == nil {
		return
	}

	for i := range n.ChildCount() {
		c := n.Child(i)

		switch c.Kind() {
		case "comment", "block_comment", "cf_comment":
		default:
			continue
		}

		if c.StartByte() >= from.EndByte() && c.EndByte() <= to.StartByte() {
			f.pendingBlockComments = append(f.pendingBlockComments, c)
		}
	}
}

// elseBody returns the else clause's body, queueing any comments that precede
// it for emission inside that body. Such a comment is a named child like any
// other, so taking the first named child picked the comment as the body and
// rendered it in the body's place:
//
//	else //we have summary
//	{ ... }
//
// It cannot be left where it sits either — a "//" comment printed before the
// "{" swallows the brace.
func (f *Formatter) elseBody(alt *sitter.Node) *sitter.Node {
	var lead []*sitter.Node

	for i := range alt.NamedChildCount() {
		c := alt.NamedChild(i)

		switch c.Kind() {
		case "comment", "block_comment", "cf_comment":
			lead = append(lead, c)

			continue
		}

		f.pendingBlockComments = append(f.pendingBlockComments, lead...)

		return c
	}

	return nil
}

// flushBlockComments emits any comments queued by deferBlockComments. It is
// called just inside an opened block, so they land at the body's indentation
// exactly as a comment written there would — which is where a second format
// pass finds them, keeping the output stable.
func (f *Formatter) flushBlockComments() bool {
	pending := f.pendingBlockComments
	f.pendingBlockComments = nil

	for _, c := range pending {
		f.formatScriptNode(c)
	}

	return len(pending) > 0
}

// scriptWrite writes s, prepending indentation if we are at the beginning of
// a line. This is the primary output primitive for script formatting.
func (f *Formatter) scriptWrite(s string) { f.write(s) }
func (f *Formatter) scriptNL()            { f.nl() }

// keptLineComments reports whether rendered treats every `//` comment in n's
// subtree the way a line comment has to be treated: still present, and still
// the last thing on its line.
//
// Both halves are needed, and both come from the same cause. An expression is
// rebuilt from its named fields, and a comment sitting between two of them is
// not one of those fields — so depending on the shape it is either dropped, or
// kept and the operand that followed it in the source is folded up onto its
// line, where the comment swallows it. The second is the more dangerous of the
// two, because only whitespace changed and a character-level comparison cannot
// see it.
//
// Which shapes are safe is not worth enumerating: `a && b` loses its comments
// where `a or b` keeps them, and a nested parenthesised operand folds where a
// flat one does not. Checking the result against the source catches all of
// them, including the ones nobody has hit yet. Callers reproduce the source
// when this returns false.
func (f *Formatter) keptLineComments(n *sitter.Node, rendered string) bool {
	for _, c := range f.lineComments(n) {
		_, after, ok := strings.Cut(rendered, c)
		if !ok {
			return false
		}

		rest := after
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
			rest = rest[:nl]
		}

		if trimSpace(rest) != "" {
			return false
		}
	}

	return true
}

// lineComments returns the text of every `//` comment in n's subtree, so a
// rendering can be checked for the ones it dropped.
func (f *Formatter) lineComments(n *sitter.Node) []string {
	if n == nil {
		return nil
	}

	var out []string

	if f.isLineCommentNode(n) {
		return append(out, trimSpace(f.text(n)))
	}

	for i := range n.ChildCount() {
		out = append(out, f.lineComments(n.Child(i))...)
	}

	return out
}

// containsLineComment reports whether n's subtree holds a `//` comment. Only the
// line form counts: a delimited comment can be moved and re-inlined, while a
// line comment runs to end of line and takes whatever follows it with it.
func (f *Formatter) containsLineComment(n *sitter.Node) bool {
	if n == nil {
		return false
	}

	if f.isLineCommentNode(n) {
		return true
	}

	for i := range n.ChildCount() {
		if f.containsLineComment(n.Child(i)) {
			return true
		}
	}

	return false
}

// isLineCommentNode reports whether n is a `//` comment.
//
// The kind alone does not say: the document grammar reports a `/* … */` inside
// a <cfset> as a plain "comment" too, the same kind it gives "//". Only the
// line form is unmovable — a delimited comment can be re-emitted inline, and
// treating one as unmovable would send perfectly ordinary expressions down the
// reproduce-verbatim path for no reason.
func (f *Formatter) isLineCommentNode(n *sitter.Node) bool {
	if n.Kind() != "comment" {
		return false
	}

	return strings.HasPrefix(trimSpace(f.text(n)), "//")
}

// parenExpr renders an expression wrapped in parens, avoiding double-wrapping
// when the node is already a parenthesized_expression.
// Long conditions are broken at logical operators.
func (f *Formatter) parenExpr(n *sitter.Node) string {
	var inner string
	if n != nil && n.Kind() == "parenthesized_expression" {
		inner = f.expr(n)
	} else {
		inner = "(" + f.opts.condPad() + f.expr(n) + f.opts.condPad() + ")"
	}

	verbatim := func() string {
		if n.Kind() == "parenthesized_expression" {
			return f.text(n)
		}

		return "(" + f.opts.condPad() + f.text(n) + f.opts.condPad() + ")"
	}

	// A `//` comment between two operands is neither the left nor the right
	// field of the binary_expression holding them, so rebuilding the condition
	// from those fields drops it — silently, and only for some shapes, which is
	// why a condition written with `or` keeps its comments while the same
	// condition written with `&&` loses all but the last. Rather than guess
	// which shapes are safe, the rendered condition is checked against the
	// source's own comments and reproduced as written when any went missing.
	if !f.keptLineComments(n, inner) {
		return verbatim()
	}

	// If the condition is too long, break at logical operators.
	// Use the larger of lineLen or indent estimate for the check.
	col := f.lineLen
	if col == 0 {
		col = len(f.opts.indent(f.level))
	}

	// A condition too long for the line is reflowed at its logical operators,
	// which keeps each surviving `//` comment at the end of the line it was
	// written on.
	if col+len(inner) > f.opts.LineWidth {
		return f.normalizeCond(inner)
	}

	// A short condition is emitted on one line, which a `//` comment makes
	// unsafe however well it survived above: everything after it on that line
	// becomes part of the comment, the closing paren included. A line comment is
	// only safe where the source already put it — the same reasoning that gives
	// collection literals their hasLineComment rule.
	if f.containsLineComment(n) {
		return verbatim()
	}

	return inner
}

// iLine emits one indented line inside a script context.
func (f *Formatter) iLine(s string) {
	f.scriptNL()
	f.writeIndent()
	f.scriptWrite(s)
}

// scriptChildren iterates named statement-level children and dispatches each.
//
// leadEmitted says whether something was already written into this body — a
// comment carried in from the construct's header by flushBlockComments. That
// comment is a sibling of the statements that follow and has to count as one:
// on the first format it was emitted outside this loop, so the statement after
// it saw no predecessor and got no blank line, while on the second format the
// comment had become an ordinary child and the blank line appeared. The file
// alternated between the two.
func (f *Formatter) scriptChildren(n *sitter.Node, leadEmitted bool) {
	prevWasBlock := false
	prevWasMultiLine := false
	prevWasNamed := leadEmitted
	prevEndRow := int(n.StartPosition().Row)

	for i := range n.ChildCount() {
		c := n.Child(i)
		if !c.IsNamed() {
			continue
		}
		// Insert blank line after a block/multi-line statement, before a block, or when the source had one.
		startRow := int(c.StartPosition().Row)

		sourceGap := startRow-prevEndRow > 1
		if prevWasBlock || prevWasMultiLine || (prevWasNamed && isScriptBlockStmt(c)) || (prevWasNamed && sourceGap) {
			f.scriptWrite("\n")
		}

		outBefore := f.out.Len()
		f.formatScriptNode(c)

		prevWasBlock = isScriptBlockStmt(c)
		// Judge "was multi-line" from what was actually emitted, not from the
		// node's span in the source. The formatter expands single-line
		// statements into blocks (`function f() {}` becomes a braced body), so
		// reading the source made the separating blank line appear only once
		// the file had already been formatted — the output was not idempotent.
		//
		// The emitted text is trimmed first: every statement starts by opening
		// a new line, and counting that leading newline would class all of them
		// as multi-line and insert a blank line between every pair.
		prevWasMultiLine = false
		if f.out.Len() > outBefore {
			prevWasMultiLine = bytes.Contains(trimSpaceBytes(f.out.Bytes()[outBefore:]), []byte{'\n'})
		}

		prevWasNamed = true
		prevEndRow = int(c.EndPosition().Row)
	}
}

func isScriptBlockStmt(n *sitter.Node) bool {
	switch n.Kind() {
	case "if_statement", "for_statement", "for_in_statement", "for_of_statement",
		"while_statement", "do_statement", "switch_statement", "try_statement":
		return true
	}

	return false
}

// writeOpenBrace writes a block's opening brace: after beforeBrace on the line
// just written, or — under braceStyle "next-line" — alone on the next line at
// the construct's own indent. The caller has not yet incremented f.level, so
// the brace lines up with the header rather than with the body.
//
// An empty beforeBrace means the caller has already placed the brace at the
// start of a fresh line and had no choice about it: a header ending in a `//`
// annotation comment would otherwise swallow the brace. That is already
// next-line, so both styles leave it alone — adding a second newline would put
// a blank line between the header and the brace.
func (f *Formatter) writeOpenBrace(beforeBrace string) {
	if beforeBrace != "" && f.opts.nextLineBraces() {
		f.scriptWrite("\n")
		f.writeIndent()
		f.scriptWrite("{")

		return
	}

	f.scriptWrite(beforeBrace + "{")
}

// attachedKeyword introduces a clause that continues a statement after a
// closing brace — `else`, `catch`, `finally`. It follows the brace on the same
// line normally; under braceStyle "next-line" it starts its own line, which is
// what lets that clause's own brace sit under the keyword instead of dangling
// off the end of `} else`.
//
// `while` in a do/while is deliberately not one of these: it terminates the
// statement rather than introducing a block, so there is no brace for it to
// line up with and it stays attached in both styles.
func (f *Formatter) attachedKeyword(keyword string) string {
	if f.opts.nextLineBraces() {
		f.scriptWrite("\n")
		f.writeIndent()

		return keyword
	}

	return " " + keyword
}

// scriptBlock renders a `{ ... }` block, indenting its contents.
func (f *Formatter) scriptBlock(n *sitter.Node) {
	f.scriptBlockWith(n, " ")
}

// scriptBlockWith renders a block, writing beforeBrace before the opening
// brace. Every caller wants a space there — the brace follows the construct's
// header on the same line — except a header ending in a `//` annotation
// comment, which has already had to break the line and needs the brace at the
// start of the next one rather than one column into it.
func (f *Formatter) scriptBlockWith(n *sitter.Node, beforeBrace string) {
	f.writeOpenBrace(beforeBrace)
	f.scriptBlockBody(n)
}

// scriptBareBlock renders a `{ ... }` that is a statement in its own right
// rather than some construct's body — the braced body of a `case`, or a block
// the grammar could not attach to anything.
//
// Being a statement, it opens its own line and closes with a newline, like
// every other statement renderer. It used to do neither: it wrote " {" wherever
// the cursor happened to be and left it there, so `case 1: { ... }` came out as
// a bare `{` in column one under the label, and the next thing written — the
// switch's own closing brace — landed on the same line as the block's, `}}`.
// The result is whitespace-only and idempotent, so the guard passed it and the
// corpus counted the file clean.
//
// braceStyle does not reach it: "next-line" means "under the header", and this
// block has no header to go under.
//
// Moving it anyway was not merely cosmetic. It made formatting non-idempotent
// on Lucee's one-word `elseif`, as spelled in its own Query.cfc: the grammar
// has no such keyword, so it reads `elseif ( ... )` as an ordinary call and
// leaves the block that follows standing alone. The newline the brace gained on
// the first pass became a real source line break, which the second pass padded
// again.
func (f *Formatter) scriptBareBlock(n *sitter.Node) {
	f.iLine("{")
	f.scriptBlockBody(n)
	f.scriptWrite("\n")
}

// scriptBlockBody writes everything after a block's opening brace: the padded,
// indented contents and the closing brace back at the outer level.
func (f *Formatter) scriptBlockBody(n *sitter.Node) {
	f.blockPadOpen()

	f.level++
	lead := f.flushBlockComments()
	f.scriptChildren(n, lead)
	f.blockPadClose()

	f.level--
	f.writeIndent()
	f.scriptWrite("}")
}

// blockPadOpen ends the line a block's opening brace is on, and writes the
// blank line under it that the formatter has always put there.
// blankLinesInBlocks false keeps the line break and drops the blank.
func (f *Formatter) blockPadOpen() {
	if f.opts.BlankLinesInBlocks {
		f.scriptWrite("\n\n")

		return
	}

	f.scriptWrite("\n")
}

// blockPadClose is the same at the other end: a newline that becomes the blank
// line above the closing brace, the last statement having already ended its own
// line.
//
// Unpadded it has to be scriptNL rather than a plain newline. Not every body
// leaves the cursor at the start of a line — a brace-less clause is emitted
// verbatim and stops where its text does — and there this newline is what puts
// the closing brace on a line of its own instead of a blank line.
func (f *Formatter) blockPadClose() {
	if f.opts.BlankLinesInBlocks {
		f.scriptWrite("\n")

		return
	}

	f.scriptNL()
}

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

// exprQuery renders a `queryExecute(...)` call.
//
// The grammar gives this one call a node of its own — `query_expression`, with
// the SQL as a `query_text` child between two quote tokens — rather than the
// `call_expression`/`arguments` shape every other call has. With no case for it
// the expression fell to expr's default arm, `return f.text(n)`, and was
// emitted verbatim: a queryExecute call was never formatted at all. Its
// arguments kept whatever indentation the source had, none of parenSpacing,
// commaPosition or indentWidth reached them, and where the source had no
// indentation the output had none either. 137 of the 5,624 corpus files
// contain one; in script-syntax CFML this is the replacement for `<cfquery>`.
//
// The SQL is re-emitted exactly as written, quotes included. It is a string
// literal, so its interior — including the line breaks of a multi-line query —
// is content rather than layout, and re-indenting it would change what the
// query says. That is also why a query holding a newline always breaks the
// argument list: there is no folding it back onto one line.
//
// The node only ever covers the positional form with a double-quoted first
// argument. `queryExecute(sql = "...")` and a single-quoted query are ordinary
// call_expressions and were already formatted.
func (f *Formatter) exprQuery(n *sitter.Node) string {
	callee, parts, ok := f.queryParts(n)
	if !ok {
		return f.text(n)
	}

	multiline := false

	for _, p := range parts {
		if strings.Contains(p, "\n") {
			multiline = true

			break
		}
	}

	inline := callee + "(" + f.padded(strings.Join(parts, ", ")) + ")"
	if !multiline && len(parts) <= 3 && len(inline) <= f.opts.LineWidth {
		return inline
	}

	// Re-render at the deeper level, as exprArgs does, so a nested split
	// indents against the position the argument will actually occupy.
	f.level++
	_, parts, ok = f.queryParts(n)
	indent := f.opts.indent(f.level)
	f.level--

	if !ok {
		return f.text(n)
	}

	var sb strings.Builder

	sb.WriteString(callee)
	sb.WriteString("(\n")

	leading := f.opts.CommaPosition == "before"

	for i, p := range parts {
		sb.WriteString(indent)

		if leading && i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString(p)

		if !leading && i < len(parts)-1 {
			sb.WriteString(",")
		}

		sb.WriteString("\n")
	}

	sb.WriteString(f.opts.indent(f.level))
	sb.WriteString(")")

	return sb.String()
}

// queryParts splits a query_expression into its callee and rendered arguments,
// reporting false for any shape this renderer should not touch.
//
// The SQL literal arrives as three children — an opening quote, the query_text,
// and a closing quote — so it is taken as the source span from one quote to the
// other rather than rendered. A comment among the arguments returns false: it
// has nowhere to go in a rendered list and would swallow whatever followed it
// on the line, and reproducing the call as written is what happened before this
// renderer existed.
//
// The SQL may be concatenated — `"SELECT …" & ( sort ? " ORDER BY x" : "" )` —
// and the operand after each `&` belongs to the same argument. The `&` has no
// field and is not named, so it used to fall through every case below, and the
// operands came out as separate arguments: `"SELECT …", ( … )`, which the
// guard refuses.
func (f *Formatter) queryParts(n *sitter.Node) (callee string, parts []string, ok bool) {
	var open *sitter.Node

	concat := false
	add := func(p string) {
		if concat && len(parts) > 0 {
			parts[len(parts)-1] += " & " + p
		} else {
			parts = append(parts, p)
		}

		concat = false
	}

	for i := range n.ChildCount() {
		c := n.Child(i)

		switch {
		case isCommentKind(c.Kind()):
			return "", nil, false
		case callee == "":
			callee = f.text(c)
		case c.Kind() == "(" || c.Kind() == ")" || c.Kind() == ",":
			continue
		case c.Kind() == "&":
			concat = true
		case c.Kind() == `"` || c.Kind() == "'":
			if open == nil {
				open = c

				continue
			}

			add(string(f.src[open.StartByte():c.EndByte()]))
			open = nil
		case open != nil:
			// query_text, already covered by the span its quotes delimit.
			continue
		case c.IsNamed():
			add(f.expr(c))
		}
	}

	// An unterminated literal means the walk above did not see the shape it
	// expects; emitting the pieces it did collect would drop the rest.
	if open != nil || callee == "" || len(parts) == 0 {
		return "", nil, false
	}

	return callee, parts, true
}

func (f *Formatter) exprArgs(args *sitter.Node) string {
	if args == nil {
		return "()"
	}

	// Lucee accepts a tag call whose attributes are separated partly by commas
	// and partly by spaces — `cflog(file="#name#" text="load test", type="error")`
	// in its own LDEV4128 test. Joining with either separator changes the other
	// gaps, which the guard rejects, so such a list is kept as written.
	if mixedArgSeparators(args) {
		return f.text(args)
	}

	var parts []string

	var isComment []bool

	hasLineComment := false

	for i := range args.NamedChildCount() {
		c := args.NamedChild(i)
		parts = append(parts, f.expr(c))
		isComment = append(isComment, isCommentKind(c.Kind()))

		if c.Kind() == "comment" {
			hasLineComment = true
		}
	}

	// A script-syntax CF tag call separates its attributes with spaces, not commas
	// (`cfdirectory(directory="#dir#" action="create")`), and the grammar hands both
	// forms over as an `arguments` node full of assignment_expressions. Joining with
	// ", " regardless inserted commas that were never in the source — a non-whitespace
	// change, so the guard rejected the file and format-on-save did nothing.
	useCommas := !tagStyleArgs(args)

	sep := ", "
	if !useCommas {
		sep = " "
	}

	// Space-separated tag-style attributes have no comma to preserve.
	trailing := useCommas && hasTrailingComma(args)

	inlineJoined := strings.Join(parts, sep)
	// Only when the final entry is the final argument: a comma written after a
	// trailing comment lands inside it, or turns it into an empty argument.
	if last := lastArgument(isComment); trailing && last >= 0 && last == len(parts)-1 {
		inlineJoined += ","
	}

	inline := "(" + f.padded(inlineJoined) + ")"
	// Break onto separate lines if >3 arguments or inline exceeds line width.
	// A line comment forces the break unconditionally: joined inline it runs to
	// end of line and comments out every argument after it.
	shouldBreak := hasLineComment || len(parts) > 3 ||
		(len(parts) > 0 && len(inline) > f.opts.LineWidth)
	if shouldBreak && !hasLineComment && len(parts) <= 3 && f.closuresHug(args, parts, inline) {
		return inline
	}

	if shouldBreak {
		return f.brokenArgs(args, isComment, useCommas, trailing)
	}

	return inline
}

// brokenArgs writes an argument list one argument per line, re-formatting
// each at the deeper level so nested splits indent correctly.
func (f *Formatter) brokenArgs(args *sitter.Node, isComment []bool, useCommas, trailing bool) string {
	f.level++

	parts := make([]string, 0, args.NamedChildCount())
	for i := range args.NamedChildCount() {
		parts = append(parts, f.expr(args.NamedChild(i)))
	}

	indent := f.opts.indent(f.level)
	f.level--
	outerIndent := f.opts.indent(f.level)

	var sb strings.Builder

	sb.WriteString("(\n")

	// Space-separated attributes carry no separator to place, so neither comma
	// position applies to them.
	if !useCommas {
		for _, p := range parts {
			sb.WriteString(indent)
			sb.WriteString(p)
			sb.WriteString("\n")
		}

		sb.WriteString(outerIndent)
		sb.WriteByte(')')

		return sb.String()
	}

	leading := f.opts.CommaPosition == "before"
	last := lastArgument(isComment)

	for i, p := range parts {
		sb.WriteString(indent)

		if leading {
			if !isComment[i] && anyArgument(isComment[:i]) {
				sb.WriteString(", ")
			}

			sb.WriteString(p)

			if trailing && !isComment[i] && i == last {
				sb.WriteString(",")
			}
		} else {
			sb.WriteString(p)

			if !isComment[i] && (anyArgument(isComment[i+1:]) || trailing) {
				sb.WriteString(",")
			}
		}

		sb.WriteString("\n")
	}

	sb.WriteString(outerIndent)
	sb.WriteByte(')')

	return sb.String()
}

// joinSignatureAttrs lays out a function declaration's annotations — the
// `cache="true" cacheTimeout="10"` a ColdBox handler carries between its
// parameter list and its body.
//
// They are normally joined with a space. That is wrong the moment one of them
// is a `//` comment, which CFML allows between annotations and which ColdBox's
// own test handlers use to say what each one is for: joined onto one line, the
// first comment swallows every annotation after it *and* the brace that opens
// the body, leaving code that no longer parses. A comment therefore ends its
// line, and the annotations that follow continue on the next.
func joinSignatureAttrs(attrs []string, indent string) string {
	var b strings.Builder

	for i, a := range attrs {
		if i > 0 {
			if isLineCommentText(attrs[i-1]) {
				b.WriteString("\n")
				b.WriteString(indent)
			} else {
				b.WriteString(" ")
			}
		}

		b.WriteString(a)
	}

	return b.String()
}

// isLineCommentText reports whether s is a `//` comment, which runs to the end
// of the line it is written on and so cannot have anything placed after it.
func isLineCommentText(s string) bool {
	return strings.HasPrefix(trimSpace(s), "//")
}

// lastArgument returns the index of the final non-comment entry, or -1 when
// every entry is a comment. A trailing comma belongs after that entry, not
// after a comment that happens to follow it.
func lastArgument(isComment []bool) int {
	for i, comment := range slices.Backward(isComment) {
		if !comment {
			return i
		}
	}

	return -1
}

// commentsBetween returns the text of any comment child of n lying strictly
// between byte offsets from and to.
func (f *Formatter) commentsBetween(n *sitter.Node, from, to uint) []string {
	var out []string

	for i := range n.ChildCount() {
		c := n.Child(i)
		if !isCommentKind(c.Kind()) {
			continue
		}

		if c.StartByte() < from || c.EndByte() > to {
			continue
		}

		out = append(out, trimSpace(f.text(c)))
	}

	return out
}

// isCommentKind reports whether a node kind is one of the comment forms that
// can appear among a call's arguments. Only cf_comment used to be recognised,
// so a cfscript `//` comment was given a trailing comma and swallowed the
// arguments that followed it.
func isCommentKind(kind string) bool {
	switch kind {
	case "comment", "block_comment", "cf_comment":
		return true
	}

	return false
}

func (f *Formatter) exprString(n *sitter.Node) string {
	// Return the raw text; the grammar preserves the quote style.
	return f.text(n)
}

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
