package parser

import (
	"slices"
	"testing"
)

// Lucee lets a script hold text that is not script: a triple-backtick island,
//
//	query datasource=ds {``` CREATE TABLE t ( v VARCHAR(255) ) ```}
//	```<cfquery name="q"> select sum(x) from t where a = #f(y)# </cfquery>```
//
// and a `java { … }` block. The scanner knows neither, so the SQL and the Java
// were read as code and `VARCHAR(255)`, `sum(x)` and a method's signature were
// recorded as calls: findings in `unresolved` for functions nobody called. The
// SQL's own `#...#` still runs, and is still a call.
func TestTextInAScriptIsNotScript(t *testing.T) {
	for _, c := range []struct {
		name string
		src  string
		want []string
	}{
		{
			"a query body",
			"component { function f() {\n\tquery datasource=ds {```\n\t\tCREATE TABLE t ( id INT, v VARCHAR(255) NOT NULL );\n\t```}\n\tafter();\n} }",
			[]string{"after"},
		},
		{
			"a query body with an interpolation",
			"component { function f() {\n\tquery datasource=ds {```\n\t\tSELECT * FROM t WHERE id = #val( x )# AND n = count(*)\n\t```}\n} }",
			[]string{"val"},
		},
		{
			"a cfquery in an island",
			"component { function f() {\n\t```\n\t<cfquery name=\"q\">\n\t\tselect sum(x), count(*) from t where a = #pick( y )#\n\t</cfquery>\n\t<cfset z = other()>\n\t```\n} }",
			[]string{"other", "pick"},
		},
		{
			"an island with no query is still read",
			"component { function f() {\n\t```<cfset z = other()>```\n} }",
			[]string{"other"},
		},
		{
			"a java block",
			"<cfscript>\nobj = java{\n\tpublic class c{\n\t\tpublic String execute() { return \"x\"; }\n\t}\n}\nwriteOutput( obj.execute() );\n</cfscript>",
			[]string{"execute", "writeOutput"},
		},
		{
			// A fence inside a string opens no island, so what follows is script.
			"a fence in a string",
			"component { function f() {\n\tvar s = \"```\";\n\tbefore();\n\tvar t = \"```\";\n\tafter();\n} }",
			[]string{"after", "before"},
		},
		{
			"a query that is not an island",
			"component { function f() {\n\tquery name=\"q\" datasource=\"ds\" { writeOutput( 1 ); }\n} }",
			[]string{"writeOutput"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := tagFileCalls(t, c.src)
			if !slices.Equal(got, c.want) {
				t.Errorf("got %v want %v", got, c.want)
			}
		})
	}
}
