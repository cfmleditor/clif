//go:build !unix

package flock

import (
	"os"
	"os/exec"
)

// Group is a command's process group; nothing here.
type Group struct{}

// NewGroup does nothing here: a console's Ctrl-C reaches every process
// attached to it.
func NewGroup(*exec.Cmd) *Group { return &Group{} }

// Restore does nothing here.
func (*Group) Restore() {}

// SignalGroup signals p alone, as far as the platform can.
func SignalGroup(p *os.Process, sig os.Signal) error { return p.Signal(sig) }
