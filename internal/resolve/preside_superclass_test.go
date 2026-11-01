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

// TestAPresideServiceHasPresideSuperClassFields: the methods virtual
// inheritance gives a presideService component read PresideSuperClass's
// fields, and the service reads them too: `$helpers.isTrue( x )`, which
// PresideSuperClass sets to the helper class it is handed. A component
// without presideService does not have them.
func TestAPresideServiceHasPresideSuperClassFields(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"system/services/PresideSuperClass.cfc":  `component { function init() { this.$helpers = new PresideHelperClass(); } }`,
		"system/services/PresideHelperClass.cfc": `component { function isTrue( v ) {} }`,
		"system/services/Audit.cfc":              "/**\n * @presideservice true\n */\ncomponent { function f() { $helpers.isTrue( 1 ); $helpers.nope(); } }",
		"system/services/Plain.cfc":              `component { function f() { $helpers.isTrue( 1 ); } }`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "system/services/Audit.cfc"), map[string]string{
		"$helpers.isTrue": "",
		"$helpers.nope":   "method 'nope' not found in PresideHelperClass",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "system/services/Plain.cfc"), map[string]string{
		"$helpers.isTrue": "variable '$helpers' has no component ref",
	})
}
