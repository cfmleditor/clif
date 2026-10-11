package path

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const assumedWheelsApplication = `component {
 this.wheels.rootPath=GetDirectoryFromPath(GetCurrentTemplatePath());
 this.vendorDir=expandPath("../vendor/");
 this.wheelsDir=this.vendorDir & "wheels/";
 this.mappings["/wheels"]=this.wheelsDir;
 function onApplicationStart(){
 application.wheelsdi=new wheels.Injector("wheels.Bindings");
 application.wo=application.wheelsdi.getInstance("global");
 }
}`

func TestAssumedWheelsPublicRoot(t *testing.T) {
	for _, mode := range []string{"layout", "different proxy", "missing framework", "conditional startup", "unrelated app", "declared config", "dynamic root", "explicit root"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			app := assumedWheelsApplication

			proxy := `component extends="Application" {}`
			if mode == "different proxy" {
				proxy = `component extends="Other" {}`
			}

			if mode == "unrelated app" {
				app = `component {}`
			}

			if mode == "conditional startup" {
				app = `component {function onApplicationStart(){if(flag){application.wheelsdi=new wheels.Injector("wheels.Bindings");application.wo=application.wheelsdi.getInstance("global");}}}`
			}

			if mode == "dynamic root" {
				app = strings.Replace(assumedWheelsApplication, "this.wheels.rootPath", `this.mappings["/"]=unknown; this.wheels.rootPath`, 1)
			}

			files := map[string]string{"public/Application.cfc": app, "public/ApplicationProxy.cfc": proxy, "vendor/wheels/Injector.cfc": `component {}`, "vendor/wheels/Bindings.cfc": `component {}`, "vendor/wheels/tests/Application.cfc": `component extends="ApplicationProxy" {}`}
			if mode == "missing framework" {
				delete(files, "vendor/wheels/Injector.cfc")
			}

			if mode == "declared config" {
				files["server.json"] = `{}`
			}

			for name, content := range files {
				file := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
					t.Fatal(err)
				}

				if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			InvalidateAppMappingsCache()
			t.Cleanup(InvalidateAppMappingsCache)

			initial := map[string]string{}
			if mode == "explicit root" {
				initial[""] = filepath.Join(root, "other")
			}

			got := LoadAppMappingsWithInitial(filepath.Join(root, "vendor", "wheels", "tests"), initial)

			want := mode == "layout"
			if (got[""] == filepath.Join(root, "public")) != want {
				t.Fatalf("unexpected assumption: %v", got)
			}

			if want && got["wheels"] != filepath.Join(root, "vendor", "wheels") {
				t.Fatalf("parent mappings not bootstrapped: %v", got)
			}
		})
	}
}
