package resolve

import "testing"

// The application scope of a component belongs to the application its request
// runs in, and a file that names no application does not start one. Galaxie
// Blog's org/camden/blog/Application.cfm only aborts a request for its ini file,
// and tags/Application.cfm is a bare <cfabort>; the root Application.cfc is
// where application.blog is set. The lookup took the nearest Application file
// and stopped there, so every application.blog in blog.cfc had no component ref
// (100 findings). A file that names an application, with this.name or
// <cfapplication name>, is its own application and still ends the walk.
func TestAnApplicationFileThatNamesNoApplicationDoesNotEndTheLookup(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Tool.cfc": `component { function go(){} }`,
		"Application.cfc": `<cfcomponent>
<cffunction name="onApplicationStart">
	<cfset application.tool = new Tool()>
	<cfset application.dyn = createObject("component", "#application.dynPath#")>
</cffunction>
</cfcomponent>`,

		"guard/Application.cfm":        `<cfif cgi.script_name is "x.ini">` + "\n" + `<cfabort></cfif>`,
		"guard/Caller.cfc":             `<cfcomponent><cffunction name="f"><cfset application.tool.go()><cfset application.tool.missing()><cfset application.dyn.anything()></cffunction></cfcomponent>`,
		"guard/deeper/Caller.cfc":      `<cfcomponent><cffunction name="f"><cfset application.tool.go()></cffunction></cfcomponent>`,
		"guard/deeper/Application.cfm": `<cfabort>`,

		"named/Application.cfm": `<cfapplication name="other" sessionmanagement="true">`,
		"named/Caller.cfc":      `<cfcomponent><cffunction name="f"><cfset application.tool.go()></cffunction></cfcomponent>`,

		"scripted/Application.cfc": `component { this.name = "scripted"; }`,
		"scripted/Caller.cfc":      `component { function f(){ application.tool.go(); } }`,
	})

	// A guard file: the root's refs are in force.
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "guard/Caller.cfc"), map[string]string{
		"application.tool.go":      "",
		"application.tool.missing": "method 'missing' not found in Tool",
		"application.dyn.anything": "",
	})

	// Two guard files in a row.
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "guard/deeper/Caller.cfc"), map[string]string{
		"application.tool.go": "",
	})

	// A file that names an application is its own application.
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "named/Caller.cfc"), map[string]string{
		"application.tool.go": "variable 'application.tool' has no component ref",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "scripted/Caller.cfc"), map[string]string{
		"application.tool.go": "variable 'application.tool' has no component ref",
	})
}
