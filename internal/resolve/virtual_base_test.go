package resolve

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestAHandlerGetsEventHandlerWhateverItExtends: ColdBox registers every
// handler with EventHandler as its virtual base, so a handler extending a base
// of its own that extends nothing still calls runEvent() bare. Preside's admin
// handlers all extend preside.system.base.AdminHandler: about 600 findings.
// A method neither the chain nor EventHandler declares is still reported, and
// a model, which ColdBox gives no virtual base, gets nothing.
func TestAHandlerGetsEventHandlerWhateverItExtends(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"base/AdminHandler.cfc": `component { function adminHelp() {} }`,
		"handlers/Pages.cfc":    `component extends="base.AdminHandler" { function index() { adminHelp(); runEvent( "x" ); nope(); getRequestContext().getValue( "x" ); getRequestContext().nope2(); } }`,
		"models/Base.cfc":       `component {}`,
		"models/User.cfc":       `component extends="models.Base" { function f() { runEvent( "x" ); } }`,
	})

	implicit := func(path string) string {
		p := filepath.ToSlash(path)
		if strings.Contains(p, "/handlers/") {
			return "coldbox.system.EventHandler"
		}

		return ""
	}

	r := &Resolver{ImplicitExtends: implicit, Mappings: map[string]string{"base": filepath.Join(dir, "base"), "models": filepath.Join(dir, "models")}}
	expectReasons(t, reasonsWith(t, r, dir, "handlers/Pages.cfc"), map[string]string{
		"adminHelp": "",
		"runEvent":  "",
		"nope":      "not found in extends chain",
		// A chain headed by one of them, as the call alone is.
		"getRequestContext.getValue": "",
		"getRequestContext.nope2":    "method 'nope2' not found in coldbox.system.web.context.RequestContext",
	})

	r = &Resolver{ImplicitExtends: implicit, Mappings: map[string]string{"base": filepath.Join(dir, "base"), "models": filepath.Join(dir, "models")}}
	expectReasons(t, reasonsWith(t, r, dir, "models/User.cfc"), map[string]string{
		"runEvent": "not found in extends chain",
	})
}
