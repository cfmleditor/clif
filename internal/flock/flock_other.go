//go:build aix || (!unix && !windows)

package flock

import "os"

// TryLock is unsupported here: wasm, and AIX, which has no flock (an fcntl
// lock would do, but nothing has needed one there).
func TryLock(*os.File) (bool, error) { return false, ErrUnsupported }

// alive cannot tell here, so takes any PID for a live one.
func alive(pid int) bool { return pid > 0 }

// Inherited returns nil here.
func Inherited(int, string, string) *os.File { return nil }
