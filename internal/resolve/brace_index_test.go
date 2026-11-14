package resolve

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// oracleBlockClosed is blockClosedBetween as it was: the braces of the span
// scanned from its start.
func oracleBlockClosed(content string, fromLine, toLine int) bool {
	from, ok := lineOffset(content, fromLine)
	if !ok {
		return false
	}

	to, ok := lineOffset(content, toLine)
	if !ok || to <= from {
		return false
	}

	sc := parser.NewSourceScanner(content[from:to])
	depth := 0

	for {
		switch tok := sc.NextSkipComments(); tok.Kind {
		case parser.TokEOF:
			return false
		case parser.TokLBrace:
			depth++
		case parser.TokRBrace:
			depth--
			if depth < 0 {
				return true
			}
		default:
		}
	}
}

var braceLines = []string{
	"{", "}", "} else {", "x = 1;", "if (a) {", "foo({a: 1});", "// }", "/* { */", `s = "}";`, `s = "{";`,
	"y = {};", "}}", "{{", "", "function f() {", "return x;", "z = [1, 2];",
}

// The index answers what a scan of the span answered, for every span of
// generated scripts, braces in comments and strings included. A span that
// begins inside a comment spanning lines is left out: the scan began in the
// middle of the comment and read it as code, where the index, which reads the
// text from its top, is right.
func TestBraceIndexMatchesAScanOfTheSpan(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 9))
	closed, open := 0, 0

	for range 2_000 {
		lines := make([]string, 4+rng.IntN(20))
		for i := range lines {
			lines[i] = braceLines[rng.IntN(len(braceLines))]
		}

		content := "component {\n" + strings.Join(lines, "\n") + "\n}\n"
		n := len(lines) + 3

		for range 12 {
			from, to := rng.IntN(n)+1, rng.IntN(n)+1
			got, want := blockClosedBetween(content, from, to), oracleBlockClosed(content, from, to)

			if got != want {
				t.Fatalf("lines %d to %d of\n%s\ngot %v, want %v", from, to, content, got, want)
			}

			if want {
				closed++
			} else {
				open++
			}
		}
	}

	if closed < 500 || open < 500 {
		t.Errorf("%d closing and %d open spans: the generated scripts do not reach both answers", closed, open)
	}
}
