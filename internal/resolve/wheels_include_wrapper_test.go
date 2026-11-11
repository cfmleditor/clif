package resolve

import (
	"strings"
	"testing"
)

const wheelsWrapperGlobal = `component {
	public void function $include(required string template) {
		$tryIncludeTemplate($resolveGlobalIncludeTemplate(arguments.template));
	}
	public void function $includeAndOutput(required string template) {
		$tryIncludeTemplate($resolveGlobalIncludeTemplate(arguments.template));
	}
	public string function $resolveGlobalIncludeTemplate(required string template) {
		var normalized = Replace(arguments.template, "\", "/", "all");
		if (!Len(normalized)) {
			return normalized;
		}
		if (Left(normalized, 1) == "/") {
			return LCase(normalized);
		}
		return normalized;
	}
	public void function $tryIncludeTemplate(required string template) {
		var resolved = arguments.template;
		var state = {done = false};
		try {
			include "#resolved#";
			state.done = true;
		} catch (any e) {
			rethrow;
		}
	}
	public string function urlFor() { return ""; }
}`

// TestATemplateAWheelsWrapperIncludesIsItsIncluders: public/Application.cfc
// runs `application.wo.$includeAndOutput( template = "/wheels/events/debug.cfm" )`,
// so debug.cfm runs in Global and calls its methods bare. A wrapper whose body
// is not the pinned one, or a computed template, proves nothing.
func TestATemplateAWheelsWrapperIncludesIsItsIncluders(t *testing.T) {
	files := map[string]string{
		"Application.cfc":         `component { this.mappings["/wheels"] = getDirectoryFromPath(getCurrentTemplatePath()) & "wheels/"; }`,
		"wheels/Global.cfc":       wheelsWrapperGlobal,
		"wheels/events/debug.cfm": `<cfoutput>#urlFor()# #missing()#</cfoutput>`,
		"wheels/events/other.cfm": `<cfoutput>#urlFor()#</cfoutput>`,
		"public/Page.cfm":         `<cfscript>application.wo.$includeAndOutput(template = "/wheels/events/debug.cfm"); application.wo.$includeAndOutput(template = "/wheels/events/#x#.cfm");</cfscript>`,
	}

	dir := t.TempDir()
	writeFiles(t, dir, files)
	expectReasons(t, reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "wheels/events/debug.cfm"), map[string]string{
		"urlFor":  "",
		"missing": "no qualifier, not in file",
	})

	if got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "wheels/events/other.cfm")["urlFor"]; got == "" {
		t.Errorf("other.cfm: resolved through a computed template")
	}

	files["wheels/Global.cfc"] = strings.Replace(wheelsWrapperGlobal, `include "#resolved#";`, ``, 1)

	dir = t.TempDir()
	writeFiles(t, dir, files)

	if got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "wheels/events/debug.cfm")["urlFor"]; got == "" {
		t.Errorf("debug.cfm: resolved through a wrapper that includes nothing")
	}
}

// Older Wheels includes the template in the wrapper itself,
// `include "#LCase(arguments.template)#"`, which matches no pinned body; the
// parser's include-wrapper fact proves it. A wrapper that includes some other
// argument, or one that is not first, proves nothing for a template= call.
func TestAnOlderWheelsWrapperIsProvenByItsOwnInclude(t *testing.T) {
	global := func(wrapper string) string {
		return `component {
	` + wrapper + `
	public string function urlFor() { return ""; }
}`
	}

	for _, tc := range []struct {
		name, wrapper string
		proven        bool
	}{
		{"its own include", `public void function $include(required string template) { include "#LCase(arguments.template)#"; }`, true},
		{"a bare argument", `public void function $include(required string template) { include "#arguments.template#"; }`, true},
		{"another argument", `public void function $include(required string template, string other = "") { include "#arguments.other#"; }`, false},
		{"not first", `public void function $include(string other = "", required string template) { include "#arguments.template#"; }`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{
				"Application.cfc":         `component { this.mappings["/wheels"] = getDirectoryFromPath(getCurrentTemplatePath()) & "wheels/"; }`,
				"wheels/Global.cfc":       global(tc.wrapper),
				"wheels/events/debug.cfm": `<cfoutput>#urlFor()#</cfoutput>`,
				"public/Page.cfm":         `<cfscript>application.wo.$include(template = "/wheels/events/debug.cfm");</cfscript>`,
			}

			dir := t.TempDir()
			writeFiles(t, dir, files)

			got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "wheels/events/debug.cfm")["urlFor"]
			if (got == "") != tc.proven {
				t.Errorf("urlFor: %q, proven %v", got, tc.proven)
			}
		})
	}
}
