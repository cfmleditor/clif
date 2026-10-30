package path

import (
	"os"
	"path/filepath"
	"testing"
)

// A file that appears is seen once the app-mapping cache is invalidated,
// which the server does when a watched file changes: probeRead remembers a
// missing file until then, and must not outlive that.
func TestInvalidatingTheMappingsForgetsWhatTheWalkRead(t *testing.T) {
	file := filepath.Join(t.TempDir(), "server.json")

	if _, err := probeRead(file); err == nil {
		t.Fatal("read a file that does not exist")
	}

	if err := os.WriteFile(file, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := probeRead(file); err == nil {
		t.Fatal("a missing file was not remembered until invalidation")
	}

	InvalidateAppMappingsCache()

	data, err := probeRead(file)
	if err != nil || string(data) != "{}" {
		t.Fatalf("after invalidation: %q, %v; want the file", data, err)
	}
}
