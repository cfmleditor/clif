package resolve

import "testing"

// TestAPresideObjectIsWhatItsDecoratorMakesIt: a Preside service calls
// $getPresideObject( "x" ).selectData(), which PresideSuperClass answers with
// PresideObjectService.getObject(), which returns the object decorated by
// PresideObjectDecorator: its injected methods and an onMissingMethod handing
// the data API to the service. The chain's head is found as the call alone is
// (on PresideSuperClass), and what it returns is the decorator. A
// getObject() outside a Preside checkout's PresideObjectService says nothing.
func TestAPresideObjectIsWhatItsDecoratorMakesIt(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"system/services/PresideSuperClass.cfc": `component {
	property name="presideObjectService" inject="presideObjectService";
	public any function $getPresideObject() { return presideObjectService.getObject( argumentCollection=arguments ); }
	public any function $getOther() { return new other.Factory().getObject( "x" ); }
}`,
		"system/services/presideObjects/PresideObjectService.cfc": `component {
	public any function getObject( required string objectName ) { var obj = _getObject( arguments.objectName ); return obj.decoratedInstance; }
}`,
		"system/services/presideObjects/PresideObjectDecorator.cfc": `component {
	public any function onMissingMethod( required string missingMethodName, required struct missingMethodArguments ) {}
	public string function getName() { return ""; }
}`,
		"system/services/Audit.cfc": "/**\n * @presideservice true\n */\ncomponent { function f() { $getPresideObject( \"x\" ).selectData(); $getOther().deleteData(); } }",
		"other/Factory.cfc":         `component { public any function getObject( required string objectName ) { return objects[ objectName ]; } }`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "system/services/Audit.cfc"), map[string]string{
		"$getPresideObject.selectData": "",
		"$getOther.deleteData":         "method '$getOther' has no component return type (chain to 'deleteData')",
	})
}
