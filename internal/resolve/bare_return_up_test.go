package resolve

import "testing"

// TestABareReturnTypeIsLookedUpAboveTheDeclaringFile: CFML looks a bare
// component name up from the component's own directory and then from the
// application's root. spreadsheet-cfml's helpers/base.cfc declares
// `Spreadsheet function library()` with Spreadsheet.cfc one level up, and
// every call chained on library() (188) was "no component return type".
func TestABareReturnTypeIsLookedUpAboveTheDeclaringFile(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Spreadsheet.cfc":  `component { function addRow(){} }`,
		"helpers/base.cfc": `component { Spreadsheet function library(){ return variables.lib; } function f(){ library().addRow(); library().definitelyMissing(); } }`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{WorkspaceFolders: []string{dir}}, dir, "helpers/base.cfc"), map[string]string{
		"library.addRow":            "",
		"library.definitelyMissing": "method 'definitelyMissing' not found in Spreadsheet",
	})
}
