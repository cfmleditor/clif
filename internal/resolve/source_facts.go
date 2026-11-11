package resolve

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

type sourceAssignment struct {
	line      int
	name, rhs string
}

type lexicalSourceFacts struct {
	starts      []int
	assignments []sourceAssignment
	queryWords  [][2]int
}

// One pass records line boundaries, word occurrences and line assignments.
// Facts contain no inferred types or caller context and are immutable.
func scanSourceFacts(content string) lexicalSourceFacts {
	facts := lexicalSourceFacts{starts: []int{0}}
	line, from := 0, 0

	// The last character of the last line holding any: after a `(` or a
	// `,` the line goes on an argument list, and `name = x` on it is a named
	// argument, not an assignment. Preside's handlers pass
	// `event = "admin.x"` to runEvent() a line each.
	var prevEnd byte

	addAssignment := func(end int) {
		text := strings.TrimRight(content[from:end], " \t\r")
		if prevEnd != '(' && prevEnd != ',' {
			if name, rhs, ok := matchSourceAssignment(content[from:end]); ok {
				facts.assignments = append(facts.assignments, sourceAssignment{line: line, name: name, rhs: rhs})
			}
		}

		// A comment line is not code: `// Turn code block into array,`.
		if code := strings.TrimLeft(text, " \t"); code != "" && !strings.HasPrefix(code, "//") && !strings.HasPrefix(code, "/*") && !strings.HasPrefix(code, "*") && !strings.HasPrefix(code, "<!---") {
			prevEnd = text[len(text)-1]
		}
	}

	for i := 0; i < len(content); {
		if content[i] == '\n' {
			addAssignment(i)
			i++
			from = i
			line++

			facts.starts = append(facts.starts, i)

			continue
		}

		if lex.IsWord(content[i]) {
			start := i
			for i < len(content) && lex.IsWord(content[i]) {
				i++
			}

			word := content[start:i]
			if strings.EqualFold(word, "query") || strings.EqualFold(word, "cfquery") || strings.EqualFold(word, "cfdbinfo") {
				facts.queryWords = append(facts.queryWords, [2]int{start, i})
			}

			continue
		}

		i++
	}

	addAssignment(len(content))

	return facts
}

func (r *Resolver) lexicalSource(pr *parser.ParseResult) *lexicalSourceFacts {
	if r.indexer != nil {
		return r.indexer.lexicalSource(pr)
	}

	facts := r.sourceFacts.get(&r.mu, pr, func() []lexicalSourceFacts { return []lexicalSourceFacts{scanSourceFacts(pr.Content)} })

	return &facts[0]
}

func (r *Resolver) sourceLineStarts(pr *parser.ParseResult) []int {
	return r.lexicalSource(pr).starts
}

func sourceLine(content string, starts []int, line int) string {
	end := len(content)
	if line+1 < len(starts) {
		end = starts[line+1] - 1
	}

	return content[starts[line]:end]
}

func (r *Resolver) localAssignmentAt(pr *parser.ParseResult, name string, start, header int) (string, int, bool) {
	return r.lexicalSource(pr).assignmentAt(pr.Content, name, start, header)
}

// assignmentAt is localAssignmentAt over facts, the table read from content.
func (facts *lexicalSourceFacts) assignmentAt(content, name string, start, header int) (string, int, bool) {
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

// localAssignmentIn is localAssignment over pr's cached assignment table
// (localAssignmentAt), rather than a walk of the lines above header.
func (r *Resolver) localAssignmentIn(pr *parser.ParseResult, name string, start, header int) (string, bool) {
	rhs, _, ok := r.localAssignmentAt(pr, name, start, header)

	return rhs, ok
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

	return trimAssignmentRHS(parser.WithoutTrailingComment(rhs))
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
