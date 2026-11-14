package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// CLIF_TAGPROBLEMS_CORPUS=<dirs> reports what TagProblems finds in real
// code: pages that run should give none, so each finding is read by hand.
func TestTagProblemsCorpus(t *testing.T) {
	env := os.Getenv("CLIF_TAGPROBLEMS_CORPUS")
	if env == "" {
		t.Skip("CLIF_TAGPROBLEMS_CORPUS not set")
	}

	for dir := range strings.SplitSeq(env, string(os.PathListSeparator)) {
		files, byKind := 0, map[TagProblemKind]int{}

		var samples []string

		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			switch strings.ToLower(filepath.Ext(path)) {
			case ".cfc", ".cfm", ".cfml":
			default:
				return nil
			}

			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			files++

			content := string(b)
			for _, p := range TagProblems(content) {
				byKind[p.Kind]++

				line := strings.Count(content[:p.Start], "\n") + 1
				samples = append(samples, fmt.Sprintf("%s:%d %s %s", strings.TrimPrefix(path, dir), line, p.Kind, p.Message))
			}

			return nil
		})

		sort.Strings(samples)
		t.Logf("%s: %d files, %v", filepath.Base(dir), files, byKind)

		for i, s := range samples {
			if i < 14 {
				t.Logf("    %s", s)
			}
		}
	}
}
