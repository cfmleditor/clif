package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestSelectedApplicationLifecycle(t *testing.T) {
	for _, form := range []string{"script", "tag"} {
		for _, mode := range []string{"inherited", "override", "super", "guarded"} {
			t.Run(form+"/"+mode, func(t *testing.T) {
				root := t.TempDir()
				write := startupFixtureWriter(t, root)
				base := `component {
 application.constructor=new Service();
 function unused(){application.unused=new Service();include "unused.cfm";} function onApplicationStart(){
 application.lifecycle=new Service();
} }`
				child := `component extends="shared.Base" {}`

				if mode != "inherited" {
					body := ""
					if mode == "super" {
						body = "super.onApplicationStart();"
					}

					if mode == "guarded" {
						body = "if (unknown) {super.onApplicationStart();}"
					}

					child = `component extends="shared.Base" {function onApplicationStart(){` + body + `}}`
				}

				if form == "tag" {
					base = `<cfcomponent><cfset application.constructor=new Service()><cffunction name="unused"><cfset application.unused=new Service()><cfinclude template="unused.cfm"></cffunction><cffunction name="onApplicationStart"><cfset application.lifecycle=new Service()></cffunction></cfcomponent>`

					child = `<cfcomponent extends="shared.Base"></cfcomponent>`
					if mode != "inherited" {
						child = `<cfcomponent extends="shared.Base"><cffunction name="onApplicationStart"></cffunction></cfcomponent>`
					}
					// Super expansion is deliberately supported only for script statements.
				}

				write("parent/Base.cfc", base)
				write("parent/Service.cfc", `component {function go(){}}`)
				write("parent/unused.cfm", `<cfset application.fromUnused=new Service()>`)
				write("app/Application.cfc", child)
				cfpath.InvalidateAppMappingsCache()

				r := &resolve.Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{root}, Mappings: map[string]string{"shared": filepath.Join(root, "parent")}}

				got := reasonsFor(t, r, filepath.Join(root, "app", "page.cfm"), `<cfset a=application.constructor.go()>
<cfset b=application.lifecycle.go()>
<cfset c=application.unused.go()>
<cfset d=application.fromUnused.go()>`)
				if got["application.constructor.go"] != "" {
					t.Fatal("parent constructor: ", got)
				}

				want := mode == "inherited" || mode == "super" && form == "script"
				if (got["application.lifecycle.go"] == "") != want {
					t.Fatalf("lifecycle want %v: %v", want, got)
				}

				if got["application.unused.go"] == "" || got["application.fromUnused.go"] == "" {
					t.Fatal("unused body supplied facts: ", got)
				}
			})
		}
	}
}

func startupFixtureWriter(t *testing.T, root string) func(string, string) {
	t.Helper()

	return func(name, content string) {
		t.Helper()

		file := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMissingMappedParentIsDiscoveryDependency(t *testing.T) {
	root := t.TempDir()
	write := startupFixtureWriter(t, root)
	write("app/Application.cfc", `component extends="shared.Later" {}`)

	dependencies := &resolve.DiscoveryDependencies{}
	r := &resolve.Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{filepath.Join(root, "app")}, Mappings: map[string]string{"shared": filepath.Join(root, "parent")}, Discovery: dependencies}

	parent := filepath.Join(root, "parent", "Later.cfc")
	if !r.DiscoveryAffected(parent) {
		t.Fatal("new mapped parent would not rebuild discovery")
	}

	write("parent/Later.cfc", `component {this.mappings["/models"]=expandPath("./models");}`)
	cfpath.InvalidateAppMappingsCache()
	r.InvalidatePaths()

	if got := r.EffectiveMappings(filepath.Join(root, "app"))["models"]; got != filepath.Join(root, "parent", "models") {
		t.Fatal(got)
	}

	if err := os.Remove(parent); err != nil {
		t.Fatal(err)
	}

	cfpath.InvalidateAppMappingsCache()
	r.InvalidatePaths()

	if got := r.EffectiveMappings(filepath.Join(root, "app"))["models"]; got != "" {
		t.Fatal(got)
	}
}

func TestIncludedLifecycleDeclarationsAreSelected(t *testing.T) {
	for _, mode := range []string{"selected", "override", "guarded declaration", "unused declaration", "repeated guarded declaration", "repeated declaration"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			write := startupFixtureWriter(t, root)

			include := `include "../shared/lifecycle.cfm";`
			if mode == "guarded declaration" {
				include = `if(flag){include "../shared/lifecycle.cfm";}`
			}

			if mode == "repeated guarded declaration" {
				include += `if(flag){include "../shared/lifecycle.cfm";}`
			}

			if mode == "repeated declaration" {
				include += include
			}

			write("parents/Base.cfc", `component {`+include+`}`)

			child := `component extends="parents.Base" {}`
			if mode == "override" {
				child = `component extends="parents.Base" {function onApplicationStart(){}}`
			}

			write("Application.cfc", child)

			name := "onApplicationStart"
			if mode == "unused declaration" {
				name = "unused"
			}

			write("shared/lifecycle.cfm", `<cfscript>function `+name+`(){include "start.cfm";}</cfscript>`)
			write("shared/start.cfm", `<cfset application.service=new Service()>`)
			write("shared/Service.cfc", `component {function ready(){}}`)
			write("Service.cfc", `component {function unrelated(){}}`)

			r := &resolve.Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{root}}

			got := reasonsFor(t, r, filepath.Join(root, "page.cfm"), `<cfset result=application.service.ready()>`)
			if (got["application.service.ready"] == "") != (mode == "selected") {
				t.Fatalf("%s: %v", mode, got)
			}
		})
	}
}
