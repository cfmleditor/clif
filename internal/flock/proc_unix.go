//go:build unix

package flock

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// Group runs a command as a process group of its own: SignalGroup reaches it
// and everything it runs at once.
type Group struct {
	foreground bool // it was given the terminal, which Restore takes back
}

// NewGroup makes cmd start as the leader of a process group of its own. When
// this process is in the terminal's foreground, the new group is put there
// in its place, as a shell does a job: a Ctrl-C then reaches the command's
// processes once, and they can read the terminal (a git credential prompt)
// and write to it under `stty tostop` without being stopped. Ctrl-Z is
// ignored, by this process and the command alike (an ignored signal stays
// ignored across exec): a run stopped holding the lock, with the terminal
// and no shell job to resume it by, would hold it for good. Call Restore once
// the command has ended.
func NewGroup(cmd *exec.Cmd) *Group {
	signal.Ignore(syscall.SIGTSTP)

	g := &Group{}
	attr := &syscall.SysProcAttr{Setpgid: true}

	if inForeground() {
		attr.Foreground, attr.Ctty = true, 0 // stdin, in the child too
		g.foreground = true
	}

	cmd.SysProcAttr = attr

	return g
}

// Restore takes the terminal back for this process's group, if NewGroup gave
// it away. Doing so from a background group raises SIGTTOU, ignored here.
func (g *Group) Restore() {
	if !g.foreground {
		return
	}

	signal.Ignore(syscall.SIGTTOU)
	takeForeground()
}

// SignalGroup sends sig to the process group p leads (NewGroup).
func SignalGroup(p *os.Process, sig os.Signal) error {
	s, ok := sig.(syscall.Signal)
	if !ok {
		return p.Signal(sig)
	}

	return unix.Kill(-p.Pid, s)
}
