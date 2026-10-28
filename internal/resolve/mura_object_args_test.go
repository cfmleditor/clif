package resolve

import "testing"

// TestAMuraDisplayObjectReadsWhatItsIncludeIsPassed: Mura's display objects
// are included by `$.dspObject_include( theFile = "x/y.cfm", iterator = it )`,
// which includes the template inside the function, so the template's
// arguments.iterator, and a bare iterator it never sets, are what each call
// passes. Every call naming the template and passing the name types it where
// the call is made; a computed theFile names every template it could, and a
// call passing an argumentCollection, which could hold the name, leaves it
// untyped. A call that omits the name says nothing about it, nor does one
// naming its file by position, which may never include the template; a name
// the template sets itself is its own.
func TestAMuraDisplayObjectReadsWhatItsIncludeIsPassed(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"mura/content/contentRenderer.cfc": `<cfcomponent>
<cffunction name="dspObject_Include" output="false">
	<cfargument name="theFile" type="string" />
	<cfinclude template="#arguments.theFile#" />
</cffunction>
</cfcomponent>`,
		"mura/content/contentIterator.cfc":  `component { function hasNext() {} function next() {} }`,
		"mura/content/calendarIterator.cfc": `component { function hasNext() {} function getDay() {} }`,
		"mura/user/userIterator.cfc":        `component { function hasNext() {} }`,
		"modules/feed/index.cfm": `<cfset variables.iterator = new mura.content.contentIterator()>
<cfoutput>#$.dspObject_include( theFile = 'collection/includes/dsp_list.cfm', iterator = variables.iterator )#</cfoutput>
<cfoutput>#$.dspObject_include( theFile = 'collection/includes/dsp_pagination.cfm', iterator = variables.iterator )#</cfoutput>
<cfoutput>#$.dspObject_include( theFile = 'collection/includes/dsp_list.cfm' )#</cfoutput>
<cfoutput>#$.dspObject_include( siteid, object, objectid, fileName )#</cfoutput>`,
		"modules/calendar/index.cfm": `<cfset iterator = new mura.content.calendarIterator()>
<cfoutput>#$.dspObject_include( theFile = 'collection/includes/dsp_list.cfm', iterator = iterator )#</cfoutput>
<cfoutput>#$.dspObject_include( theFile = 'collection/layouts/#objectParams.layout#/index.cfm', iterator = iterator )#</cfoutput>`,
		"modules/users/index.cfm": `<cfset args = { iterator = new mura.user.userIterator() }>
<cfoutput>#$.dspObject_include( theFile = 'users/dsp_users.cfm', argumentCollection = args )#</cfoutput>
<cfoutput>#$.dspObject_include( theFile = 'users/dsp_users.cfm', iterator = new mura.user.userIterator() )#</cfoutput>`,
		"modules/collection/includes/dsp_list.cfm": `<cfset more = arguments.iterator.hasNext()>
<cfset arguments.iterator.nowhere()>`,
		"modules/collection/includes/dsp_pagination.cfm": `<cfset more = iterator.hasNext()>`,
		"modules/collection/layouts/grid/index.cfm":      `<cfset day = arguments.iterator.getDay()>`,
		"modules/users/dsp_users.cfm":                    `<cfset more = arguments.iterator.hasNext()>`,
		"modules/own/dsp_own.cfm": `<cfset arguments.iterator = makeIterator()>
<cfset more = arguments.iterator.hasNext()>`,
		"modules/own/index.cfm": `<cfoutput>#$.dspObject_include( theFile = 'own/dsp_own.cfm', iterator = new mura.user.userIterator() )#</cfoutput>`,
	})

	r := func() *Resolver { return &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)} }

	expectReasons(t, reasonsWith(t, r(), dir, "modules/collection/includes/dsp_list.cfm"), map[string]string{
		"arguments.iterator.hasNext": "",
		"arguments.iterator.nowhere": "method 'nowhere' not found in calendarIterator|contentIterator",
	})
	expectReasons(t, reasonsWith(t, r(), dir, "modules/collection/includes/dsp_pagination.cfm"), map[string]string{
		"iterator.hasNext": "",
	})
	expectReasons(t, reasonsWith(t, r(), dir, "modules/collection/layouts/grid/index.cfm"), map[string]string{
		"arguments.iterator.getDay": "",
	})
	expectReasons(t, reasonsWith(t, r(), dir, "modules/users/dsp_users.cfm"), map[string]string{
		"arguments.iterator.hasNext": "variable 'arguments.iterator' has no component ref",
	})
	expectReasons(t, reasonsWith(t, r(), dir, "modules/own/dsp_own.cfm"), map[string]string{
		"arguments.iterator.hasNext": "variable 'arguments.iterator' has no component ref",
	})
}

// TestADisplayObjectHandedOnTwiceTypesAnAssignmentFromIt: MasaCMS's content
// list sets `arguments.item = arguments.iterator.next()`, and the iterator
// reaches it from the collection through two layout templates. Typing the
// assignment costs a level, each hand-off another, and the lookup's bound ran
// out one template short of the collection: 68 findings on the list's items.
// The hand-off is bounded a level further than an assignment.
func TestADisplayObjectHandedOnTwiceTypesAnAssignmentFromIt(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"mura/content/contentRenderer.cfc": `<cfcomponent>
<cffunction name="dspObject_Include" output="false">
	<cfargument name="theFile" type="string" />
	<cfinclude template="#arguments.theFile#" />
</cffunction>
</cfcomponent>`,
		"models/Item.cfc":     `component { function itemOnly() {} }`,
		"models/Iterator.cfc": `component { function hasNext() {} models.Item function next() {} }`,
		"modules/collection/index.cfm": `<cfset variables.iterator = new models.Iterator()>
<cfoutput>#$.dspObject_include( theFile = 'collection/layouts/default/index.cfm', iterator = variables.iterator )#</cfoutput>`,
		"modules/collection/layouts/default/index.cfm": `<cfoutput>#$.dspObject_include( theFile = 'collection/layouts/list/index.cfm', iterator = arguments.iterator )#</cfoutput>`,
		"modules/collection/layouts/list/index.cfm":    `<cfoutput>#$.dspObject_include( theFile = 'collection/includes/dsp_list.cfm', iterator = arguments.iterator )#</cfoutput>`,
		"modules/collection/includes/dsp_list.cfm": `<cfloop condition="arguments.iterator.hasNext()">
<cfset arguments.item = arguments.iterator.next()>
<cfset arguments.item.itemOnly()>
<cfset arguments.item.nope()>
</cfloop>`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "modules/collection/includes/dsp_list.cfm"), map[string]string{
		"arguments.item.itemOnly": "",
		"arguments.item.nope":     "method 'nope' not found in models.Item",
	})
}
