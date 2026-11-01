package resolve

import (
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// TestAStructMemberHoldsADynamicOrNativeValue: a member written through a
// struct (`instance.x = …`, the private-data idiom of older ColdBox code) kept
// only a component, so an unstubbed Java object and a native value left it
// untyped, where a plain variable assigned the same is `$any` or the native
// type. cbi18n keeps its Java formatters this way: over 750 findings in
// ColdBox's samples. A method the native value lacks is still reported.
func TestAStructMemberHoldsADynamicOrNativeValue(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Script.cfc": `component {
function init() {
	instance.tz = createObject( "java", "java.util.TimeZone" );
	instance.q = queryNew( "a" );
	instance.placeholder = structNew();
	return this;
}
function a() { instance.tz.getAvailableIDs(); instance.q.recordCount(); instance.q.definitelyMissing(); instance.placeholder.getDisplayFormat(); }
}`,
		"Tag.cfc": `<cfcomponent>
<cffunction name="init"><cfset instance.tz = createObject( "java", "java.util.TimeZone" )><cfreturn this></cffunction>
<cffunction name="a"><cfset instance.tz.getAvailableIDs()></cffunction>
</cfcomponent>`,
	})

	r := &Resolver{}
	parse := func(file string, o *parser.ParseOptions) { o.BuiltinReturnLookup = r.BuiltinReturnLookupFor(file) }

	got := reasonsWithOptions(t, r, dir, "Script.cfc", parse)
	expectReasons(t, got, map[string]string{
		"instance.tz.getAvailableIDs":  "",
		"instance.q.recordCount":       "",
		"instance.q.definitelyMissing": "method 'definitelyMissing' not found on builtin querynew",
		// A struct is the placeholder a component is handed in for, so it
		// does not type the member (tassweb's cfparam default="#StructNew()#").
		"instance.placeholder.getDisplayFormat": "variable 'instance.placeholder' has no component ref",
	})

	expectReasons(t, reasonsWithOptions(t, r, dir, "Tag.cfc", parse), map[string]string{
		"instance.tz.getAvailableIDs": "",
	})
}
