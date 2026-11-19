package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoutesInvalidFormatPreservesReport(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "report.txt")
	const original = "previous report\n"
	writeTestFile(t, report, original)

	if out, code := runClif(t, dir, "routes", "--format", "jsno", "--out", report, dir); code != 1 || !strings.Contains(out, "unknown format") {
		t.Fatalf("unexpected exit %d: %s", code, out)
	}

	got, err := os.ReadFile(report)
	if err != nil || string(got) != original {
		t.Fatalf("report is %q, error %v", got, err)
	}
}

func TestRouteTextZeroLimitShowsEveryExample(t *testing.T) {
	resolved := []routeFinding{
		{Route: "home.first", Targets: []string{"first-view.cfm"}},
		{Route: "home.second", Targets: []string{"second-view.cfm"}},
	}
	unresolved := []routeFinding{
		{Route: "home.missing.one", File: "first-site.cfm", Line: 1},
		{Route: "home.missing.two", File: "second-site.cfm", Line: 2},
	}

	var report strings.Builder
	if err := writeRouteText(&report, resolved, unresolved, 4, false, 0); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"first-view.cfm", "second-view.cfm", "first-site.cfm:1", "second-site.cfm:2"} {
		if !strings.Contains(report.String(), want) {
			t.Errorf("report omitted %q:\n%s", want, report.String())
		}
	}
}

func TestRouteScanOrdersSitesForTheSameRoute(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".clif.json"), `{"routes":{"attributes":["data-view"],"views":[{"path":"views/${1}/${2}.cfm"}]}}`)
	names := []string{"a.cfm", "b.cfm", "c.cfm", "d.cfm", "e.cfm", "f.cfm"}
	for _, name := range names {
		writeTestFile(t, filepath.Join(dir, name), `<a data-view="home.missing"></a>`)
	}

	for range 4 {
		_, unresolved, scanned := scanRoutes([]string{dir}, true)
		if scanned != len(names) || len(unresolved) != len(names) {
			t.Fatalf("scanned %d, unresolved %d, want %d", scanned, len(unresolved), len(names))
		}
		for i, name := range names {
			if unresolved[i].File != name {
				t.Fatalf("site %d is %q, want %q", i, unresolved[i].File, name)
			}
		}
	}
}
