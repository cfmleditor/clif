package resolve

import (
	"strings"
	"testing"
)

// sourceLines and contentBeforeLine replaced whole-file strings.Split calls
// in the assignment lookups; they must give what the split gave for every
// window, including past the end and on a trailing newline.
func TestSourceLinesMatchSplittingTheFile(t *testing.T) {
	for _, content := range []string{"", "a", "a\n", "a\nb", "a\nb\n", "\n\n", "x = 1;\ny = f(\n  2 );\nz = y\n"} {
		lines := strings.Split(content, "\n")

		for from := 0; from <= len(lines)+1; from++ {
			for to := from; to <= len(lines)+2; to++ {
				var want []string
				if from < len(lines) {
					want = lines[from:min(to, len(lines))]
				}

				if got := sourceLines(content, from, to); strings.Join(got, "\x00") != strings.Join(want, "\x00") || len(got) != len(want) {
					t.Errorf("sourceLines(%q, %d, %d) = %q, want %q", content, from, to, got, want)
				}
			}

			if from <= len(lines) {
				if got, want := contentBeforeLine(content, from), strings.Join(lines[:from], "\n"); got != want {
					t.Errorf("contentBeforeLine(%q, %d) = %q, want %q", content, from, got, want)
				}
			}
		}
	}
}

// The case-insensitive searches that replaced lowercasing a whole file agree
// with lowercasing it.
func TestContainsFoldMatchesLoweringTheText(t *testing.T) {
	for _, tc := range []struct{ text, lower string }{
		{"<cfif IsInstanceOf(x, 'A')>", "isinstanceof"},
		{"isInstanceOF", "isinstanceof"},
		{"isInstance Of", "isinstanceof"},
		{"", "a"},
		{"abc", ""},
		{"Core/Utilities/FormBuilder/Templates\"", "utilities/formbuilder/templates\""},
		{"templates'", "utilities/formbuilder/templates'"},
		{"ÉA", "éa"},
	} {
		want := strings.Contains(strings.ToLower(tc.text), tc.lower)

		if got := containsFoldString(tc.text, tc.lower); got != want {
			t.Errorf("containsFoldString(%q, %q) = %v, want %v", tc.text, tc.lower, got, want)
		}

		if got := containsFoldBytes([]byte(tc.text), tc.lower); got != want {
			t.Errorf("containsFoldBytes(%q, %q) = %v, want %v", tc.text, tc.lower, got, want)
		}
	}
}
