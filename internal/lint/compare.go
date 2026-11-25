package lint

import (
	"maps"
	"path/filepath"
	"strings"

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
