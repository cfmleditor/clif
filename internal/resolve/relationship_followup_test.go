package resolve

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestUnknownArgumentCollectionIsNotOmitted(t *testing.T) {
	for _, source := range []string{`f(argumentCollection=unknown)`, `f(argumentCollection:unknown)`} {
		expr, passed := callArgumentAt(allTokens(source), 0, 0, "value")
		if !passed || len(expr) != 0 {
			t.Fatalf("%s: unknown collection must supply unknown evidence, got %v, %v", source, expr, passed)
		}
	}
}

func TestProviderCallbackRejectsParameterMutation(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		resolved   bool
	}{
		{"unchanged", `x.go();`, true},
		{"before", `x=unknown;x.go();`, false},
		{"scoped before", `arguments.x=unknown;x.go();`, false},
		{"after", `x.go();x=unknown;`, true},
		{"escape", `external(x);x.go();`, false},
		{"escaped hashes", `var text="##external(x)##";x.go();`, true},
		{"nested interpolation escape", `var text="#wrap('#external(x)#')#";x.go();`, false},
		{"interpolated escape", `var text="#external(x)#";x.go();`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"Service.cfc": `component {function go(){}}`, "Provider.cfc": `component {function direct(callback){callback(new Service());}}`})
			r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
			src := `component {function run(){var p=new Provider();p.direct(function(x){` + tc.body + `});}}`
			pr := parser.ParseWithOptions(cfpath.ToURI(filepath.Join(dir, "Page.cfc")), src, &parser.ParseOptions{ExtractCalls: true})
			checked := 0

			for _, call := range pr.AllCalls() {
				if call.FuncName == "go" {
					checked++

					if got := r.CanResolveCall(&call, pr, dir); (got == "") != tc.resolved {
						t.Fatalf("got %q, want resolved %v", got, tc.resolved)
					}
				}
			}

			if checked != 1 {
				t.Fatalf("checked %d calls", checked)
			}
		})
	}
}

func TestSourceInvalidationRefreshesIncludeHost(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"A.cfc": `component {function onlyA(){}}`, "B.cfc": `component {function onlyB(){}}`,
		"Host.cfc": `component {function run(required A value){include "view.cfm";}}`,
		"view.cfm": `<cfoutput>#value.onlyA()# #value.onlyB()#</cfoutput>`,
	})

	r := &Resolver{}

	got := templateReasonsWith(t, r, dir, "view.cfm")
	if got["value.onlyA"] != "" || got["value.onlyB"] == "" {
		t.Fatal(got)
	}

	writeFiles(t, dir, map[string]string{"Host.cfc": `component {function run(required B value){include "view.cfm";}}`})
	r.InvalidatePaths()

	file := filepath.Join(dir, "view.cfm")

	pr := parser.ParseWithOptions(cfpath.ToURI(file), `<cfoutput>#value.onlyA()# #value.onlyB()#</cfoutput>`, &parser.ParseOptions{ExtractCalls: true})
	for _, call := range pr.AllCalls() {
		why := r.CanResolveCall(&call, pr, dir)
		if (why == "") != strings.EqualFold(call.FuncName, "onlyB") {
			t.Fatalf("%s: %q", call.FuncName, why)
		}
	}
}

func TestSourceInvalidationRefreshesDirectHandlerParse(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Host.cfc")
	writeFiles(t, dir, map[string]string{"Host.cfc": `component {function run(required A value){}}`})

	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	if pr := r.handlerParse(file); pr == nil || !strings.Contains(pr.Content, "required A") {
		t.Fatal("initial parse")
	}

	writeFiles(t, dir, map[string]string{"Host.cfc": `component {function run(required B value){}}`})
	r.InvalidatePaths()

	if pr := r.handlerParse(file); pr == nil || !strings.Contains(pr.Content, "required B") {
		t.Fatal("stale handler parse")
	}
}

func TestReporterIncludeTransfersQualifiedArgument(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"system/TestResult.cfc": `component {function getTotalDuration(){}}`,
		"system/reports/DocReporter.cfc": `component {
 any function runReport(
 required testbox.system.TestResult results,
 struct options = {},
 boolean justReturn=false
 ) {
 savecontent variable="local.report" { include "assets/doc.cfm"; }
 return local.report;
 }
}`,
		"system/reports/assets/doc.cfm": `<cfoutput>#results.getTotalDuration()#</cfoutput>`,
		"box.json":                      `{"slug":"testbox"}`,
	})

	if got := templateReasons(t, dir, "system/reports/assets/doc.cfm"); got["results.getTotalDuration"] != "" {
		t.Fatal(got)
	}
}

func TestModuleConfigurationLifecycle(t *testing.T) {
	for _, tc := range []struct{ name, source, folder string }{
		{"configure conventions", `function configure(){variables.conventions={modelsLocation:"domain"};}`, "domain"},
		{"unscoped conventions", `function configure(){conventions={modelsLocation:"domain"};}`, "domain"},
		{"bracket conventions", `function configure(){variables["conventions"]={};variables.conventions["modelsLocation"]="domain";}`, "domain"},
		{"public conventions ignored", `this.conventions={modelsLocation:"domain"};`, "models"},
		{"replacement resets", `function configure(){conventions={modelsLocation:"domain"};conventions={};}`, "models"},
		{"disabled", `this.disabled=true;`, ""},
		{"inactive", `this.activate=false;`, ""},
		{"constructor bracket flag", `this["autoMapModels"]=false;`, ""},
		{"configure public flag too late", `function configure(){this.autoMapModels=false;}`, "models"},
		{"escaped conventions", `function configure(){conventions={modelsLocation:"domain"};mutate(conventions);}`, ""},
		{"computed convention key", `function configure(){conventions[key]="domain";}`, ""},
		{"unrelated local", `function configure(){var label="local";conventions={modelsLocation:"domain"};}`, "domain"},
		{"local conventions", `function configure(){var conventions={modelsLocation:"domain"};}`, ""},
		{"conditional conventions", `function configure(){if(runtime()) conventions={modelsLocation:"domain"};}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			got := moduleModelRoot(`component {`+tc.source+`}`, dir)

			want := ""
			if tc.folder != "" {
				want = filepath.Join(dir, tc.folder)
			}

			if got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

func TestCallerApplicationFollowsSharedScopeReturn(t *testing.T) {
	dir := t.TempDir()
	library := filepath.Join(dir, "library")
	writeFiles(t, dir, map[string]string{
		"library/Application.cfc": `component {function onRequestStart(){abort;}}`,
		"library/Factory.cfc":     `component {function make(){return application.service;}}`,
		"a/Application.cfc":       `component {this.mappings["/lib"]="` + filepath.ToSlash(library) + `";function onApplicationStart(){application.service=new Service();}}`,
		"a/Service.cfc":           `component {function onlyA(){}}`,
		"b/Application.cfc":       `component {this.mappings["/lib"]="` + filepath.ToSlash(library) + `";function onApplicationStart(){application.service=new Service();}}`,
		"b/Service.cfc":           `component {function onlyB(){}}`,
	})

	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	for _, app := range []string{"a", "b", "a"} {
		got := r.FuncLookup(filepath.Join(dir, app))("lib.Factory", "make")

		want := filepath.Join(dir, app, "Service.cfc")
		if got != want {
			t.Errorf("%s got %q want %q", app, got, want)
		}
	}

	if got := r.FuncLookup(library)("Factory", "make"); got != "" {
		t.Fatal("direct library borrowed caller", got)
	}
}

func TestBuilderCallbackRejectsParameterMutation(t *testing.T) {
	dir := t.TempDir()
	src := `component extends="cborm.models.VirtualEntityService" {function run(){newCriteria().when(true,function(c){c=unknown;c.isEq("a",1);});}}`
	writeFiles(t, dir, map[string]string{"Svc.cfc": src})
	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}

	pr := parser.ParseWithOptions(cfpath.ToURI(filepath.Join(dir, "Svc.cfc")), src, &parser.ParseOptions{ExtractCalls: true})
	for _, call := range pr.AllCalls() {
		if call.FuncName == "isEq" && r.CanResolveCall(&call, pr, dir) == "" {
			t.Fatal("reassigned builder parameter retained entry type")
		}
	}
}

func TestSourceInvalidationRefreshesLazySource(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Holder.cfc")
	writeFiles(t, dir, map[string]string{"Holder.cfc": `component {function get(){return variables.value;}}`, "Service.cfc": `component {function go(){}}`})
	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}

	fd := &parser.FunctionDef{Name: "get", URI: cfpath.ToURI(file)}
	if got := r.lazyGetterReturn(fd); got != "" {
		t.Fatal(got)
	}

	writeFiles(t, dir, map[string]string{"Holder.cfc": `component {variables.value="";function load(){variables.value=new Service();}function get(){if(!isObject(variables.value)){load();}return variables.value;}}`})
	r.InvalidatePaths()

	if got := r.lazyGetterReturn(fd); got == "" {
		t.Fatal("cached absence of lazy source survived invalidation")
	}
}

func TestBlockArrowProviderCallback(t *testing.T) {
	for _, argument := range []string{`(x)=>{x.go();}`, `x=>{x.go();}`, `callback:(x)=>{x.go();}`} {
		t.Run(argument, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"Service.cfc": `component {function go(){}}`, "Provider.cfc": `component {function direct(callback){callback(new Service());}}`})
			r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
			src := `component {function run(){var p=new Provider();p.direct(` + argument + `);}}`
			pr := parser.ParseWithOptions(cfpath.ToURI(filepath.Join(dir, "Page.cfc")), src, &parser.ParseOptions{ExtractCalls: true})
			checked := 0

			for _, call := range pr.AllCalls() {
				if call.FuncName == "go" {
					checked++

					if why := r.CanResolveCall(&call, pr, dir); why != "" {
						t.Fatal(why)
					}
				}
			}

			if checked != 1 {
				t.Fatalf("checked %d", checked)
			}
		})
	}
}

func TestModuleMappingsUseOrderedPublicProperties(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"modules/users/ModuleConfig.cfc":    `component {this.modelNamespace="old";this["modelNamespace"]="current";this.cfmapping="legacy";this.classMapping="preferred";}`,
		"modules/users/models/Service.cfc":  `component {function go(){}}`,
		"modules/users/Util.cfc":            `component {function help(){}}`,
		"modules/disabled/ModuleConfig.cfc": `component {this.disabled=true;this.classMapping="disabledAlias";}`,
		"modules/disabled/Util.cfc":         `component {function help(){}}`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "modules/users/models/Service.cfc")

	for _, tc := range []struct{ id, path string }{
		{"Service@current", "modules/users/models/Service.cfc"},
		{"Service@old", ""},
		{"preferred.Util", "modules/users/Util.cfc"},
		{"legacy.Util", ""},
		{"disabledAlias.Util", ""},
	} {
		got := r.ComponentPath(tc.id, dir)

		want := ""
		if tc.path != "" {
			want = filepath.Join(dir, tc.path)
		}

		if got != want {
			t.Errorf("%s got %q want %q", tc.id, got, want)
		}
	}
}
