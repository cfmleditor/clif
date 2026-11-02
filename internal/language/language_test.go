package language

import (
	"strings"
	"testing"
)

func TestLanguageLoads(t *testing.T) {
	for _, g := range []struct {
		name    string
		grammar Grammar
	}{
		{"CFML", CFML},
		{"CFScript", CFScript},
		{"CFQuery", CFQuery},
	} {
		if Language(g.grammar) == nil {
			t.Errorf("%s language failed to load", g.name)
		}
	}
}

func TestParseCFML(t *testing.T) {
	src := []byte(`component { public void function hello() { return; } }`)

	tree := Parse(CFML, src, nil)
	defer tree.Close()

	if tree.RootNode().ChildCount() == 0 {
		t.Fatal("expected children in CFML parse tree")
	}
}

func TestParseCFScript(t *testing.T) {
	src := []byte(`function greet(name) { return "Hello " & name; }`)

	tree := Parse(CFScript, src, nil)
	defer tree.Close()

	if tree.RootNode().ChildCount() == 0 {
		t.Fatal("expected children in CFScript parse tree")
	}
}

func TestParseCFQuery(t *testing.T) {
	src := []byte(`SELECT id, name FROM users WHERE active = 1`)

	tree := Parse(CFQuery, src, nil)
	defer tree.Close()

	if tree.RootNode().ChildCount() == 0 {
		t.Fatal("expected children in CFQuery parse tree")
	}
}

// go-tree-sitter routes tree-sitter's allocations through a Go callback unless
// something resets it; the reset is this package's init.
func TestTreeSitterAllocatesThroughLibc(t *testing.T) {
	if !nativeAllocator() {
		t.Fatal("tree-sitter allocates through go-tree-sitter's Go callbacks, not libc")
	}
}

// A grammar that cannot be loaded panics, which the request running it
// recovers, rather than exiting the process and every editor with it.
func TestAGrammarThatCannotLoadPanicsRatherThanExiting(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("an unknown grammar neither panicked nor failed")
		}
	}()

	NewParser(Grammar(99))
}

// A parse that runs past MaxParseTime is cancelled and returns no tree:
// tree-sitter's parser never returned on some pathological pages, and a
// format request then held a core for good.
func TestAParsePastItsTimeLimitIsCancelled(t *testing.T) {
	prev := MaxParseTime
	MaxParseTime = 0

	t.Cleanup(func() { MaxParseTime = prev })

	src := []byte(strings.Repeat("<cfif a>", 20000) + strings.Repeat("</cfif>", 20000))

	if tree := Parse(CFML, src, nil); tree != nil {
		tree.Close()
		t.Fatal("a parse past its time limit still returned a tree")
	}
}
