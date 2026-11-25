package lint

import (
	"context"
	"encoding/json"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type cflintExpectedFixture struct {
	Issues []struct {
		ID        string `json:"id"`
		Locations []struct {
			Line int `json:"line"`
		} `json:"locations"`
	} `json:"issues"`
}

type fixtureStats struct {
	passed, total int
}

// TestCFLintFixtureReplay replays cfmleditor/cflint's rule fixture suite against
// NativeRunner when CFLINT_TESTS_DIR points at a local clone of cfmleditor/cflint's
// `src/test/resources/com/cflint/tests` directory. Skipped when unset.
//
// Usage:
//
//	CFLINT_TESTS_DIR=/path/to/cflint/src/test/resources/com/cflint/tests \
//	  go test -v -run TestCFLintFixtureReplay ./internal/lint/...
func TestCFLintFixtureReplay(t *testing.T) {
	root := strings.TrimSpace(os.Getenv("CFLINT_TESTS_DIR"))
	if root == "" {
		t.Skip("set CFLINT_TESTS_DIR to replay CFLint's own test fixtures")
	}

	byRule := map[string]*fixtureStats{}
	runner := NewNativeRunner(DefaultRegistry, "", "replay")

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".expected.txt") {
			return walkErr
		}

		replayOneFixture(t, runner, path, byRule)

		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%s): %v", root, err)
	}

	rules := make([]string, 0, len(byRule))
	for r := range byRule {
		rules = append(rules, r)
	}

	slices.Sort(rules)

	for _, r := range rules {
		st := byRule[r]
		t.Logf("%-34s %d/%d fixtures matched", r, st.passed, st.total)
	}
}

func replayOneFixture(t *testing.T, runner *NativeRunner, expectedPath string, byRule map[string]*fixtureStats) {
	t.Helper()

	srcFile := findFixtureSource(strings.TrimSuffix(expectedPath, ".expected.txt"))
	if srcFile == "" {
		return
	}

	expBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		return
	}

	var exp cflintExpectedFixture
	if json.Unmarshal(expBytes, &exp) != nil {
		return
	}

	expectedKeys, rulesInFixture, ok := extractFixtureExpectations(&exp, srcFile)
	if !ok || len(rulesInFixture) == 0 {
		return
	}

	run, err := runner.Lint(context.Background(), []string{srcFile})
	if err != nil {
		t.Logf("WARN %s: %v", srcFile, err)

		return
	}

	gotKeys := map[FindingKey]int{}

	for i := range run.Issues {
		k := keyOf(&run.Issues[i])
		if rulesInFixture[k.Rule] {
			gotKeys[k]++
		}
	}

	matched := maps.Equal(gotKeys, expectedKeys)

	for r := range rulesInFixture {
		st := byRule[r]
		if st == nil {
			st = &fixtureStats{}
			byRule[r] = st
		}

		st.total++
		if matched {
			st.passed++
		}
	}
}

func findFixtureSource(base string) string {
	for _, ext := range []string{".cfc", ".cfm"} {
		if _, err := os.Stat(base + ext); err == nil {
			return base + ext
		}
	}

	return ""
}

func extractFixtureExpectations(exp *cflintExpectedFixture, srcFile string) (map[FindingKey]int, map[string]bool, bool) {
	expectedKeys := map[FindingKey]int{}
	rulesInFixture := map[string]bool{}
	cleanFile := filepath.ToSlash(filepath.Clean(srcFile))

	for _, iss := range exp.Issues {
		code := strings.ToUpper(strings.TrimSpace(iss.ID))
		if code == "" {
			continue
		}

		if !DefaultRegistry.Has(code) {
			return nil, nil, false
		}

		rulesInFixture[code] = true

		for _, loc := range iss.Locations {
			expectedKeys[FindingKey{File: cleanFile, Line: loc.Line, Rule: code}]++
		}
	}

	return expectedKeys, rulesInFixture, true
}
