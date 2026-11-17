//go:build !unix

package flock

import (
	"os"
	"os/exec"
)

// NewGroup does nothing here: a console's Ctrl-C reaches every process
// attached to it.
func NewGroup(*exec.Cmd) {}

// SignalGroup signals p alone, as far as the platform can.
func SignalGroup(p *os.Process, sig os.Signal) error { return p.Signal(sig) }

// ProcessGone cannot tell here (a PID an MSYS shell recorded is not a Windows
// one), so it never says so.
func ProcessGone(int) bool { return false }
