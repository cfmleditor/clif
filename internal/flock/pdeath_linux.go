//go:build linux

package flock

import "syscall"

// dieWithParent has the kernel kill the command if this process dies, even by
// SIGKILL (a group-wide kill of the make that ran it, say, which no longer
// reaches the command's own process group). Only the command itself: what it
// has started carries on, and the docs scripts check the lock before each
// change (lib/underlock.sh), so they stop rather than change anything.
func dieWithParent(a *syscall.SysProcAttr) { a.Pdeathsig = syscall.SIGKILL }
