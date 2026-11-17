//go:build unix && !aix

package flock

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// TryLock takes an exclusive lock on f without waiting. It reports false,
// with no error, when another open of the file holds it. The lock belongs to
// the open file, and goes when it is closed or its process ends however it
// ends. scripts/docslock.go passes it to nothing it runs.
func TryLock(f *os.File) (bool, error) {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) {
		return false, nil
	}

	return err == nil, err
}

// CloseOnExec stops f's descriptor reaching any program this process starts.
func CloseOnExec(f *os.File) { unix.CloseOnExec(int(f.Fd())) }

// alive reports whether a process has pid: "no such process" alone counts as
// gone, never "not permitted" (a live process of another user).
func alive(pid int) bool {
	return pid > 0 && !errors.Is(unix.Kill(pid, 0), unix.ESRCH)
}
