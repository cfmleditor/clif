package formatter

import (
	"slices"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

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
