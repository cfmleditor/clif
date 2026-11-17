//go:build unix && !aix

package flock

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// TryLock takes an exclusive lock on f without waiting. It reports false,
// with no error, when another open of the file holds it. The lock belongs to
// the open file: it lasts while any process holds a descriptor for it
// (a child that inherited one included), and goes when the last is closed.
func TryLock(f *os.File) (bool, error) {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) {
		return false, nil
	}

	return err == nil, err
}

// CloseOnExec stops f's descriptor reaching any program this process starts,
// so nothing it runs (a daemonizing git, say) can keep the lock past it.
func CloseOnExec(f *os.File) { unix.CloseOnExec(int(f.Fd())) }

// ProcessGone reports whether no process has pid: only "no such process"
// counts, never "not permitted" (a live process of another user).
func ProcessGone(pid int) bool {
	return pid > 0 && errors.Is(unix.Kill(pid, 0), unix.ESRCH)
}
