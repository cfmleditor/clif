package resolve

import (
	"slices"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	"github.com/cfmleditor/clif/internal/vfs"
)

// The reaching walk's braces are the script scanner's, whatever else the
// script holds: a brace in a comment, in a string or in a quoted argument
// inside a string's #...# opens and closes nothing. A function body is a
// slice of its file, and one that writes "<script>" used to be read with the
// tag rule, which ends "#f("}")#" at its inner quote and closed a block there.
func TestBraceEventsReadOnlyTheScriptsBraces(t *testing.T) {
	const body = "function run( flag ) {\n" +
		"\tvar open = \"<script>\";\n" +
		"\tif ( flag ) { // a } in a comment\n" +
		"\t\tx = \"#f( \"}\" )#\";\n" +
		"\t\t/* { */\n" +
		"\t\ty = '<!--- { --->';\n" +
		"\t}\n" +
		"}\n"

	for _, tc := range []struct {
		name, text string
		tag        bool
	}{
		{"script", body, false},
		{"cfscript block", "<cfset a = 1>\n<cfscript>\n" + body + "</cfscript>\n<style>p { color: red; }</style>\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var opens, closes []string

			for _, ev := range blockEvents(tc.text, tc.tag) {
				line := tc.text[ev.offset:]
				line = line[:strings.IndexByte(line, '\n')]

				switch {
				case ev.op == opOpen && ev.kind == "{":
					opens = append(opens, strings.TrimSpace(line))
				case ev.op == opClose && ev.kind == "":
					closes = append(closes, strings.TrimSpace(line))
				}
			}

			if !slices.Equal(opens, []string{"{", "{ // a } in a comment"}) || !slices.Equal(closes, []string{"}", "}"}) {
				t.Errorf("opened at %q and closed at %q, want the function's and the if's braces alone", opens, closes)
			}
		})
	}
}

// Whether a scope's blocks are tags or braces is the parse's region split:
// a tag component's pseudo-constructor is tags, so a struct literal in a
// <cfset> opens no block, and a function in a template's <cfscript> is
// script.
func TestTagBlocksFollowsTheRegionSplit(t *testing.T) {
	tagComponent := parser.Parse("file:///ws/Tags.cfc", "<cfcomponent>\n<cfset x = { a = 1 }>\n<cfset y = 2>\n</cfcomponent>\n")
	if !tagBlocks(tagComponent, 1) {
		t.Error("a tag component's pseudo-constructor was read as script")
	}

	template := parser.Parse("file:///ws/page.cfm", "<p>Hi</p>\n<cfscript>\nfunction f() {\n\tif ( a ) { x = 1; }\n}\n</cfscript>\n<cfif a></cfif>\n")
	if tagBlocks(template, 2) {
		t.Error("a function in a template's <cfscript> was read as tags")
	}

	if !tagBlocks(template, 6) {
		t.Error("a template's markup after its <cfscript> was read as script")
	}

	script := parser.Parse("file:///ws/Script.cfc", "component {\n\tfunction f() {\n\t\treturn '<cffunction>';\n\t}\n}\n")
	if tagBlocks(script, 1) {
		t.Error("a script component's function was read as tags")
	}
}

// An assignment runs on to its next line while a bracket it opened is open,
// and a bracket is code: one in a string, in a quoted argument inside a
// string's #...# or in a comment opens nothing. A real one still does.
func TestAnAssignmentJoinsItsNextLineOnlyForAnOpenBracket(t *testing.T) {
	for rhs, more := range map[string]bool{
		`f( ")" )`:                false,
		`f( "{" )`:                false,
		`"#g( "[" )#"`:            false,
		`f() /* ( */`:             false,
		`f( "x" ).g(`:             true,
		`{ a: "}"`:                true,
		`[ "#g( "]" )#"`:          true,
		`f( ")", /* ) */ "(" ) (`: true,
	} {
		if _, got := joinAssignmentLine(rhs, "next()", false); got != more {
			t.Errorf("%s: joined %v, want %v", rhs, got, more)
		}
	}

	const content = "component {\n\tfunction f() {\n\t\tvar x = new A( \"#g( \"(\" )#\" );\n\t\tx.go();\n\t}\n}\n"

	if rhs, _, ok := contentFacts(content).assignmentAt(content, "x", 1, 3); !ok || rhs != `new A( "#g( "(" )#" )` {
		t.Errorf("assignment = %q, %v; want the call, whose parentheses balance", rhs, ok)
	}
}

// A guard controls the call while the block it opens is open, and the braces
// and semicolons that say so are code: one in a string or a comment between
// the guard and the call ends nothing. A real brace still does.
func TestAGuardsBlockEndsAtARealBrace(t *testing.T) {
	const guarded = "component {\n" +
		"\tfunction f() {\n" +
		"\t\tif ( structKeyExists( variables.cfg, \"onX\" ) ) {\n" +
		"\t\t\tlog( \"done; }\" ); // }\n" +
		"\t\t\t/* } */ log( \"#g( \"}\" )#\" );\n" +
		"\t\t\tvariables.cfg.onX();\n" +
		"\t\t}\n" +
		"\t}\n" +
		"}\n"
	if !(&Resolver{FS: vfs.OS{}}).guardedByExistsCheck(parser.Parse("file:///ws/G.cfc", guarded), "variables.cfg", "onX", 5) {
		t.Error("a brace or semicolon in a string or comment ended the guarded block")
	}

	const after = "component {\n" +
		"\tfunction f() {\n" +
		"\t\tif ( structKeyExists( variables.cfg, \"onX\" ) ) {\n" +
		"\t\t\tlog( \"{\" );\n" +
		"\t\t}\n" +
		"\t\tvariables.cfg.onX();\n" +
		"\t}\n" +
		"}\n"
	if (&Resolver{FS: vfs.OS{}}).guardedByExistsCheck(parser.Parse("file:///ws/G.cfc", after), "variables.cfg", "onX", 5) {
		t.Error("a call after the guarded block's real close was taken as guarded")
	}
}

// A guard is a call the parse reads: one written in a tag function's markup,
// outside any output, checks nothing, and the call after it is not guarded.
// The token walk read the markup as code and took its brace for a block.
func TestAGuardInMarkupGuardsNothing(t *testing.T) {
	const markup = "<cfcomponent>\n" +
		"<cffunction name=\"f\">\n" +
		"\t<p>if ( structKeyExists( variables.cfg, \"onX\" ) || isInstanceOf( variables.cfg, \"a.B\" ) ) {</p>\n" +
		"\t<cfset variables.cfg.onX()>\n" +
		"</cffunction>\n" +
		"</cfcomponent>\n"

	r := &Resolver{FS: vfs.OS{}}
	pr := parser.Parse("file:///ws/G.cfc", markup)

	if r.guardedByExistsCheck(pr, "variables.cfg", "onX", 3) {
		t.Error("a structKeyExists written in markup guarded the call")
	}

	if got := r.guardedInstanceOf(pr, "variables.cfg", "onX", 3); len(got) > 0 {
		t.Errorf("an isInstanceOf written in markup guarded the call: %v", got)
	}

	const tagGuard = "<cfcomponent>\n" +
		"<cffunction name=\"f\">\n" +
		"\t<cfif structKeyExists( variables.cfg, \"onX\" )>\n" +
		"\t\t<cfset variables.cfg.onX()>\n" +
		"\t</cfif>\n" +
		"</cffunction>\n" +
		"</cfcomponent>\n"
	if !r.guardedByExistsCheck(parser.Parse("file:///ws/G.cfc", tagGuard), "variables.cfg", "onX", 3) {
		t.Error("a <cfif structKeyExists(...)> did not guard the call it holds")
	}

	const page = "<p>isNull( driver.getBundleName )</p>\n" +
		"<cfset x = driver.getBundleName()>\n"
	if r.methodExistenceGuarded(parser.Parse("file:///ws/p.cfm", page), &parser.CallSite{Line: 1}, "getBundleName") {
		t.Error("an isNull written in markup guarded the call")
	}
}
