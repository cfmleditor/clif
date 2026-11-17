//go:build (darwin || dragonfly || freebsd || netbsd || openbsd) && !linux

package flock

import "syscall"

// dieWithParent: no such kernel facility here; the docs scripts' lock checks
// before each change are what stop a run whose docslock was killed.
func dieWithParent(*syscall.SysProcAttr) {}
