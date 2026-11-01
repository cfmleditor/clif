package resolve

import "testing"

// TestAPresideServiceHasPresideSuperClassMethods: Preside's WireBox config
// gives a component whose metadata, or an ancestor's, holds presideService
// the methods of PresideSuperClass by virtual inheritance, so
// $getPresideObject() is called bare in some 150 services that extend
// nothing: about 1,300 findings over Preside. A component without the key does
// not get them, and a call PresideSuperClass does not declare is still
// reported.
func TestAPresideServiceHasPresideSuperClassMethods(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"system/services/PresideSuperClass.cfc": `component { function init() {} public any function $getPresideObject() {} }`,
		"system/services/Audit.cfc":             "/**\n * @presideservice true\n */\ncomponent { function f() { $getPresideObject( \"x\" ); $nope(); } }",
		"system/services/Tagged.cfc":            `<cfcomponent presideService="true"><cffunction name="f"><cfset $getPresideObject( "x" )></cffunction></cfcomponent>`,
		"system/services/Base.cfc":              "/** @presideService */\ncomponent { }",
		"system/services/Child.cfc":             `component extends="Base" { function f() { $getPresideObject( "x" ); } }`,
		"system/services/Plain.cfc":             `component { function f() { $getPresideObject( "x" ); } }`,
		"system/services/Mention.cfc":           "component { function f() { /* @presideService */ $getPresideObject( \"x\" ); } }",
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "system/services/Audit.cfc"), map[string]string{
		"$getPresideObject": "",
		"$nope":             "no qualifier, not in file",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "system/services/Tagged.cfc"), map[string]string{
		"$getPresideObject": "",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "system/services/Child.cfc"), map[string]string{
		"$getPresideObject": "",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "system/services/Plain.cfc"), map[string]string{
		"$getPresideObject": "no qualifier, not in file",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "system/services/Mention.cfc"), map[string]string{
		"$getPresideObject": "no qualifier, not in file",
	})
}
