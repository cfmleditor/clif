package resolve

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

const (
	wheelsInlineRecord = `if (local.fns[local.f].access == "public") { ArrayAppend(local.publicMethods, {name = local.fns[local.f].name, ref = local.instance[local.fns[local.f].name]}); }`
	wheelsSplitRecord  = `if (local.fns[local.f].access == "public") { local.ref = local.instance[local.fns[local.f].name]; ArrayAppend(local.publicMethods, {name = local.fns[local.f].name, ref = local.ref}); }`
)

func TestWheelsInlineIntegrationRecord(t *testing.T) {
	for _, record := range []string{wheelsInlineRecord, wheelsSplitRecord} {
		dir := t.TempDir()
		files := controllerFiles()
		// Replace across formatting using the normalized builder fixture.
		files["wheels/global/objects.cfm"] = strings.Replace(files["wheels/global/objects.cfm"], mapperBuilder, strings.Replace(wheelsTokens(mapperBuilder), wheelsTokens(wheelsSplitRecord), wheelsTokens(record), 1), 1)
		writeFiles(t, dir, files)

		r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
		for _, tc := range []struct{ component, name string }{{"wheels.Mapper", "route"}, {"app.controllers.Posts", "redirectTo"}} {
			if d := r.ResolveFunc(tc.component, tc.name, dir); d == nil {
				t.Errorf("%s.%s missing", tc.component, tc.name)
			}
		}
	}
}

func modelIntegrationFiles() map[string]string {
	files := controllerFiles()
	files["wheels/Model.cfc"] = strings.ReplaceAll(strings.ReplaceAll(files["wheels/Controller.cfc"], `"controller"`, `"model"`), `$integrateComponents("wheels.controller"); $integrateComponents("wheels.view");`, `$integrateComponents("wheels.model");`)
	files["wheels/model/properties.cfc"] = `component { public function property() {} private function secret() {} }`
	files["wheels/model/read.cfc"] = `component { public function findOne() { property(); globalOwn(); } }`
	files["wheels/Global.cfc"] = strings.Replace(files["wheels/Global.cfc"], `include "global/objects.cfm";`, `include "global/objects.cfm"; public function globalOwn() {}`, 1)
	files["app/models/User.cfc"] = `component extends="wheels.Model" { public function property() {} function config() {findOne(); superProperty(); secret();} }`

	return files
}

func TestWheelsModelIntegration(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, modelIntegrationFiles())

	r := &Resolver{}
	expectReasons(t, reasonsWith(t, r, dir, "app/models/User.cfc"), map[string]string{"findOne": "", "superProperty": "", "secret": "not found in extends chain"})
	expectReasons(t, reasonsWith(t, r, dir, "wheels/model/read.cfc"), map[string]string{"property": "", "globalOwn": ""})

	d := r.ResolveFunc("app.models.User", "superProperty", dir)
	if d == nil || d.URI.Path() != filepath.Join(dir, "wheels", "model", "properties.cfc") {
		t.Fatalf("super source: %+v", d)
	}
}

func TestWheelsModelIntegrationBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, file, from, to string }{
		{"own init", "app/models/User.cfc", "function config()", "function init(){return this;} function config()"},
		{"own loader", "app/models/User.cfc", "function config()", "function $integrateComponents(path){} function config()"},
		{"unused integration", "wheels/Model.cfc", `$integrateComponents("wheels.model");`, ``},
		{"wrong policy", "wheels/Model.cfc", wheelsControllerCopy, ``},
		{"private provider", "wheels/model/read.cfc", "public function findOne", "private function findOne"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			f := modelIntegrationFiles()
			f[tc.file] = strings.Replace(f[tc.file], tc.from, tc.to, 1)
			writeFiles(t, dir, f)

			r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
			if d := r.ResolveFunc("app.models.User", "findOne", dir); d != nil {
				t.Fatalf("unproved loader: %+v", d)
			}
		})
	}
}

func TestWheelsInlineRecordBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, from, to string }{
		{"wrong instance", `local.instance[local.fns[local.f].name]`, `local.other[local.fns[local.f].name]`},
		{"wrong member", `ref = local.instance[local.fns[local.f].name]`, `ref = local.instance[local.fns[local.f+1].name]`},
		{"private filter", `access == "public"`, `access == "private"`},
		{"record mutation", `ArrayAppend(local.publicMethods`, `local.fns[local.f].name="other"; ArrayAppend(local.publicMethods`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			f := controllerFiles()
			body := strings.Replace(wheelsTokens(mapperBuilder), wheelsTokens(wheelsSplitRecord), wheelsTokens(wheelsInlineRecord), 1)
			// Mutate the admitted record itself, not another producer expression.
			record := strings.Replace(wheelsInlineRecord, tc.from, tc.to, 1)
			body = strings.Replace(body, wheelsTokens(wheelsInlineRecord), wheelsTokens(record), 1)
			f["wheels/global/objects.cfm"] = strings.Replace(f["wheels/global/objects.cfm"], mapperBuilder, body, 1)
			writeFiles(t, dir, f)

			r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
			if d := r.ResolveFunc("wheels.Mapper", "route", dir); d != nil {
				t.Fatalf("invalid record supplied method: %+v", d)
			}
		})
	}
}

func TestWheelsModelIntegrationSourceChanges(t *testing.T) {
	dir := t.TempDir()
	f := modelIntegrationFiles()
	writeFiles(t, dir, f)

	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	if r.ResolveFunc("wheels.Model", "findOne", dir) == nil {
		t.Fatal("missing initial method")
	}

	writeFiles(t, dir, map[string]string{"wheels/model/duplicate.cfc": `component {public function findOne(){}}`})
	r.InvalidatePaths()

	if d := r.ResolveFunc("wheels.Model", "findOne", dir); d != nil {
		t.Fatalf("ambiguous method: %+v", d)
	}

	f["wheels/model/read.cfc"] = `component {private function findOne(){}}`
	writeFiles(t, dir, map[string]string{"wheels/model/read.cfc": f["wheels/model/read.cfc"]})

	d := r.ResolveFunc("wheels.Model", "findOne", dir)
	if d == nil || d.URI.Path() != filepath.Join(dir, "wheels", "model", "duplicate.cfc") {
		t.Fatalf("changed access not observed: %+v", d)
	}
}

func TestWheelsModelProviderContextBoundaries(t *testing.T) {
	dir := t.TempDir()
	f := modelIntegrationFiles()
	f["wheels/model/read.cfc"] = `component {public function findOne(){} private function hidden(){property();} public function defaulted(x=globalOwn()){} }`
	writeFiles(t, dir, f)

	r := &Resolver{}
	expectReasons(t, reasonsWith(t, r, dir, "wheels/model/read.cfc"), map[string]string{"property": "no qualifier, not in file", "globalOwn": "no qualifier, not in file"})

	if d := r.ResolveFunc("wheels.model.read", "property", dir); d != nil {
		t.Fatalf("direct provider gained host method: %+v", d)
	}
}
