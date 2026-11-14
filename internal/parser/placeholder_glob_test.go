package parser

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

// oracleGlob is the expression a configuration's multi-placeholder `match` was
// compiled to, built as it was: each part quoted, a lazy run between.
func oracleGlob(parts []string) *regexp.Regexp {
	var b strings.Builder

	b.WriteString("(?i)")

	for i, part := range parts {
		b.WriteString(regexp.QuoteMeta(part))

		if i < len(parts)-1 {
			b.WriteString(`(.+?)`)
		}
	}

	return regexp.MustCompile(b.String())
}

var globAlphabet = []string{"a", "B", "get", "Get", "Service", "x", "", ".", "(", ")", "\"", "'", " ", "ab", "aa", "é", "$", "*", "+"}

func randomGlobText(rng *rand.Rand, n int) string {
	var b strings.Builder

	for range rng.IntN(n) {
		if rng.IntN(12) == 0 {
			b.WriteString("\n")
		} else {
			b.WriteString(globAlphabet[rng.IntN(len(globAlphabet))])
		}
	}

	return b.String()
}

// The glob answers what the expression built from the same parts answered,
// captures included: the leftmost start, and for each run the fewest characters
// that let what follows be found.
func TestPlaceholderGlobMatchesItsExpression(t *testing.T) {
	n := 100_000
	if testing.Short() {
		n = 10_000
	}

	rng := rand.New(rand.NewPCG(1, 7))
	failures, matched := 0, 0

	for range n {
		parts := make([]string, 2+rng.IntN(3))
		for i := range parts {
			parts[i] = randomGlobText(rng, 3)
		}

		re := oracleGlob(parts)
		glob := &placeholderGlob{parts: parts}

		for range 6 {
			expr := randomGlobText(rng, 8)
			if rng.IntN(2) == 0 {
				// An expression that holds the parts, with something between.
				expr = randomGlobText(rng, 3) + strings.Join(parts, randomGlobText(rng, 3)) + randomGlobText(rng, 3)
			}

			got, ok := glob.Match(expr)
			m := re.FindStringSubmatch(expr)

			var want []string
			if m != nil {
				want = m[1:]
				matched++
			}

			if ok != (m != nil) || fmt.Sprintf("%q", got) != fmt.Sprintf("%q", want) {
				t.Errorf("parts %q on %q: %q %v, want %q %v", parts, expr, got, ok, want, m != nil)

				failures++

				if failures == 8 {
					return
				}
			}
		}
	}

	if matched < n/2 {
		t.Errorf("%d matches of %d patterns; the generated texts do not reach the glob", matched, n*6)
	}
}
