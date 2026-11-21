package parser

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
		"x = \"multi\nline\"\ny = 2\nz = 'same line' w = 3",
		"x = 1 /* multi\nline comment */ y = 2\nz = 3",
	} {
		checkExpressionEnds(t, src)
	}
}

func TestCollectWritesFunctionCursorMatchesScopes(t *testing.T) {
	src := strings.Join([]string{
		"component {",
		"	variables.top = new Top();",
		"	function outer() {",
		"		var a = new Alpha();",
		"		function inner() {",
		"			var b = new Beta();",
		"		}",
		"		var c = new Gamma();",
		"	}",
		"	variables.between = new Mid();",
		"	function second() {",
		"		var d = new Delta();",
		"	}",
		"	variables.tail = new Tail();",
		"}",
	}, "\n")

	pr := ParseWithOptions("file:///test/NestedScopes.cfc", src, nil)
	writes := pr.collectWrites(nil)
	lines := NewLineTable(pr.Content)

	refFunctionAt := func(offset int) string {
		line := lines.LineOf(offset)

		idx := sort.Search(len(pr.Scopes), func(i int) bool { return pr.Scopes[i].Start > line }) - 1
		if idx >= 0 && line <= pr.Scopes[idx].End {
			return pr.scopeKey(idx)
		}

		return ""
	}

	wantFunc := map[string]string{
		"variables.top":     "",
		"a":                 "2:8",
		"b":                 "4:6",
		"c":                 "",
		"variables.between": "",
		"d":                 "10:12",
		"variables.tail":    "",
	}

	seen := map[string]string{}

	for _, w := range writes {
		if w.target != "" {
			seen[w.target] = w.function
			if ref := refFunctionAt(w.offset); w.function != ref {
				t.Errorf("write %q at offset %d: cursor got %q, reference sort.Search got %q", w.target, w.offset, w.function, ref)
			}
		}
	}

	for target, want := range wantFunc {
		got, ok := seen[target]
		if !ok {
			t.Errorf("missing write for %q", target)

			continue
		}

		if got != want {
			t.Errorf("write %q function = %q, want %q", target, got, want)
		}
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
