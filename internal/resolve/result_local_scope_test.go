package resolve

import "testing"

func TestInternalRequestLiteralLocalMode(t *testing.T) {
	for _, mode := range []string{`true`, `"modern"`, `'true'`, `"MODERN"`} {
		dir := t.TempDir()
		writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run() localmode=` + mode + ` skip=true {
 result=InternalRequest();
 result.filecontent.trim();
 }}`})

		got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
		if reason, ok := got["result.filecontent.trim"]; !ok || reason != "" {
			t.Errorf("modern %s: %q present=%v", mode, reason, ok)
		}
	}
}

func TestInternalRequestDominatingLocal(t *testing.T) {
	for _, declaration := range []string{"var result = unknown();", "local.result = unknown();"} {
		dir := t.TempDir()
		writeFiles(t, dir, map[string]string{"Use.cfc": "component {function run(){\n" + declaration + "\n result=InternalRequest();\n result.filecontent.trim();\n}}"})

		got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
		if reason, ok := got["result.filecontent.trim"]; !ok || reason != "" {
			t.Errorf("local store: %q present=%v", reason, ok)
		}
	}
}

func TestInternalRequestLocalScopeProofBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, header, pre, init string }{
		{"classic", `function run() localmode="classic" {`, "", "result=InternalRequest();"},
		{"false", `function run() localmode=false {`, "", "result=InternalRequest();"},
		{"bare modern", `function run() localmode=modern {`, "", "result=InternalRequest();"},
		{"dynamic", `function run() localmode=mode {`, "", "result=InternalRequest();"},
		{"expression", `function run() localmode="modern" & suffix {`, "", "result=InternalRequest();"},
		{"duplicate", `function run() localmode=true localmode=false {`, "", "result=InternalRequest();"},
		{"qualified", `function run() localmode=true {`, "", "variables.result=InternalRequest();"},
		{"closed local", `function run(){`, "if(flag){var result=unknown();}", "result=InternalRequest();"},
		{"cleared local", `function run(){`, "var result=unknown();structClear(local);", "result=InternalRequest();"},
		{"deleted local", `function run(){`, "var result=unknown();local.delete('result');", "result=InternalRequest();"},
		{"escaped local", `function run(){`, "var result=unknown();consume(local);", "result=InternalRequest();"},
		{"attributed capture", `function run() localmode=true {`, "function writer() localmode=true {result.filecontent=unknown;}", "result=InternalRequest();"},
		{"unbraced local", `function run(){`, "if(flag) var result=unknown();", "result=InternalRequest();"},
		{"sibling local", `function run(){`, "var first=function(){var result=unknown();};", "result=InternalRequest();"},
		{"callback mode", `function run() localmode=true {`, "var cb=function(){", "result=InternalRequest();"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"Use.cfc": "component {" + tc.header + "\n" + tc.pre + "\n" + tc.init + "\n result.filecontent.trim();\n}}"})

			got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
			if reason, ok := got["result.filecontent.trim"]; !ok || reason == "" {
				t.Errorf("unproven store: %q present=%v", reason, ok)
			}
		})
	}
}
