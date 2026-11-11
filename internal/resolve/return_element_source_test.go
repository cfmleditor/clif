package resolve

import (
	"sync"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

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
		// The returns are read from tokens, so an index written across lines
		// is a read like any other; the regular expression stopped at the
		// newline.
		{"an index with a newline", "component { function f() { return items[\n1 ]; } }", true},
		{"commented out", "component { function f() { // return items[ 1 ];\n return items; } }", false},
		{"in a string", "component { function f() { x = \"return items[ 1 ];\"; return x; } }", false},
		{"commented-out tag", `<cffunction name="f"><!--- <cfreturn items[ 1 ]> ---><cfreturn items></cffunction>`, false},
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

// The plain parse of a source is a function of its text, kept with the source
// and shared by every caller; a source with no place to keep it parses afresh.
func TestASourcesPlainParseIsKeptAndShared(t *testing.T) {
	content := "component { function f() { return items[ 1 ]; } }"

	kept := sourceInfo{content: content, derived: &sourceDerived{}}

	var parses [8]*parser.ParseResult

	var wg sync.WaitGroup

	for i := range parses {
		wg.Go(func() { parses[i] = kept.plainParse("/ws/A.cfc") })
	}

	wg.Wait()

	for i := range parses {
		if parses[i] == nil || parses[i] != parses[0] {
			t.Fatalf("caller %d got a different parse: %p vs %p", i, parses[i], parses[0])
		}
	}

	if len(parses[0].Funcs) != 1 || parses[0].Funcs[0].Name != "f" {
		t.Errorf("the kept parse holds %d functions, want f", len(parses[0].Funcs))
	}

	bare := sourceInfo{content: content}
	if a, b := bare.plainParse("/ws/A.cfc"), bare.plainParse("/ws/A.cfc"); a == b {
		t.Error("a source with nowhere to keep its parse returned one parse twice")
	}
}
