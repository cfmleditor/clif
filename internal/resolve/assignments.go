package resolve

import (
	"slices"
	"sort"
	"strings"
	"unsafe"

	"github.com/cfmleditor/clif/internal/parser"
)

// The resolver asks of a file, a function or a few lines whether they write a
// name (rc.x in an FW/1 action, prc.x before a view renders, variables.field
// in a lazy getter, a name a template sets before it is included) and what
// they write it. It walked the scanner's tokens for each question (codeWrites),
// over the whole text: a write in markup, in a <cfquery>'s SQL or in an
// HTML tag's name= attribute was one, and where a value ended was the reader's
// own rule.
//
// It reads the parse's assignments now (parser.Assignment): a parse it holds
// answers through ParseResult.Assignments, and a file it has only the path or
// text of through the file's source (sourceInfo.assignments) or, for text
// that is no file's, parser.AssignmentsOf. A writeRule says which of them a
// question takes.

// writeRule says which writes to a name a reader takes.
type writeRule struct {
	// scopes may stand before the name, at most one of them: variables.x
	// for x.
	scopes []string
	// qualified takes the name after any other qualifier as well: a.rc.x for
	// rc.x. Otherwise a name after a dot, other than one of scopes, is some
	// other variable's member.
	qualified bool
	// member takes a write to one member of the name, x.y = or x[k] =, and
	// no other.
	member bool
	// compound takes +=, -=, *=, /=, &=, ++ and -- as well as =.
	compound bool
}

// writesTo are the writes in as to name, a variable's name as written, dotted
// or not, that rule takes, in the order they are written.
func writesTo(as []parser.Assignment, name string, rule writeRule) []*parser.Assignment {
	parts := strings.Split(name, ".")

	var out []*parser.Assignment

	for i := range as {
		a := &as[i]
		if !rule.takesOp(a) {
			continue
		}

		if rule.takesTarget(targetSteps(a.Target), parts) {
			out = append(out, a)
		}
	}

	return out
}

// takesOp reports whether rule takes a's operator: = always; op=, ++ and --
// with compound, for op one of + - * / &; a declaration or a tag's attribute
// never.
func (rule writeRule) takesOp(a *parser.Assignment) bool {
	switch a.Op {
	case parser.AssignSet:
		return true
	case parser.AssignCompound, parser.AssignStep:
		return rule.compound && strings.ContainsRune("+-*/&", rune(a.Operator[0]))
	default:
		return false
	}
}

// takesTarget reports whether a target written as steps (targetSteps) writes
// the name parts as rule reads it.
func (rule writeRule) takesTarget(steps, parts []string) bool {
	want := len(parts)
	if rule.member {
		want++
	}

	for p := 0; p+want <= len(steps); p++ {
		if p+want != len(steps) || !foldEqual(steps[p:p+len(parts)], parts) {
			continue
		}

		if p == 0 || rule.qualified || p == 1 && slices.ContainsFunc(rule.scopes, func(s string) bool { return strings.EqualFold(s, steps[0]) }) {
			return true
		}
	}

	return false
}

// foldEqual reports whether a and b hold the same names, ignoring case.
func foldEqual(a, b []string) bool {
	for i := range a {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}

	return true
}

// targetSteps splits a target as parser.Assignment spells it into its steps:
// each dotted name, and each [ … ] key whole. a.b[ "x.y" ].c is a, b,
// [ "x.y" ], c.
func targetSteps(target string) []string {
	if !strings.ContainsAny(target, ".[") {
		return []string{target}
	}

	var steps []string

	start, depth := 0, 0

	for i := 0; i < len(target); i++ {
		switch target[i] {
		case '"', '\'':
			if depth > 0 {
				if end := strings.IndexByte(target[i+1:], target[i]); end >= 0 {
					i += end + 1
				}
			}
		case '[':
			if depth == 0 && i > start {
				steps = append(steps, target[start:i])
				start = i
			}

			depth++
		case ']':
			depth--
			if depth == 0 {
				steps = append(steps, target[start:i+1])
				start = i + 1
			}
		case '.':
			if depth == 0 {
				if i > start {
					steps = append(steps, target[start:i])
				}

				start = i + 1
			}
		default:
		}
	}

	if start < len(target) {
		steps = append(steps, target[start:])
	}

	return steps
}

// variableAttributes are the attributes and arguments whose string value
// names the variable a tag or call fills.
var variableAttributes = []string{"item", "index", "name", "returnvariable", "variable", "result"}

// declaresName reports whether as declare name rather than assign it: `var
// name`, a for-in loop over it, or a tag attribute or named argument that
// names it as the variable it fills (variableAttributes).
func declaresName(as []parser.Assignment, name string) bool {
	for i := range as {
		a := &as[i]

		switch {
		case a.Op == parser.AssignVar || a.Op == parser.AssignForIn || a.Local:
			if strings.EqualFold(a.Target, name) {
				return true
			}
		case a.Op == parser.AssignAttribute || a.Op == parser.AssignSet && a.HasLiteral:
			if strings.EqualFold(a.Literal, name) && slices.ContainsFunc(variableAttributes, func(v string) bool { return strings.EqualFold(v, a.Target) }) {
				return true
			}
		default:
		}
	}

	return false
}

// onLines are the assignments in as whose target is on lines from through to,
// 0-based and both included. as is in the order written.
func onLines(as []parser.Assignment, from, to int) []parser.Assignment {
	lo := sort.Search(len(as), func(i int) bool { return as[i].Line >= from })
	hi := sort.Search(len(as), func(i int) bool { return as[i].Line > to })

	if lo >= hi {
		return nil
	}

	return as[lo:hi]
}

// between are the assignments in as whose target starts at offsets from up
// to to.
func between(as []parser.Assignment, from, to int) []parser.Assignment {
	lo := sort.Search(len(as), func(i int) bool { return as[i].At >= from })
	hi := sort.Search(len(as), func(i int) bool { return as[i].At >= to })

	if lo >= hi {
		return nil
	}

	return as[lo:hi]
}

// withoutTag is as less the assignments read from a tag named tag.
func withoutTag(as []parser.Assignment, tag string) []parser.Assignment {
	if !slices.ContainsFunc(as, func(a parser.Assignment) bool { return a.Tag == tag }) {
		return as
	}

	return slices.DeleteFunc(slices.Clone(as), func(a parser.Assignment) bool { return a.Tag == tag })
}

// assignmentsIn is the assignments of content, read from file just now: its
// source's, read once, when that holds the same text, and otherwise the
// parse's reading of the text (assignmentMemo).
func (r *Resolver) assignmentsIn(file, content string) []parser.Assignment {
	if source := r.sourceInfo(file); source.derived != nil && source.content == content {
		return source.assignments()
	}

	return assignmentsOfText(file, content)
}

// assignments is the source's assignments (parser.AssignmentsOf), read once.
func (s *sourceInfo) assignments() []parser.Assignment {
	if s.derived == nil {
		return parser.AssignmentsOf("", s.content)
	}

	s.derived.assignmentsOnce.Do(func() { s.derived.assigned = parser.AssignmentsOf(s.derived.path, s.content) })

	return s.derived.assigned
}

// assignmentMemo keeps parser.AssignmentsOf's answer for the texts asked
// about most recently that are no source's (an edited buffer, a test's
// text), by whether they are read as a .cfm template.
var assignmentMemo = [2]*textCache[[]parser.Assignment]{
	{size: assignmentsSize}, {size: assignmentsSize},
}

// assignmentsOfText is parser.AssignmentsOf(file, content), remembered.
func assignmentsOfText(file, content string) []parser.Assignment {
	template := 0
	if ext := strings.ToLower(file[strings.LastIndexByte(file, '.')+1:]); ext == "cfm" || ext == "cfml" {
		template = 1
	}

	return assignmentMemo[template].of(content, func(text string) []parser.Assignment {
		return parser.AssignmentsOf(file, text)
	})
}

// assignmentsSize is roughly what a list of assignments holds beyond its text.
func assignmentsSize(as []parser.Assignment) int { return 24 + assignmentBytes*cap(as) }

// assignmentBytes is what a parser.Assignment holds.
const assignmentBytes = int(unsafe.Sizeof(parser.Assignment{}))
