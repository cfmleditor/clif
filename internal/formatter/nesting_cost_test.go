package formatter

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/language"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func nestingOptions() Options {
	o := DefaultOptions()
	o.ParseScript = func(b []byte) *sitter.Tree { return language.Parse(language.CFScript, b, nil) }
	o.ParseQuery = func(b []byte) *sitter.Tree { return language.Parse(language.CFQuery, b, nil) }
	o.ParseCFML = func(b []byte) *sitter.Tree { return language.Parse(language.CFML, b, nil) }

	return o
}

// renders formats src and reports how many expressions were rendered.
func renders(t *testing.T, src string) uint {
	t.Helper()

	o := nestingOptions()
	o.LineWidth = 20

	tree := language.Parse(language.CFML, []byte(src), nil)
	defer tree.Close()

	f := New(&o)
	f.src = []byte(src)
	f.formatNode(tree.RootNode())

	if f.parseErr != nil {
		t.Fatalf("format: %v", f.parseErr)
	}

	return f.exprCalls
}

// Laying out an argument list, a struct or a pair renders its entries at two
// levels, so the work doubled with every level of nesting: a 160-byte file of
// calls 52 deep took 38 seconds. Doubling the depth must cost polynomially
// more, not exponentially.
func TestNestingCostIsPolynomial(t *testing.T) {
	shapes := map[string]func(n int) string{
		"calls":   func(n int) string { return strings.Repeat("a(1, ", n) + "1" + strings.Repeat(")", n) },
		"structs": func(n int) string { return strings.Repeat("{ k: ", n) + "1" + strings.Repeat(" }", n) },
		"methods": func(n int) string { return strings.Repeat("o.m(1, ", n) + "1" + strings.Repeat(")", n) },
	}

	for name, shape := range shapes {
		t.Run(name, func(t *testing.T) {
			src := func(n int) string { return "component { function f() { x = " + shape(n) + "; } }" }

			small, large := renders(t, src(12)), renders(t, src(24))

			// Polynomial in depth is at most a few times the cube here; the
			// doubling it replaced is 2^12 = 4096 times.
			if ratio := float64(large) / float64(small); ratio > 64 {
				t.Errorf("rendering %s nested 24 deep cost %d, 12 deep %d: %.0fx", name, large, small, ratio)
			}
		})
	}
}

// A tree too deep to lay out is refused, not formatted: the output grows with
// depth times lines, and 16,000 nested blocks came out as 256MB.
func TestTooDeepToFormatIsRefused(t *testing.T) {
	o := nestingOptions()

	cases := map[string]struct {
		src     string
		refused bool
	}{
		"blocks within the limit": {"component { function f() { " + strings.Repeat("if (a) { ", 300) + "x = 1;" + strings.Repeat(" }", 300) + " } }", false},
		"blocks past the limit":   {"component { function f() { " + strings.Repeat("if (a) { ", 600) + "x = 1;" + strings.Repeat(" }", 600) + " } }", true},
		"calls within the limit":  {"component { function f() { x = " + strings.Repeat("a(1, ", maxLayoutNesting) + "1" + strings.Repeat(")", maxLayoutNesting) + "; } }", false},
		"calls past the limit":    {"component { function f() { x = " + strings.Repeat("a(1, ", maxLayoutNesting+1) + "1" + strings.Repeat(")", maxLayoutNesting+1) + "; } }", true},
		"structs past the limit":  {"<cfset x = " + strings.Repeat("{ k: ", maxLayoutNesting+1) + "1" + strings.Repeat(" }", maxLayoutNesting+1) + ">", true},
		"a long concatenation":    {"component { function f() { x = a" + strings.Repeat(" & a", 900) + "; } }", false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tree := language.Parse(language.CFML, []byte(tc.src), nil)
			defer tree.Close()

			_, err := Format([]byte(tc.src), tree, &o)

			refused := err != nil && strings.Contains(err.Error(), "deep")
			if refused != tc.refused {
				t.Errorf("Format = %v; refused %v, want %v", err, refused, tc.refused)
			}
		})
	}
}

// Past its deadline a format stops and is refused rather than running on.
func TestFormatGivesUpAtItsDeadline(t *testing.T) {
	previous := maxFormatTime
	maxFormatTime = time.Nanosecond

	t.Cleanup(func() { maxFormatTime = previous })

	src := "component { function f() { " + strings.Repeat("x = a(b, c) + d(e, f);\n", 3000) + " } }"
	tree := language.Parse(language.CFML, []byte(src), nil)

	defer tree.Close()

	o := nestingOptions()
	if _, err := Format([]byte(src), tree, &o); !errors.Is(err, errFormatTimeout) {
		t.Errorf("Format past its deadline = %v, want errFormatTimeout", err)
	}
}
