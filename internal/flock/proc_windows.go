//go:build windows

package flock

import (
	"os"
	"os/exec"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Group runs a command in a job object that is closed, killing everything in
// it, when this process ends however it ends (taskkill without /T included):
// so the command cannot carry on unlocked after the lock's holder is gone.
// A console's Ctrl-C reaches every process attached to it already.
type Group struct {
	p *os.Process
}

// NewGroup puts this process in a kill-on-close job before cmd starts, so
// the command, and everything it starts, is in the job from its first
// instant (a process joins its creator's job): nothing can start outside it
// in a gap before it is assigned. If the job cannot be made, the command
// runs without one.
func NewGroup(*exec.Cmd, bool) *Group {
	jobOnce.Do(joinJob)

	return &Group{}
}

// jobOnce makes the job once, however many commands this process runs.
var jobOnce sync.Once

// job is held open, never closed, until this process ends.
var job windows.Handle

func joinJob() {
	j, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE

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

// Started records the command's process.
func (g *Group) Started(p *os.Process) { g.p = p }

// Wait waits for the command and returns its exit code.
func (g *Group) Wait() (int, error) {
	st, err := g.p.Wait()
	if err != nil {
		return 1, err
	}

	return st.ExitCode(), nil
}

// SignalGroup does nothing on Windows, which cannot send a process SIGTERM or
// SIGHUP: a console's Ctrl-C, or its closing, reaches every process attached
// to it, the command's included, and the job ends them all should this one be
// killed outright.
func SignalGroup(*os.Process, os.Signal) error { return nil }

// Reraise does nothing here: the caller exits with the command's code.
func (*Group) Reraise() {}
