package resolve

import (
	"slices"
	"strings"
	"unicode/utf8"

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
	addAssignment := func(end int) {
		if name, rhs, ok := matchSourceAssignment(content[from:end]); ok {
			facts.assignments = append(facts.assignments, sourceAssignment{line: line, name: name, rhs: rhs})
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

		if sourceWord(content[i]) {
			start := i
			for i < len(content) && sourceWord(content[i]) {
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
	facts := r.lexicalSource(pr)

	starts, assignments := facts.starts, facts.assignments
	for _, a := range slices.Backward(assignments) {
		if a.line < start {
			break
		}

		if a.line >= min(header, len(starts)) || !strings.EqualFold(a.name, name) {
			continue
		}

		rhs := trimAssignmentRHS(a.rhs)
		_, tag := assignmentPrefix(trimAssignmentSpace(sourceLine(pr.Content, starts, a.line)), "<cfset")

		for j := a.line + 1; j < min(header, len(starts)) && j <= a.line+maxAssignmentLines; j++ {
			joined, more := joinAssignmentLine(rhs, sourceLine(pr.Content, starts, j), tag)
			if !more {
				break
			}

			rhs = joined
		}

		rhs = trimAssignmentRHS(withoutLineComment(rhs))
		if strings.Count(rhs, "(") != strings.Count(rhs, ")") {
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
	code := trimAssignmentRHS(withoutLineComment(rhs))

	switch {
	case strings.Count(code, "(") > strings.Count(code, ")"):
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

// withoutLineComment is s without a trailing `//` comment. Strings are
// stepped over as CFScript writes them: a doubled quote stays inside, and a
// `#...#` in one is an expression whose own strings may hold `//`. A
// `/* … */` comment is stepped over too, so a quote in it opens nothing.
func withoutLineComment(s string) string {
	if !strings.Contains(s, "//") {
		return s
	}

	if at, _, _ := lineCommentAt(s, 0, 0); at >= 0 {
		return s[:at]
	}

	return s
}

// maxCommentScanNesting bounds the strings inside interpolations
// lineCommentAt follows, as maxStringNesting does the scanner's.
const maxCommentScanNesting = 16

// lineCommentAt is the offset in s of the first `//` comment from i, or -1,
// and where the scan stopped. With depth > 0 it is inside a `#...#`, ends at
// its closing `#` (closed) and finds no comment.
func lineCommentAt(s string, i, depth int) (comment, end int, closed bool) {
	for i < len(s) {
		switch c := s[i]; {
		case depth > 0 && c == '#':
			return -1, i + 1, true
		case c == '"' || c == '\'':
			i = stringEnd(s, i+1, c, depth)
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			j := strings.Index(s[i+2:], "*/")
			if j < 0 {
				return -1, len(s), false
			}

			i += j + 4
		case depth == 0 && c == '/' && i+1 < len(s) && s[i+1] == '/':
			return i, i, false
		default:
			i++
		}
	}

	return -1, len(s), false
}

// stringEnd is the offset just past the string whose body starts at i.
func stringEnd(s string, i int, quote byte, depth int) int {
	for i < len(s) {
		switch s[i] {
		case quote:
			if i+1 < len(s) && s[i+1] == quote {
				i += 2

				continue
			}

			return i + 1
		case '#':
			if i+1 < len(s) && s[i+1] == '#' {
				i += 2

				continue
			}

			// A `#` whose expression never closes is a lone hash, which
			// Lucee tolerates (`"# Report"`): a character of the string.
			if depth >= maxCommentScanNesting {
				i++

				continue
			}

			if _, end, closed := lineCommentAt(s, i+1, depth+1); closed {
				i = end
			} else {
				i++
			}
		default:
			i++
		}
	}

	return i
}

func trimAssignmentRHS(text string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(text), ";"), "/"))
}

// Preserve the old line-assignment grammar without a regular expression.
// Whitespace and identifier characters deliberately retain its ASCII rules.
func assignmentSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n' || b == '\f'
}

func trimAssignmentSpace(s string) string {
	for s != "" && assignmentSpace(s[0]) {
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
	if rest, ok := assignmentPrefix(line, "<cfset"); ok && rest != "" && assignmentSpace(rest[0]) {
		line = trimAssignmentSpace(rest)
	}

	if rest, ok := assignmentPrefix(line, "var"); ok && rest != "" && assignmentSpace(rest[0]) {
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
	for end < len(line) && sourceWord(line[end]) {
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
	for start < len(rest) && assignmentSpace(rest[start]) {
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
		for end > start+1 && assignmentSpace(rest[end-1]) {
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

func sourceWord(c byte) bool { return callerWord(c) && c != '$' }
