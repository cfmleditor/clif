//go:build linux

package flock

import (
	"os/exec"
	"runtime"
	"syscall"
)

// DieWithParent has the kernel kill cmd should this process die, even by
// SIGKILL. Only cmd itself: what it starts carries on, and the docs scripts
// check the lock before each change (scripts/lib/underlock.sh), so they stop
// rather than change anything.
//
// Linux sends the signal when the thread that started cmd exits, not the
// process (golang/go#27505), so this locks the calling goroutine to its
// thread for good: start cmd from the same goroutine, the main one.
func DieWithParent(cmd *exec.Cmd) {
	runtime.LockOSThread()

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
}
