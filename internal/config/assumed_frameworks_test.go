package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
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

// TestAPresideCheckoutIsColdBoxWithPresidesRequestContext: Preside's source
// maps /preside to its own root (tests/Application.cfc, as the Bootstrap every
// Preside site extends does) and runs on ColdBox without shipping it. Nothing
// recognised it, so no preset was assumed: `event` was untyped in every
// handler (2,911 findings) and, once ColdBox's preset typed it, a ColdBox
// RequestContext lacking Preside's buildAdminLink(). With the checkout
// recognised, the ColdBox preset is assumed and `event` is Preside's
// decorator, and the controller Preside's own, ahead of the preset's.
func TestAPresideCheckoutIsColdBoxWithPresidesRequestContext(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		filepath.Join(root, "tests", "Application.cfc"):                    `component { currentDir = GetDirectoryFromPath( GetCurrentTemplatePath() ); this.mappings['/preside'] = currentDir & "../"; }`,
		filepath.Join(root, "system", "Bootstrap.cfc"):                     `component {}`,
		filepath.Join(root, "system", "services", "PresideSuperClass.cfc"): `component {}`,
	}

	read := func(p string) ([]byte, error) {
		if s, ok := files[p]; ok {
			return []byte(s), nil
		}

		return nil, os.ErrNotExist
	}

	r := Assumed([]string{root}, read)
	if !slices.Equal(r.Frameworks, []string{"coldbox"}) {
		t.Fatalf("frameworks = %v, want [coldbox]", r.Frameworks)
	}

	var event string

	for _, res := range r.ComponentResolvers {
		if strings.Contains(res.Prefix, "event") && strings.Contains(res.Match, "event") {
			event = res.Resolve

			break
		}
	}

	if event != presideRequestContext {
		t.Errorf("event resolves to %q first, want %q", event, presideRequestContext)
	}

	// Preside's own controller, which adds handlerExists() and the viewlet
	// API, ahead of ColdBox's for the same names.
	for _, name := range []string{"coldbox", "controller", "getController"} {
		var first string

		for _, res := range r.ComponentResolvers {
			if strings.Contains(res.Prefix, name) && strings.Contains(res.Match, name) {
				first = res.Resolve

				break
			}
		}

		if first != presideController {
			t.Errorf("%s resolves to %q first, want %q", name, first, presideController)
		}
	}

	// Preside's own id factories build what getInstance( id ) does.
	for _, call := range []string{`getSingleton( "auditService" )`, `getModel( "auditService" )`} {
		if got := parser.ResolveFromCall(call, ParserResolvers(r.ComponentResolvers)); got != "auditService" {
			t.Errorf("%s resolves to %q, want auditService", call, got)
		}
	}

	delete(files, filepath.Join(root, "system", "services", "PresideSuperClass.cfc"))

	if got := Assumed([]string{root}, read); len(got.Frameworks) != 0 {
		t.Errorf("without Preside's markers: frameworks = %v", got.Frameworks)
	}
}

// TestACommandBoxCheckoutIsCommandBox: CommandBox's own source maps
// /commandbox to src/cfml from its tests, and its commands and services write
// to `print` and build what `getInstance( "X" )` names. Nothing recognised
// the checkout, so the commandbox preset was not assumed there and `print`
// alone was 888 findings.
func TestACommandBoxCheckoutIsCommandBox(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		filepath.Join(root, "tests", "Application.cfc"):                 `component { this.mappings[ '/commandbox' ] = '../src/cfml'; }`,
		filepath.Join(root, "src", "cfml", "system", "BaseCommand.cfc"): `component {}`,
		filepath.Join(root, "src", "cfml", "system", "Shell.cfc"):       `component {}`,
	}

	read := func(p string) ([]byte, error) {
		if s, ok := files[p]; ok {
			return []byte(s), nil
		}

		return nil, os.ErrNotExist
	}

	if r := Assumed([]string{filepath.Join(root, "src", "cfml", "system")}, read); !slices.Equal(r.Frameworks, []string{"commandbox"}) {
		t.Fatalf("frameworks = %v, want [commandbox]", r.Frameworks)
	}

	delete(files, filepath.Join(root, "src", "cfml", "system", "Shell.cfc"))

	if got := Assumed([]string{root}, read); len(got.Frameworks) != 0 {
		t.Errorf("without CommandBox's markers: frameworks = %v", got.Frameworks)
	}
}
