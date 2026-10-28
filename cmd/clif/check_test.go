package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/knownissues"
	"github.com/cfmleditor/clif/internal/unresolved"
)

// checkProject writes a project with one page and, when cfg is not empty, a
// .clif.json holding it, and returns its directory and the page's path.
func checkProject(t *testing.T, cfg, page string) (dir, file string) {
	t.Helper()

	dir = t.TempDir()
	file = filepath.Join(dir, "page.cfm")

	if err := os.WriteFile(file, []byte(page), 0o600); err != nil {
		t.Fatal(err)
	}

	if cfg != "" {
		if err := os.WriteFile(filepath.Join(dir, ".clif.json"), []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	return dir, file
}

// fakeCFLint writes a CFLint stand-in that prints a JSON report holding one
// issue in file per entry of rules, or none, and returns the environment that
// makes clif run it.
func fakeCFLint(t *testing.T, file string, rules ...string) []string {
	t.Helper()

	issues := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		issues = append(issues, map[string]any{
			"severity": "WARNING",
			"id":       r,
			"message":  r,
			"locations": []map[string]any{{
				"file": file, "line": 1, "column": 1, "message": r + " here",
			}},
		})
	}

	report, err := json.Marshal(map[string]any{"version": "fake", "issues": issues, "counts": map[string]int{"totalFiles": 1}})
	if err != nil {
		t.Fatal(err)
	}

	bin := t.TempDir()
	out := filepath.Join(bin, "report.json")

	if err := os.WriteFile(out, report, 0o600); err != nil {
		t.Fatal(err)
	}

	script := filepath.Join(bin, "cflint")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat '"+out+"'\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	return []string{"CLIF_CFLINT=" + script}
}

const cleanPage = "<cfset x = 1 />\n"

// TestCheckRunsCFLintByDefault: with no check block, CFLint is the check, a
// finding fails it with 1, and none passes.
func TestCheckRunsCFLintByDefault(t *testing.T) {
	dir, file := checkProject(t, `{"workspaceName": "w"}`, cleanPage)

	out, code := runClifEnv(t, dir, fakeCFLint(t, file, "MISSING_VAR"), "check", dir)
	if code != exitCheckFailed || !strings.Contains(out, "[cflint MISSING_VAR]") || !strings.Contains(out, "cflint      FAIL") {
		t.Errorf("a CFLint finding: exit %d, want %d naming it; output:\n%s", code, exitCheckFailed, out)
	}

	out, code = runClifEnv(t, dir, fakeCFLint(t, file), "check", dir)
	if code != 0 || !strings.Contains(out, "check passed") {
		t.Errorf("no CFLint findings: exit %d, want 0; output:\n%s", code, out)
	}
}

// TestCheckSelection: the config's checks list replaces the default, so CFLint
// (which would fail here, being unavailable) does not run, and --checks
// replaces the config's list.
func TestCheckSelection(t *testing.T) {
	dir, _ := checkProject(t, `{"workspaceName": "w", "check": {"checks": ["scan"]}}`, cleanPage)
	noCFLint := []string{"CLIF_CFLINT=" + filepath.Join(dir, "missing")}

	out, code := runClifEnv(t, dir, noCFLint, "check", dir)
	if code != 0 || !strings.Contains(out, "scan ") || strings.Contains(out, "cflint") {
		t.Errorf("checks [scan]: exit %d, want 0 with only scan run; output:\n%s", code, out)
	}

	out, code = runClifEnv(t, dir, noCFLint, "check", "--checks", "scan,cflint", dir)
	if code != exitCheckError || !strings.Contains(out, "cflint      ERROR") {
		t.Errorf("--checks scan,cflint with CFLint missing: exit %d, want %d; output:\n%s", code, exitCheckError, out)
	}

	if out, code := runClif(t, dir, "check", "--checks", "lint", dir); code != exitCheckError || !strings.Contains(out, "no check named lint") {
		t.Errorf("an unknown check: exit %d, want %d naming it; output:\n%s", code, exitCheckError, out)
	}
}

// TestCheckThresholds: scan fails on a parse error unless maxFindings allows
// it, and format fails on a file it would change.
func TestCheckThresholds(t *testing.T) {
	broken := "<cfif x>\n<cfset y = >\n"

	dir, _ := checkProject(t, `{"workspaceName": "w"}`, broken)
	if out, code := runClif(t, dir, "check", "--checks", "scan", dir); code != exitCheckFailed || !strings.Contains(out, "[scan cfml]") {
		t.Errorf("a parse error: exit %d, want %d; output:\n%s", code, exitCheckFailed, out)
	}

	dir, _ = checkProject(t, `{"workspaceName": "w", "check": {"scan": {"maxFindings": 100}}}`, broken)
	if out, code := runClif(t, dir, "check", "--checks", "scan", dir); code != 0 {
		t.Errorf("a parse error under maxFindings: exit %d, want 0; output:\n%s", code, out)
	}

	dir, _ = checkProject(t, `{"workspaceName": "w"}`, "<CFSET x = 1>\n")
	if out, code := runClif(t, dir, "check", "--checks", "format", dir); code != exitCheckFailed || !strings.Contains(out, "page.cfm:1: [format] would be reformatted") {
		t.Errorf("an unformatted file: exit %d, want %d; output:\n%s", code, exitCheckFailed, out)
	}

	dir, _ = checkProject(t, `{"workspaceName": "w"}`, cleanPage)
	if out, code := runClif(t, dir, "check", "--checks", "format", dir); code != 0 {
		t.Errorf("a formatted file: exit %d, want 0; output:\n%s", code, out)
	}
}

// TestCheckCFLintBaseline: a finding the generated CFLint report lists is not
// counted; --no-baseline and "baseline": "none" count it; a named baseline
// that does not exist is an error.
func TestCheckCFLintBaseline(t *testing.T) {
	cfg := `{"workspaceName": "w", "knownIssues": {"files": [{"file": "cflint.txt", "generate": "cflint"}]}}`
	dir, file := checkProject(t, cfg, cleanPage)
	env := fakeCFLint(t, file, "MISSING_VAR")

	if err := os.WriteFile(filepath.Join(dir, "cflint.txt"), []byte("page.cfm:1:1: [warning MISSING_VAR] MISSING_VAR here\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, code := runClifEnv(t, dir, env, "check", dir)
	if code != 0 || !strings.Contains(out, "1 more in the baseline") {
		t.Errorf("a baselined finding: exit %d, want 0; output:\n%s", code, out)
	}

	if out, code := runClifEnv(t, dir, env, "check", "--no-baseline", dir); code != exitCheckFailed {
		t.Errorf("--no-baseline: exit %d, want %d; output:\n%s", code, exitCheckFailed, out)
	}

	dir, file = checkProject(t, `{"workspaceName": "w", "check": {"cflint": {"baseline": "missing.txt"}}}`, cleanPage)
	if out, code := runClifEnv(t, dir, fakeCFLint(t, file), "check", dir); code != exitCheckError || !strings.Contains(out, "baseline") {
		t.Errorf("a missing baseline: exit %d, want %d; output:\n%s", code, exitCheckError, out)
	}
}

// TestCheckJSONOutput: --format json is one document saying whether the run
// passed, with each check's findings.
func TestCheckJSONOutput(t *testing.T) {
	dir, file := checkProject(t, `{"workspaceName": "w"}`, cleanPage)

	out, code := runClifEnv(t, dir, fakeCFLint(t, file, "A", "B"), "check", "-q", "--format", "json", dir)
	if code != exitCheckFailed {
		t.Fatalf("exit %d, want %d; output:\n%s", code, exitCheckFailed, out)
	}

	// The child's stderr is in the output too; the document is the last line.
	start := strings.Index(out, "{")
	if start < 0 {
		t.Fatalf("no JSON in output:\n%s", out)
	}

	var doc struct {
		Checks []checkResult `json:"checks"`
		Passed bool          `json:"passed"`
	}

	if err := json.Unmarshal([]byte(out[start:]), &doc); err != nil {
		t.Fatalf("%v; output:\n%s", err, out)
	}

	if doc.Passed || len(doc.Checks) != 1 || doc.Checks[0].Status != checkFail || len(doc.Checks[0].Findings) != 2 {
		t.Errorf("doc = %+v, want cflint failed with two findings", doc)
	}
}

func TestGithubAnnotationEscapes(t *testing.T) {
	got := githubAnnotation("cflint", &checkFinding{File: "a,b.cfm", Line: 3, Column: 2, Rule: "R", Message: "50% off\nnow"})

	want := "::error file=a%2Cb.cfm,line=3,col=2,title=clif cflint R::50%25 off%0Anow"
	if got != want {
		t.Errorf("githubAnnotation = %q, want %q", got, want)
	}
}

// TestSubtractUnresolvedMatchesByTextNotLine: a call the baseline lists is
// dropped wherever it now is, and each entry drops one call.
func TestSubtractUnresolvedMatchesByTextNotLine(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.cfc")

	calls := []unresolved.Call{
		{File: file, Line: 9, Variable: "svc", Function: "go", Reason: "no ref"},
		{File: file, Line: 20, Variable: "svc", Function: "go", Reason: "no ref"},
		{File: file, Line: 30, Function: "other", Reason: "not found"},
	}
	baseline := knownissues.Parse("a.cfc:2: svc.go (no ref)\n", dir)

	kept, matched := subtractUnresolved(calls, baseline)
	if matched != 1 || len(kept) != 2 || kept[1].Function != "other" {
		t.Errorf("kept %+v, matched %d; want one svc.go dropped", kept, matched)
	}
}
