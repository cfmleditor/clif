package resolve

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
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

// ShareCallerIndex and the lazy build number files in the order they finish
// or are listed, which is not path order; both answer as IndexCallerFile does.
func TestEveryCallerIndexBuilderAnswersInPathOrder(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"c.cfm": `<cfset svc.save( 1 )>`,
		"a.cfm": `<cfset svc.save( 2 )><cfset load()>`,
		"b.cfm": `<cfset load()>`,
	})

	files := []string{filepath.Join(dir, "c.cfm"), filepath.Join(dir, "b.cfm"), filepath.Join(dir, "a.cfm")}
	want := map[string][]string{
		"save": {filepath.Join(dir, "a.cfm"), filepath.Join(dir, "c.cfm")},
		"load": {filepath.Join(dir, "a.cfm"), filepath.Join(dir, "b.cfm")},
	}

	check := func(how string, r *Resolver) {
		t.Helper()

		for name, paths := range want {
			if got := r.callerFiles(name); !slices.Equal(got, paths) {
				t.Errorf("%s: callers of %s: %v, want %v", how, name, got, paths)
			}
		}
	}

	shared := &Resolver{FS: vfs.OS{}, Index: index.New(), InferArgsFiles: files}
	restore := ShareCallerIndex([]*Resolver{shared})

	check("shared", shared)
	restore()

	check("built on first ask", &Resolver{FS: vfs.OS{}, Index: index.New(), InferArgsFiles: files})
}

// A name nothing calls is answered without being added to the index:
// inference asks about every prefix of a function's name.
func TestAskingForAnAbsentNameDoesNotGrowTheIndex(t *testing.T) {
	r := &Resolver{InferArgsFiles: []string{"/w/a.cfc"}}
	r.IndexCallerFile("/w/a.cfc", "component { function g() { save( 2 ); } }")

	before := len(r.callerIdx.byName)

	for _, name := range []string{"s", "sa", "sav", "nothing"} {
		if got := r.callerFiles(name); len(got) != 0 {
			t.Fatalf("callers of %s: %v", name, got)
		}
	}

	if after := len(r.callerIdx.byName); after != before {
		t.Fatalf("the index grew from %d names to %d by being asked", before, after)
	}
}

// A name's list is sorted by the first ask for it and kept, so later asks do
// not sort it again; a file added under the name afterwards is in the next
// answer, in its place.
func TestTheFirstAskSortsANameAndAnAddUnsortsIt(t *testing.T) {
	r := &Resolver{InferArgsFiles: []string{"/w/a.cfc", "/w/c.cfc", "/w/b.cfc"}}
	r.IndexCallerFile("/w/c.cfc", "component { function f() { save( 1 ); } }")
	r.IndexCallerFile("/w/a.cfc", "component { function g() { save( 2 ); } }")

	if got, want := r.callerFiles("save"), []string{"/w/a.cfc", "/w/c.cfc"}; !slices.Equal(got, want) {
		t.Fatalf("callers of save: %v, want %v", got, want)
	}

	idx := r.callerIdx
	if !idx.sorted["save"] || !slices.Equal(pathsOf(idx.byName["save"], idx.paths), []string{"/w/a.cfc", "/w/c.cfc"}) {
		t.Fatalf("the first ask did not keep save's list sorted: %v", pathsOf(idx.byName["save"], idx.paths))
	}

	r.IndexCallerFile("/w/b.cfc", "component { function h() { save( 3 ); } }")

	if idx.sorted["save"] {
		t.Fatal("adding a caller of save left its list marked sorted")
	}

	if got, want := r.callerFiles("save"), []string{"/w/a.cfc", "/w/b.cfc", "/w/c.cfc"}; !slices.Equal(got, want) {
		t.Fatalf("callers of save after an add: %v, want %v", got, want)
	}
}

// A first ask sorts a copy outside the lock and keeps it only if nothing was
// added meanwhile. The interleaving that defeats a length check: one ask
// copies a list holding a repeat, another stores the shorter compacted list,
// an add grows it back to the first ask's length, and the first ask finishes.
// Its copy predates the add and must not replace the list.
func TestASortStartedBeforeAnAddDoesNotReplaceTheList(t *testing.T) {
	r := &Resolver{InferArgsFiles: []string{"/w/a.cfc", "/w/b.cfc", "/w/c.cfc"}}
	r.IndexCallerFile("/w/a.cfc", "component { function f() { save( 1 ); } }")
	r.IndexCallerFile("/w/c.cfc", "component { function g() { save( 2 ); } }")
	r.IndexCallerFile("/w/a.cfc", "component { function f() { save( 1 ); } }")

	idx := r.callerIdx
	early := idx.unsorted("save")

	if got := idx.files("save"); !slices.Equal(got, []string{"/w/a.cfc", "/w/c.cfc"}) {
		t.Fatalf("callers of save: %v", got)
	}

	r.IndexCallerFile("/w/b.cfc", "component { function h() { save( 3 ); } }")

	if len(idx.byName["save"]) != len(early.ids) {
		t.Fatalf("the fixture no longer reproduces the case: %d ids held, %d copied", len(idx.byName["save"]), len(early.ids))
	}

	early.sort()
	idx.keep("save", early)

	if got, want := r.callerFiles("save"), []string{"/w/a.cfc", "/w/b.cfc", "/w/c.cfc"}; !slices.Equal(got, want) {
		t.Fatalf("callers of save after a stale sort finished: %v, want %v", got, want)
	}
}
