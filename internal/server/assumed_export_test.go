package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssumedFrameworkExportMatchesEditor(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{"Application.cfc": `component {this.mappings["/contentbox"]=expandPath("./modules/contentbox");}`, "modules/contentbox/ModuleConfig.cfc": `component {}`, "modules/contentbox/models/content/ContentService.cfc": `component {}`, "handlers/Main.cfc": `component {function index(event){event.getValue("x");event.definitelyMissing();}}`}

	var paths []string

	for name, body := range files {
		file := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}

		paths = append(paths, file)
	}

	s := newTestServer()
	s.WorkspaceFolders = []string{root}

	reports, _ := s.unresolvedReports(paths, []string{filepath.Join(root, "report.md")})
	if len(reports) != 1 {
		t.Fatalf("expected one report: %v", reports)
	}

	for _, r := range reports {
		if strings.Contains(r.content, "getValue") {
			t.Error("export lost assumed framework")
		}

		if !strings.Contains(r.content, "definitelyMissing") {
			t.Fatal("export dropped genuinely missing method")
		}
	}
}
