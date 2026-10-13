package parser

import (
	"testing"
	"unsafe"
)

// Each of these is allocated or copied once per parse, per sub-parse or per
// call site, and each keeps its small fields together so none is padded to a
// word on its own. Adding a bool in the middle of one moves it up a size
// class, or makes every call slice larger, without anything else noticing.
// .ai/docs/LINT-PLAN.md stage 3 has the measurements.
func TestParserStructsKeepTheirSize(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("sizes are stated for 64-bit words")
	}

	cases := []struct {
		name      string
		got, want uintptr
	}{
		{"CallSite", unsafe.Sizeof(CallSite{}), 120},
		{"Scanner", unsafe.Sizeof(Scanner{}), 128},
		// The managed-setter lookup and the flowBlocks pointer add a word
		// each, and fill the existing 384-byte allocation class: the next
		// word moves it to 416. The source-origin offset now adds that word,
		// and the chained-assignment depth a byte past the bools, padded to
		// another word, still inside the 416-byte class.
		{"scriptParser", unsafe.Sizeof(scriptParser{}), 400},
	}

	for _, c := range cases {
		if c.got > c.want {
			t.Errorf("%s is %d bytes, over %d: group its small fields rather than padding each", c.name, c.got, c.want)
		}
	}
}
