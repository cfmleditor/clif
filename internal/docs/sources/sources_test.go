package sources

import (
	"slices"
	"strings"
	"testing"
)

func TestParseReadsEntriesAndOrder(t *testing.T) {
	f, err := Parse("t", []byte("# a comment\r\n# order: cfdocs retained lucee\r\nfunction abs\tlucee\r\ntag cfstatic\tlucee\r\n\r\n"))
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"cfdocs", "retained", "lucee"}; !slices.Equal(f.Order, want) {
		t.Errorf("Order = %v, want %v", f.Order, want)
	}

	if f.Entries["function abs"] != "lucee" || f.Entries["tag cfstatic"] != "lucee" || len(f.Entries) != 2 {
		t.Errorf("Entries = %v", f.Entries)
	}
}

// TestParseRefusesWhatItCannotTrust: each of these would otherwise shrink a
// baseline, or switch a check off, without a word.
func TestParseRefusesWhatItCannotTrust(t *testing.T) {
	for name, text := range map[string]string{
		"no tab":           "function abs lucee\n",
		"no source":        "function abs\t\n",
		"no type":          "abs\tlucee\n",
		"bad type":         "Function abs\tlucee\n",
		"empty name":       "function \tlucee\n",
		"duplicate":        "function abs\tlucee\nfunction abs\tcfdocs\n",
		"empty order":      "# order: \n",
		"two orders":       "# order: a b\n# order: a b\n",
		"merge conflict":   "<<<<<<< HEAD\n",
		"blank-named type": "tag  \tlucee\n",
		"mixed case":       "function Abs\tlucee\n",
		"trailing space":   "function abs \tlucee\n",
		"space in name":    "function a bs\tlucee\n",
		"control in name":  "function a\x01bs\tlucee\n",
	} {
		if _, err := Parse("t", []byte(text)); err == nil {
			t.Errorf("%s: Parse(%q) accepted it", name, strings.TrimSpace(text))
		}
	}
}

func TestRank(t *testing.T) {
	order := []string{"cfdocs", Retained, "lucee"}
	if Rank(order, "cfdocs") != 1 || Rank(order, Retained) != 2 || Rank(order, "lucee") != 3 || Rank(order, "gone") != 0 {
		t.Error("Rank does not number the order from 1, with 0 for a source it does not name")
	}
}
