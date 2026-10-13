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
// refusal of one whose parentheses do not balance within the bound.
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
	} {
		wantRHS, wantLine, wantOK := localAssignmentAt(content, tc.name, tc.start, tc.header)
		gotRHS, gotLine, gotOK := r.localAssignmentAt(pr, tc.name, tc.start, tc.header)

		if gotRHS != wantRHS || gotLine != wantLine || gotOK != wantOK {
			t.Errorf("%s in [%d,%d): table (%q, %d, %v), line walk (%q, %d, %v)", tc.name, tc.start, tc.header, gotRHS, gotLine, gotOK, wantRHS, wantLine, wantOK)
		}
	}
}
