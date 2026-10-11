package path

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLuceeSourceAdminMappingAssumption(t *testing.T) {
	for _, mode := range []string{"layout", "missing marker", "explicit config", "source override", "dynamic override", "server config"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
				"core/src/main/cfml/context/admin/services.update.functions.cfm":              `<cfscript>function isStableVersion(){}</cfscript>`,
				"core/src/main/cfml/context/admin/ext.functions.cfm":                          `<cfscript>function toOSGiVersion(){}</cfscript>`,
				"test/tickets/Application.cfc":                                                `component {}`,
			}
			initial := map[string]string{}

			switch mode {
			case "missing marker":
				delete(files, "core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc")
			case "explicit config":
				initial["/AdMiN"] = filepath.Join(root, "custom")
			case "source override":
				files["test/tickets/Application.cfc"] = `component {this.mappings["/admin"]=expandPath("./own");}`
			case "dynamic override":
				files["test/tickets/Application.cfc"] = `component {this.mappings["/admin"]=runtimePath;}`
			case "server config":
				files["server.json"] = `{}`
			}

			for name, body := range files {
				file := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
					t.Fatal(err)
				}

				if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			InvalidateAppMappingsCache()
			t.Cleanup(InvalidateAppMappingsCache)

			got := LoadAppMappingsWithInitial(filepath.Join(root, "test", "tickets"), initial)

			switch mode {
			case "layout":
				if got["admin"] != filepath.Join(root, "core", "src", "main", "cfml", "context", "admin") {
					t.Fatalf("missing engine mapping: %v", got)
				}
			case "source override":
				if got["admin"] != filepath.Join(root, "test", "tickets", "own") {
					t.Fatalf("source override lost: %v", got)
				}
			default:
				if got["admin"] != "" {
					t.Fatalf("assumption survived explicit/unsupported context: %v", got)
				}
			}
		})
	}
}
