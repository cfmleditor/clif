package resolve

import "testing"

// TestAnInterceptorReadsWhatItsAnnounceIsPassed: a ColdBox interceptor method
// `cbadmin_postEntrySave( event, data )` is handed the struct the code
// announcing that point passed, so arguments.data.content, a bare
// data.content, and a local assigned from either are that value, typed where
// the announce is made. A computed state names every point its literal parts
// allow; an announce passing no data, or omitting the key, says nothing; one
// passing data that is not a struct literal, or passing the key under a state
// it does not spell, leaves the key untyped; and an interceptor that assigns
// its data itself reads its own value.
func TestAnInterceptorReadsWhatItsAnnounceIsPassed(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"models/Entry.cfc": `component { function publish() {} function archive() {} function retract() {} function expire() {} }`,
		"models/Page.cfc":  `component { function publish() {} function getSlug() {} }`,
		"models/Other.cfc": `component { function publish() {} function retract() {} }`,
		"handlers/Base.cfc": `component {
	function save( event ) {
		var oContent = new models.Entry();
		announce( "cbadmin_post#variables.entity#Save", { content : oContent, isNew : true } );
		announce( "cbadmin_postEntrySave", { isNew : false } );
		announce( "cbadmin_postEntrySave" );
	}
}`,
		"handlers/Pages.cfc": `component {
	function remove( event ) {
		announceInterception( state = "cbadmin_prePageRemove", interceptData = { page = new models.Page() } );
		announce( "cbadmin_localSave", { content : new models.Entry() } );
	}
}`,
		"handlers/Opaque.cfc": `component {
	function save( event ) {
		var args = { content : new models.Other() };
		announce( "cbadmin_opaqueSave", args );
		announce( "cbadmin_opaqueSave", { content : new models.Other() } );
		announce( stateName, { thing : new models.Other() } );
		announce( "cbadmin_unknownSave", { thing : new models.Other() } );
	}
}`,
		"interceptors/Notify.cfc": `component {
	function cbadmin_postEntrySave( event, data, buffer ) {
		var entry = arguments.data.content;
		entry.publish();
		entry.nowhere();
		arguments.data.content.publish();
		data.content.archive();
	}
	function cbadmin_prePageRemove( event, interceptData ) {
		arguments.interceptData.page.getSlug();
	}
	function cbadmin_opaqueSave( event, data ) {
		arguments.data.content.retract();
	}
	function cbadmin_unknownSave( event, data ) {
		arguments.data.thing.publish();
	}
	function cbadmin_localSave( event, data ) {
		data = makeData();
		data.content.expire();
	}
}`,
	})

	// Each declined case calls a method the value it would have been typed
	// as has, so a guard that let it through would resolve it.
	expectReasons(t, reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "interceptors/Notify.cfc"), map[string]string{
		"entry.publish":                        "",
		"entry.nowhere":                        "method 'nowhere' not found in Entry",
		"arguments.data.content.publish":       "",
		"data.content.archive":                 "",
		"arguments.interceptData.page.getSlug": "",
		"arguments.data.content.retract":       "variable 'arguments.data.content' has no component ref",
		"arguments.data.thing.publish":         "variable 'arguments.data.thing' has no component ref",
		"data.content.expire":                  "variable 'data.content' has no component ref",
	})
}
