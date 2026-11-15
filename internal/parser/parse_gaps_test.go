package parser

import (
	"slices"
	"strings"
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

// A function body read alone, as after an edit inside it, reads its tag
// island as the whole parse does: as tags, in the function's scope, and not
// again as script.
func TestABodyReadAloneReadsItsTagIsland(t *testing.T) {
	src := "component {\n function a() {\n  ```<cfset var x = svc.make()>```\n  x.go();\n }\n}"
	pr := ParseWithOptions("file:///t.cfc", src, &ParseOptions{ExtractCalls: true})

	names := func() []string {
		var out []string
		for _, c := range pr.FuncCalls(1, 4) {
			out = append(out, c.Variable+"."+c.FuncName)
		}

		slices.Sort(out)

		return out
	}

	before := names()

	pr.ApplyEdit(3, 2, 3, 2, " ")

	if after := names(); !slices.Equal(after, before) {
		t.Errorf("a()'s calls after an edit %v, want %v as the whole parse read them", after, before)
	}
}

// A <cffunction> in a tag island has a scope of its own, in a parse that
// reads scopes alone as in the whole parse.
func TestAnIslandFunctionHasAScope(t *testing.T) {
	src := "component {\n ```\n <cffunction name=\"tagged\">\n  <cfreturn 1>\n </cffunction>\n ```\n function b() { return 2; }\n}"

	for _, opts := range []*ParseOptions{{}, {Shallow: true, ScopesOnly: true}} {
		pr := ParseWithOptions("file:///t.cfc", src, opts)

		var names []string
		for _, s := range pr.Scopes {
			names = append(names, s.Name)
		}

		if !slices.Contains(names, "tagged") {
			t.Errorf("scopes %v (scopes only %v), want the island's tagged", names, opts.ScopesOnly)
		}
	}
}

// A <cffunction> after a comment of five dashes is a function with a scope:
// the scope pre-pass reads comments with the one comment reader.
func TestAFunctionAfterAFiveDashCommentHasAScope(t *testing.T) {
	src := "<!----->\n<cfcomponent>\n<cffunction name=\"f\">\n<cfset var x = createObject(\"component\", \"a.B\")>\n</cffunction>\n</cfcomponent>"
	pr := ParseWithOptions("file:///t.cfc", src, &ParseOptions{ExtractCalls: true})

	if len(pr.Scopes) != 1 || pr.Scopes[0].Name != "f" {
		t.Fatalf("scopes %+v, want f", pr.Scopes)
	}

	if refs, _ := pr.FuncRefs(pr.Scopes[0].Start, pr.Scopes[0].End); len(refs) != 1 {
		t.Errorf("f's refs %+v, want x", refs)
	}
}

// A '<' in an arrow's body is an operator, and markup's `value=>` is an
// attribute, not an arrow.
func TestArrowsAndMarkupEndTagsAsTheGrammarDoes(t *testing.T) {
	src := "<cfset f = (a, b) => a<b>\n<cfset g = 1>"
	if tags := CFTags(src, "cfset"); len(tags) != 2 || tags[0].Body() != "f = (a, b) => a<b" {
		t.Errorf("cfset tags %+v", tags)
	}

	if end := tagEndIndex("<option value=>None</option>"); end != len("<option value=") {
		t.Errorf("markup `value=>` ended at %d, want its '>'", end)
	}
}

// An arrow in a quoted value is none, so an HTML tag holding a CF tag is cut
// at it; and markup's `value =>` ends at its '>' in the document walk.
func TestOnlyACFTagsArrowRunsPastItsGT(t *testing.T) {
	src := `<a data-fn="(x) => x" <cfif sel>class="on"</cfif>>`
	ends := newTagEnds(src)

	if got, want := tagEnd(&ends, len("<a")), strings.Index(src, "<cfif"); got != want {
		t.Errorf("the <a> tag ends at %d, want the <cfif> at %d", got, want)
	}

	markup := "<input value =><cfset y = svc.run()>"
	ends = newTagEnds(markup)

	if got := ends.at(len("<input")); got != len(" value =") {
		t.Errorf("markup's `value =>` ended at %d, want its '>'", got)
	}

	cf := "<cfset f = (x) => x * 2>"
	ends = newTagEnds(cf)

	if got := ends.at(len("<cfset")); got != len(cf)-len("<cfset")-1 {
		t.Errorf("the <cfset>'s arrow ended the tag at %d", got)
	}
}

// An edit outside a tag island keeps the split, moved, with the island
// still blank in its script region.
func TestAnEditOutsideAnIslandKeepsTheSplit(t *testing.T) {
	src := "component {\n function a() {\n  ```<cfset var x = svc.make()>```\n  x.go();\n }\n}"
	pr := ParseWithOptions("file:///t.cfc", src, &ParseOptions{ExtractCalls: true})
	pr.FuncCalls(1, 4) // the body state the edit carries over

	pr.ApplyEdit(3, 2, 3, 2, " ")

	b := &pr.bodies
	if b.regionsOf != pr.Content {
		t.Fatal("the split was dropped rather than moved")
	}

	for _, r := range b.regions {
		if r.Kind == RegionScript && strings.Contains(r.Text, "svc.make") {
			t.Errorf("the script region holds the island's text: %q", r.Text)
		}
	}
}

// A body walked alone reads the same calls as the whole walk: the island's
// once, from its own region, and none from the script region holding it.
func TestABodyWalkReadsAnIslandOnce(t *testing.T) {
	src := "component {\n function a() {\n  ```<cfset var x = svc.make()>```\n  x.go();\n }\n}"
	pr := ParseWithOptions("file:///t.cfc", src, &ParseOptions{ExtractCalls: true})
	want := len(pr.FuncCalls(1, 4))

	pr.bodies.mu.Lock()
	walk := pr.walkBody(FuncScope{Start: 1, End: 4}, false, true)
	pr.bodies.mu.Unlock()

	if got := len(walk.AllCalls()); got != want {
		t.Errorf("the body walk read %d calls, the whole walk %d", got, want)
	}
}
