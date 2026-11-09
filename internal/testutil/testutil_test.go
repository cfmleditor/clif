package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTestdataIsTheRepositoryDirectory(t *testing.T) {
	root := Testdata()

	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		t.Fatalf("%s is not a directory: %v", root, err)
	}

	if _, err := os.Stat(Testdata("deps")); err != nil {
		t.Fatalf("a subdirectory is not found beneath it: %v", err)
	}

	if want := filepath.Join(root, "refs"); Testdata("refs") != want {
		t.Fatalf("Testdata(refs) = %s, want %s", Testdata("refs"), want)
	}
}
