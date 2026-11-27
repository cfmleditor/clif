package lint

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/cfmleditor/clif/internal/cflint"
)

// FindingKey uniquely identifies a finding for parity comparison between two
// lint engines: normalized file path, 1-based line, and uppercased rule code.
type FindingKey struct {
	File string
	Line int
	Rule string
}

// Comparison summarizes the rule-by-rule parity between a native run and a
// CFLint baseline run.
type Comparison struct {
	Matched    int
	OnlyNative []cflint.RunIssue
	OnlyCFLint []cflint.RunIssue
}

// CompareRuns diffs native and cflintRun by (file, line, rule) counts so
// corpus parity checks can pinpoint false positives (OnlyNative) and false
// negatives (OnlyCFLint).
func CompareRuns(native, cflintRun *cflint.Run) Comparison {
	var cmp Comparison

	nativeCounts := make(map[FindingKey]int)
	cflintCounts := make(map[FindingKey]int)

	if native != nil {
		for i := range native.Issues {
			nativeCounts[keyOf(&native.Issues[i])]++
		}
	}

	if cflintRun != nil {
		for i := range cflintRun.Issues {
			cflintCounts[keyOf(&cflintRun.Issues[i])]++
		}
	}

	matched := make(map[FindingKey]int)

	for k, nc := range nativeCounts {
		if cc := cflintCounts[k]; cc > 0 {
			m := min(nc, cc)
			matched[k] = m
			cmp.Matched += m
		}
	}

	if native != nil {
		rem := make(map[FindingKey]int, len(matched))
		maps.Copy(rem, matched)

		for i := range native.Issues {
			k := keyOf(&native.Issues[i])
			if rem[k] > 0 {
				rem[k]--

				continue
			}

			cmp.OnlyNative = append(cmp.OnlyNative, native.Issues[i])
		}
	}

	if cflintRun != nil {
		rem := make(map[FindingKey]int, len(matched))
		maps.Copy(rem, matched)

		for i := range cflintRun.Issues {
			k := keyOf(&cflintRun.Issues[i])
			if rem[k] > 0 {
				rem[k]--

				continue
			}

			cmp.OnlyCFLint = append(cmp.OnlyCFLint, cflintRun.Issues[i])
		}
	}

	return cmp
}

func keyOf(iss *cflint.RunIssue) FindingKey {
	loc := firstLoc(iss.Issue)

	return FindingKey{
		File: filepath.ToSlash(filepath.Clean(loc.File)),
		Line: loc.Line,
		Rule: strings.ToUpper(strings.TrimSpace(iss.ID)),
	}
}

// FormatComparison formats a human-readable timing and per-rule parity summary
// comparing a native run against a CFLint run.
func FormatComparison(cmp *Comparison, nativeDur, cflintDur time.Duration, showSamples int) string {
	var b strings.Builder

	speedup := ""
	if nativeDur > 0 && cflintDur > 0 {
		speedup = fmt.Sprintf(" (%.1fx faster)", float64(cflintDur)/float64(nativeDur))
	}

	fmt.Fprintf(&b, "Timing: native %s vs cflint %s%s\n", nativeDur.Round(time.Millisecond), cflintDur.Round(time.Millisecond), speedup)
	fmt.Fprintf(&b, "Totals: %d matched, %d only-native, %d only-cflint\n\n", cmp.Matched, len(cmp.OnlyNative), len(cmp.OnlyCFLint))

	type counts struct{ onlyNative, onlyCFLint int }

	byRule := make(map[string]*counts)
	get := func(rule string) *counts {
		c := byRule[rule]
		if c == nil {
			c = &counts{}
			byRule[rule] = c
		}

		return c
	}

	for i := range cmp.OnlyNative {
		get(strings.ToUpper(cmp.OnlyNative[i].ID)).onlyNative++
	}

	for i := range cmp.OnlyCFLint {
		get(strings.ToUpper(cmp.OnlyCFLint[i].ID)).onlyCFLint++
	}

	if len(byRule) == 0 {
		b.WriteString("Exact parity: 0 differences across all rules.\n")

		return b.String()
	}

	rules := make([]string, 0, len(byRule))
	for r := range byRule {
		rules = append(rules, r)
	}

	slices.Sort(rules)

	fmt.Fprintf(&b, "%-36s %11s %11s\n", "RULE", "ONLY-NATIVE", "ONLY-CFLINT")

	for _, r := range rules {
		c := byRule[r]
		fmt.Fprintf(&b, "%-36s %11d %11d\n", r, c.onlyNative, c.onlyCFLint)
	}

	if showSamples > 0 && len(cmp.OnlyNative) > 0 {
		b.WriteString("\nSample only-native findings:\n")

		for i := range min(showSamples, len(cmp.OnlyNative)) {
			loc := firstLoc(cmp.OnlyNative[i].Issue)
			fmt.Fprintf(&b, "  %s:%d [%s] %s\n", loc.File, loc.Line, cmp.OnlyNative[i].ID, loc.Message)
		}
	}

	if showSamples > 0 && len(cmp.OnlyCFLint) > 0 {
		b.WriteString("\nSample only-cflint findings:\n")

		for i := range min(showSamples, len(cmp.OnlyCFLint)) {
			loc := firstLoc(cmp.OnlyCFLint[i].Issue)
			fmt.Fprintf(&b, "  %s:%d [%s] %s\n", loc.File, loc.Line, cmp.OnlyCFLint[i].ID, loc.Message)
		}
	}

	return b.String()
}
