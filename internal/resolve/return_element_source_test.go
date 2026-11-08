package resolve

import "testing"

// A source holds an element return when any function in it returns a read of a
// collection's element, in script or in tags. The answer is the same whether
// or not the source has a place to remember it, and is remembered once.
func TestASourceKnowsWhetherItHoldsAnElementReturn(t *testing.T) {
	for _, c := range []struct {
		name, content string
		want          bool
	}{
		{"script", "component { function f() { return items[ 1 ]; } }", true},
		{"script, capitals", "component { function f() { RETURN Items[i]; } }", true},
		{"tag", `<cffunction name="f"><cfreturn items[ 1 ]></cffunction>`, true},
		{"returns a call", "component { function f() { return items.first(); } }", false},
		{"returns a variable", "component { function f() { return items; } }", false},
		{"an index with a newline", "component { function f() { return items[\n1 ]; } }", false},
		{"no return", "component { function f() { x = items[ 1 ]; } }", false},
		{"empty", "", false},
	} {
		bare := sourceInfo{content: c.content}
		if got := bare.holdsElementReturn(); got != c.want {
			t.Errorf("%s: without a place to remember it, %v, want %v", c.name, got, c.want)
		}

		kept := sourceInfo{content: c.content, derived: &sourceDerived{}}
		for range 2 {
			if got := kept.holdsElementReturn(); got != c.want {
				t.Errorf("%s: remembered, %v, want %v", c.name, got, c.want)
			}
		}

		if kept.derived.elementReads != c.want {
			t.Errorf("%s: the remembered answer is %v, want %v", c.name, kept.derived.elementReads, c.want)
		}
	}
}
