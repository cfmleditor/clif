package parser

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.lsp.dev/uri"
)

// These tests hold a parser change to "the same answers, faster" against a
// real corpus. They skip unless CFML_CORPUS names one (the variable make
// corpus uses), so CI and `make test` never run them. See
// .ai/docs/PERFORMANCE-CORPUS-FOLLOWUPS.md for how they are used.
//
//	CFML_CORPUS=/src/Lucee:/src/ContentBox CLIF_DUMP=/tmp/before.txt \
//	    go test -count=1 -run TestCorpusResultDump ./internal/parser/
//
// Run it on the commit before a change and on the commit after, then diff the
// two files. A file that differs is a file whose answers changed. Delete the
// output first: a run that fails to build leaves the last one behind, and
// comparing against a stale file reads as "identical".
//
// CLIF_DUMP_SNIPPETS=1 adds the CFML found in this package's own tests. Those
// differ between two commits whenever the tests do, so use it to compare a
// working-tree change with its parent, not two distant commits.

var corpusRawString = regexp.MustCompile("(?s)`[^`]*`")

// corpusFiles lists the CFML files under the corpus directories, in a fixed order.
func corpusFiles(tb testing.TB) []string {
	tb.Helper()

	spec := os.Getenv("CFML_CORPUS")
	if spec == "" {
		tb.Skip("CFML_CORPUS not set")
	}

	var files []string

	for dir := range strings.SplitSeq(spec, string(os.PathListSeparator)) {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				if name := d.Name(); strings.HasPrefix(name, ".") && name != "." && name != ".." || name == "node_modules" {
					return filepath.SkipDir
				}

				return nil
			}

			switch strings.ToLower(filepath.Ext(path)) {
			case ".cfc", ".cfm", ".cfml":
				files = append(files, path)
			default:
			}

			return nil
		})
		if err != nil {
			tb.Fatal(err)
		}
	}

	slices.Sort(files)

	return files
}

// dumpParse writes what a parse of src found that the collection and member
// passes decide: each function's return type, component and sources, its
// collection values, and every component ref.
func dumpParse(sb *strings.Builder, mode, name, src string, opts *ParseOptions) {
	pr := ParseWithOptions(uri.File("/t/"+filepath.Base(name)+".cfc"), src, opts)

	fmt.Fprintf(sb, "== %s %s\n", mode, name)

	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		fmt.Fprintf(sb, "F %s rt=%q rc=%q rs=%v cv=%v\n", f.Name, f.ReturnType, f.ReturnComponent, f.ReturnSources, f.CollectionValues)
	}

	fmt.Fprintf(sb, "refs %v\n", pr.ComponentRefs)
	fmt.Fprintf(sb, "calls %d insts %d\n", len(pr.AllCalls()), len(pr.Instantiations))

	keys := make([]string, 0, len(pr.funcRefsMap))
	for k := range pr.funcRefsMap {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	for _, k := range keys {
		fmt.Fprintf(sb, "fr %s %v\n", k, pr.funcRefsMap[k])
	}
}

// testSnippets is every raw string in this package's tests that looks like
// CFML: hundreds of small cases written for these passes.
func testSnippets(t *testing.T) []string {
	t.Helper()

	tests, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}

	for _, tf := range tests {
		src, err := os.ReadFile(tf)
		if err != nil {
			t.Fatal(err)
		}

		for _, m := range corpusRawString.FindAllString(string(src), -1) {
			if m = m[1 : len(m)-1]; len(m) > 40 && (strings.Contains(m, "function") || strings.Contains(m, "<cf")) {
				seen[m] = true
			}
		}
	}

	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}

	slices.Sort(out)

	return out
}

func TestCorpusResultDump(t *testing.T) {
	out := os.Getenv("CLIF_DUMP")
	if out == "" {
		t.Skip("CLIF_DUMP not set")
	}

	files := corpusFiles(t)

	var snippets []string
	if os.Getenv("CLIF_DUMP_SNIPPETS") == "1" {
		snippets = testSnippets(t)
	}

	modes := []struct {
		name   string
		lookup func(string, string) string
	}{
		{"nolookup", nil},
		{"lookup", func(string, string) string { return "" }},
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "files %d snippets %d\n", len(files), len(snippets))

	for _, mode := range modes {
		opts := func() *ParseOptions { return &ParseOptions{ExtractCalls: true, FuncLookup: mode.lookup} }

		for _, name := range files {
			if b, err := os.ReadFile(name); err == nil && len(b) <= 8<<20 {
				dumpParse(&sb, mode.name, name, string(b), opts())
			}
		}

		for _, snip := range snippets {
			// Named by content, so two commits whose snippet sets differ
			// still line up the snippets they share.
			h := fnv.New64a()
			_, _ = h.Write([]byte(snip))

			dumpParse(&sb, mode.name, "snippet-"+strconv.FormatUint(h.Sum64(), 16), snip, opts())
		}
	}

	if err := os.WriteFile(out, []byte(sb.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Logf("wrote %d bytes for %d files", sb.Len(), len(files))
}

// TestCorpusSlowestParses names the files that cost most to parse, each timed
// as the best of three. Profile those rather than a synthetic fixture.
//
//	CFML_CORPUS=/src/Lucee CLIF_SLOWEST=25 go test -count=1 -run TestCorpusSlowestParses -v ./internal/parser/
func TestCorpusSlowestParses(t *testing.T) {
	n, _ := strconv.Atoi(os.Getenv("CLIF_SLOWEST"))
	if n <= 0 {
		t.Skip("CLIF_SLOWEST not set")
	}

	type timed struct {
		name  string
		bytes int
		best  time.Duration
	}

	var all []timed

	for _, name := range corpusFiles(t) {
		b, err := os.ReadFile(name)
		if err != nil || len(b) > 8<<20 {
			continue
		}

		src := string(b)
		best := time.Duration(1 << 62)

		for range 3 {
			start := time.Now()

			ParseWithOptions(uri.File("/t/"+filepath.Base(name)), src, &ParseOptions{ExtractCalls: true})

			best = min(best, time.Since(start))
		}

		all = append(all, timed{name, len(src), best})
	}

	slices.SortFunc(all, func(a, b timed) int { return int(b.best - a.best) })

	for _, f := range all[:min(n, len(all))] {
		t.Logf("%9.2fms %8d bytes %6.1f MB/s  %s", float64(f.best)/1e6, f.bytes, float64(f.bytes)/f.best.Seconds()/1e6, f.name)
	}
}

// BenchmarkCorpusParse parses every corpus file once per iteration, as the
// editor and `unresolved` do, and BenchmarkCorpusStructure computes the fold
// spans of each. Alternate two builds of the test binary on these, not one
// build then the other.
func BenchmarkCorpusParse(b *testing.B) {
	sources := corpusSources(b)

	b.ReportAllocs()
	b.SetBytes(totalBytes(sources))

	for b.Loop() {
		for _, s := range sources {
			ParseWithOptions("file:///bench.cfc", s, &ParseOptions{ExtractCalls: true})
		}
	}
}

func BenchmarkCorpusStructure(b *testing.B) {
	sources := corpusSources(b)

	b.ReportAllocs()
	b.SetBytes(totalBytes(sources))

	for b.Loop() {
		for _, s := range sources {
			StructureSpans(s)
		}
	}
}

func corpusSources(b *testing.B) []string {
	b.Helper()

	var sources []string

	for _, name := range corpusFiles(b) {
		if data, err := os.ReadFile(name); err == nil && len(data) <= 8<<20 {
			sources = append(sources, string(data))
		}
	}

	return sources
}

func totalBytes(sources []string) int64 {
	var n int64

	for _, s := range sources {
		n += int64(len(s))
	}

	return n
}
