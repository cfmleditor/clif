//go:build aix || (!unix && !windows)

package flock

import "os"

// TryLock is unsupported here: wasm, and AIX, which has no flock (an fcntl
// lock would do, but nothing has needed one there).
func TryLock(*os.File) (bool, error) { return false, ErrUnsupported }
