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

// CloseOnExec does nothing on Windows, which passes a child no handle it was
// not given.
func CloseOnExec(*os.File) {}

// ProcessGone reports whether no process has pid, as far as Windows will
// say: a process it refuses to open (another user's) counts as alive.
func ProcessGone(pid int) bool {
	if pid <= 0 {
		return false
	}

	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return errors.Is(err, windows.ERROR_INVALID_PARAMETER)
	}

	defer func() { _ = windows.CloseHandle(h) }()

	var code uint32
	if windows.GetExitCodeProcess(h, &code) != nil {
		return false
	}

	return code != 259 // STILL_ACTIVE
}
