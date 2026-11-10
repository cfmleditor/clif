package path

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()

	for name, content := range files {
		file := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// Each way an Application file sets this.customTagPaths that the corpus
// holds: RustCFML's getDirectoryFromPath(getCurrentTemplatePath()) & "tags/",
// Mura's listAppend onto a value read from an ini file, Lucee's array, and a
// parent Application.cfc's paths a child appends to.
func TestLoadAppCustomTagPaths(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  []string // relative to the app directory
	}{
		{
			name:  "the template's directory joined to a folder",
			files: map[string]string{"Application.cfc": `component { this.customtagpaths = getDirectoryFromPath(getCurrentTemplatePath()) & "tags/ctpathroot/"; }`},
			want:  []string{"tags/ctpathroot"},
		},
		{
			name:  "a comma-delimited list",
			files: map[string]string{"Application.cfc": `component { this.customTagPaths = expandPath("./a") & "," & expandPath("./b"); }`},
			want:  []string{"a", "b"},
		},
		{
			name:  "an array with an unknown element left out",
			files: map[string]string{"Application.cfc": `component { this.customTagPaths = [ expandPath("./a"), getSetting("x"), "b" ]; }`},
			want:  []string{"a", "b"},
		},
		{
			name: "listAppend onto a value that is not a path",
			files: map[string]string{"Application.cfc": `component {
	variables.baseDir = getDirectoryFromPath(getCurrentTemplatePath());
	this.customtagpaths = evalSetting(getINIProperty("customtagpaths", ""));
	this.customtagpaths = listAppend(this.customtagpaths, variables.baseDir & "/core/mura/customtags/");
}`},
			want: []string{"core/mura/customtags"},
		},
		{
			name: "a parent's paths appended to in tag syntax",
			files: map[string]string{
				"Application.cfc": `<cfcomponent extends="Base"><cfset this.customTagPaths = this.customTagPaths & "," & expandPath("./mine")></cfcomponent>`,
				"Base.cfc":        `component { this.customTagPaths = expandPath("./shared"); }`,
			},
			want: []string{"shared", "mine"},
		},
		{
			name:  "Lucee's struct form",
			files: map[string]string{"Application.cfc": `component { this.customtagpaths["/one"] = getDirectoryFromPath(getCurrentTemplatePath()) & "lib"; this.customTagPaths["/two"] = expandPath("./more"); }`},
			want:  []string{"lib", "more"},
		},
		{
			name:  "assigned inside a method, which the constructor never runs",
			files: map[string]string{"Application.cfc": `component { function onRequestStart() { this.customTagPaths = "x"; } }`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, tc.files)
			InvalidateAppMappingsCache()

			var want []string
			for _, rel := range tc.want {
				want = append(want, filepath.Join(dir, rel))
			}

			if got := LoadAppCustomTagPaths(dir, nil); !slices.Equal(got, want) {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

// Lucee's .CFConfig.json declares the administrator's custom tag paths in
// customTagMappings and deep search in customTagDeepSearch, as Lucee's own
// LDEV-5758 test writes them.
func TestServerCustomTagRoots(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		".cfconfig.json": `{
	"customTagMappings": [
		{"physical": "./tags", "inspectTemplate": "never"},
		{"physical": "{lucee-config}/customtags/"},
		{"physical": "${TAGS}"},
		{"archive": "x.lar", "primary": "archive"}
	],
	"customTagDeepSearch": "true"
}`,
	})
	InvalidateAppMappingsCache()

	got, deep := ServerCustomTagRoots(dir)
	want := []TagRoot{{Dir: filepath.Join(dir, "tags"), Recursive: true}}

	if !slices.Equal(got, want) || !deep {
		t.Fatalf("got %+v, deep %v; want %+v, deep", got, deep, want)
	}
}

// CommandBox's cfconfig format names the directories customTagPaths, where
// Lucee's own file says customTagMappings; both are read.
func TestServerCustomTagRootsReadCommandBoxsKey(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		".cfconfig.json": `{"customTagMappings": [{"physical": "./a"}], "customTagPaths": [{"physical": "./b", "name": "b"}]}`,
	})
	InvalidateAppMappingsCache()

	got, _ := ServerCustomTagRoots(dir)
	want := []TagRoot{{Dir: filepath.Join(dir, "a")}, {Dir: filepath.Join(dir, "b")}}

	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// The custom tag paths come from the walk that reads the mappings and are
// kept beside them: once the mappings are loaded, asking for the paths reads
// nothing, so a file gone since is not noticed until the cache is dropped.
func TestCustomTagPathsComeFromTheMappingWalk(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Application.cfc": `component { this.mappings["/m"] = expandPath("./m"); this.customTagPaths = expandPath("./tags"); }`})
	InvalidateAppMappingsCache()

	if LoadAppMappings(dir)["m"] == "" {
		t.Fatal("the mapping was not read")
	}

	if err := os.Remove(filepath.Join(dir, "Application.cfc")); err != nil {
		t.Fatal(err)
	}

	probeReads.Clear()

	if got := LoadAppCustomTagPaths(dir, nil); !slices.Equal(got, []string{filepath.Join(dir, "tags")}) {
		t.Fatalf("got %q: the paths were read again rather than kept from the mapping walk", got)
	}

	InvalidateAppMappingsCache()

	if got := LoadAppCustomTagPaths(dir, nil); got != nil {
		t.Fatalf("after invalidation got %q, want none", got)
	}
}
