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

// localAssignmentAt walks the text in place; it must find what reading the
// split lines found, for every window.
func TestLocalAssignmentAtMatchesTheSplitLines(t *testing.T) {
	content := "x = new A();\n\nx = f(\n  1,\n  2 );\ny = 1\nx = g()\n"
	lines := strings.Split(content, "\n")

	split := func(start, header int) (string, int, bool) {
		for i := min(header, len(lines)) - 1; i >= max(start, 0); i-- {
			name, value, ok := matchSourceAssignment(lines[i])
			if !ok || !strings.EqualFold(name, "x") {
				continue
			}

			rhs := trimAssignmentRHS(value)
			for j := i + 1; strings.Count(rhs, "(") > strings.Count(rhs, ")") && j < min(header, len(lines)) && j <= i+maxAssignmentLines; j++ {
				rhs = strings.TrimSpace(rhs + " " + strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(lines[j]), ";"), "/")))
			}

			if strings.Count(rhs, "(") != strings.Count(rhs, ")") {
				return "", 0, false
			}

			return rhs, i, true
		}

		return "", 0, false
	}

	for start := -1; start <= len(lines)+1; start++ {
		for header := 0; header <= len(lines)+2; header++ {
			wantRHS, wantAt, wantOK := split(start, header)
			gotRHS, gotAt, gotOK := localAssignmentAt(content, "x", start, header)

			if gotRHS != wantRHS || gotAt != wantAt || gotOK != wantOK {
				t.Errorf("localAssignmentAt(%d, %d) = %q, %d, %v; want %q, %d, %v", start, header, gotRHS, gotAt, gotOK, wantRHS, wantAt, wantOK)
			}
		}
	}
}

// lineIndex replaced counting line breaks from the top for each match.
func TestLineIndexCountsAsStringsCountDoes(t *testing.T) {
	for _, s := range []string{"", "a", "\n", "a\nb\n\nc", "\n\nx\n"} {
		idx := newLineIndex(s)

		for off := 0; off <= len(s); off++ {
			if got, want := idx.lineOf(off), strings.Count(s[:off], "\n"); got != want {
				t.Errorf("lineOf(%q, %d) = %d, want %d", s, off, got, want)
			}
		}
	}
}
