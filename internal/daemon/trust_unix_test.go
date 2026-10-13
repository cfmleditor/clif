//go:build unix

package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFindConfigSkipsAConfigOthersCanWrite: a .clif.json any local user can
// write, as one planted in /tmp would be, does not govern the project below
// it, and the walk goes on to the trusted config further up.
func TestFindConfigSkipsAConfigOthersCanWrite(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "shared")
	writeConfig(t, parent, `{"workspaceName":"trusted"}`)

	planted := filepath.Join(child, ".clif.json")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(planted, []byte(`{"workspaceName":"planted"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(planted, 0o666); err != nil {
		t.Fatal(err)
	}

	cfg, err := FindConfig(child)
	if err != nil {
		t.Fatal(err)
	}

	if cfg == nil || cfg.Name != "trusted" {
		t.Fatalf("found %+v; want the parent's trusted config", cfg)
	}
}
