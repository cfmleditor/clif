package resolve

import (
	"strings"
	"testing"
)

func TestQueryOutputReplacesLiteralInitializer(t *testing.T) {
	for _, output := range []string{`cfquery(name="rows"){echo("select 1");}`, `query name="local.rows" {echo("select 1");}`, `query name="rows" {echo("select 1");}`, `cfquery(name="rows",sql="select 1");`, `cfdbinfo(name="rows",type="tables",datasource="db");`} {
		t.Run(output, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 var rows="";
 ` + output + `
 rows.columnData("id").len();
 rows.each(function(row,index,collection){row.keyArray();index.toString();collection.columnArray();});
 }
 function negative(){
 var rows="";
 ` + output + `
 rows.absent();
 }}`})

			got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
			for _, name := range []string{"rows.columnData.len", "row.keyArray", "index.toString", "collection.columnArray"} {
				if reason, ok := got[name]; !ok || reason != "" {
					t.Errorf("%s: %q present=%v", name, reason, ok)
				}
			}

			if !strings.Contains(got["rows.absent"], "native.query") {
				t.Errorf("wrong query identity: %q", got["rows.absent"])
			}
		})
	}
}

func TestQueryOutputScopeAndUnknownWriteBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, output, call string }{
		{"short circuit", `unknown && cfdbinfo(name="rows",type="tables");`, "rows.columnData(\"id\");"},
		{"unbraced conditional", `if(unknown) cfdbinfo(name="rows",type="tables");`, "rows.columnData(\"id\");"},
		{"closing branch on call line", `if(unknown){query name="rows" {echo("select 1");}`, "}rows.columnData(\"id\");"},
		{"conditional", `if(unknown){query name="rows" {echo("select 1");}}`, "rows.columnData(\"id\");"},
		{"catch", `try{query name="rows" {echo("select 1");}}catch(any e){`, "rows.columnData(\"id\");}"},
		{"later assignment", `query name="rows" {echo("select 1");}\n rows=runtimeValue();`, "rows.columnData(\"id\");"},
		{"duplicate name", `query name="other" name="rows" {echo("select 1");}`, "rows.columnData(\"id\");"},
		{"colliding outputs", `query name="rows" result="rows" {echo("select 1");}`, "rows.columnData(\"id\");"},
		{"different scope", `query name="variables.rows" {echo("select 1");}`, "rows.columnData(\"id\");"},
		{"unfinished parenthesized output", `cfquery(name="rows"){`, "rows.columnData(\"id\");}"},
		{"unfinished output", `query name="rows" {`, "rows.columnData(\"id\");}"},
		{"comment", `//query name="rows" {echo("select 1");}`, "rows.columnData(\"id\");"},
		{"query literal function", `query(name="rows");`, "rows.columnData(\"id\");"},
		{"dynamic name", `query name="#destination#" {echo("select 1");}`, "rows.columnData(\"id\");"},
		{"attribute collection", `query attributeCollection=attrs {echo("select 1");}`, "rows.columnData(\"id\");"},
		{"HQL default", `query name="rows" dbtype="hql" {echo("from Entity");}`, "rows.columnData(\"id\");"},
		{"dynamic database type", `query name="rows" dbtype=kind {echo("select 1");}`, "rows.columnData(\"id\");"},
		{"async", `query name="rows" async=true {echo("select 1");}`, "rows.columnData(\"id\");"},
		{"listener", `query name="rows" listener=handler {echo("select 1");}`, "rows.columnData(\"id\");"},
		{"different return type", `query name="rows" returntype="array" {echo("select 1");}`, "rows.columnData(\"id\");"},
		{"later output", `query name="rows" {echo("select 1");}\n cfsavecontent(variable="rows"){echo("text");}`, "rows.columnData(\"id\");"},
		{"other output body write", `query name="rows" {echo("select 1");}\n savecontent variable="text" {mutate(rows);}`, "rows.columnData(\"id\");"},
		{"later escape", `query name="rows" {echo("select 1");}\n replace(rows);`, "rows.columnData(\"id\");"},
		{"sibling closure", `var first=function(){query name="rows" {echo("select 1");}};\n var second=function(){`, "rows.columnData(\"id\");};"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 var rows="";
 ` + strings.ReplaceAll(tc.output, `\n`, "\n") + "\n" + tc.call + "\n }}"})

			got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
			if reason, ok := got["rows.columnData"]; !ok || reason == "" || strings.Contains(reason, "not found on builtin native.query") {
				t.Errorf("unknown output acquired query: %q present=%v", reason, ok)
			}
		})
	}
}

func TestQueryOutputInsideBranchAndResultMetadata(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 var rows="";
 var metadata="";
 try{
 query name="rows" result="metadata" {echo("select 1");}
 rows.columnData("id");metadata.keyArray();metadata.columnData("id");
 }catch(any e){}
 var untouched="a";
 query name="other" {echo("select 1");}
 untouched.length();
 }}`})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
	for _, name := range []string{"rows.columnData", "metadata.keyArray", "untouched.length"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}

	if !strings.Contains(got["metadata.columnData"], "native.struct") {
		t.Errorf("wrong metadata identity %q", got["metadata.columnData"])
	}
}

func TestQueryOutputLookupBeforeAssignmentWrite(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 var rows="";
 cfdbinfo(name="rows",type="tables");
 if(unknown){
 rows=rows.columnData("id");
 }
 }}`})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
	if reason, ok := got["rows.columnData"]; !ok || reason != "" {
		t.Fatalf("RHS query lookup: %q present=%v", reason, ok)
	}
}

func TestQueryOutputShadowedWriter(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {
 function cfdbinfo(){return "";}
 function run(){
 var rows="";
 cfdbinfo(name="rows",type="tables");
 rows.columnData("id");
 }
 }`})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
	if reason, ok := got["rows.columnData"]; !ok || reason == "" || strings.Contains(reason, "native.query") {
		t.Fatalf("shadowed writer inferred query: %q present=%v", reason, ok)
	}
}
