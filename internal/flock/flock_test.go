package flock

import (
	"os"
	"path/filepath"
	"strconv"
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

func TestVerify(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = f.Close() }()

	if ok, err := TryLock(f); !ok || err != nil {
		t.Fatalf("TryLock = %v, %v", ok, err)
	}

	if err := Record(f, "tok1"); err != nil {
		t.Fatal(err)
	}

	if pid, tok := Holder(path); pid != strconv.Itoa(os.Getpid()) || tok != "tok1" {
		t.Errorf("Holder = %q, %q", pid, tok)
	}

	if !Verify(path, "tok1") {
		t.Error("Verify(the holder's token) = false")
	}

	for _, tok := range []string{"", "tok2"} {
		if Verify(path, tok) {
			t.Errorf("Verify(%q) = true", tok)
		}
	}

	// Released: the record is still there, but nobody holds the lock.
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if Verify(path, "tok1") {
		t.Error("Verify = true with the lock released")
	}
}
