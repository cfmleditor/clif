//go:build windows

package flock

import (
	"errors"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// job is held open, never closed, until this process ends.
var job windows.Handle

// JoinJob puts this process in a job that is closed, killing everything in
// it, when this process ends however it ends (taskkill without /T included),
// before it starts anything: a process joins its creator's job, so all a run
// starts is in it from its first instant and cannot carry on unlocked after
// docslock is gone. Breakaway is allowed, for Breakaway. If the job cannot be
// made, the run goes on without it.
func JoinJob() {
	j, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_BREAKAWAY_OK

	if _, err := windows.SetInformationJobObject(j, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(j)

		return
	}

	if windows.AssignProcessToJobObject(j, windows.CurrentProcess()) != nil {
		_ = windows.CloseHandle(j)

		return
	}

	job = j
}

// Breakaway starts cmd outside the job, so it is not killed with this
// process: for the generator, better left to finish its write (both renames)
// than cut off between them, which would leave the generated files out of
// step. Only from the job JoinJob made: with none, there is nothing to break
// away from, and a job around this process that forbids breakaway (an IDE's,
// a CI agent's) would refuse to start cmd at all. It reports whether it asked
// for breakaway, so a caller whose start is refused anyway (a job around this
// one that forbids it) can start cmd again without (NoBreakaway).
func Breakaway(cmd *exec.Cmd) bool {
	if job == 0 {
		return false
	}

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	cmd.SysProcAttr.CreationFlags |= windows.CREATE_BREAKAWAY_FROM_JOB

	return true
}

// NoBreakaway undoes Breakaway on a command not yet started.
func NoBreakaway(cmd *exec.Cmd) {
	if cmd.SysProcAttr != nil {
		cmd.SysProcAttr.CreationFlags &^= windows.CREATE_BREAKAWAY_FROM_JOB
	}
}

// BreakawayRefused reports whether err, from starting a command Breakaway
// marked, is Windows refusing the breakaway (a job around this process that
// forbids it), the one failure worth starting it again without.
func BreakawayRefused(err error) bool { return errors.Is(err, windows.ERROR_ACCESS_DENIED) }
