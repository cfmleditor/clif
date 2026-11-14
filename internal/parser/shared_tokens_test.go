package parser

import (
	"reflect"
	"strings"
	"sync"
	"testing"
)

func largeScript(n int) string {
	var b strings.Builder

	b.WriteString("component {\n")

	for i := range n {
		b.WriteString("  function f")
		b.WriteString(strings.Repeat("x", i%7))
		b.WriteString("() { var a = {k: \"v #i#\"}; return a.k; /* c */ }\n")
	}

	b.WriteString("}\n")

	return b.String()
}

// A large text's tokens are tokenize's, held for the next parse of the same
// text, and the list handed out is the one list.
func TestScriptTokensOfIsTokenizeKept(t *testing.T) {
	text := largeScript(400)
	if len(text) < sharedTokenMin {
		t.Fatalf("the text is %d bytes, under the %d a text must have to be kept", len(text), sharedTokenMin)
	}

	first := scriptTokensOf(text)
	if want := tokenize(text); !reflect.DeepEqual(*first, want) {
		t.Fatalf("the tokens differ from tokenize's: %d, want %d", len(*first), len(want))
	}

	if second := scriptTokensOf(strings.Clone(text)); second != first {
		t.Error("a copy of the text was tokenized again")
	}

	if other := scriptTokensOf(largeScript(401)); other == first {
		t.Error("another text was given the first one's tokens")
	}

	// The same length, another text: recognised by its bytes, not its size.
	same := strings.Replace(text, "function f(", "function g(", 1)
	if len(same) != len(text) || same == text {
		t.Fatal("the generated text is not a same-length variant")
	}

	if got := scriptTokensOf(same); got == first || !reflect.DeepEqual(*got, tokenize(same)) {
		t.Error("a text of the same length was given the first one's tokens")
	}

	short := "component { function f() {} }"
	if a, b := scriptTokensOf(short), scriptTokensOf(short); a == b {
		t.Error("a short text is kept: it tokenizes faster than it is compared")
	}
}

// What is held stays within its budget however many texts pass through, and
// the lists handed out stay right when their slots are taken.
func TestSharedTokensStayWithinTheirSlots(t *testing.T) {
	var kept [][]Token

	texts := make([]string, 3*sharedTokenSlots)
	for i := range texts {
		texts[i] = largeScript(300 + i)
		kept = append(kept, *scriptTokensOf(texts[i]))
	}

	sharedTokens.mu.Lock()
	held, slots := sharedTokens.held, 0

	for i := range sharedTokens.slots {
		if sharedTokens.slots[i].toks != nil {
			slots++
		}
	}
	sharedTokens.mu.Unlock()

	if slots > sharedTokenSlots || held > sharedTokenBytes {
		t.Errorf("%d slots and %d bytes held, over %d and %d", slots, held, sharedTokenSlots, sharedTokenBytes)
	}

	for i, text := range texts {
		if !reflect.DeepEqual(kept[i], tokenize(text)) {
			t.Fatalf("the tokens of text %d changed after its slot was reused", i)
		}
	}
}

// Workers parsing the same large file at once read one list, which none of
// them writes: run under -race.
func TestSharedTokensAreReadByManyParsesAtOnce(t *testing.T) {
	src := largeScript(500)

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			pr := ParseWithOptions(testURI, strings.Clone(src), &ParseOptions{ExtractCalls: true, ScanAllScopes: true})
			if len(pr.Assignments()) == 0 {
				t.Error("the parse read no assignments")
			}
		})
	}

	wg.Wait()
}

// Slots are taken oldest first while the bytes held exceed the limit, and a
// list is held only when it fits.
func TestTokenShareHoldsWithinItsLimit(t *testing.T) {
	c := tokenShare{limit: 1000}

	for i, text := range []string{"a", "b", "c", "d"} {
		toks := &[]Token{{Value: text}}
		c.store(text, toks, 400)

		if c.held > c.limit {
			t.Fatalf("after %d stores %d bytes are held, over %d", i+1, c.held, c.limit)
		}
	}

	if c.find("a") != nil || c.find("b") != nil {
		t.Error("the oldest texts are held past the limit")
	}

	if got := c.find("d"); got == nil || (*got)[0].Value != "d" {
		t.Error("the newest text is not held")
	}
}
