package vfs

import (
	"os"
	"path/filepath"
	"testing"
)

// ReadString through vfs.OS, directly and through the batch cache, gives the
// file's text, an empty file included.
func TestReadStringGivesTheFilesText(t *testing.T) {
	dir := t.TempDir()

	for name, content := range map[string]string{"a.cfm": "<cfset x = 1>\n", "empty.cfm": ""} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		for _, fsys := range []FS{OS{}, CacheMetadata(OS{})} {
			got, err := ReadString(fsys, path)
			if err != nil || got != content {
				t.Errorf("%T: ReadString(%s) = %q, %v; want %q", fsys, name, got, err, content)
			}
		}
	}

	if _, err := ReadString(OS{}, filepath.Join(dir, "missing.cfm")); err == nil {
		t.Error("a missing file read without an error")
	}
}
