package parser

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// expressionEnds answers scriptReturnExpression for every token from one
// pass; at every token of each source it must give what a scan from there
// gives.
func checkExpressionEnds(t *testing.T, src string) int {
	t.Helper()

	ends := newExpressionEnds(src)
	sc := NewScanner(src)
	sc.interpStrings = true
	bad := 0

	for {
		state := sc.Save()
		tok := sc.NextSkipComments()

		probe := NewScanner(src)
		probe.interpStrings = true
		probe.Restore(state)

		if want, got := scriptReturnExpression(probe), ends.from(probe.PeekSkipComments().Offset); want != got {
			bad++
			if bad <= 3 {
				t.Errorf("at %d: got %q, want %q", tok.Offset, got, want)
			}
		}

		if tok.Kind == TokEOF {
			return bad
		}
	}
}

func TestExpressionEndsMatchScriptReturnExpression(t *testing.T) {
	for _, src := range []string{
		"x = a.b(c, d)\n y = {k = 1, j = [2, 3]}; z = f()",
		"function f() { var x = 1\n var y = g(x)\n return y }",
		"a = (b) ) ; c = d } e = f",
		"",
		"x = '#y(\"z\")#'; w = 1",
	} {
		checkExpressionEnds(t, src)
	}
}

func TestExpressionEndsOverCorpus(t *testing.T) {
	root := os.Getenv("ZZ_CORPUS")
	if root == "" {
		t.Skip("set ZZ_CORPUS to a directory of CFML")
	}

	files, bad := 0, 0
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(p, ".cfc") || files >= 300 {
			return nil
		}

		data, _ := os.ReadFile(p)
		if len(data) > 60000 {
			return nil
		}

		files++
		bad += checkExpressionEnds(t, string(data))

		return nil
	})

	t.Logf("%d files, %d differences", files, bad)
}
