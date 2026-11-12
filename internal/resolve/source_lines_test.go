package resolve

import (
	"strings"
	"testing"
)

// sourceText and contentBeforeLine replaced whole-file strings.Split calls
// in the assignment lookups; they must give what the split gave for every
// window, including past the end and on a trailing newline.
func TestSourceTextMatchesSplittingTheFile(t *testing.T) {
	for _, content := range []string{"", "a", "a\n", "a\nb", "a\nb\n", "\n\n", "x = 1;\ny = f(\n  2 );\nz = y\n"} {
		lines := strings.Split(content, "\n")

		for from := -1; from <= len(lines)+1; from++ {
			for to := from; to <= len(lines)+2; to++ {
				want := ""
				if f := max(from, 0); f < len(lines) && to > f {
					want = strings.Join(lines[f:min(to, len(lines))], "\n")
				}

				if got := sourceText(content, from, to); got != want {
					t.Errorf("sourceText(%q, %d, %d) = %q, want %q", content, from, to, got, want)
				}
			}

			if from >= 0 && from <= len(lines) {
				if got, want := contentBeforeLine(content, from), strings.Join(lines[:from], "\n"); got != want {
					t.Errorf("contentBeforeLine(%q, %d) = %q, want %q", content, from, got, want)
				}
			}
		}
	}
}
