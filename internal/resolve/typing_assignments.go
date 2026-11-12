package resolve

import (
	"slices"
	"sort"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

// Receiver typing asks what a name was last assigned before a line: the
// right-hand side a call's receiver holds (`x = svc.get(); x.run()`), whether
// a native type a ref claimed was overwritten before the use, and which
// assignments reach a use (reachingUnion). It found them by reading lines:
// one that began, after `<cfset` and `var`, with the name and `=`, its value
// joined over the lines after it while a bracket stayed open, a conditional
// or a chain went on (joinAssignmentLine). That read markup, JavaScript and
// comments as code, took a named argument written on its own line or a struct
// key for an assignment, and missed one that was not first on its line.
//
// It reads the parse's assignments now (parser.Assignment, through
// ParseResult.Assignments, read once per parse): a write the parse reads where a statement stands, and
// its value as the parse ends it, written as one line (ValueLine).

// assignsName reports whether a is a statement's `name = value`, with name
// written bare, after `var`, or after `local.` or `variables.`: what receiver
// typing reads as the name's value. A member's write (name.x =), an inner
// write (`a = name = 1`), a compound one, a tag's attribute and a for loop's
// initializer (`for ( var i = 1; …`) are not.
func assignsName(a *parser.Assignment, content, name string) bool {
	if a.Op != parser.AssignSet || !a.StartsStatement() && (!a.Local || forAt(content, a.Statement)) {
		return false
	}

	target := a.Target
	if dot := strings.IndexByte(target, '.'); dot >= 0 {
		if scope := target[:dot]; !strings.EqualFold(scope, "local") && !strings.EqualFold(scope, "variables") {
			return false
		}

		target = target[dot+1:]
	}

	return strings.EqualFold(target, name)
}

// forAt reports whether the statement at offset at in content is a for loop:
// a `var` in it is its initializer's.
func forAt(content string, at int) bool {
	return at >= 0 && at+3 < len(content) && strings.EqualFold(content[at:at+3], "for") && !lex.IsWord(content[at+3])
}

// lastAssignmentTo is the last assignment to name (assignsName) in as whose
// target is on a line from start to before header, and its value written as
// one line and cut at header's start. It answers none when that value's
// parentheses do not balance before header: the statement has not ended
// where it is asked about.
func lastAssignmentTo(as []parser.Assignment, content, name string, start, header int) (*parser.Assignment, string, bool) {
	start = max(start, 0)
	before := sort.Search(len(as), func(i int) bool { return as[i].Line >= header })

	for i := range slices.Backward(as[:before]) {
		a := &as[i]
		if a.Line < start {
			break
		}

		if !assignsName(a, content, name) {
			continue
		}

		limit := len(content)
		if at, ok := lineOffset(content, header); ok && at > 0 {
			limit = at - 1
		}

		rhs := a.ValueLine(content, limit)
		if parens, _, _ := parser.OpenGroups(rhs); parens != 0 {
			return nil, "", false
		}

		return a, rhs, true
	}

	return nil, "", false
}

// lastAssignment is lastAssignmentTo's value and line.
func lastAssignment(as []parser.Assignment, content, name string, start, header int) (string, int, bool) {
	a, rhs, ok := lastAssignmentTo(as, content, name, start, header)
	if !ok {
		return "", 0, false
	}

	return rhs, a.Line, true
}

// localAssignmentAt is the right-hand side of the last assignment to name on
// a line from start to before header (lastAssignmentTo), and its line.
func (r *Resolver) localAssignmentAt(pr *parser.ParseResult, name string, start, header int) (string, int, bool) {
	return lastAssignment(pr.Assignments(), pr.Content, name, start, header)
}

// localAssignmentIn is localAssignmentAt without the line.
func (r *Resolver) localAssignmentIn(pr *parser.ParseResult, name string, start, header int) (string, bool) {
	rhs, _, ok := r.localAssignmentAt(pr, name, start, header)

	return rhs, ok
}
