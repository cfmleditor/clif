package resolve

import (
	"fmt"
	"regexp"
	"testing"
)

func TestACompiledPatternIsCompiledOnce(t *testing.T) {
	a := compiledPattern(`(?i)\bfoo\d+\b`)
	b := compiledPattern(`(?i)\bfoo\d+\b`)

	if a != b {
		t.Fatal("the same pattern compiled twice")
	}

	if c := compiledPattern(`(?i)\bbar\b`); c == a {
		t.Fatal("two patterns shared one expression")
	}
}

func TestACompiledPatternMatchesWhatItsSourceDoes(t *testing.T) {
	for _, pat := range []string{
		`(?i)\bfunction\s+` + regexp.QuoteMeta("get.Name") + `\s*\(`,
		`arguments\t\.\t` + regexp.QuoteMeta("mode") + `\t(?:eq|=\t=)\t"([^"#]*)"`,
		`(?i)^a|b$`,
	} {
		want := regexp.MustCompile(pat)
		got := compiledPattern(pat)

		for _, s := range []string{"function get.Name (", "FUNCTION getXName(", "arguments\t.\tmode\teq\t\"x\"", "a", "b", ""} {
			if fmt.Sprint(got.FindStringSubmatchIndex(s)) != fmt.Sprint(want.FindStringSubmatchIndex(s)) {
				t.Errorf("%q on %q differs from regexp.MustCompile", pat, s)
			}
		}
	}
}

func TestThePatternCacheStartsAgainPastItsBound(t *testing.T) {
	first := compiledPattern(`zz_bound_first`)

	for i := range maxPatterns + 10 {
		compiledPattern(fmt.Sprintf(`zz_bound_%d`, i))
	}

	if n := patternCount.Load(); n > maxPatterns {
		t.Fatalf("the cache holds %d patterns, past its bound of %d", n, maxPatterns)
	}

	if again := compiledPattern(`zz_bound_first`); !again.MatchString("zz_bound_first") || again == nil {
		t.Fatal("a pattern evicted by the restart no longer compiles")
	}

	_ = first
}
