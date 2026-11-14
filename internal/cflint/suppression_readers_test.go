package cflint

import (
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The expressions suppressions.go replaced, kept as the oracle each reader is
// held to.
var (
	oracleIgnoreTag  = regexp.MustCompile(`@CFLintIgnore\s+([\w,]+)`)
	oracleLineIgnore = regexp.MustCompile(`//\s*cflint\s+ignore:([\w,]+)`)
)

func oracleLists(re *regexp.Regexp, s string) []string {
	var out []string
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		out = append(out, m[1])
	}

	return out
}

var suppressionCases = []string{
	"// @CFLintIgnore MISSING_VAR",
	"<!--- @CFLintIgnore A,B --->", "/* @CFLintIgnore A, B */", "@CFLintIgnore\tA\n@CFLintIgnore\nB",
	"@CFLintIgnore", "@CFLintIgnore ", "@CFLintIgnore ,", "@CFLintIgnore  ,,,x", "@CFLintIgnoreX", "@cflintignore A", "@CFLintIgnore@CFLintIgnore A",
	"@CFLintIgnore @CFLintIgnore A", "@CFLintIgnore A@CFLintIgnore B", "@CFLintIgnore\vA", "@CFLintIgnore\fA",
	"// cflint ignore:MISSING_VAR", "//cflint ignore:A,B", "//   cflint   ignore:A", "// cflint ignore: A", "// cflint ignore:",
	"// cflint ignore:,", "///cflint ignore:A", "////cflint ignore:A", "/ /cflint ignore:A", "// cflintignore:A", "// cflint ignore :A",
	"// CFLINT ignore:A", "// cflint IGNORE:A", "//\ncflint ignore:A", "// cflint\nignore:A", "//cflint\tignore:A",
	"// cflint ignore:A// cflint ignore:B", "x // cflint ignore:A\ny // cflint ignore:B,C\n", "//", "// cflint", "// cflint ignore",
	"// @CFLintIgnore A\n// cflint ignore:B", "é @CFLintIgnore é", "\xff@CFLintIgnore A\xff", "@CFLintIgnore é", "// cflint ignore:é",
}

var suppressionFragments = []string{
	"//", "/", "/*", "*/", "<!---", "--->", "@CFLintIgnore", "@CFLint", "Ignore", "cflint", "CFLINT", "ignore", "ignore:", ":", " ", "  ",
	"\t", "\n", "\r", "\f", "\v", ",", ", ", "A", "MISSING_VAR", "B_2", "x", "1", "_", "-", ".", "é", "\xff", "@",
}

func randomSuppression(rng *rand.Rand) string {
	if rng.IntN(2) == 0 {
		s := suppressionCases[rng.IntN(len(suppressionCases))]

		for range rng.IntN(4) {
			i := rng.IntN(len(s) + 1)
			j := min(len(s), i+rng.IntN(3))

			switch rng.IntN(3) {
			case 0:
				s = s[:i] + suppressionFragments[rng.IntN(len(suppressionFragments))] + s[i:]
			case 1:
				s = s[:i] + s[j:]
			default:
				s = s[:i] + suppressionFragments[rng.IntN(len(suppressionFragments))] + s[j:]
			}
		}

		return s
	}

	var b strings.Builder

	for range 1 + rng.IntN(14) {
		b.WriteString(suppressionFragments[rng.IntN(len(suppressionFragments))])
	}

	return b.String()
}

func suppressionDisagreement(s string) []string {
	var bad []string

	if got, want := fmt.Sprintf("%q", ignoreTagLists(s)), fmt.Sprintf("%q", oracleLists(oracleIgnoreTag, s)); got != want {
		bad = append(bad, fmt.Sprintf("ignoreTagLists(%q) = %s, want %s", s, got, want))
	}

	if got, want := fmt.Sprintf("%q", lineIgnoreLists(s)), fmt.Sprintf("%q", oracleLists(oracleLineIgnore, s)); got != want {
		bad = append(bad, fmt.Sprintf("lineIgnoreLists(%q) = %s, want %s", s, got, want))
	}

	return bad
}

// Both readers answer what their expressions answered, on the forms CFLint
// honours, their near misses, and texts built from fragments of them; a text in
// twenty or more holds a suppression, so the comparison is not of two refusals.
func TestSuppressionReadersMatchTheirExpressions(t *testing.T) {
	n := 200_000
	if testing.Short() {
		n = 20_000
	}

	texts := append([]string(nil), suppressionCases...)

	rng := rand.New(rand.NewPCG(1, 7))
	for range n {
		texts = append(texts, randomSuppression(rng))
	}

	failures, found := 0, 0

	for _, s := range texts {
		for _, msg := range suppressionDisagreement(s) {
			t.Error(msg)

			if failures++; failures == 10 {
				return
			}
		}

		if len(oracleLists(oracleIgnoreTag, s))+len(oracleLists(oracleLineIgnore, s)) > 0 {
			found++
		}
	}

	if found < len(texts)/20 {
		t.Errorf("%d of %d texts hold a suppression; the generated texts do not reach the readers", found, len(texts))
	}
}

// FuzzSuppressionReaders holds each reader to its expression on any text.
func FuzzSuppressionReaders(f *testing.F) {
	for _, s := range suppressionCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		for _, msg := range suppressionDisagreement(s) {
			t.Error(msg)
		}
	})
}

// TestSuppressionReadersAgreeOnACorpus is the same comparison over every CFML
// file under CLIF_SUPPRESSIONS_CORPUS (directories, as PATH is). Skipped
// without it, so it never runs in CI.
func TestSuppressionReadersAgreeOnACorpus(t *testing.T) {
	corpus := os.Getenv("CLIF_SUPPRESSIONS_CORPUS")
	if corpus == "" {
		t.Skip("CLIF_SUPPRESSIONS_CORPUS not set")
	}

	files, found := 0, 0

	for _, dir := range filepath.SplitList(corpus) {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}

			switch strings.ToLower(filepath.Ext(path)) {
			case ".cfc", ".cfm", ".cfml":
			default:
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil //nolint:nilerr // a file that cannot be read is not compared
			}

			files++

			for _, msg := range suppressionDisagreement(string(data)) {
				t.Errorf("%s: %s", path, msg)
			}

			if len(oracleLists(oracleIgnoreTag, string(data)))+len(oracleLists(oracleLineIgnore, string(data))) > 0 {
				found++
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Logf("%d files compared, %d holding a suppression", files, found)
}
