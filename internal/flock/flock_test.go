package flock

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestProcessGone(t *testing.T) {
	if ProcessGone(os.Getpid()) {
		t.Error("ProcessGone(this process) = true")
	}

	if ProcessGone(0) || ProcessGone(-1) {
		t.Error("ProcessGone of a non-PID = true")
	}
}

func TestHolds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = f.Close() }()

	if ok, err := TryLock(f); !ok || err != nil {
		t.Fatalf("TryLock = %v, %v", ok, err)
	}

	if !Holds(f, path) {
		t.Error("Holds = false for the locked file itself")
	}

	if !Holds(nil, path) {
		t.Error("Holds(nil) = false while the file is locked")
	}

	// Deleted and created again: f locks a file no longer at path. Windows
	// will not delete a file a process has open.
	if runtime.GOOS == "windows" {
		return
	}

	// Deleted and created again: f locks a file no longer at path.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if Holds(f, path) {
		t.Error("Holds = true after the file was replaced")
	}

	if Holds(nil, path) {
		t.Error("Holds(nil) = true for a replacement nobody locks")
	}
}
