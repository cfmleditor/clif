package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoTestdata(t *testing.T) string {
	t.Helper()

	_, file, _, _ := runtime.Caller(0)

	return filepath.Join(filepath.Dir(file), "..", "..", "testdata")
}

func TestMCPFlagsChooseTheTools(t *testing.T) {
	plain := mcpServer(&mcpFlags{}, nil)
	if plain.Unresolved == nil || plain.FindRefs == nil || plain.Explain == nil {
		t.Error("a plain server lacks a task tool")
	}

	if plain.Lint != nil {
		t.Error("lint offered without --allow-lint")
	}

	if mcpServer(&mcpFlags{allowLint: true}, nil).Lint == nil {
		t.Error("--allow-lint did not offer lint")
	}

	mapOnly := mcpServer(&mcpFlags{mapOnly: true, db: "x"}, nil)
	if mapOnly.Unresolved != nil || mapOnly.FindRefs != nil || mapOnly.Explain != nil || mapOnly.Lint != nil {
		t.Error("--map-only offered a tool that reads source")
	}
}

func TestAMissingPathIsAnErrorNotAnEmptyAnswer(t *testing.T) {
	root := t.TempDir()

	if _, err := mcpUnresolved(root, []string{filepath.Join(root, "nope")}, false, 10); err == nil {
		t.Error("unresolved over a missing path answered; it would read as no findings")
	}

	if _, err := mcpFindRefs(root, "f", []string{filepath.Join(root, "nope")}, 10); err == nil {
		t.Error("refs over a missing path answered")
	}
}

// TestMCPToolsStayInsideTheRoot: a tool's paths come from whatever drives the
// assistant, and every tool accepted any absolute path, so a prompt could have
// it scan, lint or quote a file anywhere the user can read. A symlink in the
// workspace that leads out is refused the same way.
func TestMCPToolsStayInsideTheRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "ws")
	outside := filepath.Join(base, "outside")

	for _, d := range []string{root, outside} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	secret := filepath.Join(outside, "secret.cfm")
	if err := os.WriteFile(secret, []byte("<cfset x = f()>\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{outside, secret, filepath.Join(root, "link"), filepath.Join(root, "..", "outside")} {
		if _, err := mcpUnresolved(root, []string{p}, false, 10); err == nil || !strings.Contains(err.Error(), "outside the workspace root") {
			t.Errorf("unresolved over %s: %v, want a refusal", p, err)
		}

		if _, err := mcpFindRefs(root, "f", []string{p}, 10); err == nil {
			t.Errorf("refs over %s answered", p)
		}

		if _, err := newMCPLinter(root)([]string{p}, 10); err == nil {
			t.Errorf("lint over %s answered", p)
		}
	}

	if _, err := explainAt(root, nil, explainConfig{}, secret, 1, ""); err == nil || !strings.Contains(err.Error(), "outside the workspace root") {
		t.Errorf("explain_call on a file outside the root: %v, want a refusal", err)
	}

	if _, err := mcpUnresolved(root, []string{root}, false, 10); err != nil {
		t.Errorf("the root itself was refused: %v", err)
	}
}

// TestMCPLinesAreOneBased: the parser numbers lines from zero, and explain_call,
// editors and the CLI's text from one. Each line reported must hold its call.
func TestMCPLinesAreOneBased(t *testing.T) {
	td := repoTestdata(t)

	res, err := mcpUnresolved(td, []string{td}, false, 1000)
	if err != nil {
		t.Fatal(err)
	}

	m, _ := res.(map[string]any)

	calls, _ := m["calls"].([]unresolvedCall)
	if len(calls) == 0 {
		t.Fatal("no unresolved calls in testdata; the check below would pass vacuously")
	}

	// Every one, not any: a neighbouring line can hold the same name.
	for _, c := range calls {
		if c.Unchecked > 0 {
			continue // the entry for a broken extends chain sits on the extends line
		}

		name := c.Call[strings.LastIndex(c.Call, ".")+1:] + "("
		if !strings.Contains(lineOf(t, c.File, int(c.Line)), name) {
			t.Errorf("%s:%d does not hold %s; lines are off by one", c.File, c.Line, name)
		}
	}

	refsRes, err := mcpFindRefs(td, "GetData", []string{filepath.Join(td, "refs")}, 10)
	if err != nil {
		t.Fatal(err)
	}

	rm, _ := refsRes.(map[string]any)
	refs, _ := rm["references"].([]reference)

	if len(refs) == 0 {
		t.Fatal("no references to GetData in testdata/refs")
	}

	// The call text is the whole line, and the line above GetData's call is
	// GetData's declaration, so a name check would pass off by one.
	for _, r := range refs {
		if got := strings.TrimSpace(lineOf(t, r.File, int(r.Line))); got != strings.TrimSpace(r.Call) {
			t.Errorf("%s:%d is %q, want the reference %q; lines are off by one", r.File, r.Line, got, r.Call)
		}
	}
}

func lineOf(t *testing.T, file string, line int) string {
	t.Helper()

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(string(data), "\n")
	if line < 1 || line > len(lines) {
		return ""
	}

	return lines[line-1]
}
