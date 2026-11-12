package parser

import (
	"slices"
	"testing"
)

func calledNames(pr *ParseResult) []string {
	calls := pr.AllCalls()
	out := make([]string, 0, len(calls))

	for i := range calls {
		out = append(out, calls[i].Variable+"."+calls[i].FuncName)
	}

	return out
}

func assignedValues(pr *ParseResult) map[string]string {
	out := map[string]string{}

	for i := range pr.Assignments() {
		a := &pr.Assignments()[i]
		out[a.Target] = a.ValueLine(pr.Content, len(pr.Content))
	}

	return out
}

// A Lucee tag island in script is tags, as the grammar reads it: a <cfset>
// in one is an assignment ending at the tag, a <cffunction> is a function,
// and a local declared in one is the script function's around it.
func TestATagIslandInScriptIsReadAsTags(t *testing.T) {
	src := "component {\n function a() {\n  ```<cfset var x = createObject(\"component\", \"a.B\")>```\n  x.go();\n }\n ```<cffunction name=\"tagged\"><cfreturn svc.run()></cffunction>```\n function b() { tagged(); }\n}"
	pr := ParseWithOptions("file:///t.cfc", src, &ParseOptions{ExtractCalls: true})

	if got := assignedValues(pr)["x"]; got != `createObject("component", "a.B")` {
		t.Errorf("x is assigned %q, want the <cfset>'s value alone", got)
	}

	var names []string
	for _, f := range pr.Funcs {
		names = append(names, f.Name)
	}

	if !slices.Contains(names, "tagged") {
		t.Errorf("functions %v, want the island's tagged", names)
	}

	if calls := calledNames(pr); !slices.Contains(calls, "x.go") || !slices.Contains(calls, "svc.run") {
		t.Errorf("calls %v, want x.go and the island's svc.run", calls)
	}

	refs, _ := pr.FuncRefs(1, 4)
	if len(refs) == 0 || refs[0].Variable != "x" {
		t.Errorf("a()'s refs %+v, want the island's x", refs)
	}
}

// A ``` in a string or a comment is no island, and a query's SQL island is
// left to the SQL blanking.
func TestTagIslandsAreCodeOnly(t *testing.T) {
	if got := tagIslands("s = \"```<cfset a = 1>```\"; // ```<b>```\n"); got != nil {
		t.Errorf("islands %v in a string and a comment", got)
	}

	if got := tagIslands("query name=\"q\" {```select 1```}"); got != nil {
		t.Errorf("islands %v in a query's SQL", got)
	}
}

// An arrow function's `=>` in a <cfset> does not end the tag.
func TestAnArrowInCfsetDoesNotEndTheTag(t *testing.T) {
	src := "<cfset f = (x) => x * 2>\n<cfset s = {a: () => svc.run()}>\n<cfset y = f(3)>"
	pr := ParseWithOptions("file:///t.cfm", src, &ParseOptions{ExtractCalls: true})

	got := assignedValues(pr)
	if got["f"] != "(x) => x * 2" || got["y"] != "f(3)" {
		t.Errorf("assignments %v", got)
	}

	if calls := calledNames(pr); !slices.Contains(calls, "svc.run") {
		t.Errorf("calls %v, want the arrow body's svc.run", calls)
	}

	if end := tagEndIndex("<cfset b = x >= y>"); end != len("<cfset b = x ") {
		t.Errorf("`>=` should still end the tag at its '>', got %d", end)
	}
}

// Code after a comment of dashes alone is read.
func TestCodeAfterAnAllDashCommentIsRead(t *testing.T) {
	pr := ParseWithOptions("file:///t.cfm", "<!------------------>\n<cfset a = svc.run()>", &ParseOptions{ExtractCalls: true})

	if calls := calledNames(pr); !slices.Contains(calls, "svc.run") {
		t.Errorf("calls %v, want svc.run", calls)
	}
}
