// Package flock takes an exclusive, non-blocking lock on an open file: the
// operating system's own (flock on Unix, LockFileEx on Windows), so it is
// released however the holder ends, kill -9 and a power cut included, with no
// owner to record or guess at. scripts/docslock.go holds docs/.lock with it.
package flock

import (
	"errors"
	"os"
)

// ErrUnsupported is returned where the platform has no file lock (wasm).
var ErrUnsupported = errors.ErrUnsupported

// Holds reports whether f is an open of the file at path as it is now on
// disk, so a lock taken on f still guards that file: false once the file has
// been deleted, or deleted and created again, since f was opened. Where a
// child is passed no descriptor (Windows, f nil) it reports whether some
// process holds the lock on path, by failing to take it.
func Holds(f *os.File, path string) bool {
	if f == nil {
		g, err := os.OpenFile(path, os.O_RDWR, 0) //nolint:gosec // the lock file the caller names, opened only to test its lock
		if err != nil {
			return false
		}

		defer func() { _ = g.Close() }()

		free, err := TryLock(g)

		return err == nil && !free
	}

	held, err := f.Stat()
	if err != nil {
		return false
	}

	onDisk, err := os.Stat(path)

	return err == nil && os.SameFile(held, onDisk)
}
