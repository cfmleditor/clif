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

// alive reports whether a process has pid, as far as Windows says: one it
// will not open (another user's) counts as alive.
func alive(pid int) bool {
	if pid <= 0 {
		return false
	}

	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return !errors.Is(err, windows.ERROR_INVALID_PARAMETER)
	}

	defer func() { _ = windows.CloseHandle(h) }()

	var code uint32
	if windows.GetExitCodeProcess(h, &code) != nil {
		return true
	}

	return code == 259 // STILL_ACTIVE
}

// Inherited returns nil on Windows, which passes a child no descriptor: the
// caller checks the lock by its token instead.
func Inherited(int, string, string) *os.File { return nil }

// Unlock releases a lock TryLock took on f, explicitly: Windows releases one
// still held when its handle closes only after an unspecified delay, during
// which a run starting would be refused.
func Unlock(f *os.File) {
	var ol windows.Overlapped
	ol.OffsetHigh = 1 << 30 // the byte TryLock locks

	_ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &ol)
}
