package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/cfmleditor/clif/internal/config"
)

// TestConfigAccessorsAgreeWithResolve: the CLI reads .clif.json through these
// accessors and the standalone editor through config.Resolve. They are two
// readings of one schema, and a key read differently by one is a project that
// resolves differently in the editor than in `clif unresolved`. Every setting
// resolution depends on is compared here, from one rich config.
//
// The accessors add one thing Resolve does not: the folder holding the config
// maps under its own name (Mappings' doc), which is removed before comparing.
// Formatting is left out on purpose: an absent block means defaults to the
// accessors and zero values to Resolve, and the server restores the former
// (restoreUnmentionedBlocks).
func TestConfigAccessorsAgreeWithResolve(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "myapp")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}

	body := `{
 "workspaceName":"x","workspacePaths":["."],
 "mappings":{"shared":"../shared","/Lib":"lib"},
 "startupFiles":["init.cfm","/abs/up.cfm"],
 "expressionMappings":{"#a#":"b."},
 "servicePropertyResolvers":{"k":"v.${name}"},
 "componentResolvers":[{"match":"a","resolve":"b","prefix":"a"},{"match":"","resolve":"x"}],
 "javaStubsPath":"stubs",
 "frameworks":["coldbox"],
 "propertyResolvers":[{"match":"m","resolve":"r","attribute":"at"},{"match":"m"}],
 "beanPaths":{"":"beans","ns":"nsbeans"},
 "linting":{"enabled":true,"minSeverity":"ERROR"},"references":{"enabled":true},
 "features":{"folding":true},"completions":{"tagSnippets":false},"docBlock":{"gap":true},
 "callerInferenceLimits":{"maxCandidateFiles":7}
}`

	path := filepath.Join(dir, ".clif.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	var raw config.JSON
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatal(err)
	}

	c := &Config{Path: path, Name: "x"}
	r := config.Resolve(&raw, dir)

	mappings := c.Mappings()
	delete(mappings, filepath.Base(dir))

	for name, pair := range map[string][2]any{
		"Mappings":                 {mappings, r.Mappings},
		"StartupFiles":             {c.StartupFiles(), r.StartupFiles},
		"ExpressionMappings":       {c.ExpressionMappings(), r.ExpressionMappings},
		"ServicePropertyResolvers": {c.ServicePropertyResolvers(), r.ServicePropertyResolvers},
		"ComponentResolvers":       {c.ComponentResolvers(), r.ComponentResolvers},
		"Frameworks":               {c.Frameworks(), r.Frameworks},
		"BeanPaths":                {c.BeanPaths(), r.BeanPaths},
		"CallerInferenceLimits":    {c.CallerInferenceLimits(), r.CallerInferenceLimits},
		"Features":                 {c.ResolvedFeatures(), r.Features},
		"Linting":                  {c.Linting(), r.Linting},
		"LintMinSeverity":          {c.LintMinSeverity(), r.LintMinSeverity},
		"References":               {c.References(), r.References},
		"Routes":                   {c.Routes(), r.Routes},
		"CodeMap":                  {c.CodeMap(), r.CodeMap},
		"KnownIssues":              {c.KnownIssues(), r.KnownIssues},
		"TagSnippets":              {c.ResolvedCompletions().TagSnippets, r.TagSnippets},
		"FunctionSnippets":         {c.ResolvedCompletions().FunctionSnippets, r.FunctionSnippets},
		"GlobalFunctionResolution": {c.ResolvedCompletions().GlobalFunctionResolution, r.GlobalFunctionResolution},
		"DocBlock":                 {c.ResolvedDocBlock(), r.DocBlock},
	} {
		if !reflect.DeepEqual(pair[0], pair[1]) {
			t.Errorf("%s differs:\n  daemon.Config:  %#v\n  config.Resolve: %#v", name, pair[0], pair[1])
		}
	}

	if got := c.Mappings()[filepath.Base(dir)]; got != dir {
		t.Errorf("the config's own folder should map under its name: %q", got)
	}
}
