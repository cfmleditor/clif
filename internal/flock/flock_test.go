package flock

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTryLockExcludesASecondOpen: a second open of the same file cannot take
// the lock while the first holds it, and can once the first is closed, which
// is all a crashed holder amounts to.
func TestTryLockExcludesASecondOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")

	open := func() *os.File {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			t.Fatal(err)
		}

		return f
	}

	first := open()

	if ok, err := TryLock(first); !ok || err != nil {
		t.Fatalf("first TryLock = %v, %v; want true", ok, err)
	}

	second := open()
	defer func() { _ = second.Close() }()

	if ok, err := TryLock(second); ok || err != nil {
		t.Fatalf("second TryLock while the first holds it = %v, %v; want false, nil", ok, err)
	}

	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	if ok, err := TryLock(second); !ok || err != nil {
		t.Fatalf("TryLock after the holder closed = %v, %v; want true", ok, err)
	}
}
