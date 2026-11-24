package formatter

// cfscript_formatter.go — recursive pretty-printer for the cfscript sub-grammar.

import (
	"bytes"
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
