package knownissues

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The expressions entryLine and entryTag replaced, kept as the oracle each
// reader is held to.
var (
	oracleEntryLine = regexp.MustCompile(`^((?:[A-Za-z]:)?[^:]+):(\d+)(?::(\d+))?:\s*(.+)$`)
	oracleEntryTag  = regexp.MustCompile(`^\[(?:(?i:(error|warning|warn|information|info|hint))\s+)?([A-Za-z0-9_.:-]+)\]\s*(.*)$`)
)

var entryCases = []string{
	"a.cfc:1: msg", "a.cfc:1:2: msg", "a/b/c.cfm:10:5: [warning CFQUERYPARAM_REQ] <cfquery> should use <cfqueryparam/>", "a.cfc:1: [TODO] move this",
	"a.cfc:12:34", "a.cfc:12:34:", "a.cfc:12:34: ", "a.cfc:12:", "a.cfc:12", "a.cfc:", "a.cfc", ":1: x", "::", ":", "a.cfc::1: x", "a.cfc:1::x",
	"a:12:5: msg", "a:12: msg", "a:12:34: x", "C:\\x\\y.cfc:12: msg", "C:/x/y.cfc:1:2: msg", "C:x.cfc:1: m", "C::1: m", "C:1: m", "ab:1: m", "1:2: m", "é:1: m",
	"a.cfc:1:\tmsg", "a.cfc:1:   msg  ", "a.cfc:1:\n", "a.cfc:1:\nx", "a.cfc:1: x\ny", "a.cfc:1: \n", "a.cfc:1: \n x", "a.cfc:1:2: \n", "a.cfc:1:2:  ",
	"a.cfc:1:\u00a0x", "a.cfc:1:\v", "a.cfc:1:\vx", "a.cfc:1:\f", "a.cfc:1:\r", "a.cfc:1: x:2: y", "a.cfc:1:2:3: x", "a.cfc:01:002: x",
	"a b.cfc:1: x", "../b/c.cfc:3: x", "/abs/x.cfc:1: x", "\\\\srv\\x.cfc:1: x", "a.cfc:99999999999999999999: x", "a.cfc:1:99999999999999999999: x",
	"[error] x", "[error x] y", "[ERROR x] y", "[warning X-1] y", "[warn x] y", "[info x] y", "[information x] y", "[hint x] y", "[infos x] y", "[error  \tx]",
	"[error]", "[error] ", "[x]", "[x] ", "[x]y", "[x]\n", "[x]\ny", "[x] \ny", "[]", "[ ]", "[error ]", "[error x", "[x", "x]", "[x y]", "[a.b:c-d_e]", "[é]", "[a é]",
	"[error x]\u00a0y", "[error\u00a0x] y", "[error\nx] y", "[error x] y\nz", "[warning warning] y", "[warning] warning", "[hint info] z", "[TODO]", "[ TODO]",
	"[error x][y]", "[[x]]", "[x]]", "[error x] [y] z", "[wArN x]", "[INFO\tx]", "", " ", "\xff", "[\xff]", "[error \xff]",
}

var entryFragments = []string{
	"a", "b.cfc", "x/y", "C", "C:", "a:", ":", "::", ":1", ":12", ":1:2", ": ", " ", "  ", "\t", "\n", "\r", "\f", "\v", "\\", "/", "..", "1", "22", "0", "9999999999",
	"[", "]", "[error", "[warning", "[warn", "[info", "[information", "[hint", " x]", "error", "ERROR", "warn", "info", "hint", "CODE_1", "-", ".", "_", "é", "\u00a0", "\xff",
}

func randomEntry(rng *rand.Rand) string {
	if rng.IntN(2) == 0 {
		s := entryCases[rng.IntN(len(entryCases))]

		for range rng.IntN(4) {
			i := rng.IntN(len(s) + 1)
			j := min(len(s), i+rng.IntN(3))

			switch rng.IntN(3) {
			case 0:
				s = s[:i] + entryFragments[rng.IntN(len(entryFragments))] + s[i:]
			case 1:
				s = s[:i] + s[j:]
			default:
				s = s[:i] + entryFragments[rng.IntN(len(entryFragments))] + s[j:]
			}
		}

		return s
	}

	var b strings.Builder

	for range 1 + rng.IntN(12) {
		b.WriteString(entryFragments[rng.IntN(len(entryFragments))])
	}

	return b.String()
}

func entryDisagreement(s string) []string {
	var bad []string

	check := func(name, got, want string) {
		if got != want {
			bad = append(bad, fmt.Sprintf("%s(%q) = %s, want %s", name, s, got, want))
		}
	}

	path, line, col, msg, ok := entryLine(s)
	m := oracleEntryLine.FindStringSubmatch(s)
	want := "false"

	if m != nil {
		want = fmt.Sprintf("true %q %q %q %q", m[1], m[2], m[3], m[4])
	}

	got := "false"
	if ok {
		got = fmt.Sprintf("true %q %q %q %q", path, line, col, msg)
	}

	check("entryLine", got, want)

	severity, code, rest, ok := entryTag(s)
	m = oracleEntryTag.FindStringSubmatch(s)
	want = "false"

	if m != nil {
		want = fmt.Sprintf("true %q %q %q", m[1], m[2], m[3])
	}

	got = "false"
	if ok {
		got = fmt.Sprintf("true %q %q %q", severity, code, rest)
	}

	check("entryTag", got, want)

	return bad
}

// unfoldedEntryRune is a character a regular expression's (?i) folds to ASCII
// and the reader, folding in ASCII alone, does not: the Kelvin sign.
func unfoldedEntryRune(s string) bool { return strings.ContainsAny(s, "Kſ") }

// Both readers answer what their expressions answered, groups included, on
// the lines the format writes, near misses of each, and texts built from their
// fragments; at least one in ten of each is a match, so the comparison is not
// of two refusals.
func TestEntryReadersMatchTheirExpressions(t *testing.T) {
	n := 300_000
	if testing.Short() {
		n = 30_000
	}

	texts := append([]string(nil), entryCases...)

	rng := rand.New(rand.NewPCG(1, 7))
	for range n {
		texts = append(texts, randomEntry(rng))
	}

	failures, lines, tags := 0, 0, 0

	for _, s := range texts {
		if unfoldedEntryRune(s) {
			continue
		}

		for _, msg := range entryDisagreement(s) {
			t.Error(msg)

			if failures++; failures == 10 {
				return
			}
		}

		if oracleEntryLine.MatchString(s) {
			lines++
		}

		if oracleEntryTag.MatchString(s) {
			tags++
		}
	}

	if lines < len(texts)/10 || tags < len(texts)/20 {
		t.Errorf("%d lines and %d tags of %d texts; the generated texts do not reach the readers", lines, tags, len(texts))
	}
}

// FuzzEntryReaders holds each reader to its expression on any text.
func FuzzEntryReaders(f *testing.F) {
	for _, s := range entryCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		if unfoldedEntryRune(s) {
			t.Skip()
		}

		for _, msg := range entryDisagreement(s) {
			t.Error(msg)
		}
	})
}

// TestEntryReadersAgreeOnRealFiles compares the readers to the expressions on
// every line of the files named by CLIF_KNOWNISSUES_FILES (a list, as PATH
// is): the reports `clif unresolved` writes, and any known-issues file.
// Skipped without it, so it never runs in CI.
func TestEntryReadersAgreeOnRealFiles(t *testing.T) {
	list := os.Getenv("CLIF_KNOWNISSUES_FILES")
	if list == "" {
		t.Skip("CLIF_KNOWNISSUES_FILES not set")
	}

	lines, entries, tagged := 0, 0, 0

	for _, path := range filepath.SplitList(list) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		for raw := range strings.SplitSeq(string(data), "\n") {
			line := strings.TrimSpace(raw)
			lines++

			for _, msg := range entryDisagreement(line) {
				t.Errorf("%s: %s", path, msg[:min(len(msg), 200)])
			}

			if m := oracleEntryLine.FindStringSubmatch(line); m != nil {
				entries++

				if oracleEntryTag.MatchString(strings.TrimSpace(m[4])) {
					tagged++
				}

				for _, msg := range entryDisagreement(strings.TrimSpace(m[4])) {
					t.Errorf("%s: %s", path, msg[:min(len(msg), 200)])
				}
			}
		}
	}

	t.Logf("%d lines, %d entries, %d with a tag", lines, entries, tagged)
}
