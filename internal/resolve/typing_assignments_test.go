package resolve

import (
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

// TestReceiverTypingReadsTheParsesAssignments: the last assignment receiver
// typing reads is one the parse reads, where a statement can stand. The line
// reader it replaced took a line beginning `name =` for one wherever it was
// (JavaScript in a template, a commented-out block, a struct literal's key, a
// script-syntax tag's attribute on a line of its own) and missed one that was
// not first on its line.
func TestReceiverTypingReadsTheParsesAssignments(t *testing.T) {
	for _, tc := range []struct {
		file, content, name string
		use                 int
		want                string
	}{{
		// Not first on its line.
		"a.cfc", "component {\n\tfunction f() {\n\t\tvar x = \"\";\n\t\tif ( ok ) { x = new A(); }\n\t\tx.run();\n\t}\n}\n",
		"x", 4, "new A()",
	}, {
		"a.cfm", "<cfset x = \"\">\n<cfif ok><cfset x = new A()></cfif>\n<cfset x.run()>\n",
		"x", 2, "new A()",
	}, {
		// JavaScript in a template is not CFML.
		"a.cfm", "<cfset x = new A()>\n<script>\n\tvar x = $( this );\n</script>\n<cfset x.run()>\n",
		"x", 4, "new A()",
	}, {
		// Nor is a commented-out block.
		"a.cfc", "component {\n\tfunction f() {\n\t\tvar x = new A();\n\t\t/*\n\t\tx = new B();\n\t\t*/\n\t\tx.run();\n\t}\n}\n",
		"x", 6, "new A()",
	}, {
		// A struct literal's key on a line of its own.
		"a.cfm", "<cfscript>\nfirst = new A();\nresults = {\n\tfirst = first.get(),\n\tsecond = 2\n};\nfirst.run();\n</cfscript>\n",
		"first", 6, "new A()",
	}, {
		// A script-syntax tag's attributes on lines of their own.
		"a.cfc", "component {\n\tfunction f() {\n\t\tvar name = new A();\n\t\tlock\n\t\t\tname = \"l\"\n\t\t\ttype = \"exclusive\" {\n\t\t}\n\t\tname.run();\n\t}\n}\n",
		"name", 7, "new A()",
	}, {
		// A value that runs on with an operator at the start of a line.
		"a.cfm", "<cfset title = a.b\n\t& \" - \" & c>\n<cfset title.run()>\n",
		"title", 2, `a.b & " - " & c`,
	}, {
		// A declaration after a statement with no `;`, which the parse runs
		// on into it: still the declaration.
		"a.cfc", "component {\n\tfunction f() {\n\t\tvar n = 0\n\t\tn++\n\t\tvar x = new A()\n\t\tx.run()\n\t}\n}\n",
		"x", 5, "new A()",
	}, {
		// A file that begins with a byte-order mark.
		"a.cfm", "\xef\xbb\xbf<cfset x = new A()>\n<cfset x.run()>\n",
		"x", 1, "new A()",
	}} {
		pr := parser.ParseWithOptions(uri.URI("file:///w/"+tc.file), tc.content, &parser.ParseOptions{})

		got, _, _ := (&Resolver{}).localAssignmentAt(pr, tc.name, 0, tc.use)
		if got != tc.want {
			t.Errorf("%q: %s is %q, want %q", tc.content, tc.name, got, tc.want)
		}
	}
}

// TestAForLoopsInitializerIsNotTheVariablesValue: `for ( var i = 1; …` is
// not read as what i holds after the loop, as the line reader did not read it
// either.
func TestAForLoopsInitializerIsNotTheVariablesValue(t *testing.T) {
	const content = "component {\n\tfunction f() {\n\t\tfor ( var i = 1; i <= 3; i++ ) {\n\t\t}\n\t\ti.run();\n\t}\n}\n"

	pr := parser.ParseWithOptions("file:///w/a.cfc", content, &parser.ParseOptions{})
	if got, _, ok := (&Resolver{}).localAssignmentAt(pr, "i", 0, 4); ok {
		t.Errorf("i is %q", got)
	}
}

// TestAReassignmentFarDownALongFileIsSeen: whether a native type a ref claims
// was overwritten before a use is read from the parse's assignments. The
// line reader's check that an assignment was not a named argument read the
// whole text above it, and gave up past 4,096 tokens, or when markup above
// left a bracket open: a reassignment there was never seen.
func TestAReassignmentFarDownALongFileIsSeen(t *testing.T) {
	var b strings.Builder

	b.WriteString("component {\n")

	for range 1200 {
		b.WriteString("\tfunction g() { return [ 1, 2 ]; }\n")
	}

	start := 1201

	b.WriteString("\tfunction f() {\n\t\tvar x = [];\n\t\tx = new A();\n\t\tx.run();\n\t}\n}\n")

	content := b.String()
	pr := parser.ParseWithOptions("file:///w/a.cfc", content, &parser.ParseOptions{})

	if !nativeReassignedAt(pr.Assignments(), content, "x", start+2, start+3) {
		t.Error("x = new A() was not seen")
	}

	// A named argument is no reassignment.
	const named = "component {\n\tfunction f() {\n\t\tvar x = [];\n\t\tg(\n\t\t\tx = 1\n\t\t);\n\t\tx.len();\n\t}\n}\n"

	pr = parser.ParseWithOptions("file:///w/b.cfc", named, &parser.ParseOptions{})
	if nativeReassignedAt(pr.Assignments(), named, "x", 3, 6) {
		t.Error("a named argument was read as a reassignment")
	}
}
