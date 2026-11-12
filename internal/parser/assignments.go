package parser

import (
	"slices"
	"strings"
)

// The resolver asks of a file, a function or a few lines whether they write a
// name, and what they write it: rc.x in an FW/1 action, prc.x before a view
// renders, variables.field in a lazy getter, a name a template sets before it
// is included. It answered by walking the scanner's tokens itself, once per
// question, over the whole text: markup, a <cfquery>'s SQL and a Java block
// read as code, and each reader with its own idea of where a statement or a
// value ends.
//
// The parse reads those writes instead, over its own regions: each script
// region's tokens with its SQL and Java islands blanked (blankNonScript), the
// body of each <cfset> the walk's tag reader finds (CFTags), and the
// attributes of the CF tags that name a variable they fill. It takes the
// parse's rules: a write stands only where a statement can
// (atStatementLevel), so a named argument and a struct literal's key are
// none; `ident ident =` is a script-syntax tag whose attributes are not
// assignments (looksLikeTagAttrs); and a value ends where the parse ends an
// expression (newlineEndsStatement, a ";", a "," or the bracket around it).
// It is recorded only when asked: ParseOptions.RecordAssignments records it
// during the parse, and Assignments reads it from any parse, once per
// version; AssignmentsOf reads it from text there is no parse of. A record
// is a function of the text alone, so the function holding one is the
// parse's FuncScopeAt(Line).

// AssignOp is how an Assignment writes its target.
type AssignOp uint8

const (
	// AssignSet is `target = value`.
	AssignSet AssignOp = iota
	// AssignCompound is `target op= value`: +=, -=, *=, /=, &=, %= or ^=.
	AssignCompound
	// AssignStep is `target++` or `target--`, with no value.
	AssignStep
	// AssignVar is `var target` with no value: a declaration.
	AssignVar
	// AssignForIn is the loop variable of `for ( [var] target in … )`.
	AssignForIn
	// AssignAttribute is an attribute that names the variable its tag fills
	// (variableAttributes): a CF tag's, a script-syntax tag's, or a call's
	// named argument (cfhttp( result = "r" )). Target is the attribute's
	// name, lowercased, and Literal the variable it names when it is a
	// quoted string.
	AssignAttribute
)

// Assignment is one write the parse read.
type Assignment struct {
	// Target is the name written, as written: its dotted names joined by "."
	// with no space, a scope included (variables.x, arguments.rc.user), and
	// each [ … ] key as its text (prc[ "x" ], a.b[ i ].c). For an
	// AssignAttribute it is the attribute's name, lowercased.
	Target string
	// Operator is the operator as written: "=", "+=", "++", …; "" for
	// AssignVar, AssignForIn and AssignAttribute.
	Operator string
	// Tag is the CF tag the write was read from, lowercased ("cfset",
	// "cfloop", …), or "" in script; for an AssignAttribute in script, the
	// script-syntax tag ("param", "lock") or the function called.
	Tag string
	// Literal is the value when it is one quoted string and nothing else,
	// without its quotes, as written; HasLiteral says it is one.
	Literal string
	// At is the offset of the target's first byte in the content (a CF
	// tag's attribute: its value's), and Line its 0-based line.
	At, Line int
	// Statement is where the statement holding the write begins: At when the
	// write is the first thing its statement does, the `var` before it, or
	// something earlier (`a = b = 1`, `f( x = 1 )`).
	Statement int
	// Value is the [start, end) offsets of the value written, from its first
	// token's first byte to its last token's last; start == end for none.
	Value      [2]int
	Op         AssignOp
	HasLiteral bool
	// Local is a `var` declaration's.
	Local bool
}

// StartsStatement reports whether the write is the first thing its statement
// does: not the inner write of `a = b = 1`, a named argument, or a write a
// `var` declares.
func (a *Assignment) StartsStatement() bool { return a.Statement == a.At }

// ValueText is the value's text in content, the text the parse read.
func (a *Assignment) ValueText(content string) string {
	if a.Value[1] <= a.Value[0] || a.Value[1] > len(content) {
		return ""
	}

	return content[a.Value[0]:a.Value[1]]
}

// ValueLine is the value's text written as one line, up to the offset limit in
// content: its tokens as written, where what separates two of them on
// different lines, or holds a comment, is one space, or nothing before a `.`
// that carries a chain on (`svc.load()` over a line each for `.iterate()` is
// `svc.load().iterate()`). A value that runs on past limit is cut there, so a
// call whose arguments go on past it is left open. Receiver typing reads a
// value as text, and this is the text it read when it joined an assignment's
// lines itself.
func (a *Assignment) ValueLine(content string, limit int) string {
	from, to := a.Value[0], min(a.Value[1], limit, len(content))
	if to <= from {
		return ""
	}

	src := content[from:to]
	if !strings.ContainsAny(src, "\n/*<") {
		return src
	}

	var b strings.Builder

	b.Grow(len(src))

	prev := -1

	for _, t := range tokenize(src) {
		if t.Kind == TokEOF {
			break
		}

		if prev >= 0 {
			gap := src[prev:t.Offset]

			switch {
			case gap == "" || !strings.ContainsAny(gap, "\n/*<"):
				b.WriteString(gap)
			case t.Kind == TokDot && strings.Contains(gap, "\n"):
			default:
				b.WriteByte(' ')
			}
		}

		b.WriteString(t.Value)
		prev = t.Offset + len(t.Value)
	}

	return b.String()
}

// assignments is Assignments' answer for the content of.
type assignments struct {
	of   string
	list []Assignment
}

const assignmentsMemoKey = "parser.assignments"

// Assignments is every write the parse reads in the file, in the order they
// are written. A parse made with ParseOptions.RecordAssignments recorded them
// as it ran; any other reads them once per version of its content. What is
// returned is shared and must not be changed.
func (pr *ParseResult) Assignments() []Assignment {
	if pr == nil {
		return nil
	}

	if a := pr.assignments; a != nil && a.of == pr.Content {
		return a.list
	}

	list, _ := pr.Memo(assignmentsMemoKey, func() any { return pr.readAssignments() }).([]Assignment)

	return list
}

// readAssignments reads the assignments of pr's content over its regions,
// split afresh when an edit has moved the content on from them.
func (pr *ParseResult) readAssignments() []Assignment {
	regions, lines := pr.Regions, pr.contentLineIdx
	if pr.regionsOf != pr.Content {
		regions, lines = pr.classifyRegions()
	}

	return recordAssignments(pr.Content, regions, lines)
}

// AssignmentsOf is Assignments for text there is no parse of: the
// assignments-only reading of the parse's regions. file is the URI or path
// the text came from, or "": it is read only for its extension, as the parse
// reads it, so a tag-free .cfm page of markup is not read as script.
func AssignmentsOf(file, content string) []Assignment {
	regions, lines := ClassifyRegionsIdx(content)
	if isMarkupTemplate(file, content) {
		regions, lines = splitCFScriptBlocks(content)
	}

	return recordAssignments(content, regions, lines)
}

// recordAssignments reads the assignments in content's regions, in order.
func recordAssignments(content string, regions []Region, lines LineTable) []Assignment {
	if lines == nil {
		lines = NewLineTable(content)
	}

	var out []Assignment

	for i := range regions {
		r := &regions[i]

		switch r.Kind {
		case RegionScript:
			src := blankNonScript(r.Text)
			w := assignmentWalk{src: src, lines: lines, base: r.Offset, toks: tokenize(src)}
			out = w.walk(out)
		case RegionTag:
			out = recordTagAssignments(r, lines, out)
		default:
		}
	}

	slices.SortStableFunc(out, func(a, b Assignment) int { return a.At - b.At })

	return out
}

// variableAttributes are the attributes whose quoted value names the variable
// a CF tag fills: <cfloop index="i">, <cfquery name="q">, <cfinvoke
// returnVariable="r">, <cfsavecontent variable="s">, <cfhttp result="h">.
var variableAttributes = [...]string{"item", "index", "name", "returnvariable", "variable", "result"}

// recordTagAssignments reads the assignments of a tag region: each <cfset>'s
// body as script, and the attributes of every other CF tag that name a
// variable.
func recordTagAssignments(r *Region, lines LineTable, out []Assignment) []Assignment {
	for _, t := range CFTags(r.Text) {
		if t.Close {
			continue
		}

		start := r.Offset + t.Start

		if t.Name == "cfset" {
			body := t.Body()
			if body == "" {
				continue
			}

			at := start + strings.Index(t.text, body)
			w := assignmentWalk{src: body, lines: lines, base: at, toks: tokenize(body), tag: "cfset"}
			out = w.walk(out)

			continue
		}

		for _, attr := range variableAttributes {
			s, e, ok := tagAttrValue(t.text, attr, cfAttrs)
			if !ok || s == 0 || e >= len(t.text) || !isQuoteByte(t.text[s-1]) || t.text[e] != t.text[s-1] {
				continue
			}

			at := start + s
			out = append(out, Assignment{
				Target: attr, Tag: t.Name, Literal: t.text[s:e], HasLiteral: true,
				At: at, Line: lines.LineOf(at), Statement: at, Value: [2]int{at, start + e}, Op: AssignAttribute,
			})
		}
	}

	return out
}

func isQuoteByte(c byte) bool { return c == '"' || c == '\'' }

// assignmentWalk reads the assignments in one run of script tokens: a script
// region's, or a <cfset>'s body, starting at base in content.
type assignmentWalk struct {
	src   string // the text toks were read from, at base in the content
	lines LineTable
	toks  []Token
	tag   string
	base  int

	// statement is the index of the token the current statement begins at.
	statement int
	// opens are the open brackets, each with what it holds (bracketKind), and
	// openAt the index of each.
	opens  []bracketKind
	openAt []int
	// attrs is, when the current statement is a script-syntax tag (`param
	// name="x" default="";`, `lock name="l" { … }`), its name lowercased:
	// what follows it is a run of attributes, not assignments.
	attrs string
	// labelled is whether the current statement began with case or default,
	// whose ':' ends the label and begins a statement.
	labelled bool
}

// walk appends the assignments in w.toks to out.
func (w *assignmentWalk) walk(out []Assignment) []Assignment {
	toks := w.toks

	for i := 0; i < len(toks) && toks[i].Kind != TokEOF; i++ {
		t := toks[i]
		if i > 0 && w.endsStatementBefore(i) {
			w.statement, w.labelled = i, false
		}

		if i == w.statement {
			w.labelled = t.Kind == TokIdent && classify(t.Value)&wordCase != 0
			w.attrs = w.scriptTagAt(i)
		}

		w.track(i)

		if t.Kind != TokIdent || i > 0 && (toks[i-1].Kind == TokDot || toks[i-1].Kind == TokDoubleColon) {
			continue
		}

		out = w.at(i, out)
	}

	return out
}

// scriptTagAt is the name of the script-syntax tag a statement starting at
// toks[i] is, lowercased, or "": `ident ident =`, the parse's reading of one
// (looksLikeTagAttrs, startsAttributes).
func (w *assignmentWalk) scriptTagAt(i int) string {
	toks := w.toks
	if i+2 >= len(toks) || toks[i].Kind != TokIdent || toks[i+1].Kind != TokIdent || toks[i+2].Kind != TokEquals {
		return ""
	}

	if !startsAttributes(classify(toks[i].Value), classify(toks[i+1].Value)) {
		return ""
	}

	return strings.ToLower(toks[i].Value)
}

// attribute reads the attribute of a script-syntax tag at toks[i], `name =
// value`, recording it when it names the variable the tag fills.
func (w *assignmentWalk) attribute(i int, out []Assignment) []Assignment {
	toks := w.toks
	if i == w.statement || i+2 >= len(toks) || toks[i+1].Kind != TokEquals || !slices.ContainsFunc(variableAttributes[:], func(a string) bool { return identEq(toks[i].Value, a) }) {
		return out
	}

	at := w.base + toks[i].Offset
	a := Assignment{Target: strings.ToLower(toks[i].Value), Tag: w.attrs, At: at, Line: w.lines.LineOf(at), Statement: w.base + toks[w.statement].Offset, Op: AssignAttribute}

	v := toks[i+2]
	a.Value = [2]int{w.base + v.Offset, w.base + v.Offset + len(v.Value)}

	if v.Kind == TokString && len(v.Value) >= 2 {
		a.Literal, a.HasLiteral = v.Value[1:len(v.Value)-1], true
	}

	return append(out, a)
}

// endsStatementBefore reports whether a statement begins at toks[i]: after a
// ";" or a brace, a condition's closing parenthesis, else or do, a case
// label's ":", or a line break where CFScript ends a statement, which a
// script-syntax tag's attributes run on over.
func (w *assignmentWalk) endsStatementBefore(i int) bool {
	prev, t := w.toks[i-1], w.toks[i]

	switch prev.Kind {
	case TokSemicolon, TokLBrace, TokRBrace:
		return true
	case TokColon:
		return w.labelled
	case TokIdent:
		if identEq(prev.Value, "else") || identEq(prev.Value, "do") {
			return true
		}
	default:
	}

	return w.attrs == "" && newlineEndsStatement(w.src, prev, t)
}

// bracketKind is what an open bracket holds.
type bracketKind uint8

const (
	// bracketBlock is a brace around statements: a function's, a block's, a
	// closure's.
	bracketBlock bracketKind = iota
	// bracketFor is the parentheses of a for loop, whose parts are statements.
	bracketFor
	// bracketCondition is the parentheses after if, while, catch or switch: the
	// statement they govern begins after them.
	bracketCondition
	// bracketCall is a call's argument list: `name = value` in it is a named
	// argument, not an assignment.
	bracketCall
	// bracketValue is any other bracket: a grouping, an index, a struct or array
	// literal, whose `key = value` is a key and not an assignment.
	bracketValue
)

// track follows the brackets at toks[i]: what each holds, and the statement
// a condition governs, which begins after it closes.
func (w *assignmentWalk) track(i int) {
	t := w.toks[i]

	switch t.Kind {
	case TokLParen, TokLBracket, TokLBrace:
		w.opens, w.openAt = append(w.opens, w.opened(i)), append(w.openAt, i)
	case TokRParen, TokRBracket, TokRBrace:
		if n := len(w.opens); n > 0 {
			kind := w.opens[n-1]
			w.opens, w.openAt = w.opens[:n-1], w.openAt[:n-1]

			if kind == bracketCondition && i+1 < len(w.toks) {
				w.statement, w.labelled = i+1, false
			}
		}
	default:
	}
}

// opened is what the bracket at toks[i] holds. A brace is a block unless it
// follows what only a value follows (structLiteralAt); a parenthesis after a
// name or a closing bracket is a call's, or a for loop's or a condition's
// after those words.
func (w *assignmentWalk) opened(i int) bracketKind {
	t := w.toks[i]

	switch t.Kind {
	case TokLBrace:
		// A case label's block: `case "x": {`.
		if i > 0 && w.toks[i-1].Kind == TokColon && w.statement == i {
			return bracketBlock
		}

		if structLiteralAt(w.src, t.Offset) {
			return bracketValue
		}

		return bracketBlock
	case TokLParen:
		if i == 0 {
			return bracketValue
		}

		switch prev := w.toks[i-1]; prev.Kind {
		case TokIdent:
			switch {
			case identEq(prev.Value, "for"):
				return bracketFor
			case identEq(prev.Value, "if") || identEq(prev.Value, "while") || identEq(prev.Value, "catch") || identEq(prev.Value, "switch"):
				return bracketCondition
			case classify(prev.Value)&wordOperator != 0 || identEq(prev.Value, "return"):
				return bracketValue
			default:
				return bracketCall
			}
		case TokRParen, TokRBracket:
			return bracketCall
		default:
			return bracketValue
		}
	default:
		return bracketValue
	}
}

// writable reports whether a write may stand where the walk is: in a block,
// a for loop's parentheses or outside any bracket, not in an argument list,
// a literal or a grouping. It is the parse's atStatementLevel.
func (w *assignmentWalk) writable() bool {
	n := len(w.opens)

	return n == 0 || w.opens[n-1] == bracketBlock || w.opens[n-1] == bracketFor
}

// namedArgument reads the argument of a call at toks[i], `name = value`,
// recording it when it names the variable the call fills, as a tag's
// attribute does: cfhttp( url = u, result = "r" ).
func (w *assignmentWalk) namedArgument(i int, out []Assignment) []Assignment {
	toks := w.toks
	if len(w.opens) == 0 || w.opens[len(w.opens)-1] != bracketCall || i == 0 || toks[i-1].Kind != TokLParen && toks[i-1].Kind != TokComma {
		return out
	}

	callee := ""
	if at := w.openAt[len(w.openAt)-1]; at > 0 && toks[at-1].Kind == TokIdent {
		callee = strings.ToLower(toks[at-1].Value)
	}

	saved := w.attrs
	w.attrs = callee
	out = w.attribute(i, out)
	w.attrs = saved

	return out
}

// at reads an assignment starting at the name toks[i], if one does: `var`
// and a name, a for-in loop's variable, or a target and an operator.
func (w *assignmentWalk) at(i int, out []Assignment) []Assignment {
	toks := w.toks
	if w.attrs != "" {
		return w.attribute(i, out)
	}

	if !w.writable() {
		return w.namedArgument(i, out)
	}

	if i > 0 && isIdentTok(toks[i-1], "var") {
		return out // read with its var
	}

	stmt := toks[w.statement].Offset
	local := false
	start := i

	if identEq(toks[i].Value, "var") && i+1 < len(toks) && toks[i+1].Kind == TokIdent {
		local, start = true, i+1
	}

	end, target := w.target(start)
	if end < 0 {
		return out
	}

	a := Assignment{Target: target, Tag: w.tag, At: w.base + toks[start].Offset, Statement: w.base + stmt, Local: local}
	a.Line = w.lines.LineOf(a.At)
	a.Value = [2]int{a.At, a.At}

	// for ( [var] x in …
	if forIn := start - 1 - btoi(local); forIn >= 1 && toks[forIn].Kind == TokLParen && isIdentTok(toks[forIn-1], "for") && isIdentTok(toks[end], "in") {
		a.Op = AssignForIn

		return append(out, a)
	}

	op, width := w.operator(end)
	if width == 0 {
		if local && start+1 == end {
			a.Op = AssignVar

			return append(out, a)
		}

		return out
	}

	a.Operator = op

	switch {
	case op == "=":
		a.Op = AssignSet
	case len(op) == 2 && op[0] == op[1]:
		a.Op = AssignStep

		return append(out, a)
	default:
		a.Op = AssignCompound
	}

	from := end + width
	to := w.valueEnd(from)

	if from < to {
		last := toks[to-1]
		a.Value = [2]int{w.base + toks[from].Offset, w.base + last.Offset + len(last.Value)}

		if to == from+1 && toks[from].Kind == TokString && len(toks[from].Value) >= 2 {
			a.Literal, a.HasLiteral = toks[from].Value[1:len(toks[from].Value)-1], true
		}
	}

	return append(out, a)
}

func btoi(b bool) int {
	if b {
		return 1
	}

	return 0
}

// isIdentTok reports whether t is the identifier word, ignoring case.
func isIdentTok(t Token, word string) bool {
	return t.Kind == TokIdent && identEq(t.Value, word)
}

// target reads the target written from toks[i]: a name, then any run of
// .name and [ … ] steps. It answers the index after it and the target as
// Assignment.Target spells it, or -1.
func (w *assignmentWalk) target(i int) (int, string) {
	toks := w.toks

	var b strings.Builder

	b.WriteString(toks[i].Value)

	j := i + 1

	for j < len(toks) {
		switch toks[j].Kind {
		case TokDot:
			if j+1 >= len(toks) || toks[j+1].Kind != TokIdent {
				return -1, ""
			}

			b.WriteByte('.')
			b.WriteString(toks[j+1].Value)

			j += 2
		case TokLBracket:
			closing := bracketEnd(toks, j)
			if closing < 0 {
				return -1, ""
			}

			last := toks[closing]
			b.WriteString(w.src[toks[j].Offset : last.Offset+len(last.Value)])

			j = closing + 1
		default:
			return j, b.String()
		}
	}

	return j, b.String()
}

// bracketEnd is the index of the ']' closing the '[' at toks[i], or -1.
func bracketEnd(toks []Token, i int) int {
	depth := 0

	for j := i; j < len(toks); j++ {
		switch toks[j].Kind {
		case TokLBracket:
			depth++
		case TokRBracket:
			depth--
			if depth == 0 {
				return j
			}
		case TokEOF:
			return -1
		default:
		}
	}

	return -1
}

// adjacentToks reports whether b is written straight after a.
func adjacentToks(a, b Token) bool { return a.Offset+len(a.Value) == b.Offset }

// operator reads an assignment's operator at toks[i]: "=" (not "=="),
// op= with op one of + - * / & % ^, or ++ or --, and how many tokens it
// takes; width 0 for none.
func (w *assignmentWalk) operator(i int) (string, int) {
	toks := w.toks
	if i >= len(toks) {
		return "", 0
	}

	equalsAt := func(j int) bool {
		return j < len(toks) && toks[j].Kind == TokEquals &&
			(j+1 >= len(toks) || toks[j+1].Kind != TokEquals || !adjacentToks(toks[j], toks[j+1]))
	}

	switch t := toks[i]; t.Kind {
	case TokEquals:
		if equalsAt(i) && i+1 < len(toks) && toks[i+1].Kind != TokEOF {
			return "=", 1
		}
	case TokPlus, TokMinus:
		if i+1 < len(toks) && toks[i+1].Kind == t.Kind && adjacentToks(t, toks[i+1]) {
			return t.Value + toks[i+1].Value, 2
		}

		fallthrough
	case TokStar, TokSlash, TokAmpersand, TokPercent, TokCaret:
		if i+1 < len(toks) && adjacentToks(t, toks[i+1]) && equalsAt(i+1) && i+2 < len(toks) && toks[i+2].Kind != TokEOF {
			return t.Value + "=", 2
		}
	default:
	}

	return "", 0
}

// valueEnd is the index after a value starting at toks[from]: at a ";", a
// "," or a bracket closing around it, or a line break where CFScript ends a
// statement, outside any bracket the value opens.
func (w *assignmentWalk) valueEnd(from int) int {
	toks := w.toks
	src := w.src
	depth := 0

	for i := from; i < len(toks); i++ {
		t := toks[i]

		if depth == 0 && i > from && newlineEndsStatement(src, toks[i-1], t) {
			return i
		}

		switch t.Kind {
		case TokEOF:
			return i
		case TokLParen, TokLBracket, TokLBrace:
			depth++
		case TokRParen, TokRBracket, TokRBrace:
			if depth == 0 {
				return i
			}

			depth--
		case TokSemicolon, TokComma:
			if depth == 0 {
				return i
			}
		default:
		}
	}

	return len(toks)
}
