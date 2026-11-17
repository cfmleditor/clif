//go:build windows

package flock

import (
	"os"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Group runs a command in a job object that is closed, killing everything in
// it, when this process ends however it ends (taskkill without /T included):
// so the command cannot carry on unlocked after the lock's holder is gone.
// A console's Ctrl-C reaches every process attached to it already.
type Group struct {
	p   *os.Process
	job windows.Handle
}

// NewGroup does nothing to cmd: the job is made once it has started.
func NewGroup(*exec.Cmd) *Group { return &Group{} }

// Started puts the command in a kill-on-close job. What it starts from then
// on joins the job too. If the job cannot be made, the command runs without.
func (g *Group) Started(p *os.Process) {
	g.p = p

	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE

	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)

		return
	}

	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(p.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)

		return
	}

	defer func() { _ = windows.CloseHandle(h) }()

	if windows.AssignProcessToJobObject(job, h) != nil {
		_ = windows.CloseHandle(job)

		return
	}

	g.job = job // held open, never closed, until this process ends
}

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
