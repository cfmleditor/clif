package resolve

import (
	"strings"
	"testing"
)

func wheelsModelFiles() map[string]string {
	files := wheelsFactoryFiles()
	files["wheels/WheelsTest.cfc"] = `component {` + wheelsTestInlineBinder + `}`
	files["wheels/global/objects.cfm"] += `<cfscript>
 public any function model(required string name){` + wheelsModelFactory + `}
 public any function $cachedModelLookup(required string name){` + wheelsCachedModelLookup + `}
 public any function $createModelClass(required string name,string modelPaths=application.wheels.modelPath,string type="model"){` + wheelsCreateModelClass + `}
 </cfscript>`
	files["wheels/events/init/views.cfm"] = `<cfscript>application.wheels.modelPath="/app/models";</cfscript>`
	files["wheels/tests/runner.cfm"] = `<cfscript>local.AssetPath="/wheels/tests/_assets/";application.wo.set(modelPath=local.AssetPath & "models");</cfscript>`
	files["app/models/Post.cfc"] = `component {function appOnly(){}}`
	files["wheels/tests/_assets/models/Post.cfc"] = `component {function testOnly(){} function findOne(){return this;}}`
	files["wheels/tests/_assets/models/Model.cfc"] = `component {function base(){}}`
	files["wheels/tests/Spec.cfc"] = `component extends="wheels.WheelsTest" {
 function run(){
 var p=model("Post");p.testOnly();p.appOnly();p.missing();
 model(name="Post").findOne().testOnly();
 var b=model("Missing");b.base();
 var u=model(runtimeName);u.testOnly();
 }
 }`

	return files
}

func TestWheelsSourceModelFactory(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, wheelsModelFiles())
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "wheels/tests/Spec.cfc"), map[string]string{
		"p.testOnly": "", "p.appOnly": "method 'appOnly' not found in Post", "p.missing": "method 'missing' not found in Post", "model.findOne.testOnly": "", "b.base": "", "u.testOnly": "variable 'u' has no component ref",
	})
}

func TestWheelsSourceModelFactoryBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, file, from, to string }{
		{"overridden loader", "wheels/tests/Spec.cfc", `function run(){`, `function $createModelClass(){return other;} function run(){`},
		{"changed loader", "wheels/global/objects.cfm", `method = "$initModelClass"`, `method = selected`},
		{"changed cache", "wheels/global/objects.cfm", `return application.wheels.models[arguments.name];`, `return other;`},
		{"changed factory", "wheels/global/objects.cfm", `execute = "$createModelClass"`, `execute = "$createOtherClass"`},
		{"changed root factory", "wheels/global/objects.cfm", `local.rv = Invoke(local.instance, local.method, local.argumentCollection);`, `local.rv = incoming;`},
		{"changed default", "wheels/global/objects.cfm", `modelPaths=application.wheels.modelPath`, `modelPaths=other`},
		{"computed path", "wheels/tests/runner.cfm", `modelPath=local.AssetPath & "models"`, `modelPath=runtimePath`},
		{"restored path", "wheels/tests/runner.cfm", `</cfscript>`, `application.wheels.modelPath=saved;</cfscript>`},
		{"missing asset", "wheels/tests/runner.cfm", `"/wheels/tests/_assets/"`, `"/uninstalled/"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			files := wheelsModelFiles()
			files[tc.file] = strings.Replace(files[tc.file], tc.from, tc.to, 1)
			writeFiles(t, dir, files)
			expectReasons(t, reasonsWith(t, &Resolver{}, dir, "wheels/tests/Spec.cfc"), map[string]string{"p.testOnly": "variable 'p' has no component ref"})
		})
	}
}

func TestWheelsLegacyRootFactory(t *testing.T) {
	dir := t.TempDir()
	files := wheelsModelFiles()
	files["wheels/global/objects.cfm"] = strings.Replace(files["wheels/global/objects.cfm"], wheelsRootFactory, wheelsLegacyRootFactory, 1)
	writeFiles(t, dir, files)
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "wheels/tests/Spec.cfc"), map[string]string{"p.testOnly": "", "model.findOne.testOnly": ""})
}

func TestWheelsLargeReadOnlyPathSetting(t *testing.T) {
	for _, write := range []bool{false, true} {
		t.Run(map[bool]string{false: "read", true: "write"}[write], func(t *testing.T) {
			dir := t.TempDir()
			files := wheelsModelFiles()
			tail := `var path=application.wheels.modelPath;`
			expected := ""

			if write {
				tail = `application.wheels.modelPath=runtimePath;`
				expected = "variable 'p' has no component ref"
			}

			files["wheels/tests/Large.cfc"] = `component {function f(){` + strings.Repeat(`var x=1;`, 900) + tail + `}}`
			writeFiles(t, dir, files)
			expectReasons(t, reasonsWith(t, &Resolver{}, dir, "wheels/tests/Spec.cfc"), map[string]string{"p.testOnly": expected})
		})
	}
}
