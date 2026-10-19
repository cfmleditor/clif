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

// The attribute form names its variable in name=: `param name="variables.x"
// default="";` defaults variables.x, not a variable called name. Read as the
// alias form, its target was the attribute keyword, so a real default on the
// guarded field was never seen and getConsoleUtil() kept a return type the
// field cannot promise. A param on another variable still leaves it alone.
func TestAParamStatementsAttributeFormNamesItsTarget(t *testing.T) {
	for tc, c := range map[string]struct {
		other  string
		listed bool
	}{
		"defaults the guarded field":   {`function prepare(){ param name="variables.consoleUtils" default=""; }`, true},
		"type before name":             {`function prepare(){ param type="any" name="variables.consoleUtils" default=""; }`, true},
		"names another variable":       {`function prepare(){ param name="url.page" default="1"; }`, false},
		"alias form on another target": {`function prepare(){ param name = 1; }`, false},
		"alias form on the field":      {`function prepare(){ param variables.consoleUtils = ""; }`, true},
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

	` + c.other + `
}`,
			})

			got := reasonsWith(t, &Resolver{}, dir, "Page.cfc")["getConsoleUtil.getBanner"]
			if c.listed && got == "" {
				t.Error("getConsoleUtil().getBanner() accepted, though a param can default the field to a string")
			}

			if !c.listed && got != "" {
				t.Errorf("getConsoleUtil().getBanner(): %q, want it accepted", got)
			}
		})
	}
}
