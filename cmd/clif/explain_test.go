package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func explainFixture(t *testing.T) (root, file string) {
	t.Helper()

	root = t.TempDir()

	for name, src := range map[string]string{
		"models/Item.cfc": `component { function ping(){} }`,
		"svc/Svc.cfc": `component {
function use( required thing ){ arguments.thing.ping(); }
function run(){ use( new models.Item() ); }
}`,
	} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	return root, filepath.Join(root, "svc", "Svc.cfc")
}

// TestExplainTypesAnArgumentFromItsCallersByDefault: explain traces the verdict
// the unresolved report gives, and the report types an untyped argument from
// what every caller passes it. A trace without that said "unresolved" about a
// call the report accepted. --no-infer-args is the report's flag, for the
// trace of a run that used it.
func TestExplainTypesAnArgumentFromItsCallersByDefault(t *testing.T) {
	root, file := explainFixture(t)

	out := captureStdout(t, func() { cmdExplain([]string{"--root", root, file, "2"}) })
	if !strings.Contains(out, "what every caller of use passes it") || !strings.Contains(out, "=> resolved") {
		t.Errorf("explain did not infer the argument:\n%s", out)
	}

	if !strings.HasSuffix(strings.TrimSpace(out), "report: counted as resolved") {
		t.Errorf("no report line:\n%s", out)
	}

	out = captureStdout(t, func() { cmdExplain([]string{"--root", root, "--no-infer-args", file, "2"}) })
	if strings.Contains(out, "what every caller of use passes it") || !strings.Contains(out, "=> unresolved") {
		t.Errorf("--no-infer-args still inferred the argument:\n%s", out)
	}
}
