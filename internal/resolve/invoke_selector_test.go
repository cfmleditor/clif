package resolve

import "testing"

// TestAComputedInvokeSelectorPassesItsArguments: qb's Blueprint compiles each
// command with invoke( getGrammar(), "compile#command.getType()#", {
// blueprint: this } ), so every compileX(blueprint) in the grammar is called
// with a Blueprint, and no call site names one. The computed selector was
// read as naming some other method, the grammar's methods had no caller, and
// every call on the blueprint was "no component ref". A selector whose literal
// start the method's name begins with is a call to it; one that cannot name
// it, or names nothing literal, is not.
func TestAComputedInvokeSelectorPassesItsArguments(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"schema/Blueprint.cfc": `component {
function getGrammar(){ return new schema.Grammar(); }
function getTable(){ return "t"; }
function toSql( command ){
	return invoke( getGrammar(), "compile#command.getType()#", { blueprint: this } );
}
}`,
		"schema/Grammar.cfc": `component {
function compileCreate( required blueprint ){ return blueprint.getTable(); }
function renderDrop( required blueprint ){ return blueprint.render(); }
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "schema/Grammar.cfc"), map[string]string{
		"blueprint.getTable": "",
		"blueprint.render":   "variable 'blueprint' has no component ref",
	})

	if !invokeSelects("compile#x#", "compileCreate") || invokeSelects("compile#x#", "renderDrop") || invokeSelects("#x#", "compileCreate") || invokeSelects("compile#x#Now", "compileCreate") || !invokeSelects("compile#x#Now", "compileDropNow") {
		t.Error("invokeSelects matched the wrong names")
	}
}

// TestAnInvokeOnAnUntypedTargetReachesTheOnlyFamilyDeclaringTheMethod: qb's
// Blueprint invokes compileX() on a grammar WireBox builds from a setting,
// so the target has no type, and every grammar's compileX() had no caller.
// When every component declaring the name is in one class tree (a base and
// the grammars overriding it), the invoke calls it there whatever the target
// is. A name some unrelated component declares too still gives nothing.
func TestAnInvokeOnAnUntypedTargetReachesTheOnlyFamilyDeclaringTheMethod(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"schema/Blueprint.cfc": `component {
function getTable(){ return "t"; }
function toSql( command, grammar ){
	return invoke( grammar, "compile#command.getType()#", { blueprint: this } );
}
}`,
		"g/Base.cfc": `component {
function compileCreate( required blueprint ){ return blueprint.getTable(); }
function compileShared( required blueprint ){ return blueprint.getTable(); }
}`,
		"g/MySQL.cfc":     `component extends="Base" { function compileCreate( required blueprint ){ return blueprint.getTable(); } }`,
		"g/Postgres.cfc":  `component extends="Base" { function compileCreate( required blueprint ){ return blueprint.getTable(); } }`,
		"other/Thing.cfc": `component { function compileShared( required blueprint ){} }`,
		"g/Shared.cfc": `component extends="Base" {
function compileShared( required blueprint ){ return blueprint.nope(); }
}`,
	})

	r := &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}
	expectReasons(t, reasonsWith(t, r, dir, "g/MySQL.cfc"), map[string]string{
		"blueprint.getTable": "",
	})
	expectReasons(t, reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "g/Shared.cfc"), map[string]string{
		"blueprint.nope": "variable 'blueprint' has no component ref",
	})
}

// TestAComputedInvokeOnAnUnplacedTargetIsPassedOver: ColdBox somewhere runs
// invoke( x, "get#name#" ) on a target nothing types, and once computed
// selectors were read, that invoke withheld inference for every get…()
// method in the workspace: HandlerService.getHandler( ehBean ) lost the
// EventHandlerBean its literal callers pass. A computed selector on a target
// that cannot be placed is passed over, as before selectors were read.
func TestAComputedInvokeOnAnUnplacedTargetIsPassedOver(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"models/Bean.cfc": `component { function getMethod(){} }`,
		"Service.cfc": `component {
function getHandler( required ehBean ){ arguments.ehBean.getMethod(); }
function run(){ getHandler( new models.Bean() ); }
}`,
		"Proxy.cfc": `component {
function forward( target, name ){ return invoke( arguments.target, "get#arguments.name#" ); }
}`,
		"Elsewhere.cfc": `component { function getHandler( name ){} }`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Service.cfc"), map[string]string{
		"arguments.ehBean.getMethod": "",
	})
}
