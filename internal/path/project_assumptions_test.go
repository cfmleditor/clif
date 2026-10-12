package path

import (
	"os"
	"path/filepath"
	"testing"
)

type projectAssumptionFixture struct {
	name, source, mapping, target string
	sentinels                     []string
}

func TestProjectLibraryNamespaceAssumptions(t *testing.T) {
	for _, profile := range []projectAssumptionFixture{
		{"testbox", "tests/Application.cfc", `component {this.mappings["/testbox"]=expandPath("../");}`, "", []string{"system/BaseSpec.cfc", "system/Assertion.cfc"}},
		{"coldbox", "tests/Application.cfc", `component {this.mappings["/coldbox"]=expandPath("../");}`, "", []string{"system/Bootstrap.cfc", "system/web/Controller.cfc"}},
		{"framework", "Application.cfc", `component {this.mappings["/framework"]=expandPath("./framework");}`, "framework", []string{"framework/one.cfc", "framework/ioc.cfc"}},
		{"mura", "Application.cfc", `component {include "core/appcfc/applicationSettings.cfm";}`, "core/mura", []string{"core/mura/configBean.cfc", "core/mura/bean/beanFactory.cfc"}},
		{"murawrm", "Application.cfc", `component {include "core/appcfc/applicationSettings.cfm";}`, "", []string{"core/mura/configBean.cfc", "core/mura/bean/beanFactory.cfc"}},
		{"contentbox", "Application.cfc", `component {this.mappings["/contentbox"]=expandPath("./modules/contentbox");}`, "modules/contentbox", []string{"modules/contentbox/ModuleConfig.cfc", "modules/contentbox/models/content/ContentService.cfc"}},
	} {
		for _, mode := range []string{"layout", "missing sentinel", "source override", "dynamic override", "server config"} {
			t.Run(profile.name+"/"+mode, func(t *testing.T) {
				checkProjectAssumptionFixture(t, &profile, mode)
			})
		}
	}
}

func checkProjectAssumptionFixture(t *testing.T, profile *projectAssumptionFixture, mode string) {
	t.Helper()
	root := t.TempDir()

	files := map[string]string{profile.source: profile.mapping, "nested/Application.cfc": `component {}`}
	for _, f := range profile.sentinels {
		files[f] = `component {}`
	}

	if profile.name == "mura" || profile.name == "murawrm" {
		files["core/appcfc/applicationSettings.cfm"] = `<cfscript>this.mappings["/mura"]=expandPath("../mura");this.mappings["/muraWRM"]=expandPath("../..");</cfscript>`
	}

	if mode == "missing sentinel" {
		delete(files, profile.sentinels[0])
	}

	if mode == "source override" {
		files["nested/Application.cfc"] = `component {this.mappings["/` + profile.name + `"]=expandPath("./own");}`
	}

	if mode == "dynamic override" {
		files["nested/Application.cfc"] = `component {this.mappings["/` + profile.name + `"]=unknown;}`
	}

	if mode == "server config" {
		files["server.json"] = `{}`
	}

	for name, content := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	InvalidateAppMappingsCache()
	t.Cleanup(InvalidateAppMappingsCache)

	got := LoadAppMappings(filepath.Join(root, "nested"))[profile.name]

	switch mode {
	case "layout":
		if got != filepath.Join(root, profile.target) {
			t.Fatalf("namespace not supplied: %q", got)
		}
	case "source override":
		if got != filepath.Join(root, "nested", "own") {
			t.Fatalf("source did not override: %q", got)
		}
	default:
		if got != "" {
			t.Fatalf("unsupported namespace %q", got)
		}
	}
}
