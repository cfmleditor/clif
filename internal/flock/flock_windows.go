//go:build windows

package flock

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// TryLock takes an exclusive lock on f without waiting. It reports false,
// with no error, when another handle holds it. Windows releases it when the
// handle is closed, or its process ends however it ends.
func TryLock(f *os.File) (bool, error) {
	var ol windows.Overlapped

	// A byte far past the end, not the file's first: a lock on Windows
	// also bars every other handle from reading what it covers, and the
	// holder's PID is written at the start for a refused run to name.
	ol.OffsetHigh = 1 << 30

	err := windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &ol)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return false, nil
	}

	return err == nil, err
}
