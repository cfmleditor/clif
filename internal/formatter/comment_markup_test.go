package formatter

import (
	"regexp"
	"testing"
)

// TestHoldsMarkupStartAgreesWithItsPattern: the scan replaced
// regexp `</?[A-Za-z]`; it must answer as the pattern did.
func TestHoldsMarkupStartAgreesWithItsPattern(t *testing.T) {
	pattern := regexp.MustCompile(`</?[A-Za-z]`)

	for _, s := range []string{
		"", "<", "</", "<a", "</a", "<1", "</1", "a < b", "a </ b", "x<y", "<<a", "<</a", "<//a", "< a", "</ a", "<!--", "é<é", "no markup", "<cfoutput>", "end <", "end </", "<-</Z",
	} {
		if got, want := holdsMarkupStart(s), pattern.MatchString(s); got != want {
			t.Errorf("holdsMarkupStart(%q) = %v, pattern %v", s, got, want)
		}
	}
}
