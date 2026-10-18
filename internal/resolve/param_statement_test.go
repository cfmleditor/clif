package resolve

import "testing"

// TestAParamStatementDoesNotInvalidateAComponentContract reproduces TestBox's
// BaseReporter: getConsoleUtil() lazily builds variables.consoleUtils under a
// structKeyExists guard, so the resolver types its return as that component —
// but the whole-file producer plan must stay readable for the contract to hold.
// A `param x = …;` in an unrelated function is a presence default for x alone,
// never a write the guarded field had to worry about, and the scanner's
// inability to read it was rejecting the whole component's plan.
func TestAParamStatementDoesNotInvalidateAComponentContract(t *testing.T) {
	for tc, other := range map[string]string{
		"with default":  "function prepare(){ param url = {}; }",
		"no default":    "function prepare(){ param url; }",
		"dotted target": "function prepare(){ param url.page = 1; }",
		"two params":    "function prepare(){ param url = {}; param url.page = 1; }",
	} {
		t.Run(tc, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"ConsoleUtil.cfc": `component { function getBanner(){} }`,
				"Page.cfc": `component {
	function getConsoleUtil(){
		if ( !structKeyExists( variables, "consoleUtils" ) ) {
			variables.consoleUtils = new ConsoleUtil();
		}
		return variables.consoleUtils;
	}

	function getColor(){
		return getConsoleUtil().getBanner();
	}

	` + other + `
}`,
			})

			expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Page.cfc"), map[string]string{
				"getConsoleUtil.getBanner": "",
			})
		})
	}
}
