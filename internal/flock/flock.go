// Package flock takes an exclusive, non-blocking lock on an open file: the
// operating system's own (flock on Unix, LockFileEx on Windows), so it is
// released however the holder ends, kill -9 and a power cut included, with no
// owner to record or guess at. scripts/docslock.go holds docs/.lock with it.
package flock

import "errors"

// ErrUnsupported is returned where the platform has no file lock (wasm).
var ErrUnsupported = errors.ErrUnsupported
