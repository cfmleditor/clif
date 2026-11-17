// Package flock takes an exclusive, non-blocking lock on an open file: the
// operating system's own (flock on Unix, LockFileEx on Windows), so it is
// released however the holder ends, kill -9 and a power cut included, with no
// owner to guess at. It also has the process-group helpers the holder needs.
// scripts/docslock.go holds docs/.flock with it.
package flock

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cfmleditor/clif/internal/vfs"
)

// ErrUnsupported is returned where the platform has no file lock (wasm).
var ErrUnsupported = errors.ErrUnsupported

// Record writes the holder's record into f, which this process has locked:
// its PID, for a refused run to name, and token, which the processes it
// starts are given to prove they run under it (Verify). Nothing else is
// passed to them, no descriptor included, so nothing they leave behind can
// keep the lock.
func Record(f *os.File, token string) error {
	if err := f.Truncate(0); err != nil {
		return err
	}

	_, err := f.WriteAt([]byte(fmt.Sprintf("%d %s\n", os.Getpid(), token)), 0)

	return err
}

// Holder reads the record at path: the holder's PID and token, or "" for
// each when there is none.
func Holder(path string) (pid, token string) {
	b, err := vfs.ReadRegular(path)
	if err != nil {
		return "", ""
	}

	f := strings.Fields(string(b))
	if len(f) != 2 {
		return "", ""
	}

	return f[0], f[1]
}

// Verify reports whether token names the run holding the lock on path: the
// record there holds token, and some process holds the lock, which this
// tells by failing to take it. A file deleted and created again since the
// holder locked it holds no record, and a token leaked from a run that has
// ended matches none. The token is compared first, and a holder clears its
// record as it exits (scripts/docslock.go), so the probe, which takes a free
// lock for a moment (refusing a run that starts just then), runs only on a
// record a killed holder left behind.
func Verify(path, token string) bool {
	if token == "" {
		return false
	}

	if _, got := Holder(path); got != token {
		return false
	}

	f, err := os.OpenFile(path, os.O_RDWR, 0) //nolint:gosec // the lock file the caller names, opened only to test its lock
	if err != nil {
		return false
	}

	defer func() { _ = f.Close() }()

	free, err := TryLock(f)

	return err == nil && !free
}
