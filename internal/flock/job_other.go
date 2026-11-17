//go:build !windows

package flock

import "os/exec"

// JoinJob does nothing here: see DieWithParent.
func JoinJob() {}

// Breakaway does nothing here, and reports false.
func Breakaway(*exec.Cmd) bool { return false }

// NoBreakaway does nothing here.
func NoBreakaway(*exec.Cmd) {}

// BreakawayRefused is false here, where nothing breaks away.
func BreakawayRefused(error) bool { return false }
