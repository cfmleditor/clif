package resolve

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

// The line reader receiver typing used before it read the parse's assignments
// (typingAssignments), kept as the differential's oracle
// (TestReceiverTypingAgreesWithTheLineReader) and nothing else.

type sourceAssignment struct {
	line      int
	name, rhs string
}

// lineAssignmentTable is the table the resolver read assignments from: each
// line matchSourceAssignment matches, after a line that does not end in `(`
// or `,`.
type lineAssignmentTable struct {
	starts      []int
	assignments []sourceAssignment
}

func scanLineAssignments(content string) lineAssignmentTable {
	facts := lineAssignmentTable{starts: []int{0}}
	line, from := 0, 0

	var prevEnd byte

	addAssignment := func(end int) {
		text := strings.TrimRight(content[from:end], " \t\r")
		if prevEnd != '(' && prevEnd != ',' {
			if name, rhs, ok := matchSourceAssignment(content[from:end]); ok {
				facts.assignments = append(facts.assignments, sourceAssignment{line: line, name: name, rhs: rhs})
			}
		}

		if code := strings.TrimLeft(text, " \t"); code != "" && !strings.HasPrefix(code, "//") && !strings.HasPrefix(code, "/*") && !strings.HasPrefix(code, "*") && !strings.HasPrefix(code, "<!---") {
			prevEnd = text[len(text)-1]
		}
	}

	for i := range len(content) {
		if content[i] == '\n' {
			addAssignment(i)
			from = i + 1
			line++

			facts.starts = append(facts.starts, i+1)
		}
	}

	addAssignment(len(content))

	return facts
}

// assignmentAt is localAssignmentAt over facts, the table read from content.
func (facts *lineAssignmentTable) assignmentAt(content, name string, start, header int) (string, int, bool) {
	starts, assignments := facts.starts, facts.assignments
	for _, a := range slices.Backward(assignments) {
		if a.line < start {
			break
		}

		if a.line >= min(header, len(starts)) || !strings.EqualFold(a.name, name) {
			continue
		}

		rhs := trimAssignmentRHS(a.rhs)
		_, tag := assignmentPrefix(trimAssignmentSpace(sourceLine(content, starts, a.line)), "<cfset")

		for j := a.line + 1; j < min(header, len(starts)) && j <= a.line+maxAssignmentLines; j++ {
			joined, more := joinAssignmentLine(rhs, sourceLine(content, starts, j), tag)
			if !more {
				break
			}

			rhs = joined
		}

		rhs = withoutTrailingComment(rhs, tag)
		if parens, _, _ := parser.OpenGroups(rhs); parens != 0 {
			return "", 0, false
		}

		return rhs, a.line, true
	}

	return "", 0, false
}

// joinAssignmentLine is rhs with the statement's next line added, when the
// statement runs on to it: while a parenthesis is open, and while the line
// carries the chain on. `x = $.getBean( "feed" )` over a line each for
// `.loadBy( … )` and `.getIterator()` is the iterator, not the bean its first
// line names. A conditional runs on while a line ends or starts with its
// `?` or `:`. In a <cfset> the chain's last line holds the tag's end, which
// is left off as the first line's is.
func joinAssignmentLine(rhs, line string, tag bool) (string, bool) {
	next := trimAssignmentRHS(line)

	// A trailing comment is not code: its `?` is not a conditional's, and
	// joined after it, the next line would be read as comment too.
	code := withoutTrailingComment(rhs, tag)

	switch parens, braces, brackets := parser.OpenGroups(code); {
	case parens > 0 || braces > 0 || brackets > 0:
		return strings.TrimSpace(code + " " + next), true
	case strings.HasSuffix(code, "?") || strings.HasSuffix(code, ":") && !strings.HasSuffix(code, "::") ||
		strings.HasPrefix(next, "?") || strings.HasPrefix(next, ":") && !strings.HasPrefix(next, "::"):
		// A conditional written a branch to a line.
		if tag {
			next = trimAssignmentRHS(strings.TrimSuffix(next, ">"))
		}

		return code + " " + next, true
	case strings.HasPrefix(next, ".") && !strings.HasPrefix(next, ".."):
		if tag {
			next = trimAssignmentRHS(strings.TrimSuffix(next, ">"))
		}

		return code + next, true
	default:
		return rhs, false
	}
}

// withoutTrailingComment is rhs without the comment that ends its line. In a
// <cfset> line `//` starts no comment, and the text after the tag (markup, a
// URL) is not cut at it.
func withoutTrailingComment(rhs string, tag bool) string {
	if tag {
		return rhs
	}

	return trimAssignmentRHS(textWithoutTrailingComment(rhs))
}

// textWithoutTrailingComment is s up to the end of its last token, read as
// CFScript: a trailing `//` or `/* */` comment is left off. It was the
// parser's WithoutTrailingComment, which only this reader used.
func textWithoutTrailingComment(s string) string {
	if !strings.Contains(s, "//") && !strings.Contains(s, "/*") {
		return s
	}

	sc := parser.NewScriptScanner(s)
	end := 0

	for {
		t := sc.NextSkipComments()
		if t.Kind == parser.TokEOF {
			return s[:end]
		}

		end = t.Offset + len(t.Value)
	}
}

func trimAssignmentRHS(text string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(text), ";"), "/"))
}

// trimAssignmentSpace drops leading \s. The line-assignment grammar this
// replaced was a regular expression, and its whitespace and identifier
// characters deliberately keep that expression's ASCII rules.
func trimAssignmentSpace(s string) string {
	for s != "" && lex.IsRegexpSpace(s[0]) {
		s = s[1:]
	}

	return s
}

func assignmentPrefix(s, prefix string) (string, bool) {
	at := 0
	for _, want := range prefix {
		if at >= len(s) {
			return s, false
		}

		got, n := utf8.DecodeRuneInString(s[at:])
		if !strings.EqualFold(string(got), string(want)) {
			return s, false
		}

		at += n
	}

	return s[at:], true
}

func matchSourceAssignment(line string) (name, rhs string, ok bool) {
	line = trimAssignmentSpace(line)
	if rest, ok := assignmentPrefix(line, "<cfset"); ok && rest != "" && lex.IsRegexpSpace(rest[0]) {
		line = trimAssignmentSpace(rest)
	}

	if rest, ok := assignmentPrefix(line, "var"); ok && rest != "" && lex.IsRegexpSpace(rest[0]) {
		if name, rhs, matched := matchAssignmentBody(trimAssignmentSpace(rest)); matched {
			return name, rhs, true
		}
	}

	return matchAssignmentBody(line)
}

func matchAssignmentBody(line string) (name, rhs string, ok bool) {
	for _, prefix := range []string{"local.", "variables."} {
		if rest, ok := assignmentPrefix(line, prefix); ok {
			line = rest

			break
		}
	}

	end := 0
	for end < len(line) && lex.IsWord(line[end]) {
		end++
	}

	if end == 0 {
		return "", "", false
	}

	name = line[:end]

	rest := trimAssignmentSpace(line[end:])
	if rest == "" || rest[0] != '=' {
		return "", "", false
	}

	rest = rest[1:]

	start := 0
	for start < len(rest) && lex.IsRegexpSpace(rest[start]) {
		start++
	}
	// The former expression permitted whitespace to be the first RHS byte.
	if start == len(rest) || rest[start] == '=' {
		if start == 0 {
			return "", "", false
		}

		start--
	}

	end = len(rest)
	trimEnd := func() {
		for end > start+1 && lex.IsRegexpSpace(rest[end-1]) {
			end--
		}
	}
	trimEnd()

	if end > start+1 && rest[end-1] == ';' {
		end--
	}

	trimEnd()

	if end > start+1 && rest[end-1] == '>' {
		end--
	}

	if end > start+1 && rest[end-1] == '/' {
		end--
	}

	trimEnd()

	return name, rest[start:end], true
}

// localAssignmentAt is localAssignment with the source line retained for
// consumers that must prove the assignment reaches a particular use.
func localAssignmentAt(content, name string, start, header int) (string, int, bool) {
	start = max(start, 0)

	// The lines before header, walked backwards in place: copying them out,
	// often from the top of the file, was a tenth of what a tassweb scan
	// allocated.
	region, end := content, strings.Count(content, "\n")+1
	if at, ok := lineOffset(content, header); ok {
		region, end = content[:max(at-1, 0)], header
	}

	stop := len(region)

	for i := end - 1; i >= start; i-- {
		from := strings.LastIndexByte(region[:stop], '\n') + 1
		line := region[from:stop]
		next := stop + 1 // where line i+1 starts
		stop = max(from-1, 0)

		assignedName, value, matched := matchSourceAssignment(line)
		if !matched || !strings.EqualFold(assignedName, name) {
			continue
		}

		rhs := trimAssignmentRHS(value)
		_, tag := assignmentPrefix(trimAssignmentSpace(line), "<cfset")

		for j := i + 1; j < end && j <= i+maxAssignmentLines; j++ {
			following := region[min(next, len(region)):]
			if k := strings.IndexByte(following, '\n'); k >= 0 {
				following = following[:k]
			}

			joined, more := joinAssignmentLine(rhs, following, tag)
			if !more {
				break
			}

			next += len(following) + 1
			rhs = joined
		}

		rhs = withoutTrailingComment(rhs, tag)
		if parens, _, _ := parser.OpenGroups(rhs); parens != 0 {
			return "", 0, false
		}

		return rhs, i, true
	}

	return "", 0, false
}

const maxAssignmentLines = 24

// Assignment recovery also matches named arguments and struct keys. Those
// names belong to their expression, not the caller's variable. This guard is
// bounded to separate statement lines; it is not branch dominance analysis.
func lineNativeReassignedAt(content, name string, start, header int) bool {
	for header > start {
		_, at, assigned := localAssignmentAt(content, name, start, header)
		if !assigned {
			return false
		}

		shape, ok := statementShape(contentBeforeLine(content, at))
		if !ok {
			return false
		}

		if shape.parens == 0 && shape.brackets == 0 && !shape.expressionKey() {
			return true
		}

		header = at
	}

	return false
}

// textShape is what nativeReassignedAt reads of the text before an
// assignment: whether its brackets balance and how it ends.
type textShape struct {
	parens, brackets int
	last, prev       parser.TokenKind
	tokens           int
}

// statementShape scans source once, keeping only its shape: it used to build
// a token slice of everything above the assignment to read three facts from
// it, half of producerTokens' allocation. Like producerTokens it gives up past
// 4,096 tokens.
func statementShape(source string) (textShape, bool) {
	var shape textShape

	scanner := parser.NewSourceScanner(source)

	for {
		tok := scanner.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			return shape, true
		}

		shape.tokens++
		if shape.tokens > 4096 {
			return textShape{}, false
		}

		switch tok.Kind {
		case parser.TokLParen:
			shape.parens++
		case parser.TokRParen:
			shape.parens--
		case parser.TokLBracket:
			shape.brackets++
		case parser.TokRBracket:
			shape.brackets--
		default:
		}

		shape.prev, shape.last = shape.last, tok.Kind
	}
}

// expressionKey reports whether the text ends where an assignment would be a
// named argument or a struct key rather than a statement.
func (s textShape) expressionKey() bool {
	if s.tokens == 0 {
		return false
	}

	switch s.last {
	case parser.TokComma, parser.TokLParen, parser.TokLBracket, parser.TokEquals:
		return true
	case parser.TokLBrace:
		if s.tokens > 1 {
			switch s.prev {
			case parser.TokEquals, parser.TokComma, parser.TokLParen, parser.TokLBracket:
				return true
			default:
			}
		}
	default:
	}

	return false
}

// lineNativeClaimMayNotReach is nativeClaimMayNotReach as the line reader
// answered it.
func lineNativeClaimMayNotReach(content, name string, scopeStart, refLine, useLine int) bool {
	if useLine <= refLine || !blockClosedBetween(content, refLine, useLine) {
		return false
	}

	_, _, earlier := localAssignmentAt(content, name, scopeStart, refLine)

	return earlier
}
