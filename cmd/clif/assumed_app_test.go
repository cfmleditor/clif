package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/unresolved"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestEachColdBoxAppIsReadUnderItsOwnConventions: a repository of several
// ColdBox applications, none shipping ColdBox, has no single owner for the
// project-wide detection, so no preset was assumed and every handler's `event`
// was untyped (ColdBox's samples: 436 `event`, 313 `controller`). Each
// application declares itself with config/ColdBox.cfc (here spelled Coldbox.cfc,
// as the samples do), and a module with ModuleConfig.cfc, so each directory is
// read under the nearest one's conventions. A module's build/ task runner is
// CommandBox's, as it is for a project owner.
func TestEachColdBoxAppIsReadUnderItsOwnConventions(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"apps/blog/config/Coldbox.cfc":   `component { function configure(){} }`,
		"apps/blog/handlers/Main.cfc":    `component { function index( event, rc, prc ){ event.getValue( "x" ); event.definitelyMissing(); } }`,
		"apps/shop/config/ColdBox.cfc":   `component { function configure(){} }`,
		"apps/shop/handlers/Cart.cfc":    `component { function add( event, rc, prc ){ event.setView( "cart" ); } }`,
		"modules/feeds/ModuleConfig.cfc": `component { function configure(){} }`,
		"modules/feeds/build/Build.cfc":  `component { function init(){ variables.cwd = getCWD(); } function run(){ print.line( "x" ).toConsole(); } }`,
		"loose/Page.cfc":                 `component { function f( event ){ event.looseThing( "x" ); } }`,
	}

	var sources []string

	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		sources = append(sources, p)
	}

	opt := unresolvedOptions(nil, []string{root}, &unresolvedFlags{})
	opt.ResolverFor = perFileResolver(nil)

	report := unresolved.Scan(vfs.OS{}, sources, sources, opt)

	got := map[string]bool{}
	for _, call := range report.Calls {
		got[call.Function] = true
	}

	for _, accepted := range []string{"setView", "line", "toConsole"} {
		if got[accepted] {
			t.Errorf("%s reported: %+v", accepted, report.Calls)
		}
	}

	if !got["definitelyMissing"] {
		t.Error("a method RequestContext lacks was accepted")
	}

	// A file under no application keeps the project's reading, which here is
	// none: its `event` is not ColdBox's.
	if !got["looseThing"] {
		t.Error("a file outside every application was read as ColdBox")
	}
}
