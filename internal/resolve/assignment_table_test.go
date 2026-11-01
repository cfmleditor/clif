package resolve

import (
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// TestTheAssignmentTableAnswersAsTheLineWalkDid: lookups of a variable's last
// assignment go to the per-file table (Resolver.localAssignmentAt), not the
// line walk (localAssignmentAt), which walked from the call to the top of the
// file for every receiver and was 7% of a tassweb scan's CPU. The two must
// answer alike: the same line, the same joined right-hand side, the same
// refusal of one whose parentheses do not balance within the bound, and the
// same chain written a call to a line.
func TestTheAssignmentTableAnswersAsTheLineWalkDid(t *testing.T) {
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
	}{
		{"x", 1, 9},
		{"x", 0, 3},
		{"y", 0, 9},
		{"z", 0, 9},
		{"svc", 0, 9},
		{"svc", 1, 9},
		{"w", 0, 10},
		{"missing", 0, 10},
		{"x", 0, 100},
		{"x", 5, 9},
		{"it", 0, 15},
		{"chained", 0, 15},
	} {
		wantRHS, wantLine, wantOK := localAssignmentAt(content, tc.name, tc.start, tc.header)
		gotRHS, gotLine, gotOK := r.localAssignmentAt(pr, tc.name, tc.start, tc.header)

		if gotRHS != wantRHS || gotLine != wantLine || gotOK != wantOK {
			t.Errorf("%s in [%d,%d): table (%q, %d, %v), line walk (%q, %d, %v)", tc.name, tc.start, tc.header, gotRHS, gotLine, gotOK, wantRHS, wantLine, wantOK)
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
	}
}`
	pr := parser.ParseWithOptions("file:///w/a.cfc", content, &parser.ParseOptions{})
	r := &Resolver{}

	for _, name := range []string{"event", "private"} {
		if rhs, _, ok := r.localAssignmentAt(pr, name, 0, 9); ok {
			t.Errorf("%s: read as assigned %q", name, rhs)
		}
	}

	for _, name := range []string{"rendered", "later"} {
		if _, _, ok := r.localAssignmentAt(pr, name, 0, 9); !ok {
			t.Errorf("%s: assignment not found", name)
		}
	}
}
