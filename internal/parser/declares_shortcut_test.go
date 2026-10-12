package parser

import (
	"strings"
	"testing"

	"go.lsp.dev/uri"
)

// mayDeclareVariable lets Declares skip AllVars for a name that is only ever
// called; it must never skip one the file declares.
func TestDeclaresShortcutNeverHidesADeclaration(t *testing.T) {
	for _, tc := range []struct {
		src   string
		name  string
		calls bool // the name never appears but called, so the shortcut answers false
	}{
		{"<cfscript>x = structNew(); y = StructNew ();</cfscript>", "structNew", true},
		{"<cfscript>structNew = 1; x = structNew();</cfscript>", "structNew", false},
		{"<cfscript>function f(){ var structnew = {}; }</cfscript>", "structNew", false},
		{`<cfloop index="structNew" from="1" to="2"></cfloop>`, "structNew", false},
		{"<cfscript>for (structNew in arr) {}</cfscript>", "structNew", false},
		{"<cfscript>x = myStructNew(); y = structNewer();</cfscript>", "structNew", true},
		{"<cfscript>// structNew\n x = 1;</cfscript>", "structNew", false},
	} {
		pr := Parse(uri.File("/t.cfm"), tc.src)

		if got := mayDeclareVariable(tc.src, tc.name); got == tc.calls {
			t.Errorf("mayDeclareVariable(%q) = %v", tc.src, got)
		}

		declared := false
		for _, v := range ParseVars(tc.src) {
			declared = declared || strings.EqualFold(v.Name, tc.name)
		}

		if declared && !mayDeclareVariable(tc.src, tc.name) {
			t.Errorf("%q declares %s but the shortcut says it cannot", tc.src, tc.name)
		}

		if pr.Declares(tc.name) != declared {
			t.Errorf("Declares(%q) in %q = %v, want %v", tc.name, tc.src, pr.Declares(tc.name), declared)
		}
	}
}
