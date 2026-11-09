// Arrow functions, function expressions and closure bodies, and when closures hug their call.

package formatter

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func (f *Formatter) exprArrow(n *sitter.Node) string {
	params := n.ChildByFieldName("parameters")
	body := n.ChildByFieldName("body")

	// Lucee spells a closure `=>` and a lambda `->`; they differ in what they
	// capture, so the source's own arrow has to survive formatting.
	arrow := arrowToken(n)

	var paramStr string
	if params != nil {
		paramStr = f.exprParams(params)
	} else {
		// single param without parens
		p := n.ChildByFieldName("parameter")
		if p != nil {
			paramStr = f.text(p)
		}
	}

	if body != nil && (body.Kind() == "statement_block" || body.Kind() == "block") {
		// Render block inline for arrow functions; full block would need
		// newlines which aren't valid inside an expression context here.
		return fmt.Sprintf("%s %s %s", paramStr, arrow, f.closureBody(body))
	}

	return fmt.Sprintf("%s %s %s", paramStr, arrow, f.expr(body))
}

// arrowToken returns the arrow an arrow_function was written with, defaulting
// to `=>` if the grammar ever produces one without an anonymous arrow child.
func arrowToken(n *sitter.Node) string {
	for i := range n.ChildCount() {
		if kind := n.Child(i).Kind(); kind == "=>" || kind == "->" {
			return kind
		}
	}

	return "=>"
}

func (f *Formatter) exprFunctionExpr(n *sitter.Node) string {
	name := n.ChildByFieldName("name")
	params := n.ChildByFieldName("parameters")
	body := n.ChildByFieldName("body")

	nameStr := ""
	if name != nil {
		nameStr = " " + f.text(name)
	}

	// A function expression can carry annotations between its parameters and
	// its body — `describe("x", function() labels="query" { … })`, which is how
	// TestBox and Lucee's own suite label a spec. They are children with no
	// field name, so rendering the node from name/parameters/body alone deleted
	// them. The declaration path already collects them this way; this one did
	// not, so the same annotation survived on a declaration and vanished inside
	// an argument list.
	var attrs []string

	for i := range n.ChildCount() {
		switch n.FieldNameForChild(uint32(i)) {
		case "name", "parameters", "body", "return_type":
			continue
		}

		c := n.Child(i)
		if !c.IsNamed() {
			continue
		}

		// A `//` comment can sit among the annotations, and this path renders
		// the whole expression on one line — the body's opening brace included,
		// which the comment would then swallow. The declaration path breaks the
		// line instead (joinSignatureAttrs); an expression has nowhere to break
		// to, so it is reproduced as written.
		if f.isLineCommentNode(c) {
			return f.text(n)
		}

		attrs = append(attrs, f.text(c))
	}

	attrStr := ""
	if len(attrs) > 0 {
		attrStr = " " + strings.Join(attrs, " ")
	}

	return fmt.Sprintf("function%s%s%s %s", nameStr, f.exprParams(params), attrStr, f.closureBody(body))
}

// closureBody renders the block of a function expression or an arrow function.
//
// A body written across lines goes through the statement renderer, exactly as
// a declared function's body does, indented one level under the line the
// closure's expression sits on. It used to be copied verbatim, so the body of
// every callback kept whatever indentation it was typed with: a TestBox spec,
// which is nothing but nested `describe`/`it` closures, was left almost
// untouched inside its `run()`.
//
// The block is rendered into the output buffer and cut back out, because the
// statement renderer writes rather than returns. The comments a surrounding
// construct has queued for its own body are set aside for the duration, or the
// closure's block would flush them into itself.
//
// A body written on one line — `function(x) { return x > 1; }` — is kept as
// written: it is a short lambda whose author chose the inline form, and laying
// it out as a block would turn every one-line callback into five lines. An
// empty body written across lines becomes `{}`.
func (f *Formatter) closureBody(body *sitter.Node) string {
	src := f.blockText(body)
	if body.Kind() != "statement_block" || !strings.Contains(src, "\n") {
		return src
	}

	if body.NamedChildCount() == 0 {
		return "{}"
	}

	key := closureKey{src: &f.src[0], start: body.StartByte(), end: body.EndByte(), level: f.level}
	if rendered, ok := f.closureBodies[key]; ok {
		return rendered
	}

	st := f.saveState()
	f.pendingBlockComments = nil
	start := f.out.Len()

	f.scriptWrite("{")
	f.scriptBlockBody(body)

	rendered := string(f.out.Bytes()[start:])
	f.restoreState(&st)

	if f.closureBodies == nil {
		f.closureBodies = map[closureKey]string{}
	}

	f.closureBodies[key] = rendered

	return rendered
}

// closureKey identifies one rendering of a closure body. The rendering is a
// function of the block and the indent level alone: the body's statements start
// on lines of their own, so nothing about the line the closure sits on reaches
// them. src tells a <cfscript> region's re-parsed source apart from the
// document's.
//
// The cache is what keeps nesting linear. A literal or an argument list that
// goes multi-line renders its entries twice — once to measure the inline form,
// once a level deeper to lay them out — and a closure inside renders its own
// entries twice in turn, so without it the work doubled with every level of
// callback nested in a struct or a long argument list.
type closureKey struct {
	src        *byte
	start, end uint
	level      int
}

// entryValue returns the value of a struct entry or a named argument —
// `key: value`, `key = value`, `name = value` — and n itself for anything else.
// A callback is as often passed by name, `it(title = "x", body = function() {`,
// as by position.
func entryValue(n *sitter.Node) *sitter.Node {
	if n == nil {
		return nil
	}

	switch n.Kind() {
	case "pair", "cf_pair":
		return n.ChildByFieldName("value")
	case "object_assignment_pattern", "assignment_expression":
		return n.ChildByFieldName("right")
	}

	return n
}

// isClosure reports whether n is a function expression or an arrow function
// whose body is a block — the expressions closureBody lays out across lines.
func isClosure(n *sitter.Node) bool {
	if n == nil {
		return false
	}

	switch n.Kind() {
	case "function_expression":
		return true
	case "arrow_function":
		body := n.ChildByFieldName("body")

		return body != nil && body.Kind() == "statement_block"
	}

	return false
}

// closuresHug reports whether an argument list that holds multi-line closures
// can stay on the call's own line, `describe("x", function() {`, with each
// closure's body indented under the call and its `}` closing the line the
// call's `)` is on.
//
// Measured whole, a closure's body counts toward the argument list's width, so
// any callback of more than a few statements broke the list onto one argument
// per line: `describe(`, then `"x",`, then `function() {` on lines of their
// own. Only the lines the call itself owns are measured here — the first,
// up to the opening brace, and the last, from the closing brace — and every
// other argument must be on one line.
func (f *Formatter) closuresHug(args *sitter.Node, parts []string, inline string) bool {
	multi := false

	for i, p := range parts {
		if !strings.Contains(p, "\n") {
			continue
		}

		if !isClosure(entryValue(args.NamedChild(uint(i)))) {
			return false
		}

		multi = true
	}

	if !multi {
		return false
	}

	first, _, _ := strings.Cut(inline, "\n")
	last := inline[strings.LastIndexByte(inline, '\n')+1:]

	return len(first) <= f.opts.LineWidth && len(last) <= f.opts.LineWidth
}

// blockText is the source of a function body that is kept as written.
//
// In the cfml grammar the body can end past its own closing brace. When a
// closure inside a tag expression is followed by a newline and then `}` — the
// last entry of a struct literal that spans lines — the scanner skips the
// whitespace before inserting its zero-width automatic semicolon, and the
// block's extent runs on to that point. Copied verbatim, the newline became a
// blank line before the struct's `}`, and every further format added another,
// so the file never settled (Lucee's test/tags/query/inc.cfm).
func (f *Formatter) blockText(body *sitter.Node) string {
	return strings.TrimRight(f.text(body), " \t\r\n")
}
