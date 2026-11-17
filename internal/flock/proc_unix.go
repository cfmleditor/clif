//go:build unix

package flock

import (
	"errors"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

// NewGroup makes cmd start as the leader of a process group of its own, so
// SignalGroup reaches it and everything it runs, and a terminal's Ctrl-C
// (sent to the terminal's foreground group) reaches only the caller, which
// passes it on once.
func NewGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// SignalGroup sends sig to the process group p leads (NewGroup).
func SignalGroup(p *os.Process, sig os.Signal) error {
	s, ok := sig.(syscall.Signal)
	if !ok {
		return p.Signal(sig)
	}

	return unix.Kill(-p.Pid, s)
}

// ProcessGone reports whether no process has pid: only "no such process"
// counts, never "not permitted" (a live process of another user).
func ProcessGone(pid int) bool {
	return pid > 0 && errors.Is(unix.Kill(pid, 0), unix.ESRCH)
}
