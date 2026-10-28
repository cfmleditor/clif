package config

import (
	"path/filepath"
	"slices"
	"strings"
)

// The checks `clif check` can run, by the name the config and --checks use.
const (
	CheckCFLint     = "cflint"
	CheckUnresolved = "unresolved"
	CheckScan       = "scan"
	CheckFormat     = "format"
)

// KnownChecks lists every check `clif check` can run.
func KnownChecks() []string {
	return []string{CheckScan, CheckFormat, CheckCFLint, CheckUnresolved}
}

// DefaultChecks is what `clif check` runs when neither the config nor the
// command line names any: CFLint, the check a CI job most often wants.
func DefaultChecks() []string { return []string{CheckCFLint} }

// BaselineNone is the baseline that turns off the default baseline (the
// known-issues reports the config generates for that check).
const BaselineNone = "none"

// Check is the `check` block, what `clif check` runs and what fails it:
//
//	"check": {
//	  "checks": ["cflint", "unresolved", "format"],
//	  "cflint": {"minSeverity": "WARNING"},
//	  "unresolved": {"baseline": "unresolved.txt", "maxFindings": 10}
//	}
//
// Every key is optional. A child config's block merges into its parent's key by
// key (mergeCheck), as `linting` does, so a child naming only a threshold keeps
// the checks its parent chose.
type Check struct {
	// Checks names the checks to run, from KnownChecks. Absent runs
	// DefaultChecks; a present list replaces an inherited one whole, since
	// interleaving two lists describes neither project.
	Checks     []string   `json:"checks"`
	CFLint     *CheckRule `json:"cflint"`
	Unresolved *CheckRule `json:"unresolved"`
	Scan       *CheckRule `json:"scan"`
	Format     *CheckRule `json:"format"`
}

// CheckRule is one check's threshold.
type CheckRule struct {
	// MaxFindings is how many findings the check may have and still pass. A
	// pointer so that a child config can state 0 over a parent's allowance.
	// Absent is 0: any finding fails.
	MaxFindings *int `json:"maxFindings"`
	// Baseline is a known-issues file, relative to the config's directory,
	// whose entries are not counted: only findings it does not list fail the
	// check. Absent uses the reports the config's knownIssues entries generate
	// for this check ("generate": "cflint" or "unresolved"), which is what
	// clif cflint --write and clif unresolved --write keep current; "none"
	// uses no baseline. cflint and unresolved only.
	Baseline string `json:"baseline"`
	// MinSeverity is the least severe CFLint level counted, on CFLint's own
	// scale (FATAL ... COSMETIC). Absent takes linting.minSeverity. cflint only.
	MinSeverity string `json:"minSeverity"`
}

// ResolvedCheck is the check block with defaults applied.
type ResolvedCheck struct {
	Checks     []string
	CFLint     ResolvedCheckRule
	Unresolved ResolvedCheckRule
	Scan       ResolvedCheckRule
	Format     ResolvedCheckRule
}

// ResolvedCheckRule is one check's threshold with defaults applied.
type ResolvedCheckRule struct {
	MaxFindings int
	// Baseline is an absolute path, BaselineNone, or "" for the reports the
	// config generates for the check.
	Baseline    string
	MinSeverity string
}

// Rule returns the named check's threshold, and the zero rule for a name
// that is not a check.
func (c *ResolvedCheck) Rule(name string) ResolvedCheckRule {
	switch name {
	case CheckCFLint:
		return c.CFLint
	case CheckUnresolved:
		return c.Unresolved
	case CheckScan:
		return c.Scan
	case CheckFormat:
		return c.Format
	default:
		return ResolvedCheckRule{}
	}
}

// ResolveCheck applies the defaults to a check block, a nil one included.
// dir is the config's directory, which a baseline is relative to.
func ResolveCheck(c *Check, dir string) ResolvedCheck {
	if c == nil {
		c = &Check{}
	}

	r := ResolvedCheck{
		Checks:     DefaultChecks(),
		CFLint:     resolveCheckRule(c.CFLint, dir),
		Unresolved: resolveCheckRule(c.Unresolved, dir),
		Scan:       resolveCheckRule(c.Scan, dir),
		Format:     resolveCheckRule(c.Format, dir),
	}

	if c.Checks != nil {
		r.Checks = NormalizeChecks(c.Checks)
	}

	return r
}

func resolveCheckRule(rule *CheckRule, dir string) ResolvedCheckRule {
	if rule == nil {
		return ResolvedCheckRule{}
	}

	r := ResolvedCheckRule{
		MaxFindings: IntDefault(rule.MaxFindings, 0),
		MinSeverity: rule.MinSeverity,
		Baseline:    rule.Baseline,
	}

	if r.Baseline != "" && r.Baseline != BaselineNone && !filepath.IsAbs(r.Baseline) {
		r.Baseline = filepath.Join(dir, filepath.FromSlash(r.Baseline))
	}

	return r
}

// NormalizeChecks lowercases and trims check names and drops blanks and
// repeats, keeping the order given.
func NormalizeChecks(names []string) []string {
	out := make([]string, 0, len(names))

	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		if n != "" && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}

	return out
}

// UnknownChecks returns the names in names that are not checks.
func UnknownChecks(names []string) []string {
	var out []string

	for _, n := range names {
		if !slices.Contains(KnownChecks(), n) {
			out = append(out, n)
		}
	}

	return out
}

// mergeCheck unions two check blocks key by key, with over winning wherever
// it states a value, for the reason mergeLinting gives.
func mergeCheck(base, over *Check) *Check {
	if base == nil {
		return over
	}

	if over == nil {
		return base
	}

	out := *base

	if over.Checks != nil {
		out.Checks = over.Checks
	}

	out.CFLint = mergeCheckRule(base.CFLint, over.CFLint)
	out.Unresolved = mergeCheckRule(base.Unresolved, over.Unresolved)
	out.Scan = mergeCheckRule(base.Scan, over.Scan)
	out.Format = mergeCheckRule(base.Format, over.Format)

	return &out
}

func mergeCheckRule(base, over *CheckRule) *CheckRule {
	if base == nil {
		return over
	}

	if over == nil {
		return base
	}

	out := *base

	if over.MaxFindings != nil {
		out.MaxFindings = over.MaxFindings
	}

	if over.Baseline != "" {
		out.Baseline = over.Baseline
	}

	if over.MinSeverity != "" {
		out.MinSeverity = over.MinSeverity
	}

	return &out
}
