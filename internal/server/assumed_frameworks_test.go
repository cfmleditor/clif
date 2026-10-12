package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

func TestEditorUsesAssumedContentBoxFramework(t *testing.T) {
	root := t.TempDir()
	content := `component {function index(event){event.getValue("x");event.definitelyMissing();}}`

	files := map[string]string{"Application.cfc": `component {this.mappings["/contentbox"]=expandPath("./modules/contentbox");}`, "modules/contentbox/ModuleConfig.cfc": `component {}`, "modules/contentbox/models/content/ContentService.cfc": `component {}`, "handlers/Main.cfc": content}
	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	s := newTestServer()
	s.WorkspaceFolders = []string{root}
	r := s.getResolver()
	pr := parser.ParseWithOptions(uri.File(filepath.Join(root, "handlers", "Main.cfc")), content, &parser.ParseOptions{ExtractCalls: true, Resolvers: r.Resolvers})

	calls := pr.AllCalls()
	for i := range calls {
		reason := r.CanResolveCall(&calls[i], pr, filepath.Join(root, "handlers"))
		if calls[i].FuncName == "getValue" && reason != "" {
			t.Fatal(reason)
		}

		if calls[i].FuncName == "definitelyMissing" && reason == "" {
			t.Fatal("unknown method accepted")
		}
	}

	if !r.DiscoveryAffected(filepath.Join(root, "Application.cfc")) {
		t.Fatal("assumption source not tracked")
	}

	s.ConfigPath = filepath.Join(root, ".clif.json")
	s.invalidateResolver()

	if s.getResolver().Stubs != nil {
		t.Fatal("explicit config did not suppress assumption")
	}
}

func TestAssumedToolingChecksMethodsAndTracksNewSources(t *testing.T) {
	root := t.TempDir()

	files := map[string]string{
		"Application.cfc":                                      `component {this.mappings["/contentbox"]=expandPath("./modules/contentbox");}`,
		"modules/contentbox/ModuleConfig.cfc":                  `component {}`,
		"modules/contentbox/models/content/ContentService.cfc": `component {}`,
		"modules/contentbox/migrations/Initial.cfc":            `component {function up(schema,qb){schema.create("test",function(table){table.string("title");table.definitelyMissing();});}function down(schema,qb){}}`,
		"build/Build.cfc":                                      `component {function run(){getCWD();print.line("test");print.definitelyMissing();}}`,
	}
	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	s := newTestServer()
	s.WorkspaceFolders = []string{root}
	r := s.getResolver()

	for _, name := range []string{"modules/contentbox/migrations/Initial.cfc", "build/Build.cfc"} {
		file := filepath.Join(root, name)
		pr := parser.ParseWithOptions(uri.File(file), files[name], &parser.ParseOptions{ExtractCalls: true, Resolvers: r.Resolvers})

		calls := pr.AllCalls()
		for i := range calls {
			call := &calls[i]
			if call.FuncName != "string" && call.FuncName != "line" && call.FuncName != "definitelyMissing" {
				continue
			}

			reason := r.CanResolveCall(call, pr, filepath.Dir(file))
			if name == "build/Build.cfc" && call.FuncName == "definitelyMissing" {
				_, steps := r.ExplainCall(call, pr, filepath.Dir(file))
				if reason != "" || !strings.Contains(strings.Join(steps, "\n"), "onMissingMethod") {
					t.Fatalf("print dynamic API lost: %s %v", reason, steps)
				}

				continue
			}

			if (reason == "") == (call.FuncName == "definitelyMissing") {
				t.Fatalf("%s: %s: %s", name, call.FuncName, reason)
			}
		}

		if !r.DiscoveryAffected(file) {
			t.Fatalf("source not tracked: %s", name)
		}
	}

	if !r.DiscoveryAffected(filepath.Join(root, "build", "NewTask.cfc")) {
		t.Fatal("adding a task does not invalidate source discovery")
	}

	if !r.DiscoveryAffected(filepath.Join(root, "migrations", "nested", "NewMigration.cfc")) {
		t.Fatal("failed directory reads did not track new nested sources")
	}

	if r.DiscoveryAffected(filepath.Join(root, "build-other", "NewTask.cfc")) {
		t.Fatal("directory boundary escaped")
	}

	s.ConfigPath = filepath.Join(root, ".clif.json")
	s.invalidateResolver()

	if s.getResolver().Stubs != nil {
		t.Fatal("explicit config did not suppress tooling defaults")
	}
}
