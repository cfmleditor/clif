package resolve

import "testing"

// TestASuiteIncludingEveryListedSpecSeesTheSuite: spreadsheet-cfml's
// test/suite.cfc lists specs/*.cfm and runs `include "specs/#file#";` for each
// name, so every spec runs in the suite's scope: it calls the suite's helpers
// bare and its `s` is the suite's variables.s. The include names no extension,
// so it is read only beside a directoryList of that directory for templates.
func TestASuiteIncludingEveryListedSpecSeesTheSuite(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Util.cfc": `component { function read(){} }`,
		"test/suite.cfc": `component {
	function helper(){}
	variables.s = new Util();
	function run(){
		var specs = DirectoryList( ExpandPath( "specs" ), false, "name", "*.cfm" );
		for( var file in specs ){
			include "specs/#file#";
		}
	}
}`,
		"test/specs/read.cfm": `<cfscript>helper(); s.read(); s.missing();</cfscript>`,
		"other/suite.cfc": `component {
	function otherHelper(){}
	function run(){
		var specs = DirectoryList( ExpandPath( "specs" ), true, "name", "*.cfm" );
		for( var file in specs ){ include "specs/#file#"; }
	}
}`,
		"other/specs/read.cfm": `<cfscript>otherHelper();</cfscript>`,
	})

	expectReasons(t, templateReasons(t, dir, "test/specs/read.cfm"), map[string]string{
		"helper":    "",
		"s.read":    "",
		"s.missing": "method 'missing' not found in Util",
	})

	// A recursive listing may name a file below the directory: not read.
	expectReasons(t, templateReasons(t, dir, "other/specs/read.cfm"), map[string]string{
		"otherHelper": "no qualifier, not in file",
	})
}

// TestATemplateACustomTagIncludesByAttributeSeesTheTag: RustCFML's runner
// calls `<cf_runtest file="core/test_x.cfm">` for each test, and runtest.cfm
// includes harness.cfm and then `#attributes.file#`, so each test calls the
// harness's assert() bare. A template no call passes is not included, and a
// call passing the file computed names nothing.
func TestATemplateACustomTagIncludesByAttributeSeesTheTag(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"tests/runtest.cfm": `<cfif thisTag.executionMode eq "start">
	<cfinclude template="harness.cfm">
	<cfinclude template="#attributes.file#">
</cfif>`,
		"tests/harness.cfm":        `<cfscript>function assert(x){}</cfscript>`,
		"tests/runner.cfm":         `<cf_runtest file="core/test_a.cfm"><cf_runtest file="#dynamic#">`,
		"tests/core/test_a.cfm":    `<cfscript>assert(1); notDeclared();</cfscript>`,
		"tests/core/test_b.cfm":    `<cfscript>assert(1);</cfscript>`,
		"tests/core/notTheTag.cfm": `<cfinclude template="#attributes.file#">`,
	})

	r := &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}
	expectReasons(t, templateReasonsWith(t, r, dir, "tests/core/test_a.cfm"), map[string]string{
		"assert":      "",
		"notDeclared": "no qualifier, not in file",
	})

	expectReasons(t, templateReasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "tests/core/test_b.cfm"), map[string]string{
		"assert": "no qualifier, not in file",
	})
}
