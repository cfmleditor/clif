package config

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

// TestResolveCheckDefaults: a config without the block runs CFLint alone and
// fails on any finding, using the reports the config generates as baselines.
func TestResolveCheckDefaults(t *testing.T) {
	dir := t.TempDir()

	for name, cfg := range map[string]*JSON{
		"no block":    {},
		"empty block": {Check: &Check{}},
	} {
		got := Resolve(cfg, dir).Check

		want := ResolvedCheck{Checks: DefaultChecks()}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: Resolve().Check = %+v, want %+v", name, got, want)
		}
	}

	if !slices.Equal(DefaultChecks(), []string{CheckCFLint}) {
		t.Errorf("DefaultChecks() = %v, want [cflint]", DefaultChecks())
	}
}

// TestResolveCheckReadsEveryKey: each key the JSON names reaches the resolved
// block, a baseline made absolute against the config's directory.
func TestResolveCheckReadsEveryKey(t *testing.T) {
	dir := t.TempDir()

	var cfg JSON

	src := []byte(`{"check": {
		"checks": [" Format", "unresolved", "format", ""],
		"cflint": {"minSeverity": "WARNING", "maxFindings": 3, "baseline": "none"},
		"unresolved": {"baseline": "reports/unresolved.txt", "maxFindings": 0},
		"scan": {"maxFindings": 1},
		"format": {"maxFindings": 2}
	}}`)
	if err := json.Unmarshal(src, &cfg); err != nil {
		t.Fatal(err)
	}

	got := Resolve(&cfg, dir).Check
	want := ResolvedCheck{
		Checks:     []string{CheckFormat, CheckUnresolved},
		CFLint:     ResolvedCheckRule{MinSeverity: "WARNING", MaxFindings: 3, Baseline: BaselineNone},
		Unresolved: ResolvedCheckRule{Baseline: filepath.Join(dir, "reports", "unresolved.txt")},
		Scan:       ResolvedCheckRule{MaxFindings: 1},
		Format:     ResolvedCheckRule{MaxFindings: 2},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Resolve().Check =\n%+v\nwant\n%+v", got, want)
	}

	for _, name := range KnownChecks() {
		if got.Rule(name) == (ResolvedCheckRule{}) {
			t.Errorf("Rule(%q) is the zero rule; Rule does not reach that check", name)
		}
	}
}

// TestCheckRuleFieldsMerge: a child naming one key keeps its parent's others,
// in every rule and for every key, and a stated zero overrides an allowance.
func TestCheckRuleFieldsMerge(t *testing.T) {
	three, zero := 3, 0
	base := &JSON{Check: &Check{
		Checks: []string{CheckScan},
		CFLint: &CheckRule{MinSeverity: "ERROR", MaxFindings: &three, Baseline: "a.txt"},
		Format: &CheckRule{MaxFindings: &three},
	}}
	over := &JSON{Check: &Check{
		CFLint: &CheckRule{MaxFindings: &zero},
		Scan:   &CheckRule{MaxFindings: &three},
	}}

	got := Merge(base, over).Check

	if !slices.Equal(got.Checks, []string{CheckScan}) {
		t.Errorf("checks = %v, want the parent's", got.Checks)
	}

	if got.CFLint.MinSeverity != "ERROR" || got.CFLint.Baseline != "a.txt" || *got.CFLint.MaxFindings != 0 {
		t.Errorf("cflint = %+v, want the parent's severity and baseline with the child's zero", got.CFLint)
	}

	if got.Format == nil || *got.Format.MaxFindings != 3 || got.Scan == nil || *got.Scan.MaxFindings != 3 {
		t.Errorf("format %+v / scan %+v: a rule only one side names was dropped", got.Format, got.Scan)
	}

	if got := Merge(base, &JSON{Check: &Check{Checks: []string{CheckFormat}}}).Check.Checks; !slices.Equal(got, []string{CheckFormat}) {
		t.Errorf("a child's checks list = %v, want it to replace the parent's", got)
	}

	// Every CheckRule field is merged: set each on over alone and see it win.
	rt := reflect.TypeFor[CheckRule]()
	for i := range rt.NumField() {
		rule := &CheckRule{}
		fv := reflect.ValueOf(rule).Elem().Field(i)

		switch fv.Kind() {
		case reflect.String:
			fv.SetString("x")
		case reflect.Pointer:
			fv.Set(reflect.ValueOf(&three))
		default:
			t.Fatalf("CheckRule.%s: a %s field this test does not set", rt.Field(i).Name, fv.Kind())
		}

		merged := mergeCheckRule(&CheckRule{}, rule)
		if !reflect.DeepEqual(merged, rule) {
			t.Errorf("mergeCheckRule drops CheckRule.%s", rt.Field(i).Name)
		}
	}
}

func TestUnknownChecks(t *testing.T) {
	if got := UnknownChecks([]string{"cflint", "lint", "format"}); !slices.Equal(got, []string{"lint"}) {
		t.Errorf("UnknownChecks = %v, want [lint]", got)
	}
}
