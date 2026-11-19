//go:build passcount

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// A batch scan reads its files many times over: the parse, the index, the
// resolver's readers. Each consolidation that made a reader read the parse
// cut how many, and each new reader that walks a file its own way puts them
// back without anything failing, as a re-measure found after a run of
// changes had taken ContentBox from 11.7 to 18.2 passes. This holds the scan
// of the repo's fixtures to the passes and parses it takes now.
//
// It counts only in a build with the passcount tag (the counting compiles
// away elsewhere), and must run alone, since the resolver's caches outlive a
// test: make check runs it as
//
//	go test -tags passcount -run TestABatchScanReadsEachFileFewTimes ./cmd/clif/
//
// CLIF_PASSES_CORPUS (directories, separated by the list separator) reports
// the same figures for each directory given, and holds them to nothing.
func TestABatchScanReadsEachFileFewTimes(t *testing.T) {
	if !parser.CountsScans() {
		t.Fatal("built without the passcount tag")
	}

	for _, c := range []struct {
		dir                string
		addConfig          bool
		maxPasses, maxEach float64
	}{
		{"../../testdata", false, 4.9, 2.5},
		{"../../internal/frameworkapi/stubs", true, 2.5, 2.3},
	} {
		// Keep project discovery inside the copied fixture so files and
		// configuration elsewhere in the checkout cannot affect the budget.
		root := filepath.Join(t.TempDir(), filepath.Base(c.dir))
		if err := os.CopyFS(root, os.DirFS(c.dir)); err != nil {
			t.Fatal(err)
		}

		if c.addConfig {
			writeTestFile(t, filepath.Join(root, ".clif.json"), `{"workspacePaths":["."]}`)
		}

		passes, each := scanPasses(t, root, true)
		t.Logf("%s: %.2f passes, %.2f parses per file", c.dir, passes, each)

		if passes > c.maxPasses {
			t.Errorf("%s: a batch scan reads its text %.2f times, more than %.2f: a reader walks the files again", c.dir, passes, c.maxPasses)
		}

		if each > c.maxEach {
			t.Errorf("%s: a batch scan parses each file %.2f times, more than %.2f", c.dir, each, c.maxEach)
		}
	}

	for dir := range strings.SplitSeq(os.Getenv("CLIF_PASSES_CORPUS"), string(os.PathListSeparator)) {
		if dir != "" {
			passes, each := scanPasses(t, dir, false)
			t.Logf("%s: %.2f passes, %.2f parses per file", dir, passes, each)
		}
	}
}

// scanPasses is the tokens an unresolved scan of dir produced over those of
// one plain scan of its CFML files, and the parses it ran per file.
func scanPasses(t *testing.T, dir string, checkCoverage bool) (passes, parsesPerFile float64) {
	t.Helper()

	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}

	var plain, files int64

	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		if !cfpath.IsCFMLFile(path) {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		files++

		sc := parser.NewScanner(string(b))
		for tok := sc.Next(); tok.Kind != parser.TokEOF; tok = sc.Next() {
			plain++
		}

		plain++ // the EOF

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if files == 0 || plain == 0 {
		t.Fatalf("%s holds no CFML", dir)
	}

	parser.ResetScanCounts()
	report, _, _ := scanUnresolved([]string{abs}, &unresolvedFlags{})

	tokens, parses := parser.ScanCounts()

	if checkCoverage && report.Err != nil {
		t.Fatal(report.Err)
	}

	if checkCoverage && (int64(report.Indexed) != files || int64(report.Scanned) != files || report.Skipped != 0) {
		t.Fatalf("%s: indexed %d, scanned %d, skipped %d, want all %d fixture files", dir, report.Indexed, report.Scanned, report.Skipped, files)
	}

	return float64(tokens) / float64(plain), float64(parses) / float64(files)
}
