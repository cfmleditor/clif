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

// TestVerifyRefusesAReusedPID: a record naming a live PID with another start
// time is a dead holder's, its PID given to a later process.
func TestVerifyRefusesAReusedPID(t *testing.T) {
	if _, ok := startTime(os.Getpid()); !ok {
		t.Skip("no process start time here")
	}

	path := filepath.Join(t.TempDir(), "lock")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = f.Close() }()

	if ok, err := TryLock(f); !ok || err != nil {
		t.Fatalf("TryLock = %v, %v", ok, err)
	}

	if _, err := f.WriteAt([]byte(strconv.Itoa(os.Getpid())+" tok 1.0\n"), 0); err != nil {
		t.Fatal(err)
	}

	if Verify(path, "tok") {
		t.Error("Verify = true for a record whose start time is not its PID's")
	}

	if err := Record(f, "tok"); err != nil {
		t.Fatal(err)
	}

	if !Verify(path, "tok") {
		t.Error("Verify = false for this process's own record")
	}
}
