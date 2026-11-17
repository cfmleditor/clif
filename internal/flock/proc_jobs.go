//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package flock

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// Group runs a command as a process group of its own, as a shell runs a job:
// SignalGroup reaches it and everything it runs at once, and it is given the
// terminal while it runs, if this process has it, so it reads and writes the
// terminal as any job does (a git prompt, `stty tostop`) and a Ctrl-C there
// reaches its processes once. Nothing it starts inherits an ignored signal.
type Group struct {
	tty    *os.File       // the controlling terminal, if this process has one
	pid    int            // the command's process, and its group
	handed bool           // the terminal was given to the command at some point
	ended  syscall.Signal // the signal that ended the command, if one did
}

// NewGroup makes cmd start as the leader of a process group of its own.
// Call Started once it has, then Wait.
//
// It locks the calling goroutine to its OS thread, for good: Linux sends the
// command its parent-death signal (dieWithParent) when the thread that
// started it exits, not the process (golang/go#27505), and a thread a
// goroutine is locked to cannot exit while the goroutine lives. So call it,
// Start the command and Wait from the main goroutine.
func NewGroup(cmd *exec.Cmd) *Group {
	runtime.LockOSThread()

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	dieWithParent(cmd.SysProcAttr)

	return &Group{}
}

// Started hands the terminal to the command's group if this process's group
// has it. From here on this process may be in the background, where taking
// the terminal back would raise SIGTTOU: that is ignored here, after the
// command has started, so the command does not inherit it.
func (g *Group) Started(p *os.Process) {
	g.pid = p.Pid

	signal.Ignore(syscall.SIGTTOU)

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return // no controlling terminal: nothing to hand over
	}

	g.tty = tty
	g.handOver()
}

// Wait waits for the command to end and returns its exit code, 128 + the
// signal's number if a signal ended it, doing a shell's part when it stops:
//
//   - stopped with the terminal (a Ctrl-Z): the terminal comes back, and this
//     process's own job stops with it, so the shell reports the job stopped;
//     `fg` resumes this, which hands the terminal over again and resumes the
//     command (`bg` resumes it without).
//   - stopped wanting a terminal this process has: it is handed over and the
//     command resumed.
//   - stopped wanting a terminal this process lacks (its job is in the
//     background): this job stops with it, as above.
//   - with no terminal at all, someone stopped it on purpose, and resumes it.
func (g *Group) Wait() (int, error) {
	for {
		var ws unix.WaitStatus

		_, err := unix.Wait4(g.pid, &ws, unix.WUNTRACED, nil)
		if errors.Is(err, unix.EINTR) {
			continue
		}

		if err != nil {
			return 1, err
		}

		switch {
		case ws.Exited():
			g.takeBack()

			return ws.ExitStatus(), nil
		case ws.Signaled():
			g.takeBack()
			g.ended = ws.Signal()

			return 128 + int(ws.Signal()), nil
		case ws.Stopped():
			g.stopped()
		}
	}
}

func (g *Group) stopped() {
	if g.tty == nil {
		return
	}

	switch g.foreground() {
	case g.pid: // a Ctrl-Z: stop with it
		g.takeBack()
		stopSelf()
	case unix.Getpgrp(): // it wanted the terminal, which is this group's to give
	default: // the terminal is another job's: stop until resumed
		stopSelf()
	}

	g.handOver()
	_ = unix.Kill(-g.pid, unix.SIGCONT)
}

// stopSelf stops this process's job, the make that ran it included, as a
// stopped job is; it carries on here once resumed.
func stopSelf() { _ = unix.Kill(-unix.Getpgrp(), unix.SIGSTOP) }

// foreground is the terminal's foreground process group, or -1.
func (g *Group) foreground() int {
	fg, err := unix.IoctlGetInt(int(g.tty.Fd()), unix.TIOCGPGRP)
	if err != nil {
		return -1
	}

	return fg
}

// handOver gives the command's group the terminal, if this group has it.
func (g *Group) handOver() {
	if g.tty != nil && g.foreground() == unix.Getpgrp() {
		setForeground(g.tty, g.pid)
		g.handed = true
	}
}

// Reraise ends this process the way the command ended, if a signal ended
// it; call it once the command has ended and the caller has cleaned up. Then
// whatever ran this one sees an interrupted run rather than an exit code (a
// shell loop breaks on it, `make -k` stops). A Ctrl-C reached only the command,
// which had the terminal: it is sent on to this process's whole job, the
// make that ran it included, as the terminal would have. Any other signal
// ends this process alone. It returns only if no signal ended the command.
func (g *Group) Reraise() {
	if g.ended == 0 {
		return
	}

	signal.Reset(g.ended)

	if g.ended == unix.SIGINT && g.handed {
		_ = unix.Kill(-unix.Getpgrp(), unix.SIGINT)
	} else {
		_ = unix.Kill(os.Getpid(), g.ended)
	}

	time.Sleep(time.Second) // the signal's delivery is not immediate
}

// takeBack returns the terminal to this group, if the command's has it.
func (g *Group) takeBack() {
	if g.tty != nil && g.foreground() == g.pid {
		setForeground(g.tty, unix.Getpgrp())
	}
}

// SignalGroup sends sig to the process group p leads (NewGroup).
func SignalGroup(p *os.Process, sig os.Signal) error {
	s, ok := sig.(syscall.Signal)
	if !ok {
		return p.Signal(sig)
	}

	return unix.Kill(-p.Pid, s)
}

// setForeground makes pgid the terminal's foreground process group.
func setForeground(tty *os.File, pgid int) {
	_ = unix.IoctlSetPointerInt(int(tty.Fd()), unix.TIOCSPGRP, pgid)
}
