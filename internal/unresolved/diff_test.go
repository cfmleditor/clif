package unresolved

import (
	"strings"
	"testing"
)

func call(file string, line uint32, fn, reason string) Call {
	return Call{File: file, Line: line, Function: fn, Reason: reason, Variable: "svc"}
}

// TestDiffNamesWhatMoved: one file breaking while another is fixed leaves the
// totals equal, so the findings themselves are compared.
func TestDiffNamesWhatMoved(t *testing.T) {
	before := []Call{
		call("/a/x.cfc", 1, "save", "variable 'svc' has no component ref"),
		call("/a/y.cfc", 2, "load", "variable 'svc' has no component ref"),
	}
	after := []Call{
		call("/a/y.cfc", 2, "load", "variable 'svc' has no component ref"),
		call("/a/z.cfc", 3, "drop", "method 'drop' not found in Dao"),
	}

	d := Diff(before, after, nil)

	if d.Before != 2 || d.After != 2 {
		t.Fatalf("counts: %d -> %d", d.Before, d.After)
	}

	if len(d.Removed) != 1 || d.Removed[0].File != "/a/x.cfc" || len(d.Added) != 1 || d.Added[0].File != "/a/z.cfc" {
		t.Errorf("equal totals hid the move: removed %+v added %+v", d.Removed, d.Added)
	}
}

// TestDiffTreatsDuplicatesAsAMultiset: two findings with one key are two
// findings, and one going away is a removal, not nothing.
func TestDiffTreatsDuplicatesAsAMultiset(t *testing.T) {
	c := call("/a/x.cfc", 1, "save", "variable 'svc' has no component ref")

	d := Diff([]Call{c, c}, []Call{c}, nil)
	if len(d.Removed) != 1 || len(d.Added) != 0 {
		t.Errorf("removed %d added %d, want 1 and 0", len(d.Removed), len(d.Added))
	}

	d = Diff([]Call{c}, []Call{c, c}, nil)
	if len(d.Removed) != 0 || len(d.Added) != 1 {
		t.Errorf("removed %d added %d, want 0 and 1", len(d.Removed), len(d.Added))
	}
}

// TestDiffIgnoresACountInTheReason: the entry for a broken extends chain says
// how many calls it leaves unchecked, which moves with every call that
// resolves; that is not a different finding.
func TestDiffIgnoresACountInTheReason(t *testing.T) {
	a := call("/a/x.cfc", 1, "coldbox.system.Base", "extends chain breaks; 12 inherited calls not checked")
	b := call("/a/x.cfc", 1, "coldbox.system.Base", "extends chain breaks; 9 inherited calls not checked")

	if d := Diff([]Call{a}, []Call{b}, nil); len(d.Removed)+len(d.Added) != 0 {
		t.Errorf("a changed count was reported as a change: %+v", d)
	}
}

// TestDiffComparesPathsRelativeToTheRoot: the same project scanned in two
// checkouts is one project.
func TestDiffComparesPathsRelativeToTheRoot(t *testing.T) {
	a := call("/home/me/one/models/X.cfc", 4, "f", "r")
	b := call("/tmp/two/models/X.cfc", 4, "f", "r")

	if d := Diff([]Call{a}, []Call{b}, nil); len(d.Removed) != 1 {
		t.Fatal("different absolute paths should differ without a root")
	}

	opt := &DiffOptions{Rel: func(p string) string {
		for _, root := range []string{"/home/me/one/", "/tmp/two/"} {
			if rest, ok := strings.CutPrefix(p, root); ok {
				return rest
			}
		}

		return p
	}}

	if d := Diff([]Call{a}, []Call{b}, opt); len(d.Removed)+len(d.Added) != 0 {
		t.Errorf("the same file in two checkouts differed: %+v", d)
	}
}

// TestGroupByReasonGroupsByCause: the name in a reason is particular to the
// call; the cause is not.
func TestGroupByReasonGroupsByCause(t *testing.T) {
	calls := []Call{
		call("/a", 1, "f", "variable 'svc' has no component ref"),
		call("/a", 2, "f", "variable 'dao' has no component ref"),
		call("/a", 3, "f", "variable 'x' has no component ref"),
		call("/a", 4, "g", "method 'g' not found in Dao"),
	}

	groups := GroupByReason(calls)
	if len(groups) != 2 || len(groups[0].Calls) != 3 || groups[0].Reason != "variable '_' has no component ref" {
		t.Errorf("groups: %+v", groups)
	}
}

func TestWriteDeltaListsExamplesUpToTheCap(t *testing.T) {
	var removed []Call
	for i := range 5 {
		removed = append(removed, call("/a/x.cfc", uint32(i), "f", "variable 'svc' has no component ref"))
	}

	d := Diff(removed, nil, nil)

	var b strings.Builder

	WriteDelta(&b, "masa", &d, 2, nil)

	out := b.String()
	for _, want := range []string{"5 ->      0", "removed     5", "removed 5:", "variable '_' has no component ref", "/a/x.cfc:1  svc.f", "... and 3 more"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	if strings.Contains(out, "/a/x.cfc:3") {
		t.Errorf("listed more than the cap:\n%s", out)
	}

	b.Reset()
	WriteDelta(&b, "masa", &d, 0, nil)

	if strings.Contains(b.String(), "removed 5:") || strings.Count(b.String(), "\n") != 1 {
		t.Errorf("show=0 should print the counts line alone:\n%s", b.String())
	}
}
