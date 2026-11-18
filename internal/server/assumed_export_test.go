package server

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

func TestUnresolvedExportRefusesUnreadableTargets(t *testing.T) {
	root := t.TempDir()

	path := filepath.Join(root, "large.cfm")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Truncate(path, vfs.MaxFileSize+1); err != nil {
		t.Fatal(err)
	}

	s := newTestServer()
	s.WorkspaceFolders = []string{root}

	reports, _, err := s.unresolvedReports([]string{path}, []string{filepath.Join(root, "report.md")})
	if !errors.Is(err, vfs.ErrTooLarge) || len(reports) != 0 {
		t.Errorf("reports = %v, error = %v; want no reports and the target's read failure", reports, err)
	}
}

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

	reports, _, err := s.unresolvedReports(paths, []string{filepath.Join(root, "report.md")})
	if err != nil {
		t.Fatal(err)
	}

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

// TestExportTypesAnArgumentFromItsCallers: the export is the unresolved scan
// the CLI runs, and the CLI infers an untyped argument from what every caller
// passes it. The export did not, so it listed `thing.ping()` in a report that
// `clif unresolved` of the same project does not.
func TestExportTypesAnArgumentFromItsCallers(t *testing.T) {
	root := t.TempDir()

	var paths []string

	for name, body := range map[string]string{
		"models/Item.cfc": `component { function ping(){} }`,
		"svc/Svc.cfc": `component {
function use( required thing ){ arguments.thing.ping(); arguments.thing.missing(); }
function run(){ use( new models.Item() ); }
}`,
	} {
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

	reports, _, err := s.unresolvedReports(paths, []string{filepath.Join(root, "report.md")})
	if err != nil {
		t.Fatal(err)
	}

	if len(reports) != 1 {
		t.Fatalf("expected one report: %v", reports)
	}

	content := reports[0].content
	if strings.Contains(content, "ping") {
		t.Errorf("export did not infer the argument from its caller:\n%s", content)
	}

	if !strings.Contains(content, "missing") {
		t.Errorf("export dropped a method the typed argument really lacks:\n%s", content)
	}
}
