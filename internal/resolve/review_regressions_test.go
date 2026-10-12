package resolve

import "testing"

func TestUserInvokeRetainsCallerInference(t *testing.T) {
	for _, invocation := range []string{`invoke(new Item());`, `this.invoke(new Item());`} {
		t.Run(invocation, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"Item.cfc": `component {function ready(){}}`,
				"Svc.cfc": `component {
function invoke(thing){arguments.thing.ready();}
function run(){` + invocation + `}
}`,
			})

			got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Svc.cfc")
			expectReasons(t, got, map[string]string{"arguments.thing.ready": ""})
		})
	}
}

func TestInheritedBuiltinOverrideHasNoNativeReturn(t *testing.T) {
	for _, tc := range []struct {
		name, base string
		resolved   bool
	}{
		{"override", `component {function listToArray(){return "plain text";}}`, false},
		{"builtin", `component {}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"Base.cfc": tc.base,
				"Use.cfc": `component extends="Base" {function run(){
var parts=listToArray("a,b");
parts.getDimension();
}}`,
			})

			got := builtinProducerReasons(t, dir)
			if reason, ok := got["parts.getDimension"]; !ok || (reason == "") != tc.resolved {
				t.Fatalf("resolved=%v: %v", tc.resolved, got)
			}
		})
	}
}
