package resolve

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// The regular expressions wild_pattern.go replaced, built as they were, kept as
// the oracle each pattern is held to.

func oracleInvokeSelects(selector, name string) bool {
	if !strings.Contains(selector, "#") {
		return strings.EqualFold(selector, name)
	}

	pieces := strings.Split(selector, "#")
	if len(pieces)%2 == 0 || strings.Contains(selector, "##") || pieces[0] == "" {
		return false
	}

	var pattern strings.Builder

	pattern.WriteString("(?i)^")

	for i, piece := range pieces {
		if i%2 == 1 {
			pattern.WriteString(".+")
		} else {
			pattern.WriteString(regexp.QuoteMeta(piece))
		}
	}

	pattern.WriteString("$")

	return regexp.MustCompile(pattern.String()).MatchString(name)
}

func oracleSpelled(value []parser.Token, wildcard string, trim bool) (string, bool) {
	var b strings.Builder

	literal := false
	wild := func() {
		if !strings.HasSuffix(b.String(), wildcard) {
			b.WriteString(wildcard)
		}
	}

	for _, t := range value {
		switch t.Kind {
		case parser.TokString:
			text := t.Value
			if len(text) >= 2 {
				text = text[1 : len(text)-1]
			}

			for i, part := range strings.Split(text, "#") {
				if i%2 == 1 {
					wild()

					continue
				}

				if trim && b.Len() == 0 {
					part = strings.TrimPrefix(part, "/")
				}

				if part != "" {
					b.WriteString(regexp.QuoteMeta(part))

					literal = true
				}
			}
		case parser.TokAmpersand:
		default:
			wild()
		}
	}

	return b.String(), literal
}

func oracleInterceptTarget(value []parser.Token) *regexp.Regexp {
	body, literal := oracleSpelled(value, `\w*`, false)
	if !literal {
		return nil
	}

	return regexp.MustCompile(`(?i)^` + body + `$`)
}

func oracleMuraObjectTarget(value []parser.Token) *regexp.Regexp {
	body, literal := oracleSpelled(value, `[^/]*`, true)
	if !literal {
		return nil
	}

	return regexp.MustCompile(`(?i)(?:^|/)` + body + `$`)
}

var wildStrings = []string{
	"a", "b", "ab", "A", "x/y", "/", "//", "a/", "/a", "tpl", "Tpl", "tpl.cfm", ".cfm", "on", "On", "start", "end", "_", "1", ".", "*", "\\", "(", "[", "$", "^", "é", "\n", " ", "",
	"-", "a-b", ":", "#", "9", "Z", "\t", "\r", "\x00", "+",
}

var wildSelectors = []string{
	"compile#x#", "#x#compile", "a#x#b", "a#x#b#y#c", "a##b", "a#x", "#x#", "a#", "a#x#", "abc", "ABC", "a#b#c#d#e", "a.b#x#.c", "a(#x#)", "a\n#x#b",
}

func randomWildString(rng *rand.Rand, n int) string {
	var b strings.Builder

	for range rng.IntN(n) {
		b.WriteString(wildStrings[rng.IntN(len(wildStrings))])
	}

	return b.String()
}

func randomValue(rng *rand.Rand) []parser.Token {
	var out []parser.Token

	for range 1 + rng.IntN(5) {
		switch rng.IntN(6) {
		case 0:
			out = append(out, parser.Token{Kind: parser.TokAmpersand, Value: "&"})
		case 1:
			out = append(out, parser.Token{Kind: parser.TokIdent, Value: "x"})
		default:
			quote := []string{`"`, `'`}[rng.IntN(2)]

			// A string with `#expressions#` in it: literal and computed parts alternate.
			var inner strings.Builder

			for i := range 1 + rng.IntN(5) {
				if i > 0 {
					inner.WriteString("#")
				}

				inner.WriteString(randomWildString(rng, 3))
			}

			out = append(out, parser.Token{Kind: parser.TokString, Value: quote + inner.String() + quote})
		}
	}

	return out
}

// Each pattern answers what its expression answered, on spelled values built
// from strings, `#computed#` parts and other tokens, against names built from
// the same strings; the patterns are not all empty, and a fair share of the
// names match, so the comparison is not of two refusals.
func TestWildPatternsMatchTheirExpressions(t *testing.T) {
	n := 60_000
	if testing.Short() {
		n = 10_000
	}

	rng := rand.New(rand.NewPCG(1, 7))
	failures, intercept, mura, selectors, matched := 0, 0, 0, 0, 0

	for range n {
		value := randomValue(rng)

		interceptNew, interceptOld := interceptTarget(value), oracleInterceptTarget(value)
		muraNew, muraOld := muraObjectTarget(value), oracleMuraObjectTarget(value)

		if (interceptNew == nil) != (interceptOld == nil) || (muraNew == nil) != (muraOld == nil) {
			t.Errorf("%v: nil intercept %v/%v, nil mura %v/%v", value, interceptNew == nil, interceptOld == nil, muraNew == nil, muraOld == nil)

			if failures++; failures == 10 {
				return
			}

			continue
		}

		for range 6 {
			name := randomWildString(rng, 5)
			if interceptOld != nil && rng.IntN(3) == 0 {
				// A name made of the value's own literals fits more often.
				name = oracleFill(value, rng)
			}

			if interceptOld != nil {
				intercept++

				if got, want := interceptNew.match(name), interceptOld.MatchString(name); got != want {
					t.Errorf("intercept %v on %q = %v, want %v", value, name, got, want)

					failures++
				} else if want {
					matched++
				}
			}

			if muraOld != nil {
				mura++

				path := name
				if rng.IntN(2) == 0 {
					path = randomWildString(rng, 3) + "/" + name
				}

				if got, want := muraNew.match(path), muraOld.MatchString(path); got != want {
					t.Errorf("mura %v on %q = %v, want %v", value, path, got, want)

					failures++
				} else if want {
					matched++
				}
			}
		}

		for range 3 {
			selector := wildSelectors[rng.IntN(len(wildSelectors))]
			if rng.IntN(2) == 0 {
				selector += randomWildString(rng, 2)
			}

			name := randomWildString(rng, 5)
			if rng.IntN(2) == 0 {
				name = strings.ReplaceAll(selector, "#", "") + randomWildString(rng, 2)
			}

			selectors++

			if got, want := invokeSelects(selector, name), oracleInvokeSelects(selector, name); got != want {
				t.Errorf("invokeSelects(%q, %q) = %v, want %v", selector, name, got, want)

				failures++
			} else if want {
				matched++
			}
		}

		if failures >= 10 {
			return
		}
	}

	if intercept < n/2 || mura < n/2 || selectors < n || matched < n/10 {
		t.Errorf("%d intercept, %d mura and %d selector comparisons, %d matching, of %d values; the generated texts do not reach the patterns", intercept, mura, selectors, matched, n)
	}
}

// oracleFill is a name the value could spell: each string's literal parts, and
// a few characters for what is computed.
func oracleFill(value []parser.Token, rng *rand.Rand) string {
	var b strings.Builder

	for _, t := range value {
		if t.Kind != parser.TokString {
			b.WriteString(randomWildString(rng, 2))

			continue
		}

		text := t.Value
		if len(text) >= 2 {
			text = text[1 : len(text)-1]
		}

		for i, part := range strings.Split(text, "#") {
			if i%2 == 1 {
				b.WriteString(randomWildString(rng, 2))
			} else {
				b.WriteString(part)
			}
		}
	}

	return b.String()
}

var _ = fmt.Sprint
