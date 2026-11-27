package lint

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runNativeRule(t *testing.T, filename, code, src string) []string {
	t.Helper()

	dir := t.TempDir()
	rc := `{"inheritParent": false, "includes": [{"code": "` + code + `"}]}`

	if err := os.WriteFile(filepath.Join(dir, ".cflintrc"), []byte(rc), 0o600); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(dir, filename)
	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := NewNativeRunner(DefaultRegistry, "", "test")

	run, err := runner.Lint(context.Background(), []string{file})
	if err != nil {
		t.Fatalf("Lint(%s): %v", code, err)
	}

	out := make([]string, 0, len(run.Issues))
	for i := range run.Issues {
		out = append(out, run.Issues[i].ID)
	}

	return out
}

func TestTierABannedTagsAndCalls(t *testing.T) {
	// Banned tag rule fires on open tag, ignores close tag and comments.
	got := runNativeRule(t, "page.cfm", "AVOID_USING_CFDUMP_TAG", "<!--- <cfdump var=#x#> --->\n<cfdump var=\"#a#\">\n")
	if len(got) != 1 || got[0] != "AVOID_USING_CFDUMP_TAG" {
		t.Errorf("CFDUMP: got %v, want 1", got)
	}

	// Banned script call fires on bare call, ignores member call and function decl.
	got = runNativeRule(t, "svc.cfc", "AVOID_USING_EVALUATE", "component {\nfunction evaluate() {}\nfunction run() { obj.evaluate(); evaluate('x'); }\n}\n")
	if len(got) != 1 || got[0] != "AVOID_USING_EVALUATE" {
		t.Errorf("EVALUATE: got %v, want 1", got)
	}

	// Script abort statement fires on statement, ignores member/call.
	got = runNativeRule(t, "svc.cfc", "AVOID_USING_ABORT", "component {\nfunction run() { x.abort(); abort; }\n}\n")
	if len(got) != 1 || got[0] != "AVOID_USING_ABORT" {
		t.Errorf("ABORT: got %v, want 1", got)
	}
}

func TestTierAAttributeAndFileRules(t *testing.T) {
	if got := runNativeRule(t, "page.cfm", "AVOID_USING_DEBUG_ATTR", `<cfquery name="q" debug="true">SELECT 1</cfquery><cfquery name="q2" debug="false">SELECT 1</cfquery>`); len(got) != 1 {
		t.Errorf("DEBUG_ATTR: got %v, want 1", got)
	}

	if got := runNativeRule(t, "page.cfm", "AVOID_USING_CFSETTING_DEBUG", `<cfsetting showdebugoutput="true"><cfsetting showdebugoutput="false">`); len(got) != 1 {
		t.Errorf("CFSETTING_DEBUG: got %v, want 1", got)
	}

	if got := runNativeRule(t, "page.cfm", "ADDTOKEN_ATTR", `<cflocation url="/a"><cflocation url="/b" addtoken="false">`); len(got) != 1 {
		t.Errorf("ADDTOKEN_ATTR: got %v, want 1", got)
	}

	if got := runNativeRule(t, "Upper.cfm", "FILE_SHOULD_START_WITH_LOWERCASE", `<cfset x = 1>`); len(got) != 1 {
		t.Errorf("FILE_SHOULD_START_WITH_LOWERCASE(Upper.cfm): got %v, want 1", got)
	}

	if got := runNativeRule(t, "lower.cfm", "FILE_SHOULD_START_WITH_LOWERCASE", `<cfset x = 1>`); len(got) != 0 {
		t.Errorf("FILE_SHOULD_START_WITH_LOWERCASE(lower.cfm): got %v, want 0", got)
	}

	if got := runNativeRule(t, "empty.cfm", "AVOID_EMPTY_FILES", "<!--- only a comment --->\n  \n"); len(got) != 1 {
		t.Errorf("AVOID_EMPTY_FILES: got %v, want 1", got)
	}

	if got := runNativeRule(t, "page.cfm", "NEVER_USE_QUERY_IN_CFM", `<cfquery name="q">SELECT 1</cfquery>`); len(got) != 1 {
		t.Errorf("NEVER_USE_QUERY_IN_CFM: got %v, want 1", got)
	}

	if got := runNativeRule(t, "svc.cfc", "NEVER_USE_QUERY_IN_CFM", `<cfquery name="q">SELECT 1</cfquery>`); len(got) != 0 {
		t.Errorf("NEVER_USE_QUERY_IN_CFM(.cfc): got %v, want 0", got)
	}
}

func TestTierBStructureRules(t *testing.T) {
	if got := runNativeRule(t, "bad.cfm", "PARSE_ERROR", `<cfset x = >`); len(got) == 0 {
		t.Error("PARSE_ERROR: expected error on broken CFML")
	}

	if got := runNativeRule(t, "svc.cfc", "OUTPUT_ATTR", "<cfcomponent>\n\t<cffunction name=\"a\"></cffunction>\n\t<cffunction name=\"b\" output=\"false\"></cffunction>\n</cfcomponent>\n"); len(got) != 1 {
		t.Errorf("OUTPUT_ATTR: got %v, want 1 (only function a)", got)
	}

	if got := runNativeRule(t, "page.cfm", "NESTED_CFOUTPUT", `<cfoutput query="q"><cfoutput>hi</cfoutput></cfoutput><cfoutput><cfoutput>ok</cfoutput></cfoutput><cfoutput query="q" group="x"><cfoutput>ok</cfoutput></cfoutput>`); len(got) != 1 {
		t.Errorf("NESTED_CFOUTPUT: got %v, want 1", got)
	}

	if got := runNativeRule(t, "svc.cfc", "ARG_TYPE_MISSING", "component {\n\tfunction f(required a, string b, c = 1) {}\n}\n"); len(got) != 2 {
		t.Errorf("ARG_TYPE_MISSING: got %v, want 2 (a and c)", got)
	}

	if got := runNativeRule(t, "svc.cfc", "FUNCTION_ACCESS_REMOTE", "component {\n\tremote void function api() {}\n\tpublic void function ok() {}\n}\n"); len(got) != 1 {
		t.Errorf("FUNCTION_ACCESS_REMOTE: got %v, want 1", got)
	}

	if got := runNativeRule(t, "svc.cfc", "NO_DEFAULT_INSIDE_SWITCH", "component {\n\tfunction f(x) {\n\t\tswitch (x) { case 1: break; }\n\t\tswitch (x) { case 1: break; default: break; }\n\t}\n}\n"); len(got) != 1 {
		t.Errorf("NO_DEFAULT_INSIDE_SWITCH: got %v, want 1", got)
	}

	if got := runNativeRule(t, "svc.cfc", "COMPARE_INSTEAD_OF_ASSIGN", "component {\n\tfunction f(x) { x == 1; x EQ 2; if (x == 1) {} var b = x == 1; }\n}\n"); len(got) != 2 {
		t.Errorf("COMPARE_INSTEAD_OF_ASSIGN: got %v, want 2", got)
	}

	if got := runNativeRule(t, "svc.cfc", "SQL_SELECT_STAR", `<cfcomponent><cfquery name="q">SELECT * FROM users</cfquery><cfquery name="q2">SELECT id FROM users</cfquery></cfcomponent>`); len(got) != 1 {
		t.Errorf("SQL_SELECT_STAR: got %v, want 1", got)
	}

	if got := runNativeRule(t, "svc.cfc", "GLOBAL_VAR", "component {\n\tfunction f() { return session.userId & local.name; }\n}\n"); len(got) != 1 {
		t.Errorf("GLOBAL_VAR: got %v, want 1", got)
	}
}

func TestPhase2RegisteredCount(t *testing.T) {
	codes := DefaultRegistry.Codes()
	if len(codes) < 35 {
		t.Errorf("DefaultRegistry has %d rules (%s), want at least 35 for Phase 2", len(codes), strings.Join(codes, ", "))
	}
}
