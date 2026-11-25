package lint

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTierCScopeRules(t *testing.T) {
	// MISSING_VAR: flags unscoped write, ignores var-declared, local., arguments, and params.
	src := "component {\n\tfunction f(required string id) {\n\t\tvar ok = 1;\n\t\tlocal.alsoOk = 2;\n\t\tbad = 3;\n\t\tid = 'x';\n\t}\n}\n"
	if got := runNativeRule(t, "svc.cfc", "MISSING_VAR", src); len(got) != 1 {
		t.Errorf("MISSING_VAR: got %v, want 1 (bad)", got)
	}

	// ARG_VAR_CONFLICT: `var id` conflicts with parameter `id`.
	src = "component {\n\tfunction f(required string id) {\n\t\tvar id = 'override';\n\t\tvar other = 1;\n\t}\n}\n"
	if got := runNativeRule(t, "svc.cfc", "ARG_VAR_CONFLICT", src); len(got) != 1 {
		t.Errorf("ARG_VAR_CONFLICT: got %v, want 1", got)
	}

	// ARG_VAR_MIXED: `id` used both as `arguments.id` and bare `id`.
	src = "component {\n\tfunction f(required string id) {\n\t\tvar a = arguments.id;\n\t\tvar b = id;\n\t}\n}\n"
	if got := runNativeRule(t, "svc.cfc", "ARG_VAR_MIXED", src); len(got) != 1 {
		t.Errorf("ARG_VAR_MIXED: got %v, want 1", got)
	}

	// IMPLICIT_SCOPE: bare read of `mystery` inside function.
	src = "component {\n\tfunction f(required string id) {\n\t\tvar x = 1;\n\t\treturn x + id + local.y + mystery;\n\t}\n}\n"
	if got := runNativeRule(t, "svc.cfc", "IMPLICIT_SCOPE", src); len(got) != 1 {
		t.Errorf("IMPLICIT_SCOPE: got %v, want 1 (mystery)", got)
	}
}

func TestTierDQueryParamRulesWithProductionParameters(t *testing.T) {
	dir := t.TempDir()
	rc := `{
		"inheritParent": false,
		"includes": [
			{"code": "CFQUERYPARAM_REQ"},
			{"code": "QUERYPARAM_REQ"}
		],
		"parameters": {
			"QueryParamChecker.allowVariableExpression": "(tablePrefix|^i$|qry[a-zA-Z0-9]*\\..*$)",
			"QueryParamChecker.allowLineExpression": "<cfloop[^>]*#${variable}#[^>]*>|#_obj[iI]nit[^\\(]*\\([^\\)]*${variable}[^\\)]*\\)?#"
		}
	}`

	if err := os.WriteFile(filepath.Join(dir, ".cflintrc"), []byte(rc), 0o600); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(dir, "Repo.cfc")
	src := strings.Join([]string{
		`<cfcomponent>`,
		`<cfquery name="q">`,
		`  SELECT id FROM #tablePrefix#users`,
		`  WHERE id = <cfqueryparam value="#arguments.id#" cfsqltype="cf_sql_integer">`,
		`  AND status = '#arguments.badStatus#'`,
		`</cfquery>`,
		`<cfscript>`,
		`  queryExecute("SELECT id FROM users WHERE name = '#arguments.rawName#'");`,
		`  queryExecute("SELECT id FROM #tablePrefix#users WHERE id = :id", {id: 1});`,
		`</cfscript>`,
		`</cfcomponent>`,
	}, "\n")

	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := NewNativeRunner(DefaultRegistry, "", "test")

	run, err := runner.Lint(context.Background(), []string{file})
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}

	if len(run.Issues) != 2 {
		t.Fatalf("got %d issues (%+v), want 2 (1 CFQUERYPARAM_REQ + 1 QUERYPARAM_REQ)", len(run.Issues), run.Issues)
	}

	if run.Issues[0].ID != "CFQUERYPARAM_REQ" || run.Issues[1].ID != "QUERYPARAM_REQ" {
		t.Errorf("issue IDs = [%s, %s], want [CFQUERYPARAM_REQ, QUERYPARAM_REQ]", run.Issues[0].ID, run.Issues[1].ID)
	}
}

func TestProductionCFLintRC100PercentNativeNoFallback(t *testing.T) {
	dir := t.TempDir()
	prodRC := `{
		"rule": [],
		"excludes": [],
		"includes": [
			{"code": "MISSING_VAR"}, {"code": "OUTPUT_ATTR"}, {"code": "AVOID_USING_CFDUMP_TAG"},
			{"code": "AVOID_USING_CFABORT_TAG"}, {"code": "AVOID_USING_DEBUG_ATTR"}, {"code": "AVOID_USING_WRITEDUMP"},
			{"code": "NESTED_CFOUTPUT"}, {"code": "ARG_VAR_MIXED"}, {"code": "ARG_VAR_CONFLICT"},
			{"code": "GLOBAL_VAR"}, {"code": "AVOID_USING_CFEXECUTE_TAG"}, {"code": "AVOID_USING_ABORT"},
			{"code": "AVOID_USING_CFINSERT_TAG"}, {"code": "AVOID_USING_CFUPDATE_TAG"}, {"code": "AVOID_USING_CFFORM_TAG"},
			{"code": "AVOID_USING_CFINPUT_TAG"}, {"code": "AVOID_USING_CFINVOKE_TAG"}, {"code": "AVOID_USING_CFFILE_TAG"},
			{"code": "AVOID_USING_CFDIRECTORY_TAG"}, {"code": "AVOID_USING_CFCOOKIE_TAG"}, {"code": "AVOID_USING_CFHTTP_TAG"},
			{"code": "ARG_TYPE_MISSING"}, {"code": "AVOID_USING_ISDEBUGMODE"}, {"code": "FILE_SHOULD_START_WITH_LOWERCASE"},
			{"code": "AVOID_USING_CREATEOBJECT"}, {"code": "COMPARE_INSTEAD_OF_ASSIGN"}, {"code": "AVOID_EMPTY_FILES"},
			{"code": "NO_DEFAULT_INSIDE_SWITCH"}, {"code": "PARSE_ERROR"}, {"code": "AVOID_USING_CFMODULE_TAG"},
			{"code": "AVOID_USING_CFSETTING_DEBUG"}, {"code": "AVOID_USING_CFINCLUDE_TAG"}, {"code": "AVOID_USING_EVALUATE"},
			{"code": "AVOID_USING_ISDEFINED"}, {"code": "CFQUERYPARAM_REQ"}, {"code": "QUERYPARAM_REQ"},
			{"code": "NEVER_USE_QUERY_IN_CFM"}, {"code": "SQL_SELECT_STAR"}, {"code": "ADDTOKEN_ATTR"},
			{"code": "IMPLICIT_SCOPE"}, {"code": "FUNCTION_ACCESS_REMOTE"}
		],
		"inheritParent": false
	}`

	if err := os.WriteFile(filepath.Join(dir, ".cflintrc"), []byte(prodRC), 0o600); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(dir, "Clean.cfc")
	if err := os.WriteFile(file, []byte("component output=\"false\" {\n\tpublic void function ok(required string a) output=\"false\" {\n\t\tvar x = arguments.a;\n\t}\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	fb := &fakeFallbackEngine{}
	hybrid := NewHybridRunner(NewNativeRunner(DefaultRegistry, "", "test"), fb)

	run, err := hybrid.Lint(context.Background(), []string{file})
	if err != nil {
		t.Fatalf("hybrid Lint: %v", err)
	}

	if fb.called {
		t.Fatal("expected HybridRunner to skip CFLint fallback completely for the 41-rule production .cflintrc")
	}

	if len(run.Issues) != 0 {
		t.Errorf("clean file produced %d issues: %+v", len(run.Issues), run.Issues)
	}
}

func BenchmarkNativeLintAll41Rules(b *testing.B) {
	dir := b.TempDir()
	rc := `{"inheritParent": false, "includes": [
		{"code": "MISSING_VAR"}, {"code": "OUTPUT_ATTR"}, {"code": "AVOID_USING_CFDUMP_TAG"},
		{"code": "AVOID_USING_CFABORT_TAG"}, {"code": "AVOID_USING_DEBUG_ATTR"}, {"code": "AVOID_USING_WRITEDUMP"},
		{"code": "NESTED_CFOUTPUT"}, {"code": "ARG_VAR_MIXED"}, {"code": "ARG_VAR_CONFLICT"},
		{"code": "GLOBAL_VAR"}, {"code": "AVOID_USING_CFEXECUTE_TAG"}, {"code": "AVOID_USING_ABORT"},
		{"code": "AVOID_USING_CFINSERT_TAG"}, {"code": "AVOID_USING_CFUPDATE_TAG"}, {"code": "AVOID_USING_CFFORM_TAG"},
		{"code": "AVOID_USING_CFINPUT_TAG"}, {"code": "AVOID_USING_CFINVOKE_TAG"}, {"code": "AVOID_USING_CFFILE_TAG"},
		{"code": "AVOID_USING_CFDIRECTORY_TAG"}, {"code": "AVOID_USING_CFCOOKIE_TAG"}, {"code": "AVOID_USING_CFHTTP_TAG"},
		{"code": "ARG_TYPE_MISSING"}, {"code": "AVOID_USING_ISDEBUGMODE"}, {"code": "FILE_SHOULD_START_WITH_LOWERCASE"},
		{"code": "AVOID_USING_CREATEOBJECT"}, {"code": "COMPARE_INSTEAD_OF_ASSIGN"}, {"code": "AVOID_EMPTY_FILES"},
		{"code": "NO_DEFAULT_INSIDE_SWITCH"}, {"code": "PARSE_ERROR"}, {"code": "AVOID_USING_CFMODULE_TAG"},
		{"code": "AVOID_USING_CFSETTING_DEBUG"}, {"code": "AVOID_USING_CFINCLUDE_TAG"}, {"code": "AVOID_USING_EVALUATE"},
		{"code": "AVOID_USING_ISDEFINED"}, {"code": "CFQUERYPARAM_REQ"}, {"code": "QUERYPARAM_REQ"},
		{"code": "NEVER_USE_QUERY_IN_CFM"}, {"code": "SQL_SELECT_STAR"}, {"code": "ADDTOKEN_ATTR"},
		{"code": "IMPLICIT_SCOPE"}, {"code": "FUNCTION_ACCESS_REMOTE"}
	]}`

	if err := os.WriteFile(filepath.Join(dir, ".cflintrc"), []byte(rc), 0o600); err != nil {
		b.Fatal(err)
	}

	file := filepath.Join(dir, "Service.cfc")
	src := strings.Repeat("component output=\"false\" {\n\tpublic string function greet(required string name) output=\"false\" {\n\t\tvar msg = \"Hello \" & arguments.name;\n\t\treturn msg;\n\t}\n}\n", 1)

	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		b.Fatal(err)
	}

	runner := NewNativeRunner(DefaultRegistry, "", "bench")
	files := []string{file}

	b.ResetTimer()

	for b.Loop() {
		if _, err := runner.Lint(context.Background(), files); err != nil {
			b.Fatal(err)
		}
	}
}
