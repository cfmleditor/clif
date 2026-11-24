package formatter

import (
	"fmt"
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
