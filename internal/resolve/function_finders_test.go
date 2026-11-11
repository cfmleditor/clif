package resolve

import (
	"slices"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// The finders that count a component's braces read its strings as the parse
// does. In ColdBox's Router.cfc a #...# holds a quoted "{": read with a plain
// scanner, the string ends at that quote, the { counts as a brace, and every
// function after it was lost to all three.
func TestFunctionFindersReadStringsAsTheParseDoes(t *testing.T) {
	src := `component {
	function first() {
		thisRegex = listFirst( thisRegex, "{" ) & "{#listLast( thisPattern, "{" )#)";
		return thisRegex;
	}

	function after() {
		return 1;
	}
}
`
	if names := parser.ApplicationMethodNames(src); !slices.Contains(names, "after") {
		t.Errorf("ApplicationMethodNames = %v, want after", names)
	}

	if _, ok := methodsOf(src)["after"]; !ok {
		t.Errorf("wheelsMethods lost after")
	}

	if newProducerPlans(src, parser.FunctionSpansOf("", src)).get("after") == nil {
		t.Errorf("producer plans lost after")
	}
}

// The finders read the parse's functions (parser.FunctionSpans) rather than
// finding them again, so what the parse does not read as a function no
// finder does either: a function in a comment, JavaScript in a page's
// <script> block. And what it reads, each one reads: a method declared after
// an access, a modifier and a `type[]` return, and a <cffunction> in a
// tag component with a <cfscript> island.
func TestFunctionFindersReadTheParsesFunctions(t *testing.T) {
	script := `component {
	/* function ghost() { return 1; } */
	private string[] function names() { return []; }
	public function init( entityName = "cbAuthor" ) { return this; }
}
`
	methods := methodsOf(script)
	if _, ghost := methods["ghost"]; ghost {
		t.Errorf("wheelsMethods found the commented-out ghost")
	}

	if m, ok := methods["names"]; !ok || m.public {
		t.Errorf("names = %+v, %v: want a private method", m, ok)
	}

	if got := initArgumentDefault(script, parser.FunctionSpansOf("", script), "entityName"); got != "cbAuthor" {
		t.Errorf("initArgumentDefault = %q, want cbAuthor", got)
	}

	page := `<html><script>function track() { return 1; }</script>
<cfscript>function helper() { return 2; }</cfscript></html>`

	spans := parser.FunctionSpansOf("page.cfm", page)
	if _, ok := methodsOf(page)["track"]; ok {
		t.Errorf("wheelsMethods read the page's JavaScript")
	}

	if body := functionBodyNamed(page, spans, "helper"); body != "return 2;" {
		t.Errorf("functionBodyNamed(helper) = %q", body)
	}

	tag := `<cfcomponent>
<cffunction name="load" hint="a > b"><cfargument name="entityName" default="cbPage"><cfreturn 1></cffunction>
</cfcomponent><cfscript>function scripted() { return 3; }</cfscript>`
	tagSpans := parser.FunctionSpansOf("", tag)

	if bodies := tagFunctionBodies(tag, tagSpans); strings.TrimSpace(parser.WithoutCFTags(bodies["load"], "cfargument")) != "<cfreturn 1>" {
		t.Errorf("tagFunctionBodies = %q", bodies)
	}

	if actions := handlerActions(tagSpans); len(actions) != 2 || actions[0].name != "load" || actions[1].name != "scripted" {
		t.Errorf("handlerActions = %+v", actions)
	}
}

// methodsOf is wheelsMethods over content's functions, as a source reads
// them.
func methodsOf(content string) map[string]wheelsMethod {
	return wheelsMethods(content, parser.FunctionSpansOf("", content))
}
