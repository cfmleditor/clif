package resolve

import (
	"regexp"
	"slices"
	"testing"
)

// The positions of the calls that could include a template depend on its path
// alone: a call naming no file always counts, one with a pattern counts when
// the pattern matches, and the answer is read once per path.
func TestMuraSitesForAreThoseThatCouldIncludeThePath(t *testing.T) {
	t.Parallel()

	r := &Resolver{
		muraSitesBuilt: true,
		muraSites: []muraObjectSite{
			{target: regexp.MustCompile(`/display_objects/a\.cfm$`)},
			{target: nil},
			{target: regexp.MustCompile(`/b\.cfm$`)},
			{target: regexp.MustCompile(`/display_objects/`)},
		},
	}

	cases := map[string][]int{
		"/w/display_objects/a.cfm": {0, 1, 3},
		"/w/other/b.cfm":           {1, 2},
		"/w/display_objects/c.cfm": {1, 3},
		"/w/elsewhere/d.cfm":       {1},
	}

	for path, want := range cases {
		if got := r.muraSitesFor(path); !slices.Equal(got, want) {
			t.Errorf("muraSitesFor(%q) = %v, want %v", path, got, want)
		}
	}

	if !r.muraObjectMayInclude("/w/elsewhere/d.cfm") {
		t.Error("a call that names no file may include any template")
	}

	// Read once: the second answer is the stored one, not a fresh slice.
	first := r.muraSitesFor("/w/other/b.cfm")
	if again := r.muraSitesFor("/w/other/b.cfm"); &again[0] != &first[0] {
		t.Error("the sites for a path were worked out again")
	}

	none := &Resolver{muraSitesBuilt: true, muraSites: []muraObjectSite{{target: regexp.MustCompile(`x\.cfm$`)}}}
	if none.muraObjectMayInclude("/w/y.cfm") {
		t.Error("a pattern that does not match the template counted")
	}
}
