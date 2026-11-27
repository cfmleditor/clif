package lint

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/cflint"
	"go.lsp.dev/protocol"
)

func TestLoadRCInheritanceAndParameters(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	leaf := filepath.Join(sub, "leaf")

	if err := os.MkdirAll(leaf, 0o755); err != nil {
		t.Fatal(err)
	}

	// Root config enables MISSING_VAR and OUTPUT_ATTR and sets a parameter.
	rootRC := `{
		"includes": [{"code": "MISSING_VAR", "severity": "ERROR"}, {"code": "OUTPUT_ATTR"}],
		"parameters": {"QueryParamChecker.allowVariableExpression": "prefix"}
	}`
	// Sub config excludes OUTPUT_ATTR and overrides parameter; inherits from root.
	subRC := `{
		"excludes": [{"code": "OUTPUT_ATTR"}],
		"parameters": {"QueryParamChecker.allowVariableExpression": "override", "QueryParamChecker.allowLineExpression": "<cfloop[^>]*#${variable}#[^>]*>"}
	}`
	// Leaf config stops inheritance.
	leafRC := `{
		"inheritParent": false,
		"includes": [{"code": "AVOID_USING_CFDUMP_TAG"}],
		"excludeFiles": ["vendor/.*\\.cfc$"]
	}`

	for p, body := range map[string]string{
		filepath.Join(root, ".cflintrc"): rootRC,
		filepath.Join(sub, ".cflintrc"):  subRC,
		filepath.Join(leaf, ".cflintrc"): leafRC,
	} {
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	subCfg, err := LoadRC(filepath.Join(sub, "Test.cfc"))
	if err != nil {
		t.Fatalf("LoadRC(sub): %v", err)
	}

	if !subCfg.RuleEnabled("MISSING_VAR") || subCfg.RuleEnabled("OUTPUT_ATTR") || subCfg.RuleEnabled("GLOBAL_VAR") {
		t.Errorf("sub RuleEnabled mismatch: MISSING_VAR=%v OUTPUT_ATTR=%v GLOBAL_VAR=%v",
			subCfg.RuleEnabled("MISSING_VAR"), subCfg.RuleEnabled("OUTPUT_ATTR"), subCfg.RuleEnabled("GLOBAL_VAR"))
	}

	if got := subCfg.RuleSeverity("MISSING_VAR", "WARNING"); got != "ERROR" {
		t.Errorf("sub severity override = %q, want ERROR", got)
	}

	if got := subCfg.ParamString("QueryParamChecker.allowVariableExpression"); got != "override" {
		t.Errorf("param override = %q, want override", got)
	}

	leafCfg, err := LoadRC(filepath.Join(leaf, "Test.cfc"))
	if err != nil {
		t.Fatalf("LoadRC(leaf): %v", err)
	}

	if leafCfg.RuleEnabled("MISSING_VAR") || !leafCfg.RuleEnabled("AVOID_USING_CFDUMP_TAG") {
		t.Errorf("leaf inheritParent=false failed: MISSING_VAR=%v CFDUMP=%v",
			leafCfg.RuleEnabled("MISSING_VAR"), leafCfg.RuleEnabled("AVOID_USING_CFDUMP_TAG"))
	}

	if !leafCfg.FileExcluded(filepath.Join(leaf, "vendor", "Lib.cfc")) || leafCfg.FileExcluded(filepath.Join(leaf, "App.cfc")) {
		t.Error("leaf FileExcluded mismatch")
	}
}

func TestParseSuppressions(t *testing.T) {
	src := strings.Join([]string{
		"component {",
		"\t<!--- @CFLintIgnore MISSING_VAR,OUTPUT_ATTR --->",
		"\tx = 1;",
		"\ty = 2; // cflint ignore:GLOBAL_VAR,AVOID_USING_EVALUATE",
		"\t<!--- @CFLintIgnore A, B --->",
		"\tz = 3;",
		"}",
	}, "\n")

	sup := ParseSuppressions(src)

	if !sup.IsSuppressed(3, "MISSING_VAR") || !sup.IsSuppressed(3, "output_attr") {
		t.Error("expected @CFLintIgnore on line 2 to suppress MISSING_VAR and OUTPUT_ATTR on line 3")
	}

	if sup.IsSuppressed(3, "GLOBAL_VAR") {
		t.Error("GLOBAL_VAR should not be suppressed on line 3")
	}

	if !sup.IsSuppressed(4, "GLOBAL_VAR") || !sup.IsSuppressed(4, "AVOID_USING_EVALUATE") {
		t.Error("expected inline // cflint ignore on line 4 to suppress GLOBAL_VAR and AVOID_USING_EVALUATE")
	}

	// Space ends code list, matching CFLint's parser: `@CFLintIgnore A, B` suppresses A only.
	if !sup.IsSuppressed(6, "A") || sup.IsSuppressed(6, "B") {
		t.Errorf("space-delimited list: A=%v B=%v, want true/false", sup.IsSuppressed(6, "A"), sup.IsSuppressed(6, "B"))
	}
}

func TestNativeRunnerHonorsRCAndSuppressionsAndSeverity(t *testing.T) {
	dir := t.TempDir()
	rcBody := `{"inheritParent": false, "includes": [{"code": "TEST_WARN"}, {"code": "TEST_INFO"}]}`

	if err := os.WriteFile(filepath.Join(dir, ".cflintrc"), []byte(rcBody), 0o600); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(dir, "Sample.cfc")
	src := "component {\n\t// cflint ignore:TEST_WARN\n\tx = 1;\n\ty = 2; // cflint ignore:TEST_WARN\n\tz = 3;\n}\n"

	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	reg := NewRegistry()
	reg.Register(Rule{
		Code:            "TEST_WARN",
		DefaultSeverity: "WARNING",
		Check: func(_ *FileContext) []cflint.Issue {
			return []cflint.Issue{
				{Message: "suppressed", Locations: []cflint.Location{{Line: 4, Column: 2, Message: "suppressed"}}},
				{Message: "kept", Locations: []cflint.Location{{Line: 5, Column: 2, Message: "kept"}}},
			}
		},
	})
	reg.Register(Rule{
		Code:            "TEST_INFO",
		DefaultSeverity: "INFO",
		Check: func(_ *FileContext) []cflint.Issue {
			return []cflint.Issue{
				{Message: "low", Locations: []cflint.Location{{Line: 3, Column: 2, Message: "low"}}},
			}
		},
	})

	runner := NewNativeRunner(reg, "WARNING", "test")

	run, err := runner.Lint(context.Background(), []string{file})
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}

	if run.TotalFiles != 1 || run.TotalLines != 6 {
		t.Errorf("totals = %d files / %d lines, want 1 / 6", run.TotalFiles, run.TotalLines)
	}

	if len(run.Issues) != 1 || run.Issues[0].ID != "TEST_WARN" || run.Issues[0].Locations[0].Line != 5 {
		t.Fatalf("issues = %+v, want single TEST_WARN on line 5", run.Issues)
	}

	diags, err := runner.Scan(context.Background(), file)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(diags) != 1 || diags[0].Source != protocol.NewOptional("lint") || diags[0].Code != protocol.String("TEST_WARN") {
		t.Fatalf("diags = %+v, want single lint/TEST_WARN diagnostic", diags)
	}
}

type fakeFallbackEngine struct {
	called bool
	issues []cflint.RunIssue
}

func (f *fakeFallbackEngine) Lint(_ context.Context, _ []string) (*cflint.Run, error) {
	f.called = true

	return &cflint.Run{TotalFiles: 1, Issues: f.issues}, nil
}

func (f *fakeFallbackEngine) LintContent(_ context.Context, _ []cflint.Source) (*cflint.Run, error) {
	f.called = true

	return &cflint.Run{TotalFiles: 1, Issues: f.issues}, nil
}

func (f *fakeFallbackEngine) Scan(_ context.Context, _ string) ([]protocol.Diagnostic, error) {
	f.called = true

	return nil, nil
}

func (f *fakeFallbackEngine) ScanFiles(_ context.Context, _ []string) (map[string][]protocol.Diagnostic, error) {
	f.called = true

	return nil, nil
}

func TestHybridRunnerSkipsFallbackWhenAllRulesAreNative(t *testing.T) {
	dir := t.TempDir()
	rcBody := `{"inheritParent": false, "includes": [{"code": "ONLY_NATIVE"}]}`

	if err := os.WriteFile(filepath.Join(dir, ".cflintrc"), []byte(rcBody), 0o600); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(dir, "A.cfc")
	if err := os.WriteFile(file, []byte("component {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	reg := NewRegistry()
	reg.Register(Rule{
		Code:            "ONLY_NATIVE",
		DefaultSeverity: "WARNING",
		Check: func(_ *FileContext) []cflint.Issue {
			return []cflint.Issue{{Locations: []cflint.Location{{Line: 1, Column: 1, Message: "native"}}}}
		},
	})

	fb := &fakeFallbackEngine{}
	hybrid := NewHybridRunner(NewNativeRunner(reg, "", "test"), fb)

	run, err := hybrid.Lint(context.Background(), []string{file})
	if err != nil {
		t.Fatalf("hybrid Lint: %v", err)
	}

	if fb.called {
		t.Error("expected fallback not to be called when all included rules are native")
	}

	if len(run.Issues) != 1 || run.Issues[0].ID != "ONLY_NATIVE" {
		t.Errorf("issues = %+v, want 1 ONLY_NATIVE", run.Issues)
	}
}

func makeRunIssue(file string, line, col int, rule, sev, msg string) cflint.RunIssue {
	var ri cflint.RunIssue

	ri.ID = rule
	ri.Severity = sev
	ri.Locations = []cflint.Location{{File: file, Line: line, Column: col, Message: msg}}

	return ri
}

func TestHybridRunnerMergesAndDeduplicatesFallback(t *testing.T) {
	dir := t.TempDir()
	rcBody := `{"inheritParent": false, "includes": [{"code": "NATIVE_RULE"}, {"code": "LEGACY_RULE"}]}`

	if err := os.WriteFile(filepath.Join(dir, ".cflintrc"), []byte(rcBody), 0o600); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(dir, "A.cfc")
	if err := os.WriteFile(file, []byte("component {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	reg := NewRegistry()
	reg.Register(Rule{
		Code:            "NATIVE_RULE",
		DefaultSeverity: "WARNING",
		Check: func(_ *FileContext) []cflint.Issue {
			return []cflint.Issue{{Locations: []cflint.Location{{Line: 1, Column: 1, Message: "from native"}}}}
		},
	})

	fb := &fakeFallbackEngine{
		issues: []cflint.RunIssue{
			makeRunIssue(file, 1, 1, "NATIVE_RULE", "WARNING", "dup"),
			makeRunIssue(file, 1, 2, "LEGACY_RULE", "ERROR", "from legacy"),
		},
	}

	hybrid := NewHybridRunner(NewNativeRunner(reg, "", "test"), fb)

	run, err := hybrid.Lint(context.Background(), []string{file})
	if err != nil {
		t.Fatalf("hybrid Lint: %v", err)
	}

	if !fb.called {
		t.Error("expected fallback to be called for LEGACY_RULE")
	}

	if len(run.Issues) != 2 || run.Issues[0].ID != "NATIVE_RULE" || run.Issues[1].ID != "LEGACY_RULE" {
		t.Errorf("merged issues = %+v, want [NATIVE_RULE, LEGACY_RULE]", run.Issues)
	}
}

func TestCompareRuns(t *testing.T) {
	native := &cflint.Run{Issues: []cflint.RunIssue{
		makeRunIssue("a.cfc", 10, 1, "MISSING_VAR", "WARNING", ""),
		makeRunIssue("a.cfc", 20, 1, "OUTPUT_ATTR", "WARNING", ""),
	}}
	legacy := &cflint.Run{Issues: []cflint.RunIssue{
		makeRunIssue("a.cfc", 10, 1, "MISSING_VAR", "WARNING", ""),
		makeRunIssue("b.cfc", 5, 1, "GLOBAL_VAR", "WARNING", ""),
	}}

	cmp := CompareRuns(native, legacy)
	if cmp.Matched != 1 || len(cmp.OnlyNative) != 1 || len(cmp.OnlyCFLint) != 1 {
		t.Fatalf("CompareRuns = %+v, want Matched=1 OnlyNative=1 OnlyCFLint=1", cmp)
	}
}

func TestSuppressionsOnManyCommentsAreLinear(t *testing.T) {
	var b strings.Builder

	const n = 2000
	for range n {
		b.WriteString("<!---\n@CFLintIgnore MISSING_VAR\n--->\nx = 1;\n")
	}

	sup := ParseSuppressions(b.String())

	// First block: lines 1-3 comment, line 4 code. Last block: line n*4.
	if !sup.IsSuppressed(4, "MISSING_VAR") || !sup.IsSuppressed(n*4, "MISSING_VAR") {
		t.Fatalf("expected first (line 4) and last (line %d) lines to be suppressed", n*4)
	}
}
