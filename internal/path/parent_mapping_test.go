package path

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapParentMappings(t *testing.T) {
	root := t.TempDir()
	write := func(name, src string) {
		t.Helper()

		file := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("app/Application.cfc", `component extends="shared.Base" {this.mappings["/override"]=expandPath("./child");}`)
	write("one/Base.cfc", `component {this.mappings["/inherited"]=expandPath("./models");this.mappings["/override"]=expandPath("./parent");function unused(){this.mappings["/unused"]=expandPath("./bad");include "bad.cfm";}}`)
	write("one/bad.cfm", `this.mappings["/bad"]="bad";`)
	write("two/Base.cfc", `component {this.mappings["/inherited"]=expandPath("./other");}`)

	app := filepath.Join(root, "app")

	InvalidateAppMappingsCache()

	for _, name := range []string{"one", "two", "one"} {
		got := LoadAppMappingsWithInitial(app, map[string]string{"shared": filepath.Join(root, name)})

		rel := "models"
		if name == "two" {
			rel = "other"
		}

		if got["inherited"] != filepath.Join(root, name, rel) {
			t.Fatalf("%s inherited: %v", name, got)
		}

		if got["override"] != filepath.Join(app, "child") {
			t.Fatalf("child override: %v", got)
		}

		if got["unused"] != "" || got["bad"] != "" {
			t.Fatalf("unused method executed: %v", got)
		}
	}
	// Unqualified names do not borrow a catalog's basename tie-break.
	write("app/Application.cfc", `component extends="Base" {}`)
	InvalidateAppMappingsCache()

	if got := LoadAppMappingsWithInitial(app, map[string]string{"shared": filepath.Join(root, "one")}); got["inherited"] != "" {
		t.Fatalf("unproven parent: %v", got)
	}
	// A missing physical parent is reconsidered after path invalidation.
	write("app/Application.cfc", `component extends="shared.Later" {}`)
	InvalidateAppMappingsCache()

	initial := map[string]string{"shared": filepath.Join(root, "one")}
	if got := LoadAppMappingsWithInitial(app, initial); got["created"] != "" {
		t.Fatal(got)
	}

	write("one/Later.cfc", `component {this.mappings["/created"]=expandPath("./created");}`)
	InvalidateAppMappingsCache()

	if got := LoadAppMappingsWithInitial(app, initial); got["created"] != filepath.Join(root, "one", "created") {
		t.Fatal(got)
	}

	if err := os.Remove(filepath.Join(root, "one", "Later.cfc")); err != nil {
		t.Fatal(err)
	}

	InvalidateAppMappingsCache()

	if got := LoadAppMappingsWithInitial(app, initial); got["created"] != "" {
		t.Fatal(got)
	}
}

func TestTagMethodMappingsDoNotRunInConstructor(t *testing.T) {
	root := t.TempDir()

	src := `<cfcomponent><cfset this.mappings["/constructor"]=expandPath("./ok")><cffunction name="unused"><cfset this.mappings["/unused"]=expandPath("./bad")></cffunction><cfset this.mappings["/after"]=expandPath("./after")></cfcomponent>`
	if err := os.WriteFile(filepath.Join(root, "Application.cfc"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	InvalidateAppMappingsCache()

	got := LoadAppMappings(root)
	if got["constructor"] != filepath.Join(root, "ok") || got["after"] != filepath.Join(root, "after") || got["unused"] != "" {
		t.Fatal(got)
	}
}

func TestRequestPageMappingExecution(t *testing.T) {
	for _, target := range []string{"arguments.page", "other.page"} {
		t.Run(target, func(t *testing.T) {
			root := t.TempDir()

			src := `component {function unused(){this.mappings["/unused"]="unused";} function onRequest(required string page){if (unknown) {this.mappings["/guarded"]="guarded";}this.mappings["/module"]=getDirectoryFromPath(getCurrentTemplatePath()) & "moduleDir";include ` + target + `;this.mappings["/after"]="after";}}`
			if err := os.WriteFile(filepath.Join(root, "Application.cfc"), []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}

			InvalidateAppMappingsCache()

			got := LoadAppMappings(root)
			if (got["module"] == filepath.Join(root, "moduleDir")) != (target == "arguments.page") {
				t.Fatal(got)
			}

			if got["unused"] != "" || got["guarded"] != "" || got["after"] != "" {
				t.Fatal(got)
			}
		})
	}
}
