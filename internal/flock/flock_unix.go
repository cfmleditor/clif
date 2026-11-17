//go:build unix

package flock

import (
	"errors"
	"os"
	"syscall"
)

// TryLock takes an exclusive lock on f without waiting. It reports false,
// with no error, when another open of the file holds it. The lock belongs to
// the open file: it lasts while any process holds a descriptor for it
// (a child that inherited one included), and goes when the last is closed.
func TryLock(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return false, nil
	}

	return err == nil, err
}
