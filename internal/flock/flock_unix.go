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

// alive reports whether a process has pid: "no such process" alone counts as
// gone, never "not permitted" (a live process of another user).
func alive(pid int) bool {
	return pid > 0 && !errors.Is(unix.Kill(pid, 0), unix.ESRCH)
}

// Inherited returns the descriptor fd as a file, marked close-on-exec, if it
// is an open of path as path is now on disk and the record there holds token:
// the lock's descriptor, passed to this process by its holder. Anything else
// at fd (a leaked DOCS_LOCK_FD, an unrelated descriptor) is not touched, and
// it returns nil. It looks with fstat, not through an *os.File, whose
// finalizer would close a descriptor that is not its to close.
func Inherited(fd int, path, token string) *os.File {
	if fd < 3 {
		return nil
	}

	var st unix.Stat_t
	if unix.Fstat(fd, &st) != nil {
		return nil
	}

	var onDisk unix.Stat_t
	if unix.Stat(path, &onDisk) != nil || st.Dev != onDisk.Dev || st.Ino != onDisk.Ino {
		return nil
	}

	if _, got := Holder(path); token == "" || got != token {
		return nil
	}

	unix.CloseOnExec(fd)

	return os.NewFile(uintptr(fd), path)
}

// Unlock releases a lock TryLock took on f. Closing f releases it too, but
// at once only on Unix: called explicitly, so the release never waits on the
// close.
func Unlock(f *os.File) { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN) }
