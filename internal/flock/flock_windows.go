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

	err := windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &ol)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return false, nil
	}

	return err == nil, err
}
