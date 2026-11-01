package resolve

import "testing"

// TestARuntimePathReturnIsDynamic: lucee-docs' BuildRunner.getBuilder()
// returns `new "builders.#arguments.builderName#.Builder"()`. A runtime path
// no expressionMappings entry covers is dynamic wherever it is assigned, and
// a call through the function returning it was reported as a component that
// does not exist.
func TestARuntimePathReturnIsDynamic(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"BuildRunner.cfc": `component { public any function getBuilder( required string builderName ) { var builder = new "builders.#arguments.builderName#.Builder"(); return builder; } }`,
		"Page.cfc":        `component { function f() { var r = new BuildRunner(); r.getBuilder( "html" ).renderPage(); } }`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Page.cfc"), map[string]string{
		"r.getBuilder.renderPage": "",
	})
}
