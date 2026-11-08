package parser

import (
	"slices"
	"testing"
)

func bodyRefNamed(refs []ComponentRef, variable string) (ComponentRef, bool) {
	i := slices.IndexFunc(refs, func(r ComponentRef) bool { return r.Variable == variable })
	if i < 0 {
		return ComponentRef{}, false
	}

	return refs[i], true
}

// A tag function's body is parsed by the tag parser on its own, so the lines
// it reports are relative to the body and must be moved back by where the
// function starts.
func TestATagFunctionsRefsAreOnTheLinesTheyAreWrittenOn(t *testing.T) {
	src := `<cfcomponent>
<cffunction name="load">
	<cfset a = new app.One()>
	<cfset b = createObject("component", "app.Two")>
</cffunction>
</cfcomponent>`

	pr := ParseWithOptions("file:///t.cfc", src, &ParseOptions{})

	var scope *FuncScope

	for i := range pr.Scopes {
		if pr.Scopes[i].Name == "load" {
			scope = &pr.Scopes[i]
		}
	}

	if scope == nil {
		t.Fatal("no scope for load")
	}

	refs, _ := pr.FuncRefs(scope.Start, scope.End)

	for _, want := range []struct {
		variable, component string
		line                uint32
	}{{"a", "app.One", 2}, {"b", "app.Two", 3}} {
		got, ok := bodyRefNamed(refs, want.variable)
		if !ok || got.Component != want.component || got.Line != want.line {
			t.Errorf("ref %s = %q at line %d (found %v), want %q at line %d", want.variable, got.Component, got.Line, ok, want.component, want.line)
		}
	}
}

// A function name the parse was asked to find calls of is found in a body
// parsed on its own: the scan reads the lines of the function for it, tag or
// script.
func TestFindCallsReadsAFunctionBodyParsedOnItsOwn(t *testing.T) {
	for _, c := range []struct {
		name, src string
	}{
		{"tag", `<cfcomponent>
<cffunction name="load">
	<cfset b = createObject("component", "app.Two")>
	<cfset b.save()>
</cffunction>
</cfcomponent>`},
		{"script", `component {
	function load() {
		var b = new app.Two();
		b.save();
	}
}`},
	} {
		pr := ParseWithOptions("file:///t.cfc", c.src, &ParseOptions{FindCalls: []string{"save"}})

		for i := range pr.Scopes {
			pr.FuncRefs(pr.Scopes[i].Start, pr.Scopes[i].End)
		}

		found := false

		for i := range pr.Calls {
			if pr.Calls[i].FuncName == "save" && pr.Calls[i].Variable == "b" {
				found = true
			}
		}

		if !found {
			t.Errorf("%s: no call of save on b among %d calls", c.name, len(pr.Calls))
		}
	}
}

// A chained assignment in a body parsed on its own is typed by walking its
// rest through the lookup, so the variable holds the last call's component.
func TestAChainedAssignmentInALazilyParsedBodyIsTyped(t *testing.T) {
	src := `component {
	function load() {
		var a = new app.One();
		var b = a.make().next();
	}
}`
	lookup := func(component, name string) string {
		switch component + "." + name {
		case "app.One.make":
			return "app.Two"
		case "app.Two.next":
			return "app.Three"
		}

		return ""
	}

	pr := ParseWithOptions("file:///s.cfc", src, &ParseOptions{FuncLookup: lookup})
	refs, _ := pr.FuncRefs(pr.Scopes[0].Start, pr.Scopes[0].End)

	got, ok := bodyRefNamed(refs, "b")
	if !ok || got.Component != "app.Three" || len(got.ChainRest) != 0 {
		t.Errorf("b = %q (found %v, rest %v), want app.Three with the rest consumed", got.Component, ok, got.ChainRest)
	}
}
