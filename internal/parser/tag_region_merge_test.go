package parser

import (
	"slices"
	"testing"
)

// A tag region after a script block is parsed on its own and its lines moved
// to the file's: the links it finds, the links and refs of a function that a
// script block splits across regions, and the function itself.
func TestATagRegionsFindingsAreOnTheLinesOfTheFile(t *testing.T) {
	src := `<cfscript>
	pre = 1;
</cfscript>
<cfinclude template="top.cfm">
<cffunction name="load">
	<cfset a = new app.One()>
	<cfinclude template="inner.cfm">
	<cfscript>
		mid = 1;
	</cfscript>
	<cfset b = new app.Two()>
	<cfinclude template="after.cfm">
	<cfreturn b>
</cffunction>
<cfinclude template="last.cfm">`

	pr := ParseWithOptions("file:///t.cfm", src, &ParseOptions{ExtractLinks: true})

	links := map[string]uint32{}
	for _, l := range pr.Links {
		links[l.Path] = l.Line
	}

	// Lines count from zero.
	for path, line := range map[string]uint32{"top.cfm": 3, "after.cfm": 11, "last.cfm": 14} {
		if got, ok := links[path]; !ok || got != line {
			t.Errorf("link %s on line %d (found %v), want %d", path, got, ok, line)
		}
	}

	var load *FuncScope

	for i := range pr.Scopes {
		if pr.Scopes[i].Name == "load" {
			load = &pr.Scopes[i]
		}
	}

	if load == nil {
		t.Fatal("no scope for load")
	}

	// The function's links are those the parse filed under it: after.cfm, in
	// the region after the <cfscript> block, is filed for the file (above),
	// as the walk of that region files it.
	_, funcLinks := pr.FuncRefs(load.Start, load.End)

	inFunc := map[string]uint32{}
	for _, l := range funcLinks {
		inFunc[l.Path] = l.Line
	}

	for path, line := range map[string]uint32{"inner.cfm": 6} {
		if got, ok := inFunc[path]; !ok || got != line {
			t.Errorf("function link %s on line %d (found %v), want %d", path, got, ok, line)
		}
	}

	// An unscoped <cfset> writes the variables scope: the file's refs.
	byVar := map[string]ComponentRef{}
	for i := range pr.ComponentRefs {
		byVar[pr.ComponentRefs[i].Variable] = pr.ComponentRefs[i]
	}

	if a := byVar["a"]; a.Component != "app.One" || a.Line != 5 {
		t.Errorf("a = %q on line %d, want app.One on 5", a.Component, a.Line)
	}

	if b := byVar["b"]; b.Component != "app.Two" || b.Line != 10 {
		t.Errorf("b = %q on line %d, want app.Two on 10", b.Component, b.Line)
	}

	if i := slices.IndexFunc(pr.Funcs, func(f FunctionDef) bool { return f.Name == "load" }); i < 0 || pr.Funcs[i].Line != 4 {
		t.Errorf("load is declared on line %v, want 4", pr.Funcs)
	}
}
