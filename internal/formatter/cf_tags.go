// Formatting CF tags: blocks, savecontent, conditions, if/else chains and self-closing tags.

package formatter

import (
	"slices"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func (f *Formatter) formatCFTag(n *sitter.Node) {
	// Check if this is a self-closing cf_tag (e.g. <cftransaction action="commit" />)
	for i := range n.ChildCount() {
		if n.Child(i).Kind() == "cf_start_tag_with_selfclose" {
			f.formatCFSelfCloseAttrTag(n)

			return
		}
	}

	name := f.tagName(n)
	attrs := f.collectAttrs(n)

	f.nl()
	f.writeIndent()
	f.write("<" + f.openTagName(n) + f.renderAttrs(name, attrs) + ">")
	f.write("\n")

	closed := f.hasRealCFEndTag(n)
	isBlock := closed // only a genuinely closed tag brackets an indented body

	// An empty block (e.g. <cfcatch type="any"></cfcatch>) otherwise emits the
	// padding blank line from both ends, leaving two blank lines around nothing.
	hasBody := f.hasCFBodyContent(n)

	if isBlock {
		f.level++

		if hasBody {
			f.write("\n")
		}
	}

	prevCFTagKind := ""
	prevCFTagWasComment := false

	for i := range n.ChildCount() {
		c := n.Child(i)
		switch c.Kind() {
		case "cf_start_tag", "cf_end_tag", "implicit_cf_end_tag":
			// Start and end tags are emitted around this loop. An unclosed tag
			// yields implicit_cf_end_tag, a synthetic marker whose text is the
			// trailing whitespace; emitting it added a stray blank line that
			// disappeared once the closing tag existed, making the output
			// differ between the first and second format.
		default:
			tagKind := f.nodeTagKind(c)
			if tagKind != "" && prevCFTagKind != "" && !prevCFTagWasComment && !f.nodeIsComment(c) &&
				(tagKind != prevCFTagKind || f.isBlockTagKind(c) || f.lastTagMultiLine) {
				f.write("\n")
			}

			// <cfcatch>/<cffinally> line up with their enclosing <cftry>
			// rather than sitting inside its body, matching the conventional
			// CFML style (and how <cfelse> is emitted inside <cfif>).
			dedent := name == "cftry" && f.level > 0 && f.isCFTryBranch(c)
			if dedent {
				f.level--
			}

			f.formatNode(c)

			if dedent {
				f.level++
			}

			if f.isWhitespaceNode(c) {
				continue
			}

			if tagKind != "" && !f.nodeIsComment(c) {
				prevCFTagKind = tagKind
			}

			prevCFTagWasComment = f.nodeIsComment(c)
		}
	}

	if isBlock {
		f.level--
	}

	if hasBody {
		f.write("\n")
	}

	// Only close what the source closed. Tags legal without a body — cfmodule,
	// cfhttp, cfinvoke, cffeed, cfadmin — were given a synthesised closing tag,
	// which swallowed everything after them into the tag's body.
	if closed {
		f.nl()
		f.writeIndent()
		f.write("</" + f.closeTagName(n) + ">")
	}

	f.write("\n")
}

// formatCFComponentOpen handles cf_component_open_tag (a sibling node in the tree).
func (f *Formatter) formatCFComponentOpen(n *sitter.Node) {
	attrs := f.collectAttrs(n)
	f.nl()
	f.writeIndent()
	f.write("<" + f.openTagNameOf("cfcomponent", n) + f.renderAttrs("cfcomponent", attrs) + ">")
	f.write("\n\n")

	f.level++
}

// formatCFComponentClose handles cf_component_close_tag (a sibling node in the tree).
//
// The open and close tags are siblings rather than parent and child, so the
// level is incremented by one and decremented by the other. A close tag with no
// open tag before it is accepted by the grammar without an ERROR node, and
// decrementing for it would leave the level negative for everything that
// follows, so the decrement is skipped when there is nothing to close.
func (f *Formatter) formatCFComponentClose(n *sitter.Node) {
	if f.level > 0 {
		f.level--
	}

	f.write("\n")
	f.nl()
	f.writeIndent()
	f.write("</" + f.closeTagNameOf("cfcomponent", n) + ">")
	f.write("\n")
}

// formatCFBlockTag handles specific block tags (cf_output_tag, cf_query_tag, etc.)
// that have inline children between <cf...> and </cf...>.
func (f *Formatter) formatCFBlockTag(n *sitter.Node) {
	name := f.tagName(n)
	attrs := f.collectAttrs(n)

	f.nl()
	f.writeIndent()
	f.write("<" + f.openTagName(n) + f.renderAttrs(name, attrs) + ">")
	f.write("\n")

	isBlock := true // specific block tag types are always blocks
	if isBlock {
		f.level++
		if name != "cfoutput" && (name != "cffunction" || !f.firstBodyChildIsArg(n)) {
			f.write("\n")
		}
	}

	prevTagKind := ""
	prevWasComment := false

	// Collect body nodes (skip syntax tokens).
	var bodyNodes []*sitter.Node

	for i := range n.ChildCount() {
		c := n.Child(i)

		kind := c.Kind()
		if kind == "<cf" || kind == "</cf" || kind == ">" || kind == "cf_tag_name" ||
			kind == "cf_tag_attributes" || kind == "cf_end_tag" ||
			kind == "cf_attribute" || kind == "cf_start_tag" {
			continue
		}

		bodyNodes = append(bodyNodes, c)
	}

	// If all body nodes are on a single line, emit them inline.
	inline := func() { f.emitBodyInline(bodyNodes) }
	if f.allSingleLine(bodyNodes) && !f.hasBlockChild(bodyNodes) && f.rendersOnOneLine(inline) {
		inline()
	} else {
		for i := 0; i < len(bodyNodes); {
			c := bodyNodes[i]
			kind := c.Kind()

			if f.isInlineNode(c) {
				// Collect consecutive inline nodes into a run.
				run := []*sitter.Node{c}
				j := i + 1

				for j < len(bodyNodes) && f.isInlineNode(bodyNodes[j]) {
					run = append(run, bodyNodes[j])
					j++
				}

				f.formatInlineRun(run)

				i = j

				continue
			}

			// Insert blank line between groups of different tag types.
			tagKind := f.nodeTagKind(c)
			if tagKind != "" && prevTagKind != "" && !prevWasComment && !f.nodeIsComment(c) &&
				(tagKind != prevTagKind || f.isBlockTagKind(c) || f.lastTagMultiLine) {
				f.write("\n")
			}

			if kind == "comment" {
				f.formatComment(c)
			} else {
				f.formatNode(c)
			}

			if f.isWhitespaceNode(c) {
				i++

				continue
			}

			if tagKind != "" && kind != "comment" && kind != "cf_comment" {
				prevTagKind = tagKind
			}

			prevWasComment = kind == "comment" || kind == "cf_comment"
			i++
		}
	}

	if isBlock {
		f.level--
	}

	f.nl()
	f.writeIndent()
	f.write("</" + f.closeTagName(n) + ">")
	f.write("\n")
}

// emitBodyInline writes a block tag's body without blank-line grouping: CF tags
// go through formatNode, and runs of text stay inline.
func (f *Formatter) emitBodyInline(bodyNodes []*sitter.Node) {
	var textRun []*sitter.Node

	for _, c := range bodyNodes {
		if strings.HasPrefix(c.Kind(), "cf_") {
			if len(textRun) > 0 {
				f.formatInlineRun(textRun)
				textRun = nil
			}

			f.formatNode(c)
		} else {
			textRun = append(textRun, c)
		}
	}

	if len(textRun) > 0 {
		f.formatInlineRun(textRun)
	}
}

// formatCFSavecontent emits the opening tag with proper indentation
// but preserves everything between (and including) the closing tag verbatim.
func (f *Formatter) formatCFSavecontent(n *sitter.Node) {
	name := f.tagName(n)
	attrs := f.collectAttrs(n)

	f.nl()
	f.writeIndent()
	f.write("<" + f.openTagName(n) + f.renderAttrs(name, attrs) + ">")

	// Emit everything between the tags verbatim. Collecting only the known
	// cf_savecontent_body* node kinds missed content the grammar places
	// outside them: a body consisting purely of comments parses as an *empty*
	// cf_savecontent_body with the comments following it as siblings, so the
	// entire body was dropped. Slicing the source spans whatever shape the
	// grammar chose.
	f.write(f.savecontentBody(n, name))

	// Emit closing tag
	f.write("</" + f.closeTagName(n) + ">")
	f.write("\n")
}

// savecontentBody returns the raw source between a savecontent tag's opening
// ">" and its closing tag.
func (f *Formatter) savecontentBody(n *sitter.Node, name string) string {
	start := headerEnd(f.src, int(n.StartByte()))

	end := min(int(n.EndByte()), len(f.src))

	if closeAt := lastIndexFold(f.src[:end], "</"+name); closeAt >= start {
		end = closeAt
	}

	if start < 0 || start > end {
		return ""
	}

	return string(f.src[start:end])
}

// lastIndexFold returns the offset of the final case-insensitive occurrence of
// lit in src, or -1.
func lastIndexFold(src []byte, lit string) int {
	for i := len(src) - len(lit); i >= 0; i-- {
		if hasBytesAtFold(src, i, lit) {
			return i
		}
	}

	return -1
}

func hasBytesAtFold(src []byte, pos int, lit string) bool {
	if pos < 0 || pos+len(lit) > len(src) {
		return false
	}

	for k := range len(lit) {
		if toLower(src[pos+k]) != toLower(lit[k]) {
			return false
		}
	}

	return true
}

// normalizeCond collapses internal newlines and leading whitespace in a
// multi-line condition expression into properly indented continuation lines.
// Long conditions are broken at logical operators, with indentation reflecting
// parenthesis depth for readability.
func (f *Formatter) normalizeCond(raw string) string {
	// Collapse to single line first.
	lines := strings.Split(raw, "\n")

	var parts []string

	for _, l := range lines {
		trimmed := trimSpace(l)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}

	// Break at logical operators, indenting by paren depth.
	baseIndent := f.indented() + f.opts.indent(1)

	// A "//" comment runs to the end of its line, so folding the following
	// line up onto it turns that code into part of the comment. Collapsing a
	// condition like
	//
	//	if ( a          // first
	//	    or b )      // second
	//
	// onto one line leaves everything after the first "//" commented out, and
	// because joining lines only removes whitespace the change slips past a
	// character-level comparison. Keep the author's line structure instead and
	// re-indent it.
	if anyHasLineComment(parts) {
		return strings.Join(parts, "\n"+baseIndent)
	}

	single := f.foldCondParts(parts)

	// Check if it fits on one line.
	tagPrefix := f.lineLen
	if tagPrefix+len(single) <= f.opts.LineWidth {
		return single
	}

	var result strings.Builder

	i := 0
	for i < len(single) {
		ch := single[i]

		if i > 0 {
			for _, op := range condBreakOperators {
				if matchWord(single, i, op) {
					result.WriteString("\n")
					result.WriteString(baseIndent)

					break
				}
			}
		}

		result.WriteByte(ch)

		i++
	}

	return result.String()
}

// foldCondParts joins a condition's lines back onto one.
//
// The lines are rendered output, not source, so a break after `(` or before
// `)` is one the formatter made: a call wider than the line width puts each
// argument on a line of its own. Joined with a space, that split came back as
// `f( a, b )` — padding argPad says an argument list does not get — and only
// once the call was wide enough to split, which it was not while a closure
// argument still spanned lines in the source. So the first format wrote
// `f(a, function(s) { … })`, the second measured that at full width, split it,
// and wrote `f( a, function(s) { … } )`. Folding the break with argPad makes
// the split invisible once it is folded, whichever pass makes it.
func (f *Formatter) foldCondParts(parts []string) string {
	var sb strings.Builder

	for i, p := range parts {
		if i > 0 {
			if strings.HasSuffix(parts[i-1], "(") || strings.HasPrefix(p, ")") {
				sb.WriteString(f.opts.argPad())
			} else {
				sb.WriteByte(' ')
			}
		}

		sb.WriteString(p)
	}

	return sb.String()
}

// anyHasLineComment reports whether any part carries a "//" comment, which
// makes the remainder of that part's line uncollapsible.
func anyHasLineComment(parts []string) bool {
	return slices.ContainsFunc(parts, hasLineComment)
}

// hasLineComment reports whether s opens a "//" comment outside a string
// literal. Unlike the document-wide scan the guard performs, s here is a single
// expression, so its quotes are balanced and tracking them is reliable.
func hasLineComment(s string) bool {
	var quote byte

	for i := range len(s) {
		c := s[i]

		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '/' && i+1 < len(s) && s[i+1] == '/':
			return true
		}
	}

	return false
}

// condBreakOperators are CFML logical operators where long conditions should break.
var condBreakOperators = []string{"&&", "||", "AND", "OR", "XOR", "EQV", "IMP"}

// matchWord checks if s[pos:] starts with word preceded and followed by a space.
func matchWord(s string, pos int, word string) bool {
	if pos == 0 || s[pos-1] != ' ' {
		return false
	}

	if pos+len(word) > len(s) {
		return false
	}

	if s[pos:pos+len(word)] != word {
		return false
	}

	if pos+len(word) < len(s) && s[pos+len(word)] != ' ' {
		return false
	}

	return true
}

// formatCFIfTag handles cf_if_tag with its condition, body, and optional cf_if_alt.
func (f *Formatter) formatCFIfTag(n *sitter.Node) {
	// Collect condition expression (named children before ">")
	var condParts []string

	inBody := false

	var bodyNodes []*sitter.Node

	var altNode *sitter.Node

	var trailingNodes []*sitter.Node

	for i := range n.ChildCount() {
		c := n.Child(i)

		kind := c.Kind()
		if kind == "<cf" || kind == "</cf" {
			continue
		}

		if kind == ">" {
			inBody = true

			continue
		}

		if kind == "cf_if_alt" {
			altNode = c

			continue
		}

		switch {
		case !inBody:
			if c.IsNamed() {
				condParts = append(condParts, f.expr(c))
			}
		case altNode != nil:
			// Nodes after cf_if_alt belong to the last branch (tree-sitter
			// sometimes places trailing comments as siblings of cf_if_alt).
			trailingNodes = append(trailingNodes, c)
		default:
			bodyNodes = append(bodyNodes, c)
		}
	}

	cond := f.normalizeCond(strings.Join(condParts, " "))
	f.nl()
	f.writeIndent()
	f.write("<" + f.openTagNameOf("cfif", n) + " " + cond + ">")
	f.write("\n")

	f.level++
	f.write("\n")

	f.formatNodes(bodyNodes)

	f.level--

	if altNode != nil {
		f.formatCFIfAlt(altNode)
	}

	// Emit nodes that tree-sitter placed after cf_if_alt (e.g. trailing
	// comments) — they belong to the last branch.
	if len(trailingNodes) > 0 {
		f.level++
		for _, c := range trailingNodes {
			f.formatNode(c)
		}

		f.level--
	}

	f.write("\n")
	f.nl()
	f.writeIndent()
	f.write("</" + f.closeTagNameOf("cfif", n) + ">")
	f.write("\n")
}

func (f *Formatter) formatCFIfAlt(n *sitter.Node) {
	// cf_if_alt structure:
	//   <cf (anonymous), cf_elseif_tag (condition + ">"), body nodes..., cf_if_alt (nested)
	//   OR: <cf (anonymous), cf_else_tag (">"), body nodes...
	// Body nodes are siblings of cf_elseif_tag/cf_else_tag, not children.
	var condParts []string

	var bodyNodes []*sitter.Node

	var altNode *sitter.Node

	isElse := false
	inBody := false

	for i := range n.ChildCount() {
		c := n.Child(i)

		kind := c.Kind()
		switch kind {
		case "<cf":
			continue
		case "cf_elseif_tag":
			// Extract condition from inside cf_elseif_tag
			for j := range c.ChildCount() {
				gc := c.Child(j)
				if gc.Kind() == ">" {
					break
				}

				if gc.IsNamed() {
					condParts = append(condParts, f.expr(gc))
				}
			}

			inBody = true
		case "cf_else_tag":
			isElse = true
			inBody = true
		case "cf_if_alt":
			altNode = c
		default:
			if inBody {
				bodyNodes = append(bodyNodes, c)
			}
		}
	}

	if isElse {
		f.write("\n")
		f.nl()
		f.writeIndent()
		f.write("<" + f.openTagNameOf("cfelse", n) + ">")
		f.write("\n")
	} else {
		cond := f.normalizeCond(strings.Join(condParts, " "))
		f.write("\n")
		f.nl()
		f.writeIndent()
		f.write("<" + f.openTagNameOf("cfelseif", n) + " " + cond + ">")
		f.write("\n")
	}

	f.level++
	f.write("\n")

	f.formatNodes(bodyNodes)

	f.level--

	if altNode != nil {
		f.formatCFIfAlt(altNode)
	}
}

func (f *Formatter) formatCFElseIf(n *sitter.Node) {
	// Called when cf_elseif_tag is visited directly (not via formatCFIfAlt).
	// In practice, formatCFIfAlt handles this inline, but keep for safety.
	f.write(f.text(n))
}

func (f *Formatter) formatCFElse(n *sitter.Node) {
	// Called when cf_else_tag is visited directly (not via formatCFIfAlt).
	// In practice, formatCFIfAlt handles this inline, but keep for safety.
	f.write(f.text(n))
}

// formatCFSelfCloseAttrTag handles cf_selfclose_tag (cfparam, cfargument, etc.)
// that have cf_attribute children.
func (f *Formatter) formatCFSelfCloseAttrTag(n *sitter.Node) {
	name := f.tagName(n)
	attrs := f.collectAttrs(n)

	rendered := f.renderAttrs(name, attrs)
	f.lastTagMultiLine = strings.Contains(rendered, "\n")
	f.nl()
	f.writeIndent()
	f.write("<" + f.openTagName(n) + rendered + " />")
	f.write("\n")
}

func (f *Formatter) formatCFSelfClosingTag(n *sitter.Node) {
	f.lastTagMultiLine = false

	// For specific self-closing tags (cf_set_tag, cf_return_tag),
	// reconstruct from expression children with normalized spacing.
	var exprParts []string

	for i := range n.ChildCount() {
		c := n.Child(i)

		kind := c.Kind()
		if kind == "<cf" || kind == "cf_selfclose_void_tag_end" || kind == ">" {
			continue
		}

		if c.IsNamed() {
			exprParts = append(exprParts, f.expr(c))
		}
	}

	body := strings.Join(exprParts, " ")

	// If the body spans multiple lines (e.g. multi-line function args),
	// emit with re-indentation based on the normalized expression.
	if strings.Contains(body, "\n") {
		f.lastTagMultiLine = true
		lines := strings.Split(body, "\n")

		f.nl()
		f.writeIndent()
		f.write("<" + f.openTagName(n) + " " + lines[0])
		f.write("\n")

		for i, l := range lines[1:] {
			if trimSpace(l) == "" {
				f.write("\n")

				continue
			}

			f.write(l)

			if i == len(lines)-2 {
				f.write(" />")
			}

			f.write("\n")
		}

		return
	}

	f.nl()
	f.writeIndent()

	if body != "" {
		f.write("<" + f.openTagName(n) + " " + body + " />")
	} else {
		f.write("<" + f.openTagName(n) + " />")
	}

	f.write("\n")
}
