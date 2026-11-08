package tsoracle

import (
	"bytes"
	"testing"
	"time"
)

// binaryLike is text the CFML grammar parses to a very flat tree: one root with
// a named child for every few bytes, as a zip saved as a .cfm does.
func binaryLike(n int) []byte {
	return bytes.Repeat([]byte("PK\x03\x04\x14\x00\x08\x00 x\x9b`'U\x7f\x0f\xe2\x1fP\xb0\x00\x00&\xe2\x04\x00"), n/26+1)[:n]
}

func fastest(src []byte) time.Duration {
	best := time.Duration(1<<63 - 1)

	for range 3 {
		start := time.Now()

		GrammarCalls(src)

		best = min(best, time.Since(start))
	}

	return best
}

// TestWalkingAFlatTreeIsLinear: reading a node's children by index is
// quadratic in their number (each NamedChild(i) walks the i siblings before
// it). MasaCMS keeps a 1.7MB zip as theme.zip.cfm, and the differential run
// over it ran into its 30-minute timeout. Four times the input costs about four
// times as long when the walk is linear and sixteen times as long when it is
// not; the bound is set between, so a busy runner moves both numbers together.
func TestWalkingAFlatTreeIsLinear(t *testing.T) {
	small, large := fastest(binaryLike(40_000)), fastest(binaryLike(160_000))

	if ratio := float64(large) / float64(max(small, time.Microsecond)); ratio > 9 {
		t.Fatalf("4x the input took %.1fx as long (%v then %v): the walk is not linear", ratio, small, large)
	}
}
