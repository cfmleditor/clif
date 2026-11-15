package resolve

import (
	"strings"
	"testing"
	"time"
)

// Many wildcards over a text that cannot match is a bounded search.
func TestWildPatternDoesNotBacktrackWithoutBound(t *testing.T) {
	p := &wildPattern{pieces: []string{"a", "a", "a", "a", "a", "a", "a", "a", "b"}, kind: wildLine}
	text := strings.Repeat("a", 200)

	done := make(chan bool)

	go func() { done <- p.match(text) }()

	select {
	case got := <-done:
		if got {
			t.Error("matched a text that has no b")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("match still backtracking after 5s")
	}
}

func TestSpelledStringReadsDoubledHashesAsLiterals(t *testing.T) {
	var b wildBuilder

	spelledString("a##b#x#c", &b, false)
	b.text("")

	want := []string{"a#b"}
	if len(b.pieces) != 1 || b.pieces[0] != want[0] || b.cur.String() != "c" {
		t.Errorf("pieces %q then %q, want %q then c", b.pieces, b.cur.String(), want)
	}
}

func TestAdjacentHashExpressionsAreNotAroundALiteralHash(t *testing.T) {
	var b wildBuilder

	spelledString("x.#a##b#", &b, false)

	// Two expressions in a row are one wildcard, and no literal hash is left
	// between them.
	if len(b.pieces) != 1 || b.pieces[0] != "x." || b.cur.String() != "" {
		t.Errorf("pieces %q then %q, want x. and a wildcard with nothing after", b.pieces, b.cur.String())
	}

	var lit wildBuilder

	spelledString("a##b", &lit, false)

	if lit.cur.String() != "a#b" {
		t.Errorf("a doubled hash is %q, want a#b", lit.cur.String())
	}
}
