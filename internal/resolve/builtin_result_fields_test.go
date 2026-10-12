package resolve

import (
	"strings"
	"testing"
)

func TestInternalRequestTextField(t *testing.T) {
	for _, producer := range []string{"InternalRequest", "_InternalRequest"} {
		for _, decl := range []string{"var result", "local.result"} {
			t.Run(producer+decl, func(t *testing.T) {
				dir := t.TempDir()
				writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 ` + decl + ` = ` + producer + `(
 template:"/test.cfm"
 );
 result.filecontent.trim();
 local.result.filecontent.toString();
 result.filecontent.absent();
 }}`})

				got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
				for _, name := range []string{"result.filecontent.trim", "local.result.filecontent.toString"} {
					if reason, ok := got[name]; !ok || reason != "" {
						t.Errorf("%s: %q present=%v", name, reason, ok)
					}
				}

				if !strings.Contains(got["result.filecontent.absent"], "native.string") {
					t.Errorf("missing field identity: %v", got)
				}
			})
		}
	}
}

func TestInternalRequestFieldBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, pre, init, after, engine string }{
		{"header write", "", "var result = InternalRequest();", "result.filecontent=unknown(); result.filecontent.trim();", "lucee"},
		{"sibling closure", "var first=function(){", "var result = InternalRequest();", "};var second=function(){", "lucee"},
		{"earlier capture", "var change=function(){result.filecontent=unknown();};", "var result = InternalRequest();", "change();", "lucee"},
		{"portable", "", "var result = InternalRequest();", "", ""},
		{"bare", "", "result = InternalRequest();", "", "lucee"},
		{"shared", "", "variables.result = InternalRequest();", "", "lucee"},
		{"qualified", "", "var result = service.InternalRequest();", "", "lucee"},
		{"closed branch", "if(flag){", "var result = InternalRequest();", "}", "lucee"},
		{"unbraced", "if(flag)", "var result = InternalRequest();", "", "lucee"},
		{"rebind", "", "var result = InternalRequest();", "result = unknown();", "lucee"},
		{"overwrite", "", "var result = InternalRequest();", "result.filecontent = unknown();", "lucee"},
		{"increment", "", "var result = InternalRequest();", "result.filecontent++;", "lucee"},
		{"escape", "", "var result = InternalRequest();", "mutate(result);", "lucee"},
		{"alias", "", "var result = InternalRequest();", "var alias = result;", "lucee"},
		{"dynamic write", "", "var result = InternalRequest();", "result[field] = value;", "lucee"},
		{"whole mutation", "", "var result = InternalRequest();", "result.clear();", "lucee"},
		{"closure", "", "var result = InternalRequest();", "var closure=function(){result.filecontent=unknown();};", "lucee"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"Use.cfc": "component {function run(){\n" + tc.pre + "\n" + tc.init + "\n" + tc.after + "\n result.filecontent.trim();\n}}"})

			got := reasonsWith(t, &Resolver{NativeEngine: tc.engine}, dir, "Use.cfc")
			if reason, ok := got["result.filecontent.trim"]; !ok || reason == "" {
				t.Errorf("unsafe result resolved: %v", got)
			}
		})
	}
}

func TestInternalRequestShadowed(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {
 function InternalRequest(){return {};}
 function run(){
 var result = InternalRequest();
 result.filecontent.trim();
 }}`})

	got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
	if got["result.filecontent.trim"] == "" {
		t.Errorf("custom producer resolved: %v", got)
	}
}

func TestInternalRequestInheritedAndArgumentShadow(t *testing.T) {
	for _, declaration := range []string{"function run(any InternalRequest){", "function run(){\n var InternalRequest=function(){return {};};"} {
		dir := t.TempDir()
		writeFiles(t, dir, map[string]string{"Use.cfc": "component {" + declaration + "\n var result=InternalRequest();\n result.filecontent.trim();\n}}"})

		got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
		if got["result.filecontent.trim"] == "" {
			t.Errorf("shadowed builtin resolved: %v", got)
		}
	}

	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Base.cfc": "component {function InternalRequest(){return {};}}", "Use.cfc": `component extends="Base" {function run(){
 var result=InternalRequest();
 result.filecontent.trim();
 }}`})

	got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
	if got["result.filecontent.trim"] == "" {
		t.Errorf("inherited producer resolved: %v", got)
	}
}

func TestInternalRequestLaterCapture(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 var result=InternalRequest();
 change();
 result.filecontent.trim();
 var change=function(){result.filecontent=unknown();};
 }}`})

	got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
	if got["result.filecontent.trim"] == "" {
		t.Errorf("captured producer resolved: %v", got)
	}
}

func TestInternalRequestIndependentCallbacks(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 it("one",function(){
 var result=InternalRequest();
 result.filecontent.trim();
 });
 it("two",function(){
 var result=InternalRequest();
 result.filecontent.toString();
 });
 }}`})

	got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
	for _, name := range []string{"result.filecontent.trim", "result.filecontent.toString"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("independent owner: %s=%q present=%v", name, reason, ok)
		}
	}
}

func TestInternalRequestFreshLocalLifetimes(t *testing.T) {
	for _, before := range []string{
		"result.filecontent.trim();",
		"var old = result; mutate(old);",
		"mutate(result);",
		"result.filecontent=unknown();",
		"result.status_code=unknown();",
	} {
		t.Run(before, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"Use.cfc": "component {function run(){\n local.result=InternalRequest();\n" + before + "\n local.result=InternalRequest();\n result.filecontent.trim();\n}}"})

			got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
			if reason, ok := got["result.filecontent.trim"]; !ok || reason != "" {
				t.Errorf("fresh local result: %q present=%v", reason, ok)
			}
		})
	}
}

func TestInternalRequestScalarReads(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 var result=InternalRequest();
 assertEquals(200,result.status_code);
 if(result.status_code==200){
 result.filecontent.trim();
 }
 }}`})

	got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
	if reason, ok := got["result.filecontent.trim"]; !ok || reason != "" {
		t.Errorf("scalar read: %q present=%v", reason, ok)
	}
}

func TestInternalRequestNamedCapture(t *testing.T) {
	for _, declaration := range []string{"function change(){result.filecontent=unknown();}", "function change(){mutate(result);}"} {
		dir := t.TempDir()
		writeFiles(t, dir, map[string]string{"Use.cfc": "component {function run(){\n var result=InternalRequest();\n change();\n result.filecontent.trim();\n" + declaration + "\n}}"})

		got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
		if got["result.filecontent.trim"] == "" {
			t.Errorf("named capture acquired string: %v", got)
		}
	}
}
