//go:build !linux

package flock

import "os/exec"

// DieWithParent does nothing here, where the kernel has no such facility (on
// Windows, JoinJob does the same for everything at once); the docs scripts'
// lock checks before each change are what stop a run whose docslock was
// killed.
func DieWithParent(*exec.Cmd) {}
