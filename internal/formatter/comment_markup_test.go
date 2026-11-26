package formatter

import (
	"regexp"
	"testing"
)

// markupStartOracle is the rule holdsMarkupStart implements, as the regular
// expression it replaced: a less-than sign, an optional slash, then a letter.
// It is the oracle here, not code the formatter runs.
var markupStartOracle = regexp.MustCompile(`</?[A-Za-z]`)

// TestHoldsMarkupStartIsTheIntendedRule states the rule in cases: a comment
// holding the start of a tag or end tag is left alone, and `a < b` is not
// markup. An edit that changes the rule changes these, deliberately.
func TestHoldsMarkupStartIsTheIntendedRule(t *testing.T) {
	for s, want := range map[string]bool{
		"":           false,
		"<":          false,
		"</":         false,
		"<a":         true,
		"</a":        true,
		"<cfoutput>": true,
		"x <b> y":    true,
		"a < b":      false,
		"a </ b":     false,
		"<1":         false,
		"</1":        false,
		"<!--":       false,
		"<?xml":      false,
		"<//a":       false,
		"<</a":       true,
		"é<é":        false,
		"end <":      false,
		"end </":     false,
	} {
		if got := holdsMarkupStart(s); got != want {
			t.Errorf("holdsMarkupStart(%q) = %v, want %v", s, got, want)
		}
	}
}

// TestHoldsMarkupStartAgreesWithItsOracle holds the scan to the regular
// expression it replaced on every string up to six characters of an alphabet
// that has each character the rule distinguishes.
func TestHoldsMarkupStartAgreesWithItsOracle(t *testing.T) {
	alphabet := []string{"<", "/", "a", "Z", "1", " ", "!", "é"}

	var check func(prefix string, depth int)

	check = func(prefix string, depth int) {
		if got, want := holdsMarkupStart(prefix), markupStartOracle.MatchString(prefix); got != want {
			t.Fatalf("holdsMarkupStart(%q) = %v, oracle %v", prefix, got, want)
		}

		if depth == 0 {
			return
		}

		for _, c := range alphabet {
			check(prefix+c, depth-1)
		}
	}

	check("", 6)
}
