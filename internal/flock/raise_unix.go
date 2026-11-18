//go:build unix

package flock

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Raise ends this process by sig, as a process that had not caught it would
// end, so whatever ran it sees an interrupted run (a shell loop breaks on
// it, make stops) rather than an exit code. Only StopSignals:
// one that dumps a Go process's stacks (SIGQUIT, SIGABRT) would print them
// and exit 2. It returns if sig is not one of those, or if delivery somehow
// does not end the process, for the caller to exit with a code.
func Raise(sig os.Signal) {
	s, ok := sig.(syscall.Signal)
	if !ok || !Stopping(s) {
		return
	}

	signal.Reset(s)
	_ = syscall.Kill(os.Getpid(), s)

	time.Sleep(200 * time.Millisecond) // a bound; delivery ends this process within moments
}
