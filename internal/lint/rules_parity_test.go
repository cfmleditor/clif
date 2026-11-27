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

	// Dynamic string constructor `new "#cls#"(argumentCollection = arguments)`
	// should treat named args as call arguments, not MISSING_VAR.
	dynNewSrc := "component {\n\tfunction build(required string cls) {\n\t\treturn new \"#arguments.cls#\"(argumentCollection = arguments);\n\t}\n}\n"
	if got := runNativeRule(t, "factory.cfc", "MISSING_VAR", dynNewSrc); len(got) != 0 {
		t.Errorf("MISSING_VAR on dynamic new string constructor = %v, want 0", got)
	}

	// Struct initialized on `arguments.<name> = {}` should not trigger MISSING_VAR
	// when populated via `memento[k] = v` inside the function.
	argAssignSrc := "component {\n\tfunction populate(required array items) {\n\t\targuments.memento = {};\n\t\targuments.items.each(function(item) {\n\t\t\tmemento[item] = true;\n\t\t});\n\t\treturn arguments.memento;\n\t}\n}\n"
	if got := runNativeRule(t, "pop.cfc", "MISSING_VAR", argAssignSrc); len(got) != 0 {
		t.Errorf("MISSING_VAR on arguments-assigned struct = %v, want 0", got)
	}

	// Assigning to an ARGUMENTS member inside a .cfm <cfscript> function must not panic.
	cfmArgAssign := "<cfscript>\nfunction f(any a) {\n\tARGUMENTS.a = 1;\n}\n</cfscript>\n"
	if got := runNativeRule(t, "page.cfm", "IMPLICIT_SCOPE", cfmArgAssign); len(got) != 0 {
		t.Errorf("IMPLICIT_SCOPE on .cfm ARGUMENTS.a = 1 = %v, want 0", got)
	}

	// HTML `#` characters outside <cfoutput> and CF tags must not trigger IMPLICIT_SCOPE,
	// while `#expr#` inside <cfoutput> or CF tag attributes still does.
	cfmHTMLHash := `<a href="#">hidden btn</a><div style="color:#fff">text</div><cfoutput>#realVar#</cfoutput><cfloop collection="#cfgVar#" item="k"></cfloop>`
	if got := runNativeRule(t, "view.cfm", "IMPLICIT_SCOPE", cfmHTMLHash); len(got) != 2 {
		t.Errorf("IMPLICIT_SCOPE on .cfm with HTML # outside cfoutput = %v, want 2 (realVar, cfgVar)", got)
	}

	// Pure-HTML .cfm partials with no CF tags must not be misclassified as CFScript.
	cfmPureHTML := "<!--- comment --->\n<div class=\"modal-body\">\n\t<button type=\"button\">Cancel</button>\n</div>\n"
	if got := runNativeRule(t, "partial.cfm", "IMPLICIT_SCOPE", cfmPureHTML); len(got) != 0 {
		t.Errorf("IMPLICIT_SCOPE on pure-HTML .cfm partial = %v, want 0", got)
	}

	// GLOBAL_VAR should not flag component-owned `this`, while still flagging `request`.
	globalThisSrc := "component {\n\tthis.name = 'app';\n\tfunction f() {\n\t\treturn this.name & request.id;\n\t}\n}\n"
	if got := runNativeRule(t, "app.cfc", "GLOBAL_VAR", globalThisSrc); len(got) != 1 {
		t.Errorf("GLOBAL_VAR on this + request = %v, want 1 (request only)", got)
	}

	// `<!--- CFLINT-DISABLE RULE --->` suppresses the following tag.
	disableSrc := "<cfsavecontent variable=\"x\">\n\t<!--- CFLINT-DISABLE AVOID_USING_CFDUMP_TAG --->\n\t<cfdump var=\"#x#\" />\n</cfsavecontent>\n"
	if got := runNativeRule(t, "dbg.cfc", "AVOID_USING_CFDUMP_TAG", disableSrc); len(got) != 0 {
		t.Errorf("AVOID_USING_CFDUMP_TAG under CFLINT-DISABLE = %v, want 0", got)
	}

	// Query-of-Queries (`dbtype="query"`) should not trigger CFQUERYPARAM_REQ.
	qoqSrc := "<cfquery name=\"q\" dbtype=\"query\">SELECT id FROM src ORDER BY #arguments.col#</cfquery>\n"
	if got := runNativeRule(t, "q.cfc", "CFQUERYPARAM_REQ", qoqSrc); len(got) != 0 {
		t.Errorf("CFQUERYPARAM_REQ on dbtype=query = %v, want 0", got)
	}

	// Dotted assignment `VARIABLES.persist.fn = ...` is an assignment, not an ARG_VAR_MIXED read.
	dottedVarAssign := "<cffunction name=\"init\"><cfargument name=\"persist\" type=\"any\"><cfset VARIABLES.persist = ARGUMENTS.persist><cfset VARIABLES.persist.helper = VARIABLES.helper></cffunction>\n"
	if got := runNativeRule(t, "svc.cfc", "ARG_VAR_MIXED", dottedVarAssign); len(got) != 0 {
		t.Errorf("ARG_VAR_MIXED on VARIABLES.persist.helper = ... = %v, want 0", got)
	}

	// Inline `<!--- ... --->` comment inside a multi-line `<cfset fn(...)>` call
	// must not break paren tracking and trigger ARG_VAR_CONFLICT on named args.
	cfsetCommentCall := "<cffunction name=\"run\"><cfargument name=\"ctx\" type=\"any\"><cfset helper(\n\t<!--- disabled arg --->\n\tctx=ARGUMENTS.ctx)></cffunction>\n"
	if got := runNativeRule(t, "svc.cfc", "ARG_VAR_CONFLICT", cfsetCommentCall); len(got) != 0 {
		t.Errorf("ARG_VAR_CONFLICT on cfset call with inline comment = %v, want 0", got)
	}

	// Named UDF params in `.cfm <cfscript>` should not trigger IMPLICIT_SCOPE,
	// while custom-tag `returnVariable` still lands unscoped in VARIABLES.
	cfmUDF := "<cfscript>function sortIt(any a, any k){ return a; }</cfscript>\n"
	if got := runNativeRule(t, "page.cfm", "IMPLICIT_SCOPE", cfmUDF); len(got) != 0 {
		t.Errorf("IMPLICIT_SCOPE on .cfm UDF params = %v, want 0", got)
	}

	cfmCustomTag := "<ns:lock returnVariable=\"isLocked\"></ns:lock><cfif isLocked></cfif>\n"
	if got := runNativeRule(t, "page.cfm", "IMPLICIT_SCOPE", cfmCustomTag); len(got) != 1 {
		t.Errorf("IMPLICIT_SCOPE on custom-tag returnVariable = %v, want 1", got)
	}

	// Unscoped column names inside `<cfloop query="q">` and `<cfoutput query="q">`
	// resolve to the current row and must not trigger IMPLICIT_SCOPE.
	queryLoopSrc := "<cfset q = queryNew(\"a\",\"varchar\")>\n<cfloop query=\"q\"><cfoutput>#a#</cfoutput></cfloop>\n<cfoutput query=\"q\">#a#</cfoutput>\n"
	if got := runNativeRule(t, "q.cfm", "IMPLICIT_SCOPE", queryLoopSrc); len(got) != 0 {
		t.Errorf("IMPLICIT_SCOPE inside query loops = %v, want 0", got)
	}

	// A `<cfset` inside a regex string literal in `<cfif REFindNoCase("<cfset...", ...)>`
	// must not be misparsed as a `<cfset>` tag and trigger IMPLICIT_SCOPE.
	regexCFSet := "<cffunction name=\"t\">\n\t<cfset var file = {source: ''} />\n\t<cfif REFindNoCase(\"<cfset\\s+popupurl\\s*=[^>\r\n]*=##[A-Za-z_]+##\", file.source) GT 0></cfif>\n</cffunction>\n"
	if got := runNativeRule(t, "regex.cfc", "IMPLICIT_SCOPE", regexCFSet); len(got) != 0 {
		t.Errorf("IMPLICIT_SCOPE on <cfset inside <cfif> regex string = %v, want 0", got)
	}

	// Pseudo-constructor `<cfset _doc = ArrayNew(1) />` outside any function in a
	// `.cfc` establishes a VARIABLES field and should not trigger IMPLICIT_SCOPE.
	pseudoCtor := "<cfcomponent>\n\t<cfset _doc = ArrayNew(1) />\n\t<cffunction name=\"get\">\n\t\t<cfreturn _doc />\n\t</cffunction>\n</cfcomponent>\n"
	if got := runNativeRule(t, "doc.cfc", "IMPLICIT_SCOPE", pseudoCtor); len(got) != 0 {
		t.Errorf("IMPLICIT_SCOPE on pseudo-constructor field = %v, want 0", got)
	}

	// Reading a variable before its first assignment in source order (e.g. loop
	// carry-over) must still trigger IMPLICIT_SCOPE on the first read only.
	readBeforeWrite := "<cfscript>\nfor (i = 1; i LTE 3; i++) {\n\tif (i NEQ 1) {\n\t\ts[k] = StructNew();\n\t\ts[k].n = p;\n\t}\n\tk = i;\n\tp = 1;\n}\n</cfscript>\n"
	if got := runNativeRule(t, "loop.cfm", "IMPLICIT_SCOPE", readBeforeWrite); len(got) != 2 {
		t.Errorf("IMPLICIT_SCOPE on read-before-write = %v, want 2 (k at line 4, p at line 5)", got)
	}
}
