package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestAssumedFrameworksRequireOneProjectContext(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{}

	for _, project := range []string{"one", "two"} {
		base := filepath.Join(root, project)
		files[filepath.Join(base, "tests", "Application.cfc")] = `component {this.mappings["/testbox"]=expandPath("../");}`
		files[filepath.Join(base, "system", "BaseSpec.cfc")] = `component {}`
		files[filepath.Join(base, "system", "Assertion.cfc")] = `component {}`
	}

	read := func(p string) ([]byte, error) {
		if s, ok := files[p]; ok {
			return []byte(s), nil
		}

		return nil, os.ErrNotExist
	}

	one, two := filepath.Join(root, "one"), filepath.Join(root, "two")
	if got := AssumedFrameworks([]string{one, filepath.Join(one, "system")}, read); !slices.Equal(got, []string{"testbox"}) {
		t.Fatalf("same owner: %v", got)
	}

	if got := AssumedFrameworks([]string{one, two}, read); len(got) != 0 {
		t.Fatalf("mixed owners: %v", got)
	}

	files[filepath.Join(one, "server.json")] = `{}`
	if got := AssumedFrameworks([]string{one}, read); len(got) != 0 {
		t.Fatalf("deployment config: %v", got)
	}
}

func TestAssumedMigrationContextRequiresEntryPoints(t *testing.T) {
	for _, mode := range []string{"migration", "ordinary folder", "typed primitive", "outside folder", "mixed owners"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"Application.cfc":                                      `component {this.mappings["/contentbox"]=expandPath("./modules/contentbox");}`,
				"modules/contentbox/ModuleConfig.cfc":                  `component {}`,
				"modules/contentbox/models/content/ContentService.cfc": `component {}`,
			}

			source := `component {function up(schema,qb){} function down(schema,qb){}}`
			if mode == "ordinary folder" {
				source = `component {function unrelated(){}}`
			}

			if mode == "typed primitive" {
				source = `component {function up(string schema,qb){} function down(string schema,qb){}}`
			}

			where := "modules/contentbox/migrations/Initial.cfc"
			if mode == "outside folder" {
				where = "models/Initial.cfc"
			}

			files[where] = source
			for name, body := range files {
				p := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
					t.Fatal(err)
				}

				if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			roots := []string{root}

			if mode == "mixed owners" {
				other := t.TempDir()
				for _, name := range []string{"system/BaseSpec.cfc", "system/Assertion.cfc", "tests/Application.cfc"} {
					p := filepath.Join(other, name)
					if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
						t.Fatal(err)
					}

					body := `component {}`
					if name == "tests/Application.cfc" {
						body = `component {this.mappings["/testbox"]=expandPath("../");}`
					}

					if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
						t.Fatal(err)
					}
				}

				roots = append(roots, other)
			}

			got := AssumedFrameworks(roots, os.ReadFile, os.ReadDir)
			if slices.Contains(got, "cfmigrations") != (mode == "migration") {
				t.Fatalf("%s: %v", mode, got)
			}
		})
	}
}

func TestAssumedCommandBoxContextRequiresHostEvidence(t *testing.T) {
	for _, mode := range []string{"task", "command", "run argument", "ordinary build", "comment only", "declared host", "outside folder", "deployment config"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{"Application.cfc": `component {this.mappings["/testbox"]=expandPath("./");}`, "system/BaseSpec.cfc": `component {}`, "system/Assertion.cfc": `component {}`}
			// TestBox's supported host mapping is declared in tests/Application.cfc.
			delete(files, "Application.cfc")
			files["tests/Application.cfc"] = `component {this.mappings["/testbox"]=expandPath("../");}`
			source := `component {function run(){getCWD();print.line("hello");}}`
			where := "build/Build.cfc"

			switch mode {
			case "command":
				where = "cli/src/commands/tool/run.cfc"
				source = `component extends="commandbox.system.BaseCommand" {function run(){}}`
			case "run argument":
				source = `component {function execute(run){getCWD();}}`
			case "ordinary build":
				source = `component {function run(){print.line("hello");}}`
			case "comment only":
				source = `component {function run(){/* getCWD(); */}}`
			case "declared host":
				source = `component {function getCWD(){}function run(){getCWD();}}`
			case "outside folder":
				where = "models/Build.cfc"
			case "deployment config":
				files["server.json"] = `{}`
			}

			files[where] = source
			for name, body := range files {
				p := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
					t.Fatal(err)
				}

				if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			got := AssumedFrameworks([]string{root}, os.ReadFile, os.ReadDir)

			want := mode == "task" || mode == "command"
			if slices.Contains(got, "commandbox") != want {
				t.Fatalf("%s: %v", mode, got)
			}
		})
	}
}
