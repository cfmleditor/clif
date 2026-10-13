//go:build unix

package server

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStandaloneSkipsAConfigOthersCanWrite: standalone mode's walk applies the
// same rule as daemon.FindConfig, so a world-writable config is passed over
// for the trusted one above it.
func TestStandaloneSkipsAConfigOthersCanWrite(t *testing.T) {
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	child := filepath.Join(parent, "shared")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	trusted := filepath.Join(parent, ".clif.json")
	write(t, trusted, `{"workspaceName":"trusted"}`)

	planted := filepath.Join(child, ".clif.json")
	write(t, planted, `{"workspaceName":"planted"}`)

	if err := os.Chmod(planted, 0o666); err != nil {
		t.Fatal(err)
	}

	if p, cfg := newTestServer().findConfigUpwards(child); cfg == nil || p != trusted {
		t.Fatalf("found %q; want %q", p, trusted)
	}
}
