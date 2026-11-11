// Package formatter walks a tree-sitter concrete syntax tree produced by the
// cfml grammar and re-emits well-formatted CFML source.
//
// Formatting rules applied
//   - Consistent 4-space indentation inside block-level CF tags.
//   - One attribute per line when there are more than [AttrBreakThreshold]
//     attributes, or when the tag would exceed [LineWidth] columns.
//   - Attribute values are always double-quoted.
//   - CF tag and attribute names are lower-cased.
//   - Blank lines inside CFScript blocks are preserved but capped at one.
//   - The closing </cfXxx> tag matches the indentation of the opening tag.
//   - HTML content and non-CF tags are re-emitted verbatim (pass-through).
package formatter

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cfmleditor/clif/internal/lex"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// ParseFunc parses source code and returns a tree-sitter tree.
// Used to re-parse cfscript content with the cfscript sub-grammar.
type ParseFunc func(src []byte) *sitter.Tree

// Options controls formatting behaviour.
type Options struct {
	// IndentWidth is the number of spaces per indentation level (default 4).
	IndentWidth int
	// LineWidth is the soft column limit used to decide whether to expand
	// attributes onto separate lines (default 100).
	LineWidth int
	// QueryLineWidth is the soft column limit for SQL inside cfquery (default 70).
	QueryLineWidth int
	// BlankLinesInBlocks pads a block's body with a blank line after the
	// opening brace and before the closing one. Default true, which is what the
	// formatter has always emitted.
	BlankLinesInBlocks bool
	// SwitchCaseIndent indents `case` and `default` labels one level inside the
	// switch, level with the statements under them. Default false, which keeps
	// the label pulled back to the `switch` keyword's own column.
	SwitchCaseIndent bool
	// ParamBreakThreshold is the number of parameters above which a function
	// declaration's parameter list is expanded onto separate lines. Zero — the
	// default — breaks every list that has any parameters at all, which is what
	// the formatter has always done. A list at or below the threshold still
	// breaks when it would run past LineWidth, or when it holds a comment or a
	// trailing comma.
	ParamBreakThreshold int
	// AttrBreakThreshold is the number of attributes above which they are
	// always expanded onto separate lines regardless of line width (default 4).
	AttrBreakThreshold int
	// UseTabs uses a single tab character instead of spaces for indentation.
	UseTabs bool
	// LowercaseTags lowercases CF tag names (default true).
	LowercaseTags bool
	// LowercaseAttributes lowercases attribute names (default true).
	LowercaseAttributes bool
	// DoubleQuoteAttributes normalizes attribute values to double quotes (default true).
	DoubleQuoteAttributes bool
	// QueryUppercaseKeywords uppercases SQL keywords in cfquery (default true).
	QueryUppercaseKeywords bool
	// QueryFormat controls whether cfquery content is formatted.
	// When false, query content is emitted verbatim. Default false.
	QueryFormat bool
	// ParenSpacing is the padding inside parentheses: "pad" for `( a )`,
	// "tight" for `(a)`. Empty keeps what has always been emitted, which is
	// neither consistently — see condPad and argPad.
	ParenSpacing string
	// BraceStyle is where a block's opening brace goes: "same-line" (K&R,
	// `function f() {`) or "next-line" (Allman, the brace alone on the line
	// under the header). Empty means same-line, which is what the formatter
	// has always emitted.
	BraceStyle string
	// ScopeCase controls the case of CFML scope names (variables, arguments, etc.).
	// Valid values: "upper", "lower", "leave" (default "leave").
	ScopeCase string
	// CommaPosition controls where commas appear in multi-line argument lists.
	// Valid values: "after" (trailing, default), "before" (leading).
	CommaPosition string
	// QueryCommaPosition controls where commas appear in SQL SELECT lists.
	// Defaults to the value of CommaPosition if empty.
	QueryCommaPosition string
	// ParseScript re-parses cfscript content. If nil, script blocks are
	// emitted verbatim.
	ParseScript ParseFunc
	// ParseQuery re-parses cfquery content. If nil, query blocks are
	// emitted verbatim.
	ParseQuery ParseFunc
	// ParseCFML re-parses CFML source after pre-formatting. If nil,
	// pre-formatting (e.g. converting implicit end tags to self-closing) is skipped.
	ParseCFML ParseFunc
	// SelfCloseTags controls whether void/implicit-end HTML tags are
	// converted to self-closing form (e.g. <br> → <br />). Default true.
	SelfCloseTags bool
	// WhitespaceOnly when true causes Format to return an error if the
	// output differs from the input in non-whitespace content.
	WhitespaceOnly bool
}

// condPad and argPad are the padding inside a parenthesis, for the two kinds
// the formatter emits: a condition or grouping (`if ( a )`, `( a + b ) * c`)
// and an argument or parameter list (`foo(1, 2)`).
//
// Unset they disagree, because that is what the formatter has always done and
// changing it silently would reformat every file in every project. "pad" and
// "tight" are how a project asks for one rule in both places.
func (o *Options) condPad() string {
	if o.ParenSpacing == "tight" {
		return ""
	}

	return " "
}

func (o *Options) argPad() string {
	if o.ParenSpacing == "pad" {
		return " "
	}

	return ""
}

// nextLineBraces reports whether a block's opening brace starts a line of its
// own (Allman) rather than following the construct's header (K&R). Anything
// other than "next-line", the empty string included, is same-line, so a
// project that never set this keeps byte-for-byte what it had.
func (o *Options) nextLineBraces() bool {
	return o.BraceStyle == "next-line"
}

// padded wraps a rendered argument or parameter list in the configured inner
// padding. An empty list stays "()" — "( )" is nobody's preference.
func (f *Formatter) padded(inner string) string {
	if inner == "" {
		return ""
	}

	pad := f.opts.argPad()

	return pad + inner + pad
}

// indent renders the indentation for a nesting level. A negative level is
// column zero rather than a panic: roughly thirty sites decrement f.level, each
// paired with an increment somewhere else, and a document that closes a
// construct it never opened unbalances that pairing. strings.Repeat panics on a
// negative count, so `</cfcomponent>` on its own — an ordinary mid-edit state,
// and a real file in the corpus — took the formatter down rather than emitting
// the tag. Clamping keeps this function total for every caller; the individual
// level counters are still kept balanced at their own sites.
func (o *Options) indent(level int) string {
	if level <= 0 {
		return ""
	}

	if o.UseTabs {
		return strings.Repeat("\t", level)
	}

	w := o.IndentWidth
	if w == 0 {
		w = 4
	}

	return strings.Repeat(" ", w*level)
}

func (o *Options) queryCommaLeading() bool {
	return o.QueryCommaPosition == "before"
}

func (o *Options) queryCommaPreserve() bool {
	if o.QueryCommaPosition != "" {
		return o.QueryCommaPosition == "preserve"
	}

	return true // default: preserve comma placement in SQL
}

// DefaultOptions returns Options with sensible defaults (4-space indent, 120 col width).
func DefaultOptions() Options {
	return Options{
		IndentWidth:            4,
		LineWidth:              100,
		QueryLineWidth:         70,
		AttrBreakThreshold:     4,
		UseTabs:                true,
		LowercaseTags:          true,
		LowercaseAttributes:    true,
		DoubleQuoteAttributes:  true,
		QueryUppercaseKeywords: true,
		QueryFormat:            false,
		SelfCloseTags:          true,
		BlankLinesInBlocks:     true,
	}
}

// selfClosingTags never have a separate closing tag.
// var selfClosingTags = map[string]bool{
// 	"cfset": true, "cfparam": true, "cfreturn": true,
// 	"cfthrow": true, "cfabort": true, "cfbreak": true,
// 	"cfcontinue": true, "cfinvoke": true, "cfargument": true,
// 	"cfinclude": true, "cflocation": true, "cfcookie": true,
// 	"cfheader": true, "cfcontent": true, "cfflush": true,
// 	"cflog": true, "cfsetting": true, "cfprocessingdirective": true,
// 	"cfdump": true, "cfimage": true, "cfpdf": true,
// }

// utf8BOM is the UTF-8 byte-order mark. tree-sitter reports it as trivia
// belonging to no node, so it has to be preserved explicitly.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Formatter holds state during a single formatting pass.
type Formatter struct {
	opts             Options
	src              []byte
	out              bytes.Buffer
	level            int   // current indentation level
	atBOL            bool  // at beginning of line
	lastNL           bool  // last written byte was a newline
	lineLen          int   // approximate current line length
	lastTagMultiLine bool  // last emitted tag had expanded (multi-line) attributes
	parseErr         error // first sub-parse error encountered
	pendingComma     bool  // deferred trailing comma to emit after next query item
	// pendingBlockComments holds comments found between a construct's header
	// and its body, to be emitted just inside that body once it opens.
	pendingBlockComments []*sitter.Node
	// closureBodies caches closureBody by block and level; see there.
	closureBodies map[closureKey]string
	// exprs caches expr by node, level and column; see exprKey.
	exprs map[exprKey]exprResult
	// deadline and exprCalls are overBudget's; see there.
	deadline  time.Time
	exprCalls uint
}

// maxTreeDepth is the deepest tree the formatter lays out. The deepest file in
// the six-project corpus is 421 levels, a long concatenation; real nesting is
// under 110. Past the limit the file is refused rather than formatted, since
// the output grows with depth times lines: 16,000 nested blocks, 32KB of
// source, came out as 256MB, and 100,000 took 15GB before it was stopped.
const maxTreeDepth = 1000

// maxLayoutNesting is the deepest the formatter lets argument lists, arrays and
// structs nest inside one another. Each of those may be laid out on one line or
// across several, and deciding renders its entries at both levels, so a value
// at nesting n is rendered at up to n levels with up to n levels of indent on
// every line: 300 nested calls, 1KB of source, took 1.6GB, and 300 nested
// structs 3.5GB. The deepest in the corpus is 11, and 99.9% of files are at 8
// or less.
const maxLayoutNesting = 64

// layoutKinds are the nodes maxLayoutNesting counts.
var layoutKinds = []string{"arguments", "array", "object"}

// maxFormatTime bounds one Format call. Nothing found in the corpus comes near
// it; it is there so that whatever still makes the formatter slow on some
// input costs a refused format rather than a core held until the process
// exits, since the LSP runs a format on a goroutine it does not wait for.
// A var so a test can shorten it.
var maxFormatTime = 30 * time.Second

// errFormatTimeout is a format abandoned at maxFormatTime.
var errFormatTimeout = errors.New("formatting took too long; not formatting this file")

// overBudget reports whether this format has run past its deadline, recording
// the error the first time, so every renderer can stop and return its text as
// it is. The clock is read once every 4096 expressions.
func (f *Formatter) overBudget() bool {
	if errors.Is(f.parseErr, errFormatTimeout) {
		return true
	}

	f.exprCalls++
	if f.exprCalls&0xfff != 0 || f.deadline.IsZero() || time.Now().Before(f.deadline) {
		return false
	}

	f.parseErr = errFormatTimeout

	return true
}

// treeTooDeep refuses a tree deeper than maxTreeDepth, or with argument lists
// and literals nested more than maxLayoutNesting deep, recording why, so the
// format is declined. The walk stops as soon as either is past its limit.
func (f *Formatter) treeTooDeep(context string, root *sitter.Node) bool {
	why := treeTooDeep(root)
	if why == "" {
		return false
	}

	if f.parseErr == nil {
		f.parseErr = fmt.Errorf("%s %s; not formatting it", context, why)
	}

	return true
}

// treeTooDeep says which limit root's tree is past, or "" when it is within
// both.
func treeTooDeep(root *sitter.Node) string {
	lang := root.Language()

	layout := map[uint16]bool{}

	for _, k := range layoutKinds {
		if id := lang.IdForNodeKind(k, true); id != 0 {
			layout[id] = true
		}
	}

	c := root.Walk()
	defer c.Close()

	// nested[i] is how many layout nodes are open at depth i, the root's own
	// included, so leaving a node needs no record of what it was.
	nested := []int{0}

	for {
		if c.GotoFirstChild() {
			n := nested[len(nested)-1]
			if layout[c.Node().KindId()] {
				n++
			}

			nested = append(nested, n)

			switch {
			case len(nested)-1 > maxTreeDepth:
				return fmt.Sprintf("nests more than %d levels deep", maxTreeDepth)
			case n > maxLayoutNesting:
				return fmt.Sprintf("nests argument lists and literals more than %d deep", maxLayoutNesting)
			}

			continue
		}

		for !c.GotoNextSibling() {
			if !c.GotoParent() {
				return ""
			}

			nested = nested[:len(nested)-1]
		}

		// A sibling is entered at the depth of the node it follows.
		n := nested[len(nested)-2]
		if layout[c.Node().KindId()] {
			n++
		}

		nested[len(nested)-1] = n

		if n > maxLayoutNesting {
			return fmt.Sprintf("nests argument lists and literals more than %d deep", maxLayoutNesting)
		}
	}
}

// New creates a Formatter with the given options.
func New(opts *Options) *Formatter {
	// The formatter keeps its own copy: the defaults below are written into
	// it, and must not reach the caller's struct.
	o := *opts
	opts = &o

	if opts.IndentWidth == 0 {
		opts.IndentWidth = 4
	}

	if opts.LineWidth == 0 {
		opts.LineWidth = 120
	}

	if opts.AttrBreakThreshold == 0 {
		opts.AttrBreakThreshold = 3
	}

	return &Formatter{opts: o, atBOL: true, lastNL: true}
}

// Format parses src with the provided tree-sitter parser and returns
// formatted CFML.
// opts is only read.
func Format(src []byte, tree *sitter.Tree, opts *Options) (out []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			out = nil
			err = fmt.Errorf("formatter panic: %v", r)
		}
	}()

	if opts.SelfCloseTags && opts.ParseCFML != nil {
		var owned *sitter.Tree

		src, tree, owned = preformat(src, tree, opts.ParseCFML)
		if owned != nil {
			// preformat re-parses after each conversion pass and hands back the
			// last tree it made. Nobody else can free it: the caller only knows
			// about the tree it passed in, which preformat left untouched.
			defer owned.Close()
		}
	}

	f := New(opts)
	f.src = src
	root := tree.RootNode()

	// Refuse a tree the grammar could not parse. The node walk has no
	// meaningful rendering for an ERROR node and falls through to a raw
	// emit that concatenates its children without separators, producing
	// output that is not valid CFML. Bailing out here means a parse
	// failure can never be written over the user's source.
	if root.HasError() {
		return nil, parseErrorAt("document", root, src, 0)
	}

	if f.treeTooDeep("document", root) {
		return nil, f.parseErr
	}

	f.deadline = time.Now().Add(maxFormatTime)
	f.formatNode(root)

	if f.parseErr != nil {
		return nil, f.parseErr
	}

	out = f.out.Bytes()

	// A leading UTF-8 BOM sits outside every CST node, so the walk never
	// emits it. Carry it across verbatim — dropping it rewrites the file's
	// encoding preamble, which some CFML engines are sensitive to.
	if bytes.HasPrefix(src, utf8BOM) && !bytes.HasPrefix(out, utf8BOM) {
		out = slices.Concat(utf8BOM, out)
	}

	if opts.WhitespaceOnly {
		if err := checkWhitespaceOnly(src, out, opts.SelfCloseTags, opts.DoubleQuoteAttributes); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// cfmlScopes are the built-in CFML scope names.
var cfmlScopes = map[string]bool{
	"variables":   true,
	"arguments":   true,
	"local":       true,
	"request":     true,
	"session":     true,
	"application": true,
	"server":      true,
	"form":        true,
	"url":         true,
	"cgi":         true,
	"cookie":      true,
	"client":      true,
	"this":        true,
	"super":       true,
}

// recordParseError stores the first parse error encountered during sub-parsing.
func (f *Formatter) recordParseError(context string, root *sitter.Node, src []byte, baseRow uint) {
	if f.parseErr != nil {
		return
	}

	f.parseErr = parseErrorAt(context, root, src, baseRow)
}

// parseErrorAt builds an error describing the first ERROR or MISSING node
// under root, reported against the source line baseRow is offset from.
func parseErrorAt(context string, root *sitter.Node, src []byte, baseRow uint) error {
	errNode := findFirstError(root)
	if errNode == nil {
		return fmt.Errorf("parse error in %s block", context)
	}

	pos := errNode.StartPosition()
	line := baseRow + pos.Row + 1

	snippet := string(src[errNode.StartByte():errNode.EndByte()])
	if len(snippet) > 50 {
		snippet = snippet[:50] + "..."
	}

	return fmt.Errorf("parse error in %s at line %d, col %d near %q", context, line, pos.Column+1, snippet)
}

func findFirstError(n *sitter.Node) *sitter.Node {
	if n.IsError() || n.IsMissing() {
		return n
	}

	for i := range n.ChildCount() {
		if found := findFirstError(n.Child(i)); found != nil {
			return found
		}
	}

	return nil
}

// applyScopeCase transforms a CFML scope name based on the ScopeCase option.
func (f *Formatter) applyScopeCase(text string) string {
	if f.opts.ScopeCase == "" || f.opts.ScopeCase == "leave" {
		return text
	}

	if !cfmlScopes[strings.ToLower(text)] {
		return text
	}

	if f.opts.ScopeCase == "upper" {
		return strings.ToUpper(text)
	}

	return strings.ToLower(text)
}

// ─── node text helpers ───────────────────────────────────────────────────────

func (f *Formatter) text(n *sitter.Node) string {
	return string(f.src[n.StartByte():n.EndByte()])
}

// func (f *Formatter) childByField(n *sitter.Node, field string) *sitter.Node {
// 	for i := range n.ChildCount() {
// 		c := n.Child(i)
// 		if n.FieldNameForChild(uint32(i)) == field {
// 			return c
// 		}
// 	}
// 	return nil
// }

// ─── output helpers ──────────────────────────────────────────────────────────

func (f *Formatter) write(s string) {
	if s == "" {
		return
	}

	f.out.WriteString(s)
	// track approximate line length for soft-wrap decisions
	if idx := strings.LastIndexByte(s, '\n'); idx >= 0 {
		f.lineLen = len(s) - idx - 1
		f.lastNL = s[len(s)-1] == '\n'
		f.atBOL = f.lastNL
	} else {
		f.lineLen += len(s)
		f.lastNL = false
		f.atBOL = false
	}
}

func (f *Formatter) nl() {
	if f.lastNL {
		return
	}

	if f.dropWhitespaceOnlyLine() {
		return
	}

	f.write("\n")
}

// dropWhitespaceOnlyLine removes a partial trailing line holding nothing but
// whitespace, reporting whether it removed one.
//
// Content emitted verbatim — a <cfxml> or <cfscript> body, say — ends with the
// indentation that preceded its closing tag. Starting a new line after it
// committed that indentation as a whitespace-only line, and the next format did
// the same to the line it had just created. Files grew by one line every time
// they were formatted, without limit.
func (f *Formatter) dropWhitespaceOnlyLine() bool {
	b := f.out.Bytes()

	cut := len(b)
	for cut > 0 && b[cut-1] != '\n' {
		if !isWS(b[cut-1]) {
			return false
		}

		cut--
	}

	if cut == len(b) {
		return false
	}

	f.out.Truncate(cut)
	f.lineLen = 0
	f.lastNL = true
	f.atBOL = true

	return true
}

// appendTrailingComma inserts a comma before the trailing newline(s) in the
// output buffer. Used in trailing-comma mode when a source comma appears at
// the start of a new line — it gets moved to the end of the previous line.
// Returns true if the comma was successfully appended.
func (f *Formatter) appendTrailingComma() bool {
	b := f.out.Bytes()
	// Walk backwards past trailing whitespace/newlines to find the last content line
	i := len(b) - 1
	for i >= 0 && (b[i] == '\n' || b[i] == '\r' || b[i] == ' ' || b[i] == '\t') {
		i--
	}

	if i < 0 {
		return false
	}
	// Insert comma after position i (after the last non-whitespace char)
	insertPos := i + 1

	// b aliases the buffer's backing array, so the tail has to be copied out
	// before anything is written back. Reset() keeps that array, WriteByte then
	// overwrites the very byte the following Write re-reads, and "SELECT a\n"
	// came back as "SELECT a,,".
	tail := append([]byte(nil), b[insertPos:]...)

	f.out.Truncate(insertPos)
	f.out.WriteByte(',')
	f.out.Write(tail)

	return true
}

// flushPendingComma appends a deferred trailing comma after the last emitted
// content. Called after an item has been fully emitted.
// func (f *Formatter) flushPendingComma() {
// 	if !f.pendingComma {
// 		return
// 	}
// 	f.pendingComma = false
// 	f.appendTrailingComma()
// }

func (f *Formatter) indented() string {
	return f.opts.indent(f.level)
}

func (f *Formatter) writeIndent() {
	if f.atBOL {
		f.write(f.indented())
	}
}

// countIndentLevel counts the indentation level of a line, treating each tab
// as one level and each indentWidth spaces as one level.
// func (f *Formatter) countIndentLevel(line string, indentWidth int) int {
// 	level := 0
// 	spaces := 0
// 	for _, ch := range line {
// 		switch ch {
// 		case '\t':
// 			level++
// 			spaces = 0
// 		case ' ':
// 			spaces++
// 			if spaces >= indentWidth {
// 				level++
// 				spaces = 0
// 			}
// 		default:
// 			return level
// 		}
// 	}
// 	return level
// }

// ─── attribute formatting ────────────────────────────────────────────────────

type cfAttr struct {
	name  string
	value string // empty = boolean attribute
	// comment marks an entry that is a CFML comment sitting among the
	// attributes rather than an attribute of its own. It carries its whole
	// text in name and is re-emitted verbatim.
	comment bool
}

// collectAttrs gathers all cf_attribute children from a tag node,
// searching through cf_start_tag and cf_tag_attributes.
func (f *Formatter) collectAttrs(tag *sitter.Node) []cfAttr {
	var attrs []cfAttr

	f.walkAttrs(tag, &attrs, headerEnd(f.src, int(tag.StartByte())))

	return attrs
}

func (f *Formatter) walkAttrs(n *sitter.Node, attrs *[]cfAttr, hdrEnd int) {
	for i := range n.ChildCount() {
		c := n.Child(i)
		switch c.Kind() {
		case "cf_start_tag", "cf_start_tag_with_selfclose", "cf_tag_attributes":
			f.walkAttrs(c, attrs, hdrEnd)
		case "cf_comment":
			// A comment among the attributes. Commenting an attribute out is
			// how a setting gets parked without losing it, so dropping the
			// comment throws that away silently. Carry it through in place.
			if int(c.StartByte()) < hdrEnd {
				*attrs = append(*attrs, cfAttr{name: f.text(c), comment: true})
			}
		case "cf_attribute":
			attr := cfAttr{}

			for j := range c.ChildCount() {
				gc := c.Child(j)
				switch gc.Kind() {
				case "cf_attribute_name":
					name := f.text(gc)
					if f.opts.LowercaseAttributes {
						name = strings.ToLower(name)
					}

					attr.name = name
				case "quoted_cf_attribute_value":
					if f.opts.DoubleQuoteAttributes {
						attr.value = f.normaliseAttrValue(f.text(gc))
					} else {
						attr.value = trimSpace(f.text(gc))
					}
				case "cf_attribute_value":
					val := trimSpace(f.text(gc))
					if f.opts.DoubleQuoteAttributes {
						attr.value = `"` + val + `"`
					} else {
						attr.value = val
					}
				}
			}

			*attrs = append(*attrs, attr)
		}
	}
}

// normaliseAttrValue ensures the value is wrapped in double quotes.
//
// The quote characters *inside* the value are never rewritten. Swapping them
// corrupts the value whenever the delimiter it would swap to already appears
// within: `to="#listLen(temp,"'")#"` became `to="#listLen(temp,”')#"`, an
// unbalanced literal the grammar then rejects. Re-quoting is therefore only
// applied when the value carries no double quote of its own; otherwise the
// original delimiters are kept as-is.
func (f *Formatter) normaliseAttrValue(v string) string {
	v = trimSpace(v)
	if len(v) >= 2 {
		q := v[0]
		if (q == '"' || q == '\'') && v[len(v)-1] == q {
			if q == '"' {
				return v
			}

			inner := v[1 : len(v)-1]
			if strings.Contains(inner, `"`) {
				// Converting would need the inner quotes rewritten. Leave the
				// single quotes in place rather than change the value.
				return v
			}

			return `"` + inner + `"`
		}
	}

	return `"` + v + `"`
}

func (f *Formatter) renderAttrs(tagName string, attrs []cfAttr) string {
	if len(attrs) == 0 {
		return ""
	}

	// Inline rendering
	var inline strings.Builder
	inline.WriteString(" ")

	for i, a := range attrs {
		if i > 0 {
			inline.WriteString(" ")
		}

		if a.value == "" {
			inline.WriteString(a.name)
		} else {
			fmt.Fprintf(&inline, "%s=%s", a.name, a.value)
		}
	}

	// Should we expand?
	oneLiner := "<" + tagName + inline.String()
	expand := len(attrs) > f.opts.AttrBreakThreshold ||
		f.lineLen+len(oneLiner) > f.opts.LineWidth ||
		hasMultiLineComment(attrs)

	if !expand {
		return inline.String()
	}

	// Multi-line rendering: one attribute per line, indented one level.
	indent := f.opts.indent(1)

	var sb strings.Builder
	for i, a := range attrs {
		sb.WriteString("\n")
		sb.WriteString(f.indented())
		sb.WriteString(indent)

		if a.value == "" {
			sb.WriteString(a.name)
		} else {
			sb.WriteString(a.name)
			sb.WriteByte('=')
			sb.WriteString(a.value)
		}

		if i < len(attrs)-1 {
			sb.WriteString(" ")
		}
	}

	return sb.String()
}

// ─── core traversal ──────────────────────────────────────────────────────────

func (f *Formatter) formatNode(n *sitter.Node) {
	kind := n.Kind()
	switch kind {
	case "program", "component_file":
		f.formatChildren(n)

	case "cf_component_content":
		f.formatComponentContent(n)

	case "cf_component_open_tag":
		f.formatCFComponentOpen(n)

	case "cf_component_close_tag":
		f.formatCFComponentClose(n)

	case "cf_tag":
		f.formatCFTag(n)

	case "cf_set_tag", "cf_return_tag":
		f.formatCFSelfClosingTag(n)

	case "cf_selfclose_tag":
		f.formatCFSelfCloseAttrTag(n)

	case "cf_if_tag":
		f.formatCFIfTag(n)

	case "cf_if_alt":
		f.formatCFIfAlt(n)

	case "cf_elseif_tag":
		f.formatCFElseIf(n)

	case "cf_else_tag":
		f.formatCFElse(n)

	case "cf_output_tag",
		"cf_function_tag",
		"cf_xml_tag":
		f.formatCFBlockTag(n)

	case "cf_savecontent_tag":
		f.formatCFSavecontent(n)

	case "cf_query_tag":
		f.formatCFQuery(n)

	case "cf_script_tag":
		f.formatCFScript(n)

	case "hash_expression":
		f.formatHashExpression(n)

	case "element":
		f.formatElement(n)

	case "html_text", "text":
		f.formatText(n)

	case "doctype", "xml_decl":
		f.formatDoctype(n)

	case "style_element", "script_element":
		f.formatRawTextElement(n)

	case "comment", "cf_comment":
		f.formatComment(n)

	case "cf_selfclose_void_tag_end":
		// Handled by parent (formatCFSelfClosingTag / formatCFSelfCloseAttrTag).
		// Emit verbatim if reached directly.
		f.write(f.text(n))

	case "implicit_end_tag":
		// Whitespace between tags — suppress since the formatter handles spacing.

	case "assignment_expression", "binary_expression",
		"unary_expression", "not_expression", "ternary_expression",
		"elvis_expression", "update_expression",
		"call_expression", "member_expression",
		"subscript_expression", "new_expression",
		"sequence_expression", "augmented_assignment_expression",
		"parenthesized_expression":
		// Expression nodes — normalize spacing via expr().
		f.write(f.expr(n))

	default:
		// Leaf nodes emit verbatim; non-leaf recurse into children.
		if n.ChildCount() == 0 {
			f.write(f.text(n))
		} else {
			f.formatChildren(n)
		}
	}
}

// formatterState is the mutable output state a trial render has to leave
// exactly as it found it.
type formatterState struct {
	outLen           int
	level            int
	atBOL            bool
	lastNL           bool
	lineLen          int
	lastTagMultiLine bool
	pendingComma     bool
	parseErr         error
	pendingComments  []*sitter.Node
}

func (f *Formatter) saveState() formatterState {
	return formatterState{
		outLen:           f.out.Len(),
		level:            f.level,
		atBOL:            f.atBOL,
		lastNL:           f.lastNL,
		lineLen:          f.lineLen,
		lastTagMultiLine: f.lastTagMultiLine,
		pendingComma:     f.pendingComma,
		parseErr:         f.parseErr,
		pendingComments:  slices.Clone(f.pendingBlockComments),
	}
}

func (f *Formatter) restoreState(s *formatterState) {
	f.out.Truncate(s.outLen)
	f.level = s.level
	f.atBOL = s.atBOL
	f.lastNL = s.lastNL
	f.lineLen = s.lineLen
	f.lastTagMultiLine = s.lastTagMultiLine
	f.pendingComma = s.pendingComma
	f.parseErr = s.parseErr
	f.pendingBlockComments = s.pendingComments
}

// rendersOnOneLine reports whether what emit writes lands on a single line
// inside LineWidth, by running it, measuring the result and discarding it.
//
// The question has to be asked of the text that will actually be emitted, not
// of the source. The formatter rewrites as it goes — a tag written ">" comes out
// " />", attribute values gain quotes, spacing is normalised — so measuring the
// source ran a couple of characters per tag short of the truth. Once a file had
// been formatted the measurement changed, borderline bodies crossed the limit
// and took the other branch, and formatting alternated between two layouts.
// Approximating the difference textually does not help: it only moves the
// boundary, so a different set of borderline cases flips instead.
//
// Note that the width alone cannot answer this. The emitters soft-wrap, so a
// body that is too long comes back inside LineWidth by being split across
// lines — the tell is the split itself, not the width. Both are checked.
func (f *Formatter) rendersOnOneLine(emit func()) bool {
	saved := f.saveState()
	col := f.lineLen

	emit()

	rendered := f.out.Bytes()[saved.outLen:]

	// The emitters open their own line; that first newline is expected, and
	// resets the column. Any further one means the content was wrapped.
	if len(rendered) > 0 && rendered[0] == '\n' {
		rendered = rendered[1:]
		col = 0
	}

	fits := !bytes.ContainsRune(rendered, '\n') && col+len(rendered) <= f.opts.LineWidth

	f.restoreState(&saved)

	return fits
}

// isWhitespaceNode reports whether n contributes nothing but whitespace.
//
// Whitespace between two siblings is not itself a sibling. Letting one update
// the grouping state made trailing spaces after a comment look like a
// non-comment neighbour, so the tag after it gained a blank line — which the
// same format then stripped, flipping the answer on the next pass. Files
// oscillated between the two forms forever.
func (f *Formatter) isWhitespaceNode(n *sitter.Node) bool {
	return trimSpace(f.text(n)) == ""
}

func (f *Formatter) formatChildren(n *sitter.Node) {
	children := make([]*sitter.Node, 0, n.ChildCount())
	for i := range n.ChildCount() {
		children = append(children, n.Child(i))
	}

	f.formatNodes(children)
}

// formatNodes formats siblings in order, with a blank line between two groups of
// tags (a different kind of tag, a block tag, or after a multi-line one) unless a
// comment sits between them.
func (f *Formatter) formatNodes(nodes []*sitter.Node) {
	prevTagKind := ""
	prevWasComment := false

	for _, c := range nodes {
		kind := f.nodeTagKind(c)

		if kind != "" && prevTagKind != "" && !prevWasComment && !f.nodeIsComment(c) &&
			(kind != prevTagKind || f.isBlockTagKind(c) || f.lastTagMultiLine) {
			f.write("\n")
		}

		f.formatNode(c)

		if f.isWhitespaceNode(c) {
			continue
		}

		if kind != "" && !f.nodeIsComment(c) {
			prevTagKind = kind
		}

		prevWasComment = f.nodeIsComment(c)
	}
}

// nodeTagKind returns a string identifying the "tag group" of a node for
// blank-line-between-groups logic. Returns "" for non-tag nodes.
func (f *Formatter) nodeTagKind(n *sitter.Node) string {
	kind := n.Kind()
	switch kind {
	case "cf_set_tag":
		// Distinguish <cfset var ...> from <cfset ...> for grouping.
		text := f.text(n)
		if strings.Contains(text, " var ") || strings.Contains(text, "\tvar ") {
			return "cf_set_var_tag"
		}

		return kind
	case "cf_return_tag", "cf_selfclose_tag", "cf_tag",
		"cf_if_tag", "cf_output_tag", "cf_function_tag", "cf_query_tag",
		"cf_script_tag", "cf_xml_tag", "cf_savecontent_tag",
		"comment", "cf_comment":
		return kind
	default:
		return ""
	}
}

// nodeIsComment returns true if the node is a comment. Comments are transparent
// to blank-line grouping in both directions: they neither end the run they sit
// in nor start a new one.
//
// The symmetry is what makes the output stable. Whether the grammar hands a
// comment to a tag's body or to the siblings after it depends on stray trailing
// whitespace, and only the body is grouped — so a comment gained a blank line
// before it, the format removed the whitespace, the re-parse moved the comment
// out of the body, and the blank line went away again.
func (f *Formatter) nodeIsComment(n *sitter.Node) bool {
	kind := n.Kind()

	return kind == "comment" || kind == "cf_comment"
}

// isBlockTagKind returns true if the node is a block-level CF tag that
// contains indented children (cfif, cfloop, cfoutput, etc.).
func (f *Formatter) isBlockTagKind(n *sitter.Node) bool {
	kind := n.Kind()
	switch kind {
	case "cf_if_tag", "cf_output_tag", "cf_function_tag", "cf_query_tag",
		"cf_script_tag", "cf_xml_tag", "cf_savecontent_tag":
		return true
	case "cf_tag":
		// A cf_tag with an end tag is a block tag.
		for i := range n.ChildCount() {
			if n.Child(i).Kind() == "cf_end_tag" {
				return true
			}
		}

		return false
	}

	return false
}

// firstBodyChildIsArg returns true if the first meaningful body child of a
// block tag is a cfargument tag.
//
// Whitespace-only text counts as nothing. Trailing spaces after a tag's ">"
// become a text node, so `<cffunction ...>   ` followed by a <cfargument> used
// to answer false and gain a blank line — which the format then removed,
// leaving the file to answer true and lose the blank line on the next pass.
// Formatting oscillated between the two forever.
func (f *Formatter) firstBodyChildIsArg(n *sitter.Node) bool {
	for i := range n.ChildCount() {
		c := n.Child(i)

		kind := c.Kind()
		if kind == "<cf" || kind == "</cf" || kind == ">" || kind == "cf_tag_name" ||
			kind == "cf_tag_attributes" || kind == "cf_end_tag" ||
			kind == "cf_attribute" || kind == "cf_start_tag" {
			continue
		}

		if trimSpace(f.text(c)) == "" {
			continue
		}

		if kind == "cf_selfclose_tag" {
			return f.tagName(c) == "cfargument"
		}

		return false
	}

	return false
}

// ─── CF tag formatting ───────────────────────────────────────────────────────

// tagName is a CF tag's canonical name: always lowercase, whatever the source
// wrote and whatever LowercaseTags says. It is the tag's identity — what the
// per-tag rules switch on, what gets compared against literals, what the source
// is searched for to find a matching close. What to *write* is openTagName or
// closeTagName.
//
// It used to fold in LowercaseTags, so with that off it returned things like
// "cfARGUMENT" and the bare comparison in hasArgumentChild stopped matching.
func (f *Formatter) tagName(n *sitter.Node) string {
	kind := n.Kind()

	// Generic cf_tag: look for cf_tag_name in cf_start_tag child.
	if kind == "cf_tag" || kind == "cf_start_tag" || kind == "cf_end_tag" || kind == "cf_start_tag_with_selfclose" {
		for i := range n.ChildCount() {
			c := n.Child(i)
			if c.Kind() == "cf_tag_name" {
				return "cf" + strings.ToLower(f.text(c))
			}

			if c.Kind() == "cf_start_tag" || c.Kind() == "cf_start_tag_with_selfclose" {
				return f.tagName(c)
			}
		}
	}

	// cf_selfclose_tag: extract name from source text after "<cf"
	if kind == "cf_selfclose_tag" {
		raw := f.text(n)
		if strings.HasPrefix(strings.ToLower(raw), "<cf") {
			rest := raw[3:]
			name := rest

			if end := strings.IndexAny(rest, " \t\r\n/>"); end > 0 {
				name = rest[:end]
			}

			return "cf" + strings.ToLower(name)
		}
	}

	// Specific tags: cf_set_tag → cfset, cf_if_tag → cfif, etc.
	if strings.HasPrefix(kind, "cf_") && strings.HasSuffix(kind, "_tag") && len(kind) > 7 {
		inner := kind[3 : len(kind)-4] // strip "cf_" and "_tag"

		return "cf" + strings.ReplaceAll(inner, "_", "")
	}

	return ""
}

// openTagName and closeTagName are what to write for a tag's opening and
// closing halves.
//
// With LowercaseTags on — the default — both are the canonical name and the
// source is not consulted at all, so nothing about the default output changes.
// With it off the source's own spelling is kept, which is the whole point of
// the setting and is what it did not do. The grammar folds a known tag's
// keyword into the node kind: `<CFOUTPUT>` parses as cf_output_tag with no
// token covering "OUTPUT" at all, so the name was rebuilt from the kind and
// came out lowercase however it was written. The one path that did read the
// source rebuilt the name as "cf" plus the source's suffix, which turned
// <CFDUMP> into the mongrel cfDUMP.
//
// The halves are read separately because CFML does not require them to agree —
// `<CFOUTPUT>...</cfoutput>` is legal — and "leave my casing alone" means
// leaving each as it was written.
func (f *Formatter) openTagName(n *sitter.Node) string {
	return f.openTagNameOf(f.tagName(n), n)
}

func (f *Formatter) closeTagName(n *sitter.Node) string {
	return f.closeTagNameOf(f.tagName(n), n)
}

// openTagNameOf and closeTagNameOf are the same for a writer that hardcodes the
// tag it emits — cfif, cfelse, cfcomponent, cfscript, cfquery — rather than
// reading the name off the node.
func (f *Formatter) openTagNameOf(canonical string, n *sitter.Node) string {
	return f.spelledAs(canonical, int(n.StartByte()))
}

func (f *Formatter) closeTagNameOf(canonical string, n *sitter.Node) string {
	if f.opts.LowercaseTags || canonical == "" {
		return canonical
	}

	end := min(int(n.EndByte()), len(f.src))

	at := lastIndexFold(f.src[:end], "</"+canonical)
	if at < 0 {
		return canonical
	}

	return f.spelledAs(canonical, at)
}

// spelledAs reads back how the tag name at a "<" is written in the source.
//
// It can only ever return a different *casing* of the name it was given: a
// spelling that does not case-fold equal to the canonical name is discarded, so
// a miscounted offset degrades to today's output rather than emitting some
// other tag.
func (f *Formatter) spelledAs(canonical string, lt int) string {
	if f.opts.LowercaseTags || canonical == "" || lt < 0 || lt >= len(f.src) || f.src[lt] != '<' {
		return canonical
	}

	i := lt + 1
	if i < len(f.src) && f.src[i] == '/' {
		i++
	}

	start := i
	for i < len(f.src) && lex.IsWord(f.src[i]) {
		i++
	}

	if spelled := string(f.src[start:i]); strings.EqualFold(spelled, canonical) {
		return spelled
	}

	return canonical
}

// formatCFTag handles generic cf_tag nodes (cffunction, cfloop, etc.)
// Structure: cf_start_tag + body children + cf_end_tag
// hasCFBodyContent reports whether a cf_tag has any body child with actual
// content, ignoring the start/end tags emitted around the body loop.
func (f *Formatter) hasCFBodyContent(n *sitter.Node) bool {
	for i := range n.ChildCount() {
		c := n.Child(i)
		switch c.Kind() {
		case "cf_start_tag", "cf_end_tag", "implicit_cf_end_tag":
			continue
		default:
			if trimSpace(f.text(c)) != "" {
				return true
			}
		}
	}

	return false
}

// hasRealCFEndTag reports whether the source actually closed this cf_tag. A
// tag written without one (`<cfmodule ...>`, `<cffeed ...>`, `<cfadmin ...>`)
// has either no end-tag child at all or only the grammar's synthetic
// implicit_cf_end_tag marker — inventing a `</name>` for it re-parents every
// following sibling into the tag's body.
func (f *Formatter) hasRealCFEndTag(n *sitter.Node) bool {
	for i := range n.ChildCount() {
		if n.Child(i).Kind() == "cf_end_tag" {
			return true
		}
	}

	return false
}

// isCFTryBranch reports whether n is a <cfcatch> or <cffinally> block, the// isCFTryBranch reports whether n is a <cfcatch> or <cffinally> block, the
// tags that are indented with their enclosing <cftry> rather than inside it.
func (f *Formatter) isCFTryBranch(n *sitter.Node) bool {
	if n.Kind() != "cf_tag" {
		return false
	}

	switch strings.ToLower(f.tagName(n)) {
	case "cfcatch", "cffinally":
		return true
	}

	return false
}

// ─── CFScript formatting ─────────────────────────────────────────────────────

// formatScriptRegion re-parses one cf_script_content node with the CFScript
// grammar and formats it.
//
// It is a function rather than the body of its caller's loop so that the tree
// is closed when the region is done with, not when the whole <cfscript> element
// is. A tree-sitter tree is C memory the Go collector does not account for, and
// a deferred Close inside a loop runs at function return — so every region in
// an element was held live at once, in the one place already known as the
// server's largest source of untracked memory growth. Today an element has a
// single content child and it costs nothing; the shape is what is wrong.
func (f *Formatter) formatScriptRegion(c *sitter.Node) {
	scriptSrc := f.src[c.StartByte():c.EndByte()]

	tree := f.opts.ParseScript(scriptSrc)
	if tree == nil {
		return
	}

	defer tree.Close()

	if tree.RootNode().HasError() {
		f.recordParseError("cfscript", tree.RootNode(), scriptSrc, c.StartPosition().Row)
	}

	if f.treeTooDeep("cfscript", tree.RootNode()) {
		return
	}

	origSrc := f.src
	f.src = scriptSrc
	f.formatScriptChildren(tree.RootNode())
	f.src = origSrc
}

// formatCFScript pretty-prints the contents of a <cfscript>…</cfscript> block
// by recursing into the cfscript sub-grammar nodes via cfscript_formatter.go.
func (f *Formatter) formatCFScript(n *sitter.Node) {
	f.nl()
	f.writeIndent()
	f.write("<" + f.openTagNameOf("cfscript", n) + ">\n")

	f.level++

	for i := range n.ChildCount() {
		c := n.Child(i)
		if c.Kind() == "cf_script_content" && f.opts.ParseScript != nil {
			f.formatScriptRegion(c)
		}
	}

	f.level--
	f.nl()
	f.writeIndent()
	f.write("</" + f.closeTagNameOf("cfscript", n) + ">\n")
}

func (f *Formatter) formatScriptChildren(n *sitter.Node) {
	prevEndRow := int(n.StartPosition().Row)

	for i := range n.ChildCount() {
		c := n.Child(i)

		startRow := int(c.StartPosition().Row)
		if startRow-prevEndRow > 1 {
			f.write("\n")
		}

		prevEndRow = int(c.EndPosition().Row)
		f.formatScriptNode(c)
	}
}

// ─── hash expression ─────────────────────────────────────────────────────────

// formatHashExpression emits #expr# preserving the delimiters.
// The opening # is an external token not exposed as a child node.
func (f *Formatter) formatHashExpression(n *sitter.Node) {
	f.write("#")

	for i := range n.ChildCount() {
		c := n.Child(i)
		if c.Kind() == "#" {
			f.write("#")
		} else {
			f.write(f.text(c))
		}
	}
}

// ─── component file ──────────────────────────────────────────────────────────

// formatComponentContent formats a script-based component file by re-parsing
// the content with the CFScript grammar.
func (f *Formatter) formatComponentContent(n *sitter.Node) {
	if f.opts.ParseScript != nil {
		scriptSrc := f.src[n.StartByte():n.EndByte()]

		tree := f.opts.ParseScript(scriptSrc)
		if tree != nil {
			if tree.RootNode().HasError() {
				f.recordParseError("cfscript", tree.RootNode(), scriptSrc, n.StartPosition().Row)
			}
			defer tree.Close()

			if f.treeTooDeep("cfscript", tree.RootNode()) {
				return
			}

			origSrc := f.src
			f.src = scriptSrc
			f.formatScriptChildren(tree.RootNode())
			f.src = origSrc

			return
		}
	}

	f.write(f.text(n))
}

// ─── text / comment pass-through ─────────────────────────────────────────────

func (f *Formatter) formatText(n *sitter.Node) {
	raw := f.text(n)
	// Suppress whitespace-only text nodes (spacing between tags is managed by the formatter).
	if trimSpace(raw) == "" {
		return
	}
	// Collapse all whitespace to single spaces (HTML whitespace rules) — unless
	// the text holds a `//` comment, whose line break is load-bearing.
	f.writeText(trimSpace(raw))
}

// formatDoctype emits a <!DOCTYPE ...> or <?xml ...?> declaration verbatim on its
// own line.
//
// The grammar models the doctype body (everything between "DOCTYPE" and ">") as an
// unnamed pattern token that is not exposed as a child node, so the generic
// child-walking path in formatNode would silently reconstruct the node as
// "<!DOCTYPE>" and discard the body. An xml_decl fails there differently but just as
// badly: its parts *are* children ("<?", "xml", tag_attributes..., "?>"), and the
// generic path concatenates them with no separator, turning
// <?xml version="1.0" encoding="utf-8"?> into <?xmlversion="1.0"encoding="utf-8"?>.
// The whitespaceOnly guard cannot catch that — only whitespace was removed — but it
// destroys the declaration and leaves the file unparseable.
//
// Emitting the node's own source text avoids both, and is correct regardless:
// neither declaration has internal structure worth reformatting, and an XML
// declaration is specified down to its spacing.
func (f *Formatter) formatDoctype(n *sitter.Node) {
	raw := trimSpace(f.text(n))
	if raw == "" {
		return
	}

	f.writeIndent()
	f.write(raw)
	f.write("\n")
}

func (f *Formatter) formatComment(n *sitter.Node) {
	raw := f.normalizeCFComment(trimSpace(f.text(n)))
	if raw == "" {
		return
	}

	f.nl()
	f.writeIndent()
	f.write(raw)
	f.write("\n")
}

// normalizeCFComment collapses extra internal whitespace in <!--- ... ---> comments.
func (f *Formatter) normalizeCFComment(raw string) string {
	if !strings.HasPrefix(raw, "<!---") || !strings.HasSuffix(raw, "--->") {
		return raw
	}

	inner := raw[5 : len(raw)-4]

	lines := strings.Split(inner, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	for len(lines) > 0 && trimSpace(lines[0]) == "" {
		lines = lines[1:]
	}

	for len(lines) > 0 && trimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) == 0 {
		return "<!--- --->"
	}

	if len(lines) == 1 {
		return "<!--- " + trimSpace(lines[0]) + " --->"
	}
	// Multi-line: strip common leading whitespace
	minIndent := -1

	for _, line := range lines {
		if trimSpace(line) == "" {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent > 0 {
		for i, line := range lines {
			if len(line) >= minIndent {
				lines[i] = line[minIndent:]
			}
		}
	}

	baseIndent := f.opts.indent(f.level)

	var sb strings.Builder

	sb.WriteString("<!---\n")

	for _, line := range lines {
		if trimSpace(line) == "" {
			sb.WriteString("\n")
		} else {
			sb.WriteString(baseIndent)
			sb.WriteString(f.opts.indent(1))
			sb.WriteString(line)
			sb.WriteByte('\n')
		}
	}

	sb.WriteString(baseIndent)
	sb.WriteString("--->")

	return sb.String()
}

// asciiSpace is the whitespace the formatter may trim from the edge of a piece
// of source. `strings.TrimSpace` also strips Unicode spaces, and U+00A0 is one
// of them: a no-break space at the end of a text run or after a comment is
// content, it renders, and the formatter dropped it (the whitespace-only guard
// then refused the whole file: Galaxie Blog's Udf.cfc).
const asciiSpace = " \t\n\v\f\r"

// trimSpace is strings.TrimSpace for ASCII whitespace only.
func trimSpace(s string) string {
	return strings.Trim(s, asciiSpace)
}

// trimSpaceBytes is bytes.TrimSpace for ASCII whitespace only.
func trimSpaceBytes(b []byte) []byte {
	return bytes.Trim(b, asciiSpace)
}
