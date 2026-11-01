package resolve

import "testing"

// TestAnAbstractBaseCallsWhatEverySubclassDeclares: qb's spec base calls
// selectAllColumns(), which each grammar's spec declares, and builds its
// builder with getBuilder(), which it declares only to throw. Neither is in
// the base or above it, so every such call was "not found in extends chain"
// and the builder had no type. A call every leaf subclass answers is
// accepted, and an abstract method returns what the overrides return; a leaf
// that lacks the method, or a base method that does more than throw, gives
// nothing.
func TestAnAbstractBaseCallsWhatEverySubclassDeclares(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"models/Builder.cfc": `component { function select(){ return this; } function init(){ return this; } }`,
		"models/Other.cfc":   `component { function select(){ return this; } function only(){} }`,
		"spec/Base.cfc": `component {
function run(){
	expected();
	var b = getBuilder();
	b.select();
	b.nope();
	getBuilder().select();
	partial();
	var c = concrete();
	c.select();
}
private function getBuilder(){
	throw( "Must be implemented in a subclass" );
}
private function concrete(){
	return getThing();
}
}`,
		"spec/A.cfc": `component extends="Base" {
function expected(){ return "a"; }
function partial(){}
private function getBuilder(){ var b = new models.Builder().init(); return b; }
private function concrete(){ return new models.Builder(); }
}`,
		"spec/Mid.cfc": `component extends="Base" {
function expected(){ return "m"; }
private function getBuilder(){ return new models.Builder(); }
}`,
		"spec/B.cfc": `component extends="Mid" { private function concrete(){ return new models.Builder(); } }`,

		"mock/Base.cfc": `component extends="testbox.system.BaseSpec" {
function run(){
	var b = getBuilder();
	b.select();
}
private function getBuilder(){ throw( "abstract" ); }
}`,
		"mock/A.cfc": `component extends="mock.Base" {
function chained(){ getMockBox().createMock( "models.Builder" ).init().nope(); }
private function getBuilder(){
	var b = getMockBox().createMock( "models.Builder" ).init();
	return b;
}
}`,

		"spec/Mixed.cfc": `component {
function run(){
	var b = getBuilder();
	b.select();
	b.only();
}
private function getBuilder(){ throw( "abstract" ); }
}`,
		"spec/MA.cfc": `component extends="Mixed" { private function getBuilder(){ return new models.Builder(); } }`,
		"spec/MB.cfc": `component extends="Mixed" { private function getBuilder(){ return new models.Other(); } }`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "spec/Base.cfc"), map[string]string{
		"expected":          "",
		"b.select":          "",
		"b.nope":            "method 'nope' not found in Builder",
		"getBuilder.select": "",
		"partial":           "no qualifier, not in file",
		"c.select":          "variable 'c' has no component ref",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "mock/Base.cfc"), map[string]string{
		"b.select": "",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "mock/A.cfc"), map[string]string{
		"getMockBox.createMock.init.nope": "method 'nope' not found in Builder",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "spec/Mixed.cfc"), map[string]string{
		"b.select": "",
		"b.only":   "",
	})
}

// TestAMockReportIsNotTheMock: a chained createEmptyMock( "X" ) is typed as X
// now, which exposed `mock.$callLog().deleteVar.len()`: $callLog() returns a
// struct of calls by method name, and the hop walk had every decoration
// return the mock, so deleteVar was looked for as a property of X.
func TestAMockReportIsNotTheMock(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"models/Cookies.cfc": `component { function deleteVar(){} }`,
		"spec/Test.cfc": `component extends="testbox.system.BaseSpec" {
function run(){
	var m = getMockBox().createEmptyMock( "models.Cookies" );
	m.$( "deleteVar" ).deleteVar();
	m.$callLog().deleteVar.len();
	m.$( "deleteVar" ).nope();
}
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "spec/Test.cfc"), map[string]string{
		"m.$.deleteVar":                      "",
		"m.$callLog.$property:deleteVar.len": "",
		"m.$.nope":                           "method 'nope' not found in models.Cookies",
	})
}
