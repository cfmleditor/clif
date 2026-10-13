package vfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A workspace file that is not a regular file is refused rather than read:
// a symlink to /dev/zero read without end, and took a scan to 17GB.
func TestReadRegularRefusesADevice(t *testing.T) {
	if _, err := os.Stat("/dev/zero"); err != nil {
		t.Skip("no /dev/zero here")
	}

	link := filepath.Join(t.TempDir(), "Evil.cfc")
	if err := os.Symlink("/dev/zero", link); err != nil {
		t.Fatal(err)
	}

	for name, read := range map[string]func(string) error{
		"ReadRegular": func(p string) error {
			_, err := ReadRegular(p)

			return err
		},
		"ReadFile": func(p string) error {
			_, err := OS{}.ReadFile(p)

			return err
		},
		"ReadFileString": func(p string) error {
			_, err := OS{}.ReadFileString(p)

			return err
		},
	} {
		if err := read(link); !errors.Is(err, ErrNotRegular) {
			t.Errorf("%s of a symlink to /dev/zero = %v, want ErrNotRegular", name, err)
		}
	}
}

// A FIFO is refused at once: reading one waits for a writer that never comes,
// and so did indexing a workspace that held one.
func TestReadRegularDoesNotWaitOnAFIFO(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "Fifo.cfc")
	if err := mkfifo(fifo); err != nil {
		t.Skipf("cannot make a FIFO here: %v", err)
	}

	done := make(chan error, 1)

	go func() {
		_, err := ReadRegular(fifo)
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, ErrNotRegular) {
			t.Errorf("ReadRegular of a FIFO = %v, want ErrNotRegular", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ReadRegular blocked on a FIFO")
	}
}

// A file past MaxFileSize is refused without being read.
func TestReadRegularRefusesAHugeFile(t *testing.T) {
	big := filepath.Join(t.TempDir(), "Big.cfc")

	f, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}

	// Sparse: the size is what is checked, so nothing is written.
	if err := f.Truncate(MaxFileSize + 1); err != nil {
		t.Fatal(err)
	}

	_ = f.Close()

	if _, err := ReadRegular(big); !errors.Is(err, ErrTooLarge) {
		t.Errorf("ReadRegular of a %d-byte file = %v, want ErrTooLarge", MaxFileSize+1, err)
	}
}

// An ordinary file, and one reached through a symlink, read as they are.
func TestReadRegularReadsAFile(t *testing.T) {
	dir := t.TempDir()
	want := "component { function f() {} }\n"

	file := filepath.Join(dir, "Ok.cfc")
	if err := os.WriteFile(file, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "Link.cfc")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{file, link} {
		got, err := ReadRegular(p)
		if err != nil || string(got) != want {
			t.Errorf("ReadRegular(%s) = %q, %v; want %q", filepath.Base(p), got, err, want)
		}
	}

	empty := filepath.Join(dir, "Empty.cfc")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if got, err := ReadRegular(empty); err != nil || len(got) != 0 {
		t.Errorf("ReadRegular of an empty file = %q, %v", got, err)
	}
}
