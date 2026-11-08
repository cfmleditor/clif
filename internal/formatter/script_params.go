// Parameter lists: flat and broken layouts, defaults, types and attributes.

package formatter

import (
	"fmt"
	"slices"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// exprParams renders a formal_parameters / parameter_list node.
func (f *Formatter) exprParams(params *sitter.Node) string {
	if params == nil {
		return "()"
	}
	// Check if parameters use the flat structure (function_declaration grammar)
	// where required/type/name are siblings separated by commas, rather than
	// being wrapped in required_parameter/optional_parameter nodes.
	if f.hasFlatParams(params) {
		if inline, ok := f.flatParams(params); ok {
			return "(" + f.padded(inline) + ")"
		}

		// A list this path cannot lay out on one line without breaking it —
		// see flatParams. The source already holds a form that parses, so it is
		// reproduced as written rather than reconstructed.
		return f.text(params)
	}

	var parts []string

	var isComment []bool

	for i := range params.NamedChildCount() {
		c := params.NamedChild(i)
		parts = append(parts, f.exprParam(c))
		isComment = append(isComment, isCommentKind(c.Kind()))

		// As in the flat path above: a line comment cannot be joined onto a
		// single line, because it would swallow the rest of it.
		if f.isLineCommentNode(c) {
			return f.text(params)
		}
	}

	joined := strings.Join(parts, ", ")
	// The comma belongs after the last parameter, never after a trailing
	// comment — writing it there puts the separator inside the comment.
	if last := lastArgument(isComment); last >= 0 && last == len(parts)-1 && hasTrailingComma(params) {
		joined += ","
	}

	return "(" + f.padded(joined) + ")"
}

// exprFuncDefParams renders function definition parameters: on one line when
// paramBreakThreshold allows it and they fit, otherwise each on its own line.
//
// col is the column the "(" lands in — the indent plus everything already built
// into the signature — so the one-line form can be measured against LineWidth.
// The caller is the only one that knows it.
func (f *Formatter) exprFuncDefParams(params *sitter.Node, col int) string {
	if params == nil {
		return "()"
	}

	var parts []paramPart

	if f.hasFlatParams(params) {
		parts = f.flatParamParts(params)
	} else {
		trailing := hasTrailingComma(params)

		for i := range params.NamedChildCount() {
			c := params.NamedChild(i)
			parts = append(parts, paramPart{
				text:      f.exprParam(c),
				isComment: isCommentKind(c.Kind()),
			})
		}

		// This structure carries no comma children to read the separators from,
		// so they follow the usual rule: one after every parameter that has
		// another after it, and one after the last only if the source wrote it.
		last := lastParameter(parts)
		for i := range parts {
			parts[i].commaAfter = !parts[i].isComment && (i < last || (trailing && i == last))
		}
	}

	if len(parts) == 0 {
		return "()"
	}

	if flat, ok := f.flatFuncDefParams(parts, col); ok {
		return flat
	}

	indent := f.opts.indent(f.level + 1)

	var sb strings.Builder

	sb.WriteString("(\n")

	leading := f.opts.CommaPosition == "before"

	for i, p := range parts {
		sb.WriteString(indent)

		// In leading-comma style the separator is written before the parameter
		// it follows, so it is the *previous* entry's comma that decides
		// whether one appears here. A pair the source left comma-less has none
		// to move, and stays separated by the newline that makes it parse.
		if leading && i > 0 && parts[i-1].commaAfter {
			sb.WriteString(", ")
		}

		sb.WriteString(p.text)

		if !leading && p.commaAfter {
			sb.WriteString(",")
		}

		// A trailing comma has nowhere to go in leading style but after the
		// final parameter, which is where the source put it.
		if leading && i == len(parts)-1 && p.commaAfter {
			sb.WriteString(",")
		}

		sb.WriteString("\n")
	}

	sb.WriteString(f.opts.indent(f.level))
	sb.WriteString(")")

	return sb.String()
}

// flatFuncDefParams renders the parameter list on one line and reports whether
// that is allowed here.
//
// The threshold is the first gate, and its default of zero is what keeps a
// declaration breaking the way it always has: every list has at least one
// parameter by this point, so nothing is ever at or below zero. A project
// raising it is asking for short signatures to stay on one line.
//
// Four things refuse the one-line form whatever the threshold says. A comment
// among the parameters would swallow the rest of the signature, including the
// brace. A parameter that rendered multi-line cannot be folded back up. A
// trailing comma the source wrote means nothing on one line and would have to
// be either dropped — a non-whitespace change — or emitted as `(a, b,)`. And a
// list that would run past LineWidth is what the limit is for; the measurement
// stops at the closing paren, since whatever follows is the caller's to place.
func (f *Formatter) flatFuncDefParams(parts []paramPart, col int) (string, bool) {
	if len(parts) > f.opts.ParamBreakThreshold {
		return "", false
	}

	var sb strings.Builder

	for i, p := range parts {
		if p.isComment || strings.Contains(p.text, "\n") || strings.Contains(p.text, "//") {
			return "", false
		}

		if p.commaAfter && i == len(parts)-1 {
			return "", false
		}

		if i > 0 {
			sb.WriteString(" ")
		}

		sb.WriteString(p.text)

		if p.commaAfter {
			sb.WriteString(",")
		}
	}

	out := "(" + f.padded(sb.String()) + ")"
	if col+len(out) > f.opts.LineWidth {
		return "", false
	}

	return out, true
}

// lastParameter returns the index of the final entry that is a parameter rather
// than a comment, or -1 when there is none.
func lastParameter(parts []paramPart) int {
	for i, part := range slices.Backward(parts) {
		if !part.isComment {
			return i
		}
	}

	return -1
}

// hasFlatParams returns true if formal_parameters uses the flat structure:
// required/type/name/default as direct siblings of the params node rather
// than wrapped in a single per-parameter node. required_parameter and
// optional_parameter, which the non-flat path below was written for, do not
// exist anywhere in the current grammar (cfscript/grammar.js's
// _formal_parameter never wraps a parameter in one) — every parameter is flat.
//
// A typed parameter always has a parameter_type sibling, so that alone used
// to gate this. An untyped parameter does not, but "required" on one is
// exactly as flat: `required timeUnit = "milliseconds"` parses as the
// anonymous token "required" followed by an assignment_pattern sibling, not a
// single wrapping node — so without this check, a formal_parameters made up
// entirely of untyped parameters took the non-flat path, which iterates named
// children only and silently dropped every "required" it found.
func (f *Formatter) hasFlatParams(params *sitter.Node) bool {
	for i := range params.ChildCount() {
		switch params.Child(i).Kind() {
		case "parameter_type", "required":
			return true
		}
	}

	return false
}

// paramPart is one entry of a flat formal_parameters list: its rendered text,
// whether it is nothing but a comment, and whether the source wrote a comma
// after it. The separator is recorded per entry rather than assumed, because
// CFML does not require one — see flatParamParts.
type paramPart struct {
	text       string
	isComment  bool
	commaAfter bool
}

// flatParamParts splits the flat formal_parameters structure used by
// function_declaration — `[required] [type] name [= default]` as direct
// siblings — into one entry per parameter.
//
// Two entries need care, and both come from the separator not being reliable:
//
// A comment can sit anywhere a parameter can, including after the list's final
// comma (`f(a, b, // note\n)`), and it is not a parameter: a separator placed
// after one lands inside the comment rather than after the parameter it belongs
// to.
//
// And the comma between two parameters is optional — `f(struct s = structNew()
// boolean ssl)` is valid CFML, which tree-sitter-cfml has parsed since v0.26.35
// (#49). Splitting on commas alone merged such a pair into a single entry whose
// two halves were then rejoined with a space; the grammar accepts a newline
// between them but not a space, so the formatter's own output no longer parsed.
// A parameter that already has a name therefore ends where the next one's first
// token begins, comma or not, and whether a comma followed is recorded so the
// renderers can reproduce the source's separator instead of inventing one.
//
// The two callers below rendered this walk identically, so they now share it
// rather than each carrying a copy to keep in step.
func (f *Formatter) flatParamParts(params *sitter.Node) []paramPart {
	var parts []paramPart

	var current []string

	currentAllComments := true
	currentHasName := false

	// A `//` comment trailing a parameter, held back from its text. In
	// comma-first style (`,required any image // why` then `,string type`) the
	// comma that follows it is only read on the next line, and appending it to
	// the parameter's text put it inside the comment: `image // why,`. The two
	// parameters were then no longer separated, and the guard refused the file.
	// It becomes a comment entry of its own after the parameter, the shape a
	// comma-last `image, // why` already has, so both spellings come out alike.
	// Only when a comma follows: with none, nothing is written after the text,
	// and the comment stays on the parameter's line as it always has.
	var trailing []string

	flush := func(commaAfter bool) {
		if len(current) == 0 {
			return
		}

		if !commaAfter {
			current = append(current, trailing...)
			trailing = nil
		}

		parts = append(parts, paramPart{
			text:       strings.Join(current, " "),
			isComment:  currentAllComments,
			commaAfter: commaAfter,
		})

		for _, comment := range trailing {
			parts = append(parts, paramPart{text: comment, isComment: true})
		}

		current = nil
		trailing = nil
		currentAllComments = true
		currentHasName = false
	}

	for i := range params.ChildCount() {
		c := params.Child(i)
		kind := c.Kind()

		// The parentheses delimit the list rather than belonging to any entry
		// in it, so they are skipped before the flags below are touched —
		// marking the pending entry non-comment for the opening paren made
		// every list look as though it ended in a parameter.
		if kind == "(" || kind == ")" {
			continue
		}

		// A parameter carries at most one name, so a token that can only open a
		// parameter, seen once this entry already has its name, is the next
		// parameter beginning with no comma in between.
		if currentHasName && startsParameter(kind) {
			flush(false)
		}

		if kind != "," && !isCommentKind(kind) {
			currentAllComments = false

			// Something other than the separator came after a held comment, so
			// it was inside the parameter rather than trailing it. It goes back
			// where it was, as before.
			current = append(current, trailing...)
			trailing = nil
		}

		switch kind {
		case ",":
			flush(true)
		case "required":
			current = append(current, "required")
		case "parameter_type":
			current = append(current, f.text(c.Child(0)))
		case "array_return_suffix":
			current = appendTypeSuffix(current, f.text(c))
		case "identifier":
			current = append(current, f.text(c))
			currentHasName = true
		case "assignment_pattern":
			left := c.ChildByFieldName("left")
			right := c.ChildByFieldName("right")
			current = append(current, fmt.Sprintf("%s = %s", f.expr(left), f.expr(right)))
			currentHasName = true
		default:
			if !c.IsNamed() {
				break
			}

			// Once one is held, every comment after it is on a later line and
			// is held with it, in order.
			if !currentAllComments && isCommentKind(kind) &&
				(len(trailing) > 0 || f.isLineCommentNode(c)) {
				trailing = append(trailing, f.text(c))

				break
			}

			current = append(current, f.text(c))

			// A `//` comment ends its line by definition, so it cannot share an
			// entry with whatever follows. Entries are joined with a space, so
			// a standalone one used to absorb the next parameter — `a, // why,
			// b` came back as `a,` and `// why b`, with b deleted into the
			// comment. The guard caught the lost parameter and refused the
			// file, so it simply stopped responding to format-on-save.
			//
			// Only when the entry is nothing but comments. A comment trailing a
			// parameter belongs to it, and flushing there would strip the comma
			// that has not been read yet.
			if currentAllComments && f.isLineCommentNode(c) {
				flush(false)
			}
		}
	}

	// Nothing is pending when the list ended with a comma, so the trailing
	// comma is already recorded against the entry before it.
	flush(false)

	return parts
}

// startsParameter reports whether a child kind can only appear at the start of
// a parameter, which is what makes it a boundary when one is already in hand.
func startsParameter(kind string) bool {
	switch kind {
	case "required", "parameter_type", "identifier", "assignment_pattern":
		return true
	default:
		return false
	}
}

// commaLessBoundary reports whether any parameter is followed by another with no
// comma between them. Such a pair is separated by a newline in the source and
// only parses that way, so a renderer that lays the list out on one line has to
// reproduce it verbatim instead.
func commaLessBoundary(parts []paramPart) bool {
	for i, p := range parts {
		if i < len(parts)-1 && !p.commaAfter && !p.isComment {
			return true
		}
	}

	return false
}

// flatParams renders parameters from the flat formal_parameters structure
// used by function_declaration: [required] [type] name [= default], ...
// on a single line. It reports ok=false for a list it cannot lay out that way,
// which the caller renders verbatim instead.
func (f *Formatter) flatParams(params *sitter.Node) (string, bool) {
	parts := f.flatParamParts(params)

	// Two parameters with no comma between them are separated by a newline in
	// the source, and the grammar accepts only a newline there — joining them
	// with a space produces output it can no longer read.
	if commaLessBoundary(parts) {
		return "", false
	}

	// A `//` comment among the parameters cannot go on a single line either:
	// everything after it — the remaining parameters, the closing paren, and
	// the brace opening the body — becomes part of the comment. flatParamParts
	// collects comments as entries of their own, so this path has to say no to
	// them rather than join them.
	for _, p := range parts {
		if p.isComment && isLineCommentText(p.text) {
			return "", false
		}
	}

	var b strings.Builder

	for i, p := range parts {
		if i > 0 {
			b.WriteString(" ")
		}

		b.WriteString(p.text)

		if p.commaAfter {
			b.WriteString(",")
		}
	}

	return b.String(), true
}

// appendTypeSuffix glues an array_return_suffix (`[]`, always exactly that —
// the grammar lexes it as one token) onto the type token it qualifies, so
// `string[] v` does not come back as `string [] v`. The parts are joined with
// spaces, so it cannot simply be appended as another element.
func appendTypeSuffix(parts []string, suffix string) []string {
	if len(parts) == 0 {
		return append(parts, suffix)
	}

	parts[len(parts)-1] += suffix

	return parts
}

func (f *Formatter) exprParam(n *sitter.Node) string {
	switch n.Kind() {
	case "identifier":
		return f.text(n)
	case "assignment_pattern":
		left := n.ChildByFieldName("left")
		right := n.ChildByFieldName("right")

		return fmt.Sprintf("%s = %s", f.expr(left), f.expr(right))
	case "rest_pattern":
		return "..." + f.expr(n.NamedChild(0))
	case "required_parameter", "optional_parameter":
		// cfscript-specific: [required] [type] name [= default]
		return f.cfParam(n)
	default:
		return f.text(n)
	}
}

// cfParam renders a CF-style parameter declaration inside a function signature.
func (f *Formatter) cfParam(n *sitter.Node) string {
	var parts []string

	required := n.ChildByFieldName("required")
	typ := n.ChildByFieldName("type")
	name := n.ChildByFieldName("name")
	defVal := n.ChildByFieldName("default_value")

	if required != nil {
		parts = append(parts, f.text(required))
	}

	if typ != nil {
		parts = append(parts, f.text(typ))
	}

	if name != nil {
		parts = append(parts, f.text(name))
	}

	result := strings.Join(parts, " ")
	if defVal != nil {
		result += " = " + f.expr(defVal)
	}

	return result
}
