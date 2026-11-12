package resolve

import (
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// TestTheLastAssignmentIsTheParses: a variable's last assignment before a
// line is the parse's record of it (parser.Assignment), its value written as
// one line: a call's arguments one to a line joined, a chain written a call
// to a line joined, and one whose parentheses do not balance before the line
// asked about refused. The line reader this replaced kept a <cfset>'s `>`
// on a value that ran over lines, and read `var chained = ...` in a tag
// function's body, which is text it outputs, as code.
func TestTheLastAssignmentIsTheParses(t *testing.T) {
	content := `<cfset svc = createObject("component", "a.Svc")>
<cffunction name="f">
	<cfset var x = svc.get(
		1,
		2
	)>
	<cfset local.y = x>
	<cfset VARIABLES.z = svc.make()>
	<cfset w = svc.open(
	<cfset x.run()>
	<cfset it = svc
		.load()
		.iterate()>
	var chained = svc.load( 1 )
		.iterate();
</cffunction>`
	pr := parser.ParseWithOptions("file:///w/a.cfm", content, &parser.ParseOptions{})
	r := &Resolver{}

	for _, tc := range []struct {
		name          string
		start, header int
		rhs           string
		line          int
	}{
		{"x", 1, 9, "svc.get( 1, 2 )", 2},
		{"x", 0, 3, "", -1},
		{"y", 0, 9, "x", 6},
		{"z", 0, 9, "svc.make()", 7},
		{"svc", 0, 9, `createObject("component", "a.Svc")`, 0},
		{"svc", 1, 9, "", -1},
		{"w", 0, 10, "", -1},
		{"missing", 0, 10, "", -1},
		{"x", 0, 100, "svc.get( 1, 2 )", 2},
		{"x", 5, 9, "", -1},
		{"it", 0, 15, "svc.load().iterate()", 10},
		{"chained", 0, 15, "", -1},
	} {
		rhs, line, ok := r.localAssignmentAt(pr, tc.name, tc.start, tc.header)
		if !ok {
			line = -1
		}

		if rhs != tc.rhs || line != tc.line {
			t.Errorf("%s in [%d,%d): (%q, %d), want (%q, %d)", tc.name, tc.start, tc.header, rhs, line, tc.rhs, tc.line)
		}
	}
}

// TestANamedArgumentOnItsOwnLineIsNotAnAssignment: Preside's handlers call
// runEvent() with an argument a line, `event = "admin.x"` among them, and the
// table read that line as an assignment to event, which then typed `event` in
// another function of the component as a string. A line after one ending in
// `(` or `,` continues an argument list.
func TestANamedArgumentOnItsOwnLineIsNotAnAssignment(t *testing.T) {
	content := `component {
	function a() {
		var rendered = runEvent(
			  event   = "admin.x"
			, private = true
		);
		var later = "y";
		// a comment ending in a comma,
		var after = "z";
	}
}`
	pr := parser.ParseWithOptions("file:///w/a.cfc", content, &parser.ParseOptions{})
	r := &Resolver{}

	for _, name := range []string{"event", "private"} {
		if rhs, _, ok := r.localAssignmentAt(pr, name, 0, 9); ok {
			t.Errorf("%s: read as assigned %q", name, rhs)
		}
	}

	for _, name := range []string{"rendered", "later", "after"} {
		if _, _, ok := r.localAssignmentAt(pr, name, 0, 11); !ok {
			t.Errorf("%s: assignment not found", name)
		}
	}
}
