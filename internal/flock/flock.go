// Package flock provides an OS file lock and the docs driver's stop signals.
package flock

import (
	"errors"
	"os"
	"os/signal"
	"slices"
	"syscall"
)

// ErrUnsupported is returned on platforms without OS file locks.
var ErrUnsupported = errors.ErrUnsupported

// Exit ends this process by sig (Raise), so what ran it sees an interrupted
// run, or, should it not end by it, with the shell's code for it: 128 + its
// number.
func Exit(sig os.Signal) {
	Raise(sig)

	if n, ok := sig.(syscall.Signal); ok {
		os.Exit(128 + int(n))
	}

	os.Exit(1)
}

// StopSignals are the signals that stop a docs run on purpose: a Ctrl-C, a
// kill, a hang-up. The docs driver catches these and Raise re-raises them
// after it has cleaned up and released its lock.
var StopSignals = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}

// NotifyStops relays to c those of StopSignals this process was not started
// with ignored, and returns them (none: nil). One that was ignored
// (nohup's SIGHUP, a background job's SIGINT) is left so: caught, it would
// end the process, and what it runs would no longer inherit it ignored. With
// none left, nothing is relayed: signal.Notify with no signals relays every
// one, the Go runtime's own included.
func NotifyStops(c chan<- os.Signal) []os.Signal {
	var caught []os.Signal

	for _, s := range StopSignals {
		if !signal.Ignored(s) {
			caught = append(caught, s)
		}
	}

	if len(caught) > 0 {
		signal.Notify(c, caught...)
	}

	return caught
}

// Stopping reports whether sig is one of StopSignals.
func Stopping(sig os.Signal) bool { return slices.Contains(StopSignals, sig) }
