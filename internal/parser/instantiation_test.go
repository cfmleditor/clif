package parser

import (
	"slices"
	"testing"
)

// TestInstantiationsAreRecordedWhereverWritten: the code map's instantiates
// edge came from component-level refs alone, so a local `var s = new Svc()`,
// an inline `new Svc().handle()` and one in an argument list drew none (#308).
// Every construction is recorded, in either syntax, with the line it is on.
func TestInstantiationsAreRecordedWhereverWritten(t *testing.T) {
	cases := map[string]struct {
		src   string
		lines []uint32
	}{
		"script": {
			src: "component {\nfunction f() {\n" +
				"createObject(\"component\", \"shared.Svc\").handle();\n" + // 2
				"g(new shared.Svc());\n" + // 3
				"g(createObject(\"component\", \"shared.Svc\"));\n" + // 4
				"return createObject(\"component\", \"shared.Svc\");\n" + // 5
				"}\n" +
				"function h() { new shared.Svc(); var x = [new shared.Svc()]; var y = { a: new shared.Svc() }; z = 1 + new shared.Svc().n(); }\n" + // 7, four times
				"function i() { var s = new shared.Svc(); variables.t = new shared.Svc(); writeOutput(\"new shared.Svc()\"); }\n" + // 8, twice; not the string
				"}\n",
			lines: []uint32{2, 3, 4, 5, 7, 7, 7, 7, 8, 8},
		},
		"tag": {
			src: "<cffunction name=\"tf\">\n" +
				"<cfset var s = new shared.Svc()>\n" + // 1
				"<cfset createObject(\"component\", \"shared.Svc\").handle()>\n" + // 2
				"<cfreturn new shared.Svc()>\n" + // 3
				"</cffunction>\n" +
				"<cfset application.x = new shared.Svc()>\n", // 5
			lines: []uint32{1, 2, 3, 5},
		},
	}

	for name, c := range cases {
		pr := ParseWithOptions("file:///x/A.cfc", c.src, &ParseOptions{ExtractCalls: true})

		var lines []uint32

		for _, in := range pr.Instantiations {
			if in.Component != "shared.Svc" {
				t.Errorf("%s: instantiation of %q on line %d", name, in.Component, in.Line)
			}

			lines = append(lines, in.Line)
		}

		slices.Sort(lines)

		if !slices.Equal(lines, c.lines) {
			t.Errorf("%s: instantiations on lines %v, want %v", name, lines, c.lines)
		}
	}
}

// TestAParseWithoutCallsRecordsNoInstantiations: the record is the code map's,
// and is gated as a call is.
func TestAParseWithoutCallsRecordsNoInstantiations(t *testing.T) {
	pr := Parse("file:///x/A.cfc", "component { function f() { var s = new shared.Svc(); } }")
	if len(pr.Instantiations) != 0 {
		t.Errorf("recorded %+v", pr.Instantiations)
	}
}
