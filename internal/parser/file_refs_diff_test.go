package parser

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"go.lsp.dev/uri"
)

// TestFileRefsAreTheOldReaders compares what the parse's record answers
// (file_refs.go) with the old walks' answers (file_refs_oracle_test.go), field
// by field, over every file under CFML_CORPUS: the text entry points, and a
// parse's own answers, its includes from a default parse and its links and
// custom tags from one that records them (ExtractLinks), which its walk read.
// Each difference is counted by what it is and a few are shown.
func TestFileRefsAreTheOldReaders(t *testing.T) {
	if os.Getenv("CLIF_FILE_REFS_DIFF") == "" {
		t.Skip("set CLIF_FILE_REFS_DIFF=1 and CFML_CORPUS")
	}

	files := corpusFiles(t)
	d := &refsDiff{counts: map[string]int{}, held: map[string]int{}, shown: map[string][]string{}}

	work := make(chan string)

	var wg sync.WaitGroup

	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for f := range work {
				d.file(f)
			}
		})
	}

	for _, f := range files {
		work <- f
	}

	close(work)
	wg.Wait()

	keys := make([]string, 0, len(d.counts))
	for k := range d.counts {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	t.Logf("%d files, %d answers compared", len(files), d.answers)

	held := make([]string, 0, len(d.held))
	for k := range d.held {
		held = append(held, k)
	}

	sort.Strings(held)

	for _, k := range held {
		t.Logf("%-50s %d entries in the old answers", k, d.held[k])
	}

	t.Logf("differences:")

	for _, k := range keys {
		t.Logf("%-50s %d", k, d.counts[k])

		for _, ex := range d.shown[k] {
			t.Logf("    %s", ex)
		}
	}
}

type refsDiff struct {
	mu      sync.Mutex
	answers int
	counts  map[string]int
	held    map[string]int // what the old readers answered, by class
	shown   map[string][]string
}

func (d *refsDiff) note(class, file, detail string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.counts[class]++
	if len(d.shown[class]) < 4 {
		d.shown[class] = append(d.shown[class], file+": "+detail)
	}
}

func (d *refsDiff) compare(class, file string, got, want any) {
	d.mu.Lock()
	d.answers++
	d.mu.Unlock()

	if reflect.ValueOf(got).Len() == 0 && reflect.ValueOf(want).Len() == 0 {
		return
	}

	d.mu.Lock()
	d.held[class] += reflect.ValueOf(want).Len()
	d.mu.Unlock()

	if !reflect.DeepEqual(got, want) {
		d.note(class, file, firstDifference(got, want))
	}
}

// firstDifference shows where two answers part: the first element of two
// slices that differs, with the lengths.
func firstDifference(got, want any) string {
	g, w := reflect.ValueOf(got), reflect.ValueOf(want)
	if g.Kind() != reflect.Slice {
		return fmt.Sprintf("got %v\n        want %v", trimDump(got), trimDump(want))
	}

	i := 0
	for i < g.Len() && i < w.Len() && reflect.DeepEqual(g.Index(i).Interface(), w.Index(i).Interface()) {
		i++
	}

	el := func(v reflect.Value) string {
		if i < v.Len() {
			return trimDump(v.Index(i).Interface())
		}

		return "(end)"
	}

	return fmt.Sprintf("lengths %d, %d; at %d got %s\n        want %s", g.Len(), w.Len(), i, el(g), el(w))
}

func trimDump(v any) string {
	s := fmt.Sprintf("%+v", v)
	if len(s) > 600 {
		s = s[:600] + "…"
	}

	return s
}

var diffPrefixes = []string{"cf", "cfa"}

func (d *refsDiff) file(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}

	content := string(b)

	// The text entry points.
	d.compare("text IncludeSites", path, IncludeSites(content), oldIncludeSites(content))
	d.compare("text ExtractLinks", path, ExtractLinks(content), oldExtractLinks(content))
	d.compare("text CustomTagCalls(nil)", path, CustomTagCalls(content, nil), oldCustomTagCalls(content, nil))
	d.compare("text CustomTagCalls(prefixes)", path, CustomTagCalls(content, diffPrefixes), oldCustomTagCalls(content, diffPrefixes))
	d.compare("text IncludeWrappers", path, IncludeWrappers(content), oldIncludeWrappers(content))

	// A default parse records includes as its walk goes.
	u := uri.File(path)
	pr := ParseWithOptions(u, content, nil)
	d.compare("parse IncludeSites", path, pr.IncludeSites(), oldIncludeSitesIn(content, pr.Regions))

	// One that records links reads them, and the custom tags, by its walk.
	pl := ParseWithOptions(u, content, &ParseOptions{ExtractLinks: true})
	d.compare("parse DocumentLinks", path, pl.DocumentLinks(), oldLinksInRegions(content, pl.Regions, nil))
	d.compare("parse CustomTagCalls(nil)", path, pl.CustomTagCalls(nil), oldCustomTagCallsIn(pl.Regions, nil, content, nil))
	d.compare("parse CustomTagCalls(prefixes)", path, pl.CustomTagCalls(diffPrefixes), oldCustomTagCallsIn(pl.Regions, nil, content, diffPrefixes))
	d.compare("parse IncludeSites (links parse)", path, pl.IncludeSites(), oldIncludeSitesIn(content, pl.Regions))

	// The links its walk filed (Links, and by function), as the old walk read
	// each region.
	var got []DocumentLink

	got = append(got, pl.Links...)
	for _, links := range pl.funcLinksMap {
		got = append(got, links...)
	}

	var want []DocumentLink

	for _, r := range pl.Regions {
		switch r.Kind {
		case RegionScript:
			want = append(want, oldRegionLinks(r.Text, true, r.StartLine)...)
		case RegionTag:
			for _, l := range oldRegionLinks(r.Text, false, 0) {
				l.Line += uint32(r.StartLine)
				want = append(want, l)
			}
		case RegionSkip:
		}
	}

	byPlace := func(a, b DocumentLink) int {
		if a.Line != b.Line {
			return int(a.Line) - int(b.Line)
		}

		if a.Start != b.Start {
			return int(a.Start) - int(b.Start)
		}

		return strings.Compare(a.Path, b.Path)
	}
	slices.SortFunc(got, byPlace)
	slices.SortFunc(want, byPlace)
	d.compare("parse walk's filed links", path, got, want)
}
