package unresolved

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

// The regular expressions reason_text.go replaced, kept as the oracle each
// reader is held to.
var (
	oracleFailedMethod    = regexp.MustCompile(`^method '([^']+)' not found`)
	oracleMethodNotFound  = regexp.MustCompile(`^method '[^']+' not found in (.+)$`)
	oracleMissingCompoent = regexp.MustCompile(`^component '([^']+)' does not exist`)
	oracleUnchecked       = regexp.MustCompile(`; \d+ (?:inherited )?calls? not checked`)
	oracleQuoted          = regexp.MustCompile(`'[^']*'|"[^"]*"`)
	oracleNumber          = regexp.MustCompile(`\b\d+\b`)
)

// reasonCases are the reasons the resolver writes, and near misses of each.
var reasonCases = []string{
	"method 'run' not found in mura.servletEvent|mura.event",
	"method 'getEmail' not found in Foo (chain to 'bar')",
	"method 'x' not found on builtin native.string",
	"method '' not found in X", "method 'a' not found", "method 'a' not found in ", "method 'a' not found in X\nY",
	"method 'a'  not found in X", "method 'a' not foundX", "Method 'a' not found in X", " method 'a' not found in X",
	"component 'javaLoader' does not exist (calling 'create')", "component '' does not exist", "component 'a' does not exist",
	"component 'a' does not exis", "component 'a.b.C' does not exist (chain hop 'x' to 'y')",
	"base component does not resolve; 20 inherited calls not checked", "base component does not resolve; 1 call not checked",
	"x; 3 calls not checked; 4 inherited call not checked", "; 5 calls not checked", ";  5 calls not checked", "; 5 callsnot checked",
	"; 5 inherited inherited calls not checked", "; a calls not checked", "; 5calls not checked", "; 12345678901234567890 calls not checked",
	"chained on 'getBean', which is not found (calling 'validate')", `variable "x" has no component ref`, "it's 'a", `a 'b' c "d" e 'f`,
	"line 12 col 7 in file2.cfc v3 x1 2y _4 4_ 5", "12 34", "a1 22b c33", "é1 1é", "7", "", "'", "\"'\"'", "'\n'",
}

var reasonFragments = []string{
	"method", " ", "'", "\"", " not found", " not found in ", " does not exist", "component", "x", "Y", ".", "(", ")", ";", "; ",
	"1", "23", "4567", "a1", "1a", "_", "inherited ", "call", "calls", " not checked", "\n", "\r", "\t", "é", "\xff", "#", "|",
	"method '", "component '", "chain to ",
}

func randomReason(rng *rand.Rand) string {
	if rng.IntN(2) == 0 {
		s := reasonCases[rng.IntN(len(reasonCases))]

		for range rng.IntN(4) {
			i := rng.IntN(len(s) + 1)
			j := min(len(s), i+rng.IntN(3))

			switch rng.IntN(3) {
			case 0:
				s = s[:i] + reasonFragments[rng.IntN(len(reasonFragments))] + s[i:]
			case 1:
				s = s[:i] + s[j:]
			default:
				s = s[:i] + reasonFragments[rng.IntN(len(reasonFragments))] + s[j:]
			}
		}

		return s
	}

	var b strings.Builder

	for range 1 + rng.IntN(14) {
		b.WriteString(reasonFragments[rng.IntN(len(reasonFragments))])
	}

	return b.String()
}

// agree is every reader's answer to s against its expression's.
func agree(s string) []string {
	var bad []string

	check := func(name, got, want string) {
		if got != want {
			bad = append(bad, fmt.Sprintf("%s(%q) = %s, want %s", name, s, got, want))
		}
	}

	name, ok := failedMethodName(s)
	m := oracleFailedMethod.FindStringSubmatch(s)
	check("failedMethodName", fmt.Sprint(name, ok), fmt.Sprint(firstSubmatch(m), m != nil))

	comp, ok := methodMissingFrom(s)
	m = oracleMethodNotFound.FindStringSubmatch(s)
	check("methodMissingFrom", fmt.Sprint(comp, ok), fmt.Sprint(firstSubmatch(m), m != nil))

	comp, ok = missingComponent(s)
	m = oracleMissingCompoent.FindStringSubmatch(s)
	check("missingComponent", fmt.Sprint(comp, ok), fmt.Sprint(firstSubmatch(m), m != nil))

	check("withoutUncheckedCount", fmt.Sprintf("%q", withoutUncheckedCount(s)), fmt.Sprintf("%q", oracleUnchecked.ReplaceAllString(s, "")))
	check("quotesBlanked", fmt.Sprintf("%q", quotesBlanked(s)), fmt.Sprintf("%q", oracleQuoted.ReplaceAllString(s, "'_'")))
	check("numbersBlanked", fmt.Sprintf("%q", numbersBlanked(s)), fmt.Sprintf("%q", oracleNumber.ReplaceAllString(s, "N")))

	return bad
}

func firstSubmatch(m []string) string {
	if len(m) < 2 {
		return ""
	}

	return m[1]
}

// Every reader answers what its expression answered, captures included, on
// the reasons the resolver writes and on texts built from them and from
// fragments of them.
func TestReasonReadersMatchTheirExpressions(t *testing.T) {
	n := 200_000
	if testing.Short() {
		n = 20_000
	}

	texts := append([]string(nil), reasonCases...)

	rng := rand.New(rand.NewPCG(1, 7))
	for range n {
		texts = append(texts, randomReason(rng))
	}

	failures := 0

	for _, s := range texts {
		for _, msg := range agree(s) {
			t.Error(msg)

			failures++

			if failures == 10 {
				return
			}
		}
	}
}

// FuzzReasonReaders holds each reader to its expression on any text.
func FuzzReasonReaders(f *testing.F) {
	for _, s := range reasonCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		for _, msg := range agree(s) {
			t.Error(msg)
		}
	})
}
