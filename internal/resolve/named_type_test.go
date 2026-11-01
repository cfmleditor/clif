package resolve

import "testing"

// TestABareArgumentTypeNamesTheOnlyComponentOfThatName: qb's grammars
// declare `required QueryBuilder query` for models/Query/QueryBuilder.cfc,
// another directory, and every call on the argument was "no component ref".
// The engine checks a component argument by the component's name, so a bare
// type naming exactly one component in the workspace is that component. A
// name two files answer to stays untyped, and the file beside the declaring
// one still wins.
func TestABareArgumentTypeNamesTheOnlyComponentOfThatName(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"models/Query/QueryBuilder.cfc": `component { function getFrom(){} }`,
		"a/Twice.cfc":                   `component { function one(){} }`,
		"b/Twice.cfc":                   `component { function two(){} }`,
		"models/Grammars/Grammar.cfc": `component {
function compile( required QueryBuilder query, required Twice t, required Local l ){
	query.getFrom();
	query.nope();
	t.one();
	l.near();
}
}`,
		"models/Grammars/Local.cfc": `component { function near(){} }`,
		"elsewhere/Local.cfc":       `component { function far(){} }`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "models/Grammars/Grammar.cfc"), map[string]string{
		"query.getFrom": "",
		"query.nope":    "method 'nope' not found in QueryBuilder",
		"t.one":         "variable 't' has no component ref",
		"l.near":        "",
	})
}

// TestAGeneratedGetterIsWhatInitStoresFromATypedArgument: qb's Blueprint
// takes `required SchemaBuilder schemaBuilder` in init and stores it with the
// generated setSchemaBuilder(), and every grammar reads it back as
// arguments.blueprint.getSchemaBuilder(). The getter was typed from init's
// argument only when its type was dotted; a bare type is read as an
// argument's is (namedType).
func TestAGeneratedGetterIsWhatInitStoresFromATypedArgument(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"schema/SchemaBuilder.cfc": `component { function getShouldWrap(){} }`,
		"schema/Blueprint.cfc": `component {
property name="schemaBuilder";
property name="loose";
function init( required SchemaBuilder schemaBuilder, loose ){
	setSchemaBuilder( arguments.schemaBuilder );
	setLoose( arguments.loose );
	return this;
}
}`,
		"Grammar.cfc": `component {
function compile( required schema.Blueprint blueprint ){
	arguments.blueprint.getSchemaBuilder().getShouldWrap();
	arguments.blueprint.getSchemaBuilder().nope();
	arguments.blueprint.getLoose().anything();
}
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Grammar.cfc"), map[string]string{
		"arguments.blueprint.getSchemaBuilder.getShouldWrap": "",
		"arguments.blueprint.getSchemaBuilder.nope":          "method 'nope' not found in SchemaBuilder",
		"arguments.blueprint.getLoose.anything":              "method 'getLoose' in schema.Blueprint has no component return type (chain to 'anything')",
	})
}
