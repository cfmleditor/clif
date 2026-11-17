//go:build !unix && !windows

package flock

import "os"

// TryLock is unsupported here.
func TryLock(*os.File) (bool, error) { return false, ErrUnsupported }
