package path

import (
	"os"
	"path/filepath"
	"testing"
)

// The editor's cache keeps only directories a lookup found its segment in,
// so a component written later is still found; a snapshot cache keeps every
// listing, since nothing is written while it lives.
func TestOnlyASnapshotDirCacheKeepsMisses(t *testing.T) {
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

		if found := resolveSegmentsCached(dir, segments, tc.cache) != ""; found != tc.findNew {
			t.Errorf("%s: found the new file = %v, want %v", tc.name, found, tc.findNew)
		}

		if err := os.Remove(filepath.Join(dir, "New.cfc")); err != nil {
			t.Fatal(err)
		}
	}
}
