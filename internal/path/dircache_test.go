package path

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The editor's cache keeps only directories a lookup found its segment in,
// so a component written later is still found (once MissTTL has passed); a
// snapshot cache keeps every listing, since nothing is written while it lives.
func TestOnlyASnapshotDirCacheKeepsMisses(t *testing.T) {
	old := MissTTL
	MissTTL = 20 * time.Millisecond

	t.Cleanup(func() { MissTTL = old })

	dir := t.TempDir()
	segments := []string{"New.cfc"}

	for _, tc := range []struct {
		name    string
		cache   *DirCache
		findNew bool
	}{
		{"editor", NewDirCache(), true},
		{"snapshot", NewSnapshotDirCache(), false},
	} {
		if got := resolveSegmentsCached(dir, segments, tc.cache); got != "" {
			t.Fatalf("%s: found %s before it was written", tc.name, got)
		}

		if err := os.WriteFile(filepath.Join(dir, "New.cfc"), []byte("component {}"), 0o600); err != nil {
			t.Fatal(err)
		}

		// Within MissTTL the editor's cache still answers from the listing it
		// read; neither finds the file yet.
		if resolveSegmentsCached(dir, segments, tc.cache) != "" {
			t.Errorf("%s: found the new file within MissTTL", tc.name)
		}

		time.Sleep(2 * MissTTL)

		if found := resolveSegmentsCached(dir, segments, tc.cache) != ""; found != tc.findNew {
			t.Errorf("%s: found the new file = %v, want %v", tc.name, found, tc.findNew)
		}

		if err := os.Remove(filepath.Join(dir, "New.cfc")); err != nil {
			t.Fatal(err)
		}
	}
}
