package resolve

import (
	"slices"
	"testing"
)

// The caller index stores each file as a number, assigned in the order files
// reach it, which in a scan is the order its workers finish. What callerFiles
// answers is still each file once, sorted by path: a file indexed twice is
// listed once, and the numbering does not leak into the order.
func TestCallerFilesAreSortedPathsWhateverTheirNumbers(t *testing.T) {
	r := &Resolver{InferArgsFiles: []string{"/w/a.cfc", "/w/b.cfc", "/w/c.cfc"}}

	r.IndexCallerFile("/w/c.cfc", "component { function f() { save( 1 ); } }")
	r.IndexCallerFile("/w/a.cfc", "component { function g() { save( 2 ); load(); } }")
	r.IndexCallerFile("/w/c.cfc", "component { function f() { save( 1 ); } }")
	r.IndexCallerFile("/w/b.cfc", "component { function h() { load(); } }")

	if got, want := r.callerFiles("SAVE"), []string{"/w/a.cfc", "/w/c.cfc"}; !slices.Equal(got, want) {
		t.Fatalf("callers of save: %v, want %v", got, want)
	}

	if got, want := r.callerFiles("load"), []string{"/w/a.cfc", "/w/b.cfc"}; !slices.Equal(got, want) {
		t.Fatalf("callers of load: %v, want %v", got, want)
	}

	if got := r.callerFiles("missing"); len(got) != 0 {
		t.Fatalf("callers of missing: %v", got)
	}
}
