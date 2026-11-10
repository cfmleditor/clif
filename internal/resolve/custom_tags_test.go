package resolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

// Every syntax resolves to the template the engine would run, through every
// source of custom tag roots: the caller's directory, this.customTagPaths,
// customTagPaths config entries (plain, recursive and prefixed), a cfimport
// taglib through a mapping, and the server's CFConfig with deep search on.
func TestCustomTagTemplate(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"app/Application.cfc": `component {
	this.mappings["/lib"] = expandPath("./library");
	this.customTagPaths = expandPath("/apptags");
}`,
		"app/.cfconfig.json":                 `{"customTagMappings": [{"physical": "../servertags"}], "customTagDeepSearch": true}`,
		"app/pages/index.cfm":                `<cfimport taglib="/lib/tags" prefix="ui">`,
		"app/pages/local.cfm":                ``,
		"app/pages/sub/page.cfm":             ``,
		"app/apptags/layout.cfm":             ``,
		"app/apptags/local.cfm":              ``,
		"app/apptags/inner/innertag.cfm":     ``,
		"app/library/tags/button.cfm":        ``,
		"servertags/a/b/srv.cfm":             ``,
		"configtags/widget.cfc":              ``,
		"configtags/ui/grid.cfm":             ``,
		"deeptags/x/y/deepone.cfm":           ``,
		"bridgelib/bridge.cfm":               ``,
		"lucee-library/output.cfm":           ``,
		"lucee-library/videoplayer.cfm":      ``,
		"configtags/sub/onlybelow.cfm":       ``,
		"app/pages/sub/beside/neighbour.cfm": ``,
	}
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cfpath.InvalidateAppMappingsCache()

	r := &Resolver{FS: vfs.OS{}, Index: index.New(), CustomTagPaths: config.ResolveCustomTagPaths([]config.CustomTagPath{
		{Directory: "configtags"},
		{Directory: "deeptags", Recursive: true},
		{Directory: "bridgelib", Prefix: "cfa"},
		{Directory: "lucee-library", Prefix: "cf"},
	}, dir)}

	at := func(rel string) string { return filepath.Join(dir, filepath.FromSlash(rel)) }
	indexPage := at("app/pages/index.cfm")
	page := at("app/pages/sub/page.cfm")

	cases := []struct {
		name string
		call parser.CustomTagCall
		from string
		want string
	}{
		{"cf_ in this.customTagPaths, its expandPath(\"/x\") read under the app", parser.CustomTagCall{Kind: parser.CustomTagUnderscore, Name: "LAYOUT"}, indexPage, "app/apptags/layout.cfm"},
		{"the caller's directory before every root", parser.CustomTagCall{Kind: parser.CustomTagUnderscore, Name: "local"}, indexPage, "app/pages/local.cfm"},
		{"a config root, as a .cfc", parser.CustomTagCall{Kind: parser.CustomTagUnderscore, Name: "widget"}, indexPage, "configtags/widget.cfc"},
		{"a recursive config root", parser.CustomTagCall{Kind: parser.CustomTagUnderscore, Name: "deepone"}, indexPage, "deeptags/x/y/deepone.cfm"},
		{"the server's root, searched deep", parser.CustomTagCall{Kind: parser.CustomTagUnderscore, Name: "srv"}, indexPage, "servertags/a/b/srv.cfm"},
		{"deep search reaches into the application's root too", parser.CustomTagCall{Kind: parser.CustomTagUnderscore, Name: "innertag"}, indexPage, "app/apptags/inner/innertag.cfm"},
		{"cfmodule name, a dotted path under a root", parser.CustomTagCall{Kind: parser.CustomTagModuleName, Name: "ui.grid"}, indexPage, "configtags/ui/grid.cfm"},
		{"cfmodule name does not search the caller's directory", parser.CustomTagCall{Kind: parser.CustomTagModuleName, Name: "local"}, page, "app/apptags/local.cfm"},
		{"cfmodule template, relative", parser.CustomTagCall{Kind: parser.CustomTagModuleTemplate, Name: "../local.cfm"}, page, "app/pages/local.cfm"},
		{"cfmodule template, computed", parser.CustomTagCall{Kind: parser.CustomTagModuleTemplate, Name: "#x#.cfm", Dynamic: true}, page, ""},
		{"a cfimport taglib through a mapping", parser.CustomTagCall{Kind: parser.CustomTagImported, Name: "button", Prefix: "ui", Taglib: "/lib/tags"}, indexPage, "app/library/tags/button.cfm"},
		{"a cfimport taglib relative to the caller", parser.CustomTagCall{Kind: parser.CustomTagImported, Name: "neighbour", Prefix: "n", Taglib: "beside/"}, page, "app/pages/sub/beside/neighbour.cfm"},
		{"a configured prefix", parser.CustomTagCall{Kind: parser.CustomTagPrefixed, Name: "bridge", Prefix: "cfa"}, indexPage, "bridgelib/bridge.cfm"},
		{"a Lucee library directory", parser.CustomTagCall{Kind: parser.CustomTagPrefixed, Name: "videoplayer", Prefix: "cf"}, indexPage, "lucee-library/videoplayer.cfm"},
		{"never a built-in tag", parser.CustomTagCall{Kind: parser.CustomTagPrefixed, Name: "output", Prefix: "cf"}, indexPage, ""},
		{"a plain root is not searched below", parser.CustomTagCall{Kind: parser.CustomTagUnderscore, Name: "onlybelow"}, at("elsewhere/x.cfm"), ""},
		{"nothing answers", parser.CustomTagCall{Kind: parser.CustomTagUnderscore, Name: "missing"}, indexPage, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := ""
			if tc.want != "" {
				want = at(tc.want)
			}

			if got := r.CustomTagTemplate(&tc.call, tc.from); !cfpath.SamePath(got, want) && got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}
