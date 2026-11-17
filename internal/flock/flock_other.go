//go:build aix || (!unix && !windows)

package flock

import "os"

// TryLock is unsupported here (wasm; AIX, which has no flock: an fcntl lock
// would not pass to a child, which is the point of this one).
func TryLock(*os.File) (bool, error) { return false, ErrUnsupported }

// CloseOnExec does nothing here.
func CloseOnExec(*os.File) {}

// ProcessGone cannot tell here, so it never says so.
func ProcessGone(int) bool { return false }
