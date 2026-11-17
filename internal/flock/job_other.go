//go:build !windows

package flock

import "os/exec"

// JoinJob does nothing here: see DieWithParent.
func JoinJob() {}

// Breakaway does nothing here.
func Breakaway(*exec.Cmd) {}
