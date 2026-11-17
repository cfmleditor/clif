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
