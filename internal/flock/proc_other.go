//go:build !(darwin || dragonfly || freebsd || linux || netbsd || openbsd) && !windows

package flock

import (
	"os"
	"os/exec"
	"syscall"
)

// Group is a command started as is (AIX, Solaris, wasm): no process group or
// terminal handover here.
type Group struct{ p *os.Process }

// NewGroup does nothing to cmd here.
func NewGroup(*exec.Cmd, bool) *Group { return &Group{} }

// Started records the command's process.
func (g *Group) Started(p *os.Process) { g.p = p }

// Wait waits for the command and returns its exit code, 128 + the signal's
// number if a signal ended it.
func (g *Group) Wait() (int, error) {
	st, err := g.p.Wait()
	if err != nil {
		return 1, err
	}

	if ws, ok := st.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal()), nil
	}

	return st.ExitCode(), nil
}

// SignalGroup signals p alone here.
func SignalGroup(p *os.Process, sig os.Signal) error { return p.Signal(sig) }

// Reraise does nothing here: the caller exits with the command's code.
func (*Group) Reraise() {}
