// Control-flow statements in CFScript: if/else chains, switch, loops, try/catch and tag statements.

package formatter

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// scriptIf renders if / else if / else chains.
func (f *Formatter) scriptIf(n *sitter.Node) {
	f.iLine("if ")
	f.scriptIfTail(n)
	f.scriptWrite("\n")
}

// scriptIfInline is like scriptIf but does not prefix a newline+indent
// (used for `else if` continuation on the same line).
func (f *Formatter) scriptIfInline(n *sitter.Node) {
	f.scriptWrite("if ")
	f.scriptIfTail(n)
}

// scriptIfTail renders everything after the introducing keyword: the
// condition, the body, and whatever alternative follows. It is shared by
// `if_statement` and by Lucee's `else_if_clause`, which the grammar gives the
// same three fields; the two used to be walked by a pair of near-identical
// copies of this code, and the copies had already drifted — only one of them
// handled an alternative that was a bare `if_statement`.
func (f *Formatter) scriptIfTail(n *sitter.Node) {
	cond := n.ChildByFieldName("condition")
	cons := n.ChildByFieldName("consequence")
	alt := n.ChildByFieldName("alternative")

	f.scriptWrite(f.parenExpr(cond))
	f.deferBlockComments(n, cond, cons)
	f.scriptBlockOf2(cons)

	if alt != nil {
		f.scriptElseChain(n, cons, alt)
	}
}

// scriptElseChain renders a conditional's alternative: the keyword that
// introduces it, whatever follows, and any further alternatives beyond that.
func (f *Formatter) scriptElseChain(n, cons, alt *sitter.Node) {
	lead := f.elseLead(n, cons, alt, elseKeyword(alt))

	switch alt.Kind() {
	// Lucee's one-word `elseif`, which the grammar reads as a clause of its
	// own with an if_statement's fields rather than as an `else_clause`
	// wrapping one — there is no `else` token for that wrapper to match. It
	// is emitted as written: rewriting it to `else if` is a change no setting
	// asked for, and the reverse would break ACF, which has no such keyword.
	//
	// Before the grammar grew the clause this fell through to the default
	// arm, which wrote `else` and then rendered the whole `elseif ( b ) { … }`
	// as the statement inside it — the condition and its keyword emitted as
	// code in the body.
	case "else_if_clause":
		f.scriptWrite(lead + " ")
		f.scriptIfTail(alt)
	case "else_clause":
		inner := f.elseBody(alt)
		if inner != nil && inner.Kind() == "if_statement" {
			f.scriptWrite(lead + " ")
			f.scriptIfInline(inner)

			return
		}

		f.scriptWrite(lead)
		f.scriptBlockOf2(inner)
	case "if_statement":
		f.scriptWrite(lead + " ")
		f.scriptIfInline(alt)
	default:
		f.scriptWrite(lead)
		f.scriptBlockOf2(alt)
	}
}

// elseKeyword is the word that introduces alt. Only Lucee's one-word clause
// spells it anything but `else`; an `else if` is an `else_clause` holding an
// if_statement, so its `if` is written by that statement rather than here.
func elseKeyword(alt *sitter.Node) string {
	if alt != nil && alt.Kind() == "else_if_clause" {
		return "elseif"
	}

	return "else"
}

// elseLead emits any comments sitting between the consequence and the
// alternative, and returns the text to introduce keyword with: attached to
// the closing brace normally, or starting a fresh line once a comment has
// been written between them, or under braceStyle "next-line" (attachedKeyword).
func (f *Formatter) elseLead(n, cons, alt *sitter.Node, keyword string) string {
	if cons == nil || alt == nil {
		return f.attachedKeyword(keyword)
	}

	if !f.writeInterveningComments(n, cons.EndByte(), alt.StartByte()) {
		return f.attachedKeyword(keyword)
	}

	f.scriptWrite("\n")
	f.writeIndent()

	return keyword
}

// scriptBlockOf2 renders a statement as a braced block attached to the
// current line (e.g. the body of if/while/for).
func (f *Formatter) scriptBlockOf2(body *sitter.Node) {
	if body == nil {
		f.writeOpenBrace(" ")
		f.scriptWrite("}")

		return
	}

	if body.Kind() == "statement_block" || body.Kind() == "block" {
		f.scriptBlock(body)

		return
	}

	// A single-statement body gains braces here. The padding has to match
	// scriptBlock's exactly: on a second format the braces are in the source,
	// so scriptBlock renders the same code and any difference in blank lines
	// makes formatting non-idempotent — an unchanged file kept producing a
	// new diff on every save.
	f.writeOpenBrace(" ")
	f.blockPadOpen()

	f.level++

	if f.flushBlockComments() && isScriptBlockStmt(body) {
		f.scriptWrite("\n")
	}

	f.formatScriptNode(body)
	f.blockPadClose()

	f.level--
	f.writeIndent()
	f.scriptWrite("}")
}

// scriptSwitch renders a switch statement.
func (f *Formatter) scriptSwitch(n *sitter.Node) {
	val := n.ChildByFieldName("value")
	body := n.ChildByFieldName("body")

	f.iLine("switch " + f.parenExpr(val))
	f.writeOpenBrace(" ")
	f.scriptWrite("\n")

	f.level++

	// Each clause writes its label one level back from the statements under
	// it, so the two move together: shifting the whole body in by one puts the
	// label where the statements were and the statements one further in, which
	// is the indented style. Doing it here rather than at the labels is what
	// keeps them apart — moving the label alone left it level with its own
	// statements, which is neither style.
	if f.opts.SwitchCaseIndent {
		f.level++
	}

	if body != nil {
		for i := range body.NamedChildCount() {
			clause := body.NamedChild(i)
			switch clause.Kind() {
			case "switch_case":
				val2 := clause.ChildByFieldName("value")
				f.level--
				f.iLine(fmt.Sprintf("case %s:", f.expr(val2)))
				f.scriptWrite("\n")

				f.level++

				for j := range clause.NamedChildCount() {
					child := clause.NamedChild(j)
					if val2 != nil && child.StartByte() == val2.StartByte() && child.EndByte() == val2.EndByte() {
						continue
					}

					f.formatScriptNode(child)
				}
			case "switch_default":
				f.level--
				f.iLine("default:")
				f.scriptWrite("\n")

				f.level++
				for j := range clause.NamedChildCount() {
					f.formatScriptNode(clause.NamedChild(j))
				}
			default:
				f.formatScriptNode(clause)
			}
		}
	}

	if f.opts.SwitchCaseIndent {
		f.level--
	}

	f.level--
	f.writeIndent()
	f.scriptWrite("}\n")
}

func (f *Formatter) scriptWhile(n *sitter.Node) {
	cond := n.ChildByFieldName("condition")
	body := n.ChildByFieldName("body")

	f.iLine("while " + f.parenExpr(cond))
	f.scriptBlockOf2(body)
	f.scriptWrite("\n")
}

func (f *Formatter) scriptDo(n *sitter.Node) {
	body := n.ChildByFieldName("body")
	cond := n.ChildByFieldName("condition")

	f.iLine("do")
	f.scriptBlockOf2(body)
	f.scriptWrite(fmt.Sprintf(" while %s;\n", f.parenExpr(cond)))
}

// scriptFor renders: for (init; cond; update) { body }.
func (f *Formatter) scriptFor(n *sitter.Node) {
	init := n.ChildByFieldName("initializer")
	cond := n.ChildByFieldName("condition")
	incr := n.ChildByFieldName("increment")
	body := n.ChildByFieldName("body")

	initStr := f.forClause(init)
	condStr := f.forClause(cond)
	incrStr := f.forClause(incr)

	pad := f.opts.condPad()
	f.iLine(fmt.Sprintf("for (%s%s; %s; %s%s)", pad, initStr, condStr, incrStr, pad))
	f.scriptBlockOf2(body)
	f.scriptWrite("\n")
}

// scriptForIn renders: for (var x in collection) { body }
// Also handles for...of.
func (f *Formatter) scriptForIn(n *sitter.Node) {
	left := n.ChildByFieldName("left")
	right := n.ChildByFieldName("right")
	body := n.ChildByFieldName("body")

	keyword := "in"
	if n.Kind() == "for_of_statement" {
		keyword = "of"
	}

	varKind := ""

	for i := range n.ChildCount() {
		c := n.Child(i)
		if n.FieldNameForChild(uint32(i)) == "kind" {
			varKind = f.text(c) + " "

			break
		}
	}

	pad := f.opts.condPad()
	f.iLine(fmt.Sprintf("for (%s%s%s %s %s%s)", pad, varKind, f.expr(left), keyword, f.expr(right), pad))
	f.scriptBlockOf2(body)
	f.scriptWrite("\n")
}

// forClause renders an optional for-loop clause (init/cond/incr).
func (f *Formatter) forClause(n *sitter.Node) string {
	if n == nil {
		return ""
	}

	switch n.Kind() {
	case "empty_statement":
		return ""
	case "variable_declaration":
		// Inline version: var i = 0
		keyword := declKeyword(n)

		var decls []string

		for i := range n.NamedChildCount() {
			decls = append(decls, f.declarator(n.NamedChild(i)))
		}

		return keyword + " " + strings.Join(decls, ", ")
	default:
		return f.expr(n)
	}
}

// scriptTry renders try / catch / finally.
func (f *Formatter) scriptTry(n *sitter.Node) {
	body := n.ChildByFieldName("body")
	finalizer := n.ChildByFieldName("finalizer")

	f.iLine("try")

	if body != nil {
		f.scriptBodyOrStatement(body)
	}

	// Every catch clause carries the same `handler` field name, so
	// ChildByFieldName returns only the first one — a try with several
	// catches silently lost all but the first, bodies included. Walk the
	// children instead.
	prevEnd := uint(0)
	if body != nil {
		prevEnd = body.EndByte()
	}

	for i := range n.ChildCount() {
		c := n.Child(i)
		if c.Kind() != "catch_clause" {
			continue
		}

		f.scriptCatch(c, f.clauseLead(n, prevEnd, c, "catch"))
		prevEnd = c.EndByte()
	}

	if finalizer != nil {
		fBody := finalizer.ChildByFieldName("body")
		if fBody == nil {
			fBody = finalizer
		}

		f.scriptWrite(f.clauseLead(n, prevEnd, finalizer, "finally"))
		f.scriptBodyOrStatement(fBody)
	}

	f.scriptWrite("\n")
}

// scriptBodyOrStatement renders a clause body that may or may not be braced.
//
// v0.26.34 of the grammar parses a brace-less try body — `try doIt(); catch
// (any e) {}` — so this receives the statement itself rather than a
// statement_block. Feeding that to scriptBlock wrapped it in braces the source
// never had and dropped its semicolon, because the `;` is an anonymous child
// that the named-children walk skips. Both are non-whitespace changes, so with
// the whitespaceOnly guard on (the default) the LSP simply refused to format
// any file containing one.
//
// A brace-less body is a single statement, so it is emitted verbatim: there is
// nothing to re-indent, and reproducing the source exactly is what keeps the
// guard satisfied.
func (f *Formatter) scriptBodyOrStatement(body *sitter.Node) {
	switch body.Kind() {
	case "statement_block", "block":
		f.scriptBlock(body)
	default:
		f.scriptWrite(" ")
		f.scriptWrite(trimSpace(f.text(body)))
	}
}

// clauseLead emits any comments between the previous clause and this one, and
// returns the keyword text to introduce it with — attached to the closing
// brace normally, or on a fresh line once a comment has been written, or under
// braceStyle "next-line" (attachedKeyword).
func (f *Formatter) clauseLead(parent *sitter.Node, from uint, clause *sitter.Node, keyword string) string {
	if clause == nil || from == 0 {
		return f.attachedKeyword(keyword)
	}

	if !f.writeInterveningComments(parent, from, clause.StartByte()) {
		return f.attachedKeyword(keyword)
	}

	f.scriptWrite("\n")
	f.writeIndent()

	return keyword
}

// scriptCatch renders one catch clause: `catch (<type> <param>) { ... }`.
// The exception type is a separate `type` field, not part of the parameter —
// rendering only the parameter turned `catch (java.lang.Exception e)` into
// `catch (e)`, widening what the handler catches. The same goes for the `var`
// in `catch (any var e)`, which scopes the caught variable: it is an
// anonymous child with no field, so it has to be looked for.
func (f *Formatter) scriptCatch(n *sitter.Node, lead string) {
	catchType := n.ChildByFieldName("type")
	param := n.ChildByFieldName("parameter")
	body := n.ChildByFieldName("body")

	var parts []string

	if catchType != nil {
		parts = append(parts, f.text(catchType))
	}

	for i := range n.ChildCount() {
		if c := n.Child(i); !c.IsNamed() && c.Kind() == "var" {
			parts = append(parts, "var")

			break
		}
	}

	if param != nil {
		parts = append(parts, f.exprParam(param))
	}

	if len(parts) > 0 {
		f.scriptWrite(fmt.Sprintf("%s (%s)", lead, strings.Join(parts, " ")))
	} else {
		f.scriptWrite(lead)
	}

	if body != nil {
		f.scriptBlock(body)
	}
}

// scriptTagStatement renders a script-syntax CF tag that has a body —
// `lock name="x" { … }`, `transaction { … }`, `query name="q" { … }`,
// `cfhttp(url = u) { … }` — as its header, written as it stands, and its body
// laid out as a block.
//
// These had no renderer and went to scriptRaw, which trims every line of the
// node and writes them all at the statement's own level: whatever the body held
// came out flat, nested `if`s and loops included, and none of it was
// formatted. 811 corpus components have one; `query`, `transaction`, `loop`,
// `lock`, `savecontent` and `thread` account for most.
//
// The header keeps the tag's own spelling — `name="x"`, not the `name = x`
// the formatter gives a call's named arguments — as scriptThrow does for the
// same reason. Two shapes are left to scriptRaw as before: a body written on
// one line, which the author chose to keep inline, and a header holding a `//`
// comment, where writing the opening brace after it would put the brace inside
// the comment.
func (f *Formatter) scriptTagStatement(n *sitter.Node) {
	body := n.ChildByFieldName("body")
	if body == nil || body.Kind() != "statement_block" || !strings.Contains(f.text(body), "\n") {
		f.scriptRaw(n)

		return
	}

	for i := range n.NamedChildCount() {
		if c := n.NamedChild(i); c.StartByte() < body.StartByte() && f.isLineCommentNode(c) {
			f.scriptRaw(n)

			return
		}
	}

	f.iLine(trimSpace(string(f.src[n.StartByte():body.StartByte()])))
	f.scriptBlock(body)

	if tail := trimSpace(string(f.src[body.EndByte():n.EndByte()])); tail != "" {
		f.scriptWrite(tail)
	}

	f.scriptWrite("\n")
}

// scriptPassthru re-emits a node's text re-indented (last-resort fallback).
func (f *Formatter) scriptPassthru(n *sitter.Node) {
	f.iLine(trimSpace(f.text(n)))
	f.scriptWrite("\n")
}
