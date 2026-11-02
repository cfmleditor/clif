package formatter

import "testing"

// A document whose parse was cancelled (language.MaxParseTime) has no tree,
// and is refused rather than read.
func TestParseErrorRefusesAMissingTree(t *testing.T) {
	if err := ParseError(nil, []byte("<cfset x = 1>")); err == nil {
		t.Fatal("a missing tree was not refused")
	}
}
