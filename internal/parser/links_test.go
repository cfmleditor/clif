package parser

import (
	"slices"
	"testing"
)

func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"cfinclude tag", `<cfinclude template="a.cfm">`, []string{"a.cfm"}},
		{"cfmodule tag", `<cfmodule template = "m.cfm" x="1">`, []string{"m.cfm"}},
		{"href on markup", `<p><a href="page.cfm">x</a></p>`, []string{"page.cfm"}},
		{"form action", `<form action="post.cfm"></form>`, []string{"post.cfm"}},
		{"action elsewhere is no link", `<button action="go.cfm">x</button>`, nil},
		{"a CF tag's action is a keyword", `<cfdirectory action="list" directory="d" name="q">`, nil},
		{"href in an expression tag", `<cfset href = "x.cfm">`, nil},
		{"an unquoted value", `<cfinclude template=a.cfm>`, nil},
		{"in a CFML comment", "<!--- <cfinclude template=\"c.cfm\"> --->\n<cfinclude template=\"d.cfm\">", []string{"d.cfm"}},
		{"dynamic", `<cfinclude template="#x#.cfm">`, nil},
		{"absolute URL", `<a href="https://example.com/x.cfm">x</a>`, nil},
		{"markup cut by a CF tag", `<a <cfif x>class="on"</cfif> href="p.cfm">x</a>`, []string{"p.cfm"}},
		{"a closing tag is stepped over", `<a href="one.cfm"></a><a href="two.cfm"></a>`, []string{"one.cfm", "two.cfm"}},
		{"script include", "<cfscript>\ninclude \"s.cfm\";\n</cfscript>", []string{"s.cfm"}},
		{"script include template", "<cfscript>\ninclude template=\"t.cfm\";\n</cfscript>", []string{"t.cfm"}},
		{"script cfinclude call", "<cfscript>\ncfinclude( template=\"f.cfm\" );\n</cfscript>", []string{"f.cfm"}},
		{"script cfmodule call", "<cfscript>\ncfmodule( template = \"mod.cfm\" );\n</cfscript>", []string{"mod.cfm"}},
		{"script comment", "<cfscript>\n// include \"no.cfm\";\n/* include \"no2.cfm\"; */\n</cfscript>", nil},
		{"member include is no link", "<cfscript>\narr.include(\"x.cfm\");\n</cfscript>", nil},
		{"a struct key named template", "<cfscript>\ns = { template = \"k.cfm\" };\n</cfscript>", nil},
		{"tag written in a script component", "component {\nfunction f() {\n<cfinclude template=\"p.cfm\">\n}\n}", []string{"p.cfm"}},
		{"tag-free markup file", `<link href="style.cfm">`, []string{"style.cfm"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, l := range ExtractLinks(tt.content) {
				got = append(got, l.Path)
			}

			if !slices.Equal(got, tt.want) {
				t.Errorf("links = %q, want %q", got, tt.want)
			}
		})
	}
}
