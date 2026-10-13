package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/codemap"
	"github.com/cfmleditor/clif/internal/unresolved"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestGraphAssumesTheFrameworkUnresolvedAssumes: with no .clif.json the project
// is read for the framework it is, and `unresolved` applies that framework's
// resolvers, implicit bases and bundled API. graph built its resolver from the
// config alone, so on Masa CMS (no config, the mura preset) it reported 21,962
// unresolved calls where `unresolved` reported 3,786. Both commands now count
// the same calls on the same files, including a method that really is missing.
func TestGraphAssumesTheFrameworkUnresolvedAssumes(t *testing.T) {
	root := t.TempDir()

	var sources []string

	for name, content := range map[string]string{
		"Application.cfc":                                      `component {this.mappings["/contentbox"]=expandPath("./modules/contentbox");}`,
		"modules/contentbox/ModuleConfig.cfc":                  `component {}`,
		"modules/contentbox/models/content/ContentService.cfc": `component {}`,
		"handlers/Main.cfc":                                    `component {function index(event){event.getValue("x");event.definitelyMissing();}}`,
	} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		sources = append(sources, p)
	}

	fsys := vfs.OS{}

	opt := unresolvedOptions(nil, []string{root}, &unresolvedFlags{})
	report := unresolved.Scan(fsys, sources, nil, opt)

	_, fallback, _, _ := routeWorkspace(fsys, root, &graphFlags{paths: []string{root}, quiet: true})

	m := codemap.Build(&codemap.Options{
		Root:     root,
		Files:    sources,
		FS:       fsys,
		Resolver: fallback.Resolver,
		Workers:  2,
	})

	if m.Stats.Unresolved != len(report.Calls) {
		t.Errorf("unresolved reports %d calls, the map %d: %+v", len(report.Calls), m.Stats.Unresolved, report.Calls)
	}

	if len(report.Calls) != 1 || report.Calls[0].Function != "definitelyMissing" {
		t.Errorf("the fixture should leave only definitelyMissing unresolved: %+v", report.Calls)
	}
}
