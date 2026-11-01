package resolve

import (
	"path/filepath"
	"testing"
)

// TestARemovedColdBoxMethodIsItsAliasWhenColdBoxIsStubbed: setNextEvent()
// was relocate() before ColdBox 6. A handler whose ColdBox is not in the
// workspace runs a ColdBox the stubs do not say, so the call is relocate();
// with ColdBox checked out, its EventHandler says, and a ColdBox without
// setNextEvent is still reported.
func TestARemovedColdBoxMethodIsItsAliasWhenColdBoxIsStubbed(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"handlers/Pages.cfc": `component extends="coldbox.system.EventHandler" { function index( event ) { setNextEvent( "main" ); relocate( "main" ); getModel( "x" ); nope(); } }`,
	})

	r := &Resolver{}
	expectReasons(t, reasonsWith(t, r, dir, "handlers/Pages.cfc"), map[string]string{
		"setNextEvent": "",
		"relocate":     "",
		"getModel":     "",
		"nope":         "not found in extends chain",
	})

	checkout := t.TempDir()
	writeFiles(t, checkout, map[string]string{
		"handlers/Pages.cfc":                    `component extends="coldbox.system.EventHandler" { function index( event ) { setNextEvent( "main" ); relocate( "main" ); } }`,
		"coldbox/system/EventHandler.cfc":       `component extends="coldbox.system.FrameworkSupertype" {}`,
		"coldbox/system/FrameworkSupertype.cfc": `component { function relocate( event ) {} }`,
	})

	r = &Resolver{Mappings: map[string]string{"coldbox": filepath.Join(checkout, "coldbox")}}
	expectReasons(t, reasonsWith(t, r, checkout, "handlers/Pages.cfc"), map[string]string{
		"setNextEvent": "not found in extends chain",
		"relocate":     "",
	})
}
