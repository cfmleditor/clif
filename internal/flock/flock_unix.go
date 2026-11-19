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
// ends. the docs driver passes it to no child.
func TryLock(f *os.File) (bool, error) {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) {
		return false, nil
	}

	return err == nil, err
}

// Unlock releases a lock TryLock took on f. Closing f releases it too, but
// at once only on Unix: called explicitly, so the release never waits on the
// close.
func Unlock(f *os.File) { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN) }
