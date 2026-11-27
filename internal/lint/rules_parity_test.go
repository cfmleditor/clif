package lint

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/cflint"
)

func TestRuleParityFixes(t *testing.T) {
	t.Run("ImplicitScopeComponentVarsAndFunctionField", func(t *testing.T) {
		probe := `<cfcomponent output="false">
	<cffunction name="init" access="public" output="false" returntype="any">
		<cfset VARIABLES.persist = createObject("component","persist").init() />
		<cfreturn this />
	</cffunction>
	<cffunction name="go" access="public" output="false" returntype="any">
		<cfargument name="x" type="string" required="true" />
		<cfset var r = persist.getThing(x=ARGUMENTS.x) />
		<cfset var o = helper.valLeft(ARGUMENTS.x, 2) />
		<cfreturn r />
	</cffunction>
</cfcomponent>
`
		if got := runNativeRule(t, "svc.cfc", "IMPLICIT_SCOPE", probe); len(got) != 0 {
			t.Errorf("IMPLICIT_SCOPE on component vars probe = %v, want 0", got)
		}

		// Bare unscoped read still fires and populates Location.Function.
		dir := t.TempDir()
		rc := `{"inheritParent": false, "includes": [{"code": "IMPLICIT_SCOPE"}]}`

		if err := os.WriteFile(filepath.Join(dir, ".cflintrc"), []byte(rc), 0o600); err != nil {
			t.Fatal(err)
		}

		runner := NewNativeRunner(DefaultRegistry, "", "test")

		run, err := runner.LintContent(context.Background(), []cflint.Source{{
			Path:    filepath.Join(dir, "svc.cfc"),
			Content: []byte("component {\n\tfunction myFunc() {\n\t\treturn unknownVar;\n\t}\n}\n"),
		}})
		if err != nil {
			t.Fatal(err)
		}

		if len(run.Issues) != 1 || run.Issues[0].Locations[0].Function != "myFunc" {
			t.Errorf("IMPLICIT_SCOPE issues = %+v, want 1 with Function=myFunc", run.Issues)
		}
	})

	t.Run("NestedCFOutputRequiresOuterQuery", func(t *testing.T) {
		plain := `<cfoutput><p>#now()#</p><cfoutput>#now()#</cfoutput></cfoutput>`
		if got := runNativeRule(t, "page.cfm", "NESTED_CFOUTPUT", plain); len(got) != 0 {
			t.Errorf("plain nested cfoutput = %v, want 0", got)
		}

		withQuery := `<cfoutput query="q"><cfoutput>#q.a#</cfoutput></cfoutput>`
		if got := runNativeRule(t, "page.cfm", "NESTED_CFOUTPUT", withQuery); len(got) != 1 {
			t.Errorf("query nested cfoutput = %v, want 1", got)
		}
	})

	t.Run("SuppressionCoversMultilineTagAndCFQueryNotCFTry", func(t *testing.T) {
		src := `<cfcomponent>
	<cffunction name="f" output="false">
		<!--- @CFLintIgnore SQL_SELECT_STAR,CFQUERYPARAM_REQ --->
		<cfquery name="local.q" datasource="dsn">
			SELECT *
			FROM users
			WHERE id = #arguments.id#
		</cfquery>
		<!--- @CFLintIgnore AVOID_USING_ISDEFINED --->
		<cfif isDefined("a")
			AND isDefined("b")>
		</cfif>
		<!--- @CFLintIgnore AVOID_USING_CFEXECUTE_TAG --->
		<cftry>
			<cfexecute name="ls" />
		<cfcatch></cfcatch>
		</cftry>
	</cffunction>
</cfcomponent>
`
		sup := ParseSuppressions(src)

		if !sup.IsSuppressed(5, "SQL_SELECT_STAR") || !sup.IsSuppressed(7, "CFQUERYPARAM_REQ") {
			t.Error("expected @CFLintIgnore above <cfquery> to cover its whole SQL body")
		}

		if !sup.IsSuppressed(11, "AVOID_USING_ISDEFINED") {
			t.Error("expected @CFLintIgnore above multi-line <cfif> to cover all condition lines")
		}

		if sup.IsSuppressed(15, "AVOID_USING_CFEXECUTE_TAG") {
			t.Error("expected @CFLintIgnore above <cftry> NOT to extend into its body")
		}
	})

	t.Run("NestedQuotesInHashStringsNotVariables", func(t *testing.T) {
		src := `<cfcomponent output="false">
	<cffunction name="f" output="false">
		<cfset var i = 1 />
		<cfset var j = 1 />
		<cfset var tmpquote = "'" />
		<cfset var result = [] />
		<cfset ArrayAppend(result[i].arrLabels, "#result[i]["item#j#_desc"]#: #result[i]["item#j#_flg"]#") />
		<cfset var clean = REReplaceNoCase("abc", '#tmpquote#', '&quot;', 'ALL') />
	</cffunction>
</cfcomponent>
`
		if got := runNativeRule(t, "svc.cfc", "IMPLICIT_SCOPE", src); len(got) != 0 {
			t.Errorf("IMPLICIT_SCOPE on nested-quote hash strings = %v, want 0", got)
		}
	})

	t.Run("SkipCFSFiles", func(t *testing.T) {
		dir := t.TempDir()
		rc := `{"inheritParent": false, "includes": [{"code": "PARSE_ERROR"}]}`

		if err := os.WriteFile(filepath.Join(dir, ".cflintrc"), []byte(rc), 0o600); err != nil {
			t.Fatal(err)
		}

		runner := NewNativeRunner(DefaultRegistry, "", "test")

		run, err := runner.LintContent(context.Background(), []cflint.Source{{
			Path:    filepath.Join(dir, "index.cfs"),
			Content: []byte("binary search index {{{{"),
		}})
		if err != nil {
			t.Fatal(err)
		}

		if len(run.Issues) != 0 {
			t.Errorf("expected .cfs files to be skipped, got %+v", run.Issues)
		}
	})

	t.Run("SmallerParserFalsePositives", func(t *testing.T) {
		// Closing </cfsetting> should not trigger PARSE_ERROR.
		if got := runNativeRule(t, "remote.cfm", "PARSE_ERROR", `<cfsetting enablecfoutputonly="true"></cfsetting>`); len(got) != 0 {
			t.Errorf("PARSE_ERROR on </cfsetting> = %v, want 0", got)
		}

		// <cffunction inside a cfscript or <cfset> regex string literal should not trigger OUTPUT_ATTR.
		if got := runNativeRule(t, "lib.cfc", "OUTPUT_ATTR", "component {\n\tfunction f() {\n\t\tvar re = \"<cffunction[^>]+>\";\n\t}\n}\n"); len(got) != 0 {
			t.Errorf("OUTPUT_ATTR on <cffunction in script string = %v, want 0", got)
		}

		if got := runNativeRule(t, "lib.cfm", "OUTPUT_ATTR", `<cfset re = "<cffunction name=""init""[^>]*>">`); len(got) != 0 {
			t.Errorf("OUTPUT_ATTR on <cffunction in <cfset> string = %v, want 0", got)
		}

		// Updating a declared component VARIABLES field inside a method is not MISSING_VAR.
		compField := "<cfcomponent output=\"false\">\n\t<cfset VARIABLES.ready = false />\n\t<cffunction name=\"markReady\" output=\"false\">\n\t\t<cfset ready = true />\n\t</cffunction>\n</cfcomponent>\n"
		if got := runNativeRule(t, "svc.cfc", "MISSING_VAR", compField); len(got) != 0 {
			t.Errorf("MISSING_VAR on declared component field = %v, want 0", got)
		}

		// Parameter with trailing hint="..." metadata still extracts the real parameter name for ARG_VAR_MIXED.
		paramHint := "component {\n\tpublic void function load(required string beanPath hint=\"Folder path\") {\n\t\tlocal.x = local.beanPath;\n\t}\n}\n"
		if got := runNativeRule(t, "factory.cfc", "ARG_VAR_MIXED", paramHint); len(got) != 1 {
			t.Errorf("ARG_VAR_MIXED with param hint attribute = %v, want 1", got)
		}

		// Top-level .cfm template IMPLICIT_SCOPE flags unscoped reads not established in the file.
		cfmSrc := `<cfset localVal = 1><cfoutput>#localVal# #externalVal#</cfoutput>`
		if got := runNativeRule(t, "view.cfm", "IMPLICIT_SCOPE", cfmSrc); len(got) != 1 {
			t.Errorf("IMPLICIT_SCOPE in .cfm template = %v, want 1", got)
		}

		// Script-tag form of banned tags (e.g. `cfmodule(...)`) should also be flagged.
		if got := runNativeRule(t, "svc.cfc", "AVOID_USING_CFMODULE_TAG", "component {\n\tfunction f() {\n\t\tcfmodule(template='a.cfm');\n\t}\n}\n"); len(got) != 1 {
			t.Errorf("AVOID_USING_CFMODULE_TAG on script cfmodule() = %v, want 1", got)
		}

		// Untyped parameter literally named `required` (`required = true`) should trigger ARG_TYPE_MISSING.
		if got := runNativeRule(t, "svc.cfc", "ARG_TYPE_MISSING", "component {\n\tfunction f(required = true) {}\n}\n"); len(got) != 1 {
			t.Errorf("ARG_TYPE_MISSING on param named required = %v, want 1", got)
		}

		// Built-in scopes cfthread and file should not trigger IMPLICIT_SCOPE.
		if got := runNativeRule(t, "svc.cfc", "IMPLICIT_SCOPE", "component {\n\tfunction f() {\n\t\treturn structKeyExists(cfthread, 't1') && structKeyExists(file, 'name');\n\t}\n}\n"); len(got) != 0 {
			t.Errorf("IMPLICIT_SCOPE on cfthread/file scopes = %v, want 0", got)
		}

		// Numeric struct key `.1.title` should not trigger MISSING_VAR on `title`.
		numKey := "component {\n\tfunction f() {\n\t\tvar result = {};\n\t\tvar idx = 1;\n\t\tresult.items[idx].entry.1.title = 'x';\n\t}\n}\n"
		if got := runNativeRule(t, "svc.cfc", "MISSING_VAR", numKey); len(got) != 0 {
			t.Errorf("MISSING_VAR on numeric struct key = %v, want 0", got)
		}

		// Struct literal ternary `==` should not trigger COMPARE_INSTEAD_OF_ASSIGN.
		structTern := "component {\n\tfunction f(required struct s) {\n\t\targuments.s[\"checked\"] = {\n\t\t\t\"first\" : DateCompare(1, 2, \"s\") == 1 ? true : false\n\t\t};\n\t}\n}\n"
		if got := runNativeRule(t, "svc.cfc", "COMPARE_INSTEAD_OF_ASSIGN", structTern); len(got) != 0 {
			t.Errorf("COMPARE_INSTEAD_OF_ASSIGN on struct literal ternary = %v, want 0", got)
		}
	})
}

func TestRuleParityClosureAndTagFixes(t *testing.T) {
	t.Parallel()

	// UTF-8 BOM-only files should still trigger AVOID_EMPTY_FILES.
	if got := runNativeRule(t, "empty.cfm", "AVOID_EMPTY_FILES", "\uFEFF"); len(got) != 1 {
		t.Errorf("AVOID_EMPTY_FILES on UTF-8 BOM-only file = %v, want 1", got)
	}

	// CFScript tag-as-statement (`application action="update" mappings=...;`)
	// should not treat tag attributes as MISSING_VAR or ARG_VAR_CONFLICT.
	appTagSrc := "component {\n\tfunction addMappings(required struct mappings) {\n\t\tapplication action=\"update\" customTagPaths=\"/x\" mappings=arguments.mappings;\n\t}\n}\n"
	if got := runNativeRule(t, "helper.cfc", "MISSING_VAR", appTagSrc); len(got) != 0 {
		t.Errorf("MISSING_VAR on script application tag = %v, want 0", got)
	}

	if got := runNativeRule(t, "helper.cfc", "ARG_VAR_CONFLICT", appTagSrc); len(got) != 0 {
		t.Errorf("ARG_VAR_CONFLICT on script application tag = %v, want 0", got)
	}

	// Outer function arguments captured inside closures (anonymous function,
	// block arrow, or expression arrow) must be referenced unscoped in CFML
	// and should not trigger ARG_VAR_MIXED.
	closureCaptureSrc := "component {\n\tfunction run(required array items, string prefix = '', boolean sensitive = false) {\n\t\targuments.items.filter(function(k) {\n\t\t\treturn sensitive && len(prefix);\n\t\t});\n\t\targuments.items.each((k) => {\n\t\t\twriteOutput(prefix & k);\n\t\t});\n\t\treturn arguments.items.map(k => prefix & k);\n\t}\n}\n"
	if got := runNativeRule(t, "closure.cfc", "ARG_VAR_MIXED", closureCaptureSrc); len(got) != 0 {
		t.Errorf("ARG_VAR_MIXED on outer arg captured in closure = %v, want 0", got)
	}
}
