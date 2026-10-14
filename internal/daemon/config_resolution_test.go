package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/cfmleditor/clif/internal/config"
)

func writeTestConfig(t *testing.T, dir, body string) *Config {
	t.Helper()

	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}

	p := filepath.Join(dir, ".clif.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	return &Config{Path: p, Name: "x"}
}

// TestTheCommandLineReadsAConfigAsTheEditorDoes: every accessor is the file's
// resolution, config.ResolveFile, which is also what the standalone editor
// applies. They were two readings of one schema and had drifted: the command
// line mapped the folder holding the config under its own name and the editor
// did not, and an absent formatting block meant defaults to the one and zero
// values to the other. Nothing here compares two implementations any more; it
// pins that there is one.
func TestTheCommandLineReadsAConfigAsTheEditorDoes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "myapp")
	c := writeTestConfig(t, dir, `{
 "workspaceName":"x","workspacePaths":["."],
 "mappings":{"shared":"../shared","/Lib":"lib"},
 "startupFiles":["init.cfm"],
 "componentResolvers":[{"match":"a","resolve":"b","prefix":"a"}],
 "frameworks":["coldbox"],
 "beanPaths":{"":"beans"},
 "linting":{"enabled":true},"features":{"folding":true}
}`)

	data, err := os.ReadFile(c.Path)
	if err != nil {
		t.Fatal(err)
	}

	var raw config.JSON
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	want := config.ResolveFile(&raw, c.Path)
	if got := c.Resolved(); !reflect.DeepEqual(got, want) {
		t.Errorf("Resolved differs from config.ResolveFile:\n got  %+v\n want %+v", got, want)
	}

	if got := c.Mappings()["myapp"]; got != dir {
		t.Errorf("the folder holding the config should map under its own name, got %q", got)
	}

	if !c.Linting() || !c.ResolvedFeatures().Folding || !reflect.DeepEqual(c.Frameworks(), []string{"coldbox"}) {
		t.Errorf("accessors do not read the file: linting %v folding %v frameworks %v", c.Linting(), c.ResolvedFeatures().Folding, c.Frameworks())
	}
}

// TestAConfigWithoutFormattingMeansTheDefaults: with no formatting block the
// formatter's safety guard (whitespaceOnly) is on, whoever reads the config.
func TestAConfigWithoutFormattingMeansTheDefaults(t *testing.T) {
	c := writeTestConfig(t, filepath.Join(t.TempDir(), "app"), `{"workspaceName":"x"}`)

	if got := c.ResolvedFormatting(); got != config.DefaultResolvedFormatting() {
		t.Errorf("an absent formatting block resolved to %+v, want the defaults", got)
	}
}
