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

type lineRule struct {
	line int
	rule string
}

// vendoredLineOverrides records fixtures where native lint reports the exact
// offending line while CFLint's expected.txt had an off-by-one or outer-tag line.
var vendoredLineOverrides = map[string][]lineRule{
	// Line 5 is `x.x == 2;` and line 9 is `x.x EQ 6;`; CFLint reported lines 6 and 10.
	"CompareInsteadOfAssign/Compare1.cfm": {
		{5, "COMPARE_INSTEAD_OF_ASSIGN"},
		{9, "COMPARE_INSTEAD_OF_ASSIGN"},
	},
	// Line 2 is `#writeDump(now(), 'HH:mm:ss')#`; CFLint reported line 1 (`<cfoutput>`).
	"WriteDumpChecker/output_hash_625.cfc": {
		{2, "AVOID_USING_WRITEDUMP"},
	},
	// Line 2 is `new tests.packagecase.packagecase_nokC();`; CFLint reported line 1 (`<cfscript>`).
	"PackageCase/packagecase_nokC.cfc": {
		{2, "PACKAGE_CASE_MISMATCH"},
	},
	// Line 11 / 7 is `cfhttp(..., result="local.httpResponse")`; CFLint reported line 1 (`component{`).
	"UnusedVariable/UnusedVar_593.cfc": {
		{11, "UNUSED_LOCAL_VARIABLE"},
	},
	"UnusedVariable/httpparam_593.cfc": {
		{7, "UNUSED_LOCAL_VARIABLE"},
	},
	// Line 4 is `component {`; CFLint reported line 1 (doc comment).
	"Parsing/columns/test.cfc": {
		{4, "COMPONENT_INVALID_NAME"},
		{10, "ARGUMENT_HAS_PREFIX_OR_POSTFIX"},
		{10, "ARGUMENT_IS_TEMPORARY"},
	},
	// Line 3 is `var FO = "Bar";`; native flags VAR_ALLCAPS_NAME at the declaration (line 3)
	// rather than the return statement (line 6).
	"Naming/Inconsistent_544.cfc": {
		{2, "METHOD_ALLCAPS_NAME"},
		{3, "VAR_TOO_SHORT"},
		{3, "VAR_ALLCAPS_NAME"},
		{4, "VAR_ALLCAPS_NAME"},
		{5, "SCOPE_ALLCAPS_NAME"},
	},
	"Naming/Inconsistent_545_cfml.cfm": {
		{2, "METHOD_ALLCAPS_NAME"},
		{3, "VAR_TOO_SHORT"},
		{3, "VAR_ALLCAPS_NAME"},
		{4, "VAR_ALLCAPS_NAME"},
		{5, "SCOPE_ALLCAPS_NAME"},
	},
}

// vendoredSkips records vendored fixtures intentionally not matched.
var vendoredSkips = map[string]string{
	// @CFLintIgnore placed above an HTML <div> wrapping <cfsavecontent><cfscript>;
	// native intentionally does not extend suppressions across arbitrary HTML containers.
	"Ignores/ignoreCFMLAny3_Nested.cfc": "suppression above HTML <div> wrapper",
	// Upstream .expected.txt has malformed JSON (`/"qry/"` instead of `\"qry\"`).
	"VarScoper/cfqueryparam_569.cfc": "malformed JSON in upstream expected.txt",
	// Upstream .expected.txt has malformed JSON (`/"abs/"` instead of `\"abs\"`).
	"FunctionCollision/function_abs_cfml.cfc": "malformed JSON in upstream expected.txt",
	// Multi-file fixture (companion `packagecase_nok.cfc.2` holds the `new` expression).
	"PackageCase/packagecase_nok.cfc": "multi-file .cfc.2 fixture",
	// Multi-file fixture (`<cfinclude template="Header.cfm">` accumulates literal counts across files).
	"Includes/Template.cfm": "multi-file <cfinclude> literal counter",
}

// TestCFLintVendoredFixtures asserts native lint parity against the full
// vendored cfmleditor/cflint fixture suite in testdata/cflint (BSD-3-Clause).
// Fixtures whose .cflintrc enables at least one native rule are strictly
// asserted; fixtures targeting not-yet-implemented CFLint rules are tracked
// and reported by rule code so coverage can be expanded incrementally.
func TestCFLintVendoredFixtures(t *testing.T) {
	root := filepath.Join("testdata", "cflint")
	runner := NewNativeRunner(DefaultRegistry, "", "vendored")
	checked := 0
	unimplemented := 0
	unimplementedByRule := map[string]int{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".expected.txt") {
			return walkErr
		}

		srcFile := findFixtureSource(strings.TrimSuffix(path, ".expected.txt"))
		if srcFile == "" {
			return nil
		}

		rel, err := filepath.Rel(root, srcFile)
		if err != nil {
			t.Fatalf("Rel(%s): %v", srcFile, err)
		}

		relSlash := filepath.ToSlash(rel)
		if reason, skip := vendoredSkips[relSlash]; skip {
			t.Logf("SKIP %s (%s)", relSlash, reason)

			return nil
		}

		rc, err := LoadRCForDir(filepath.Dir(srcFile))
		if err != nil {
			t.Fatalf("LoadRCForDir(%s): %v", srcFile, err)
		}

		activeNative := activeNativeRules(rc)
		if len(activeNative) == 0 {
			unimplemented++

			for code := range rc.Includes {
				unimplementedByRule[code]++
			}

			return nil
		}

		checked++

		assertOneVendoredFixture(t, runner, path, srcFile, relSlash, activeNative)

		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%s): %v", root, err)
	}

	if checked < 200 {
		t.Fatalf("expected >=200 asserted vendored CFLint fixtures, checked %d", checked)
	}

	if unimplemented > 0 {
		t.Errorf("expected 0 unimplemented vendored CFLint fixtures, got %d", unimplemented)
	}

	t.Logf("asserted=%d unimplemented=%d (across %d unimplemented rule codes)", checked, unimplemented, len(unimplementedByRule))

	codes := make([]string, 0, len(unimplementedByRule))
	for c := range unimplementedByRule {
		codes = append(codes, c)
	}

	slices.Sort(codes)

	for _, c := range codes {
		t.Logf("  UNIMPLEMENTED %-36s (%d fixtures)", c, unimplementedByRule[c])
	}
}

func assertOneVendoredFixture(t *testing.T, runner *NativeRunner, expectedPath, srcFile, relSlash string, activeNative map[string]bool) {
	t.Helper()

	expBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read %s: %v", expectedPath, err)
	}

	var exp cflintExpectedFixture
	if err := json.Unmarshal(expBytes, &exp); err != nil {
		t.Fatalf("parse %s: %v", expectedPath, err)
	}

	cleanFile := filepath.ToSlash(filepath.Clean(srcFile))

	var expectedKeys map[FindingKey]int
	if override, ok := vendoredLineOverrides[relSlash]; ok {
		expectedKeys = make(map[FindingKey]int, len(override))
		for _, lr := range override {
			expectedKeys[FindingKey{File: cleanFile, Line: lr.line, Rule: lr.rule}]++
		}
	} else {
		expectedKeys = extractFixtureExpectations(&exp, srcFile, activeNative)
	}

	run, err := runner.Lint(context.Background(), []string{srcFile})
	if err != nil {
		t.Fatalf("Lint(%s): %v", relSlash, err)
	}

	gotKeys := map[FindingKey]int{}

	for i := range run.Issues {
		k := keyOf(&run.Issues[i])
		if activeNative[k.Rule] {
			gotKeys[k]++
		}
	}

	if !maps.Equal(gotKeys, expectedKeys) {
		t.Errorf("%s mismatch:\n  expected=%v\n  got=%v", relSlash, expectedKeys, gotKeys)
	}
}

// TestCFLintFixtureReplay replays cfmleditor/cflint's full rule fixture suite
// against NativeRunner when CFLINT_TESTS_DIR points at a local clone of
// cfmleditor/cflint's `src/test/resources/com/cflint/tests` directory. Skipped when unset.
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

	rc, err := LoadRCForDir(filepath.Dir(srcFile))
	if err != nil {
		return
	}

	activeNative := activeNativeRules(rc)
	if len(activeNative) == 0 {
		return
	}

	expectedKeys := extractFixtureExpectations(&exp, srcFile, activeNative)

	run, err := runner.Lint(context.Background(), []string{srcFile})
	if err != nil {
		t.Logf("WARN %s: %v", srcFile, err)

		return
	}

	gotKeys := map[FindingKey]int{}

	for i := range run.Issues {
		k := keyOf(&run.Issues[i])
		if activeNative[k.Rule] {
			gotKeys[k]++
		}
	}

	matched := maps.Equal(gotKeys, expectedKeys)
	if !matched && os.Getenv("CFLINT_TESTS_VERBOSE") != "" {
		t.Logf("DIFF %s:\n  expected=%v\n  got=%v", srcFile, expectedKeys, gotKeys)
	}

	for r := range activeNative {
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

func activeNativeRules(rc *RCConfig) map[string]bool {
	active := map[string]bool{}

	for code := range rc.Includes {
		if DefaultRegistry.Has(code) {
			active[code] = true
		}
	}

	return active
}

func findFixtureSource(base string) string {
	for _, ext := range []string{".cfc", ".cfm"} {
		if _, err := os.Stat(base + ext); err == nil {
			return base + ext
		}
	}

	return ""
}

func extractFixtureExpectations(exp *cflintExpectedFixture, srcFile string, activeNative map[string]bool) map[FindingKey]int {
	expectedKeys := map[FindingKey]int{}
	cleanFile := filepath.ToSlash(filepath.Clean(srcFile))

	for _, iss := range exp.Issues {
		code := strings.ToUpper(strings.TrimSpace(iss.ID))
		if code == "" || !activeNative[code] {
			continue
		}

		for _, loc := range iss.Locations {
			expectedKeys[FindingKey{File: cleanFile, Line: loc.Line, Rule: code}]++
		}
	}

	return expectedKeys
}

// TestCFLintJavaPorts ports key unit test cases from cfmleditor/cflint's Java
// test suite (src/test/java/com/cflint/Test*.java) for rules that do not have
// file-based fixtures in testdata/cflint.
func TestCFLintJavaPorts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		file      string
		src       string
		rule      string
		wantLines []int
	}{
		{
			name:      "CreateObjectComponent",
			file:      "test.cfm",
			src:       "<cfscript>\nvar obj = createObject(\"component\",\"componentPath\");\n</cfscript>",
			rule:      "AVOID_USING_CREATEOBJECT",
			wantLines: []int{2},
		},
		{
			name:      "CreateObjectComponentMultiline",
			file:      "test.cfm",
			src:       "<cfscript>\nvar obj = createObject(\n\"component\",\n\"componentPath\");\n</cfscript>",
			rule:      "AVOID_USING_CREATEOBJECT",
			wantLines: []int{2},
		},
		{
			name: "CreateObjectJavaAllowed",
			file: "test.cfm",
			src:  "<cfscript>\nvar a = 23;\nvar obj = createObject(\"java\",\"javaPath\");\n</cfscript>",
			rule: "AVOID_USING_CREATEOBJECT",
		},
		{
			name: "SwitchDefaultGoodCFML",
			file: "test.cfm",
			src:  "<cfswitch expression=\"#Trim(Department)#\">\n<cfcase value=\"Sales\">s</cfcase>\n<cfdefaultcase>d</cfdefaultcase>\n</cfswitch>",
			rule: "NO_DEFAULT_INSIDE_SWITCH",
		},
		{
			name:      "SwitchDefaultBadCFML",
			file:      "test.cfm",
			src:       "<cfswitch expression=\"#Trim(Department)#\">\n<cfcase value=\"Sales\">s</cfcase>\n</cfswitch>",
			rule:      "NO_DEFAULT_INSIDE_SWITCH",
			wantLines: []int{1},
		},
		{
			name: "SwitchDefaultGoodCFScript",
			file: "test.cfc",
			src:  "component {\nfunction f() {\nswitch(x) {\ncase \"a\": break;\ndefault: break;\n}\n}\n}",
			rule: "NO_DEFAULT_INSIDE_SWITCH",
		},
		{
			name:      "ImplicitScopeCFM",
			file:      "test.cfm",
			src:       "<cfset URL.temp = \"\" />\n<cfif temp></cfif>",
			rule:      "IMPLICIT_SCOPE",
			wantLines: []int{2},
		},
		{
			name: "ImplicitScopePriorAssignNotFlagged",
			file: "test.cfm",
			src:  "<cfset mode = URL.mode />\n<cfif mode EQ 1></cfif>",
			rule: "IMPLICIT_SCOPE",
		},
		{
			name: "ImplicitScopeCfqueryNameNotFlagged",
			file: "test.cfm",
			src:  "<cfquery name=\"foo\">SELECT 1</cfquery>\n<cfif foo.recordCount GT 0></cfif>",
			rule: "IMPLICIT_SCOPE",
		},
		{
			name: "ImplicitScopeCfloopItemNotFlagged",
			file: "test.cfm",
			src:  "<cfloop collection=\"#StructNew()#\" item=\"bar\">\n<cfoutput>#bar#</cfoutput>\n</cfloop>",
			rule: "IMPLICIT_SCOPE",
		},
		{
			name: "ImplicitScopeScopeKeywordLiteralNotFlagged",
			file: "test.cfm",
			src:  "<cfif StructKeyExists(VARIABLES,\"username\")></cfif>",
			rule: "IMPLICIT_SCOPE",
		},
		{
			name: "ImplicitScopeCfoutputQueryColumnNotFlagged",
			file: "test.cfm",
			src:  "<cfquery name=\"q\">SELECT col1, col2 FROM t</cfquery>\n<cfoutput query=\"q\">#col1#</cfoutput>",
			rule: "IMPLICIT_SCOPE",
		},
		{
			name:      "AvoidUsingAbortScript",
			file:      "test.cfm",
			src:       "<cfscript>\nabort;\n</cfscript>",
			rule:      "AVOID_USING_ABORT",
			wantLines: []int{2},
		},
		{
			name:      "AvoidUsingCFExecuteTag",
			file:      "test.cfm",
			src:       "<cfexecute name=\"ls\" />",
			rule:      "AVOID_USING_CFEXECUTE_TAG",
			wantLines: []int{1},
		},
		{
			name:      "AvoidUsingCFDumpTag",
			file:      "test.cfm",
			src:       "<cfdump var=\"#x#\" />",
			rule:      "AVOID_USING_CFDUMP_TAG",
			wantLines: []int{1},
		},
		{
			name:      "AvoidUsingIsDefined",
			file:      "test.cfm",
			src:       "<cfif isDefined(\"url.x\")></cfif>",
			rule:      "AVOID_USING_ISDEFINED",
			wantLines: []int{1},
		},
		{
			name:      "AvoidUsingEvaluate",
			file:      "test.cfm",
			src:       "<cfset x = evaluate(\"1+1\") />",
			rule:      "AVOID_USING_EVALUATE",
			wantLines: []int{1},
		},
		{
			name:      "AvoidUsingCFUpdateTag",
			file:      "test.cfm",
			src:       "<cfupdate datasource=\"dsn\" tablename=\"t\" />",
			rule:      "AVOID_USING_CFUPDATE_TAG",
			wantLines: []int{1},
		},
		{
			name:      "AvoidUsingCFInsertTag",
			file:      "test.cfm",
			src:       "<cfinsert datasource=\"dsn\" tablename=\"t\" />",
			rule:      "AVOID_USING_CFINSERT_TAG",
			wantLines: []int{1},
		},
		{
			name:      "FileShouldStartWithLowercaseCFM",
			file:      "BadName.cfm",
			src:       "<cfoutput>hi</cfoutput>",
			rule:      "FILE_SHOULD_START_WITH_LOWERCASE",
			wantLines: []int{1},
		},
		{
			name: "FileShouldStartWithLowercaseCFCAllowed",
			file: "MyService.cfc",
			src:  "component {}",
			rule: "FILE_SHOULD_START_WITH_LOWERCASE",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			p := filepath.Join(dir, tc.file)

			if err := os.WriteFile(p, []byte(tc.src), 0o600); err != nil {
				t.Fatalf("write %s: %v", p, err)
			}

			rcPath := filepath.Join(dir, ".cflintrc")
			rcBody := `{"inheritParent":false,"includes":[{"code":"` + tc.rule + `"}]}`

			if err := os.WriteFile(rcPath, []byte(rcBody), 0o600); err != nil {
				t.Fatalf("write .cflintrc: %v", err)
			}

			runner := NewNativeRunner(DefaultRegistry, "", "java-port")

			run, err := runner.Lint(context.Background(), []string{p})
			if err != nil {
				t.Fatalf("Lint: %v", err)
			}

			var gotLines []int

			for i := range run.Issues {
				for _, loc := range run.Issues[i].Locations {
					gotLines = append(gotLines, loc.Line)
				}
			}

			if !slices.Equal(gotLines, tc.wantLines) {
				t.Errorf("%s (%s): got lines %v, want %v", tc.name, tc.rule, gotLines, tc.wantLines)
			}
		})
	}
}
