package parser

import "testing"

func TestComponentDeclarationBounds(t *testing.T) {
	for _, c := range []struct {
		name, src, at string // at is the text the offset points at, with what follows
		ok            bool
	}{
		{"script", "component {\n function f() {}\n}", "{\n function", true},
		{"with attributes", `component extends="a.B" accessors=true {` + "\n}", "{\n}", true},
		{"a brace in a quoted value", `component hint="a {b} c" {` + "\n}", "{\n}", true},
		{"a doc comment first", "/**\n * @return {struct}\n */\ncomponent {\n}", "{\n}", true},
		{"a modifier", "abstract component {\n}", "{\n}", true},
		{"tag", "<cfcomponent output=\"false\">\n<cffunction name=\"f\"></cffunction>", ">\n<cffunction", true},
		{"tag with a > in a value", `<cfcomponent hint="a > b">` + "\nx", ">\nx", true},
		{"upper case", "COMPONENT {\n}", "{\n}", true},
		{"no component", "<cfset x = 1>", "", false},
		{"a struct literal in a tag before the declaration", "<cfset x = {}>\n<cfcomponent>\n</cfcomponent>", ">\n</cfcomponent", true},
		{"a balanced block before the declaration", "{ x = 1 } component {}", "{}", true},
		{"an unbalanced brace before any declaration", "{ x = 1 component {}", "", false},
		{"an import before the declaration", "import a.b;\ncomponent {\n}", "{\n}", true},
		{"component as a value is no declaration", "x = component;\nimport a;", "", false},
		{"a tag whose name is only spelled cfcomponent", "<cfset cfcomponent = 1>", "", false},
		{"a script interface", "interface {\n}", "{\n}", true},
		{"a tag interface", "<cfinterface>\n</cfinterface>", ">\n</cfinterface", true},
		{"empty", "", "", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			at, _, ok := ComponentDeclarationBounds(c.src)
			if ok != c.ok {
				t.Fatalf("ok is %v, want %v", ok, c.ok)
			}

			if ok && c.src[at:min(len(c.src), at+len(c.at))] != c.at {
				t.Errorf("offset %d is at %q, want %q", at, c.src[at:], c.at)
			}
		})
	}
}

func TestComponentBodyEnd(t *testing.T) {
	for _, c := range []struct {
		name, src, at string
		tags, ok      bool
	}{
		{"script", "component {\n function f() { if (a) {} }\n}\n", "}\n", false, true},
		{"a brace in a string and a comment", "component {\n x = \"}\"; // }\n /* } */\n}\ntrailing", "}\ntrailing", false, true},
		{"nested components of closures", "component {\n f = function() { return {a: 1}; };\n}\n", "}\n", false, true},
		{"tags", "<cfcomponent>\n<cffunction name=\"f\"></cffunction>\n</cfcomponent>\n", "</cfcomponent>", true, true},
		{"tags, upper case", "<CFCOMPONENT>\n</CFCOMPONENT>", "</CFCOMPONENT>", true, true},
		{"never closed", "component {\n function f() {}\n", "", false, false},
		{"no component", "x = 1", "", false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			at, tags, ok := ComponentBodyEnd(c.src)
			if ok != c.ok || ok && tags != c.tags {
				t.Fatalf("tags %v ok %v, want %v %v", tags, ok, c.tags, c.ok)
			}

			if ok && c.src[at:min(len(c.src), at+len(c.at))] != c.at {
				t.Errorf("offset %d is at %q, want %q", at, c.src[at:], c.at)
			}
		})
	}
}

func TestAnAttributeNamedComponentIsNotTheDeclaration(t *testing.T) {
	src := "<cfinvoke component=\"x\" method=\"y\">\n<cfcomponent>\n</cfcomponent>"

	open, _, ok := ComponentDeclarationBounds(src)
	if !ok || src[open:min(len(src), open+len(">\n</cfcomponent"))] != ">\n</cfcomponent" {
		t.Errorf("offset %d is at %q, want the > of <cfcomponent>", open, src[open:])
	}
}
