package parser

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// The indexes behind RefsNamed, FuncDeclares, FuncScopeAt and CallsNamed
// answer what a walk of the whole list answered, on a file large enough that
// each takes its indexed path.
func TestNameIndexesAgreeWithTheWalk(t *testing.T) {
	var b strings.Builder

	b.WriteString("component {\n")

	for i := range 40 {
		fmt.Fprintf(&b, "function f%d( a%d ) {\n", i, i)

		for j := range 40 {
			fmt.Fprintf(&b, "\tvar v%d = new models.T%d();\n\tv%d.go( svc.Run%d() );\n", j, j%3, j, i%4)
		}

		fmt.Fprintf(&b, "\tVariables.Shared = new models.S();\n\tinvoke( this, \"f%d\" );\n\tf%d( 1 );\n}\n", (i+1)%40, (i+1)%40)
	}

	b.WriteString("}\n")

	pr := ParseWithOptions("file:///w/Big.cfc", b.String(), &ParseOptions{ExtractCalls: true})
	if len(pr.Scopes) <= maxScopeScan {
		t.Fatalf("%d scopes: the premise is a file past maxScopeScan", len(pr.Scopes))
	}

	for _, name := range []string{"v3", "V3", "shared", "svc", "nope"} {
		var want []int

		for i := range pr.ComponentRefs {
			if strings.EqualFold(pr.ComponentRefs[i].Variable, name) {
				want = append(want, i)
			}
		}

		if got := pr.RefsNamed(name); !slices.Equal(got, want) {
			t.Errorf("RefsNamed(%q) = %v, the walk %v", name, got, want)
		}
	}

	for line := -1; line <= strings.Count(b.String(), "\n")+1; line++ {
		want := findFuncScope(line, pr.Scopes)
		if got := pr.FuncScopeAt(line); got != want {
			t.Fatalf("FuncScopeAt(%d) = %+v, findFuncScope %+v", line, got, want)
		}

		if want.Start < 0 {
			continue
		}

		for _, name := range []string{"v0", "V39", "a3", "A3", "v40", "svc", "shared"} {
			wantDecl := slices.ContainsFunc(pr.FuncVars(want.Start, want.End), func(v string) bool { return strings.EqualFold(v, name) })
			if got := pr.FuncDeclares(want.Start, want.End, name); got != wantDecl {
				t.Errorf("FuncDeclares(%d, %d, %q) = %v, the walk %v", want.Start, want.End, name, got, wantDecl)
			}
		}
	}

	for _, names := range [][]string{{"f3", "invoke"}, {"F3", "F3"}, {"go"}, {"run1", "RUN2"}, {"nope"}} {
		var want []CallSite

		for _, c := range pr.AllCalls() {
			if slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, c.FuncName) }) {
				want = append(want, c)
			}
		}

		got := pr.CallsNamed(names...)
		if len(got) != len(want) {
			t.Fatalf("CallsNamed(%v): %d calls, AllCalls filtered %d", names, len(got), len(want))
		}

		for i := range got {
			if got[i].Line != want[i].Line || got[i].FuncName != want[i].FuncName || got[i].Variable != want[i].Variable {
				t.Errorf("CallsNamed(%v)[%d] = %+v, AllCalls filtered %+v", names, i, got[i], want[i])
			}
		}
	}
}
