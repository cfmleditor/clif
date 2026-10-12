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
		for j := a.line + 1; strings.Count(rhs, "(") > strings.Count(rhs, ")") && j < min(header, len(starts)) && j <= a.line+maxAssignmentLines; j++ {
			rhs = strings.TrimSpace(rhs + " " + trimAssignmentRHS(sourceLine(pr.Content, starts, j)))
		}

		if strings.Count(rhs, "(") != strings.Count(rhs, ")") {
			return "", 0, false
		}

		return rhs, a.line, true
	}

	return "", 0, false
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
