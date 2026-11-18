// Package flock takes an exclusive, non-blocking lock on an open file: the
// operating system's own (flock on Unix, LockFileEx on Windows), so it is
// released however the holder ends, kill -9 and a power cut included, with no
// owner to guess at. It also has the few process helpers its holder needs.
// scripts/docslock.go holds docs/.flock with it.
package flock

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/cfmleditor/clif/internal/vfs"
)

// ErrUnsupported is returned where the platform has no file lock (wasm).
var ErrUnsupported = errors.ErrUnsupported

// Record writes the holder's record into f, which this process has locked:
// its PID, for a refused run to name, and token, which the processes it
// starts are given to prove they run under it (Verify). The scripts are
// passed no descriptor, so nothing they leave behind can keep the lock; the
// one program docslock --then passes the lock's descriptor (the generator,
// which then holds the lock itself) checks it with Inherited and Holds.
func Record(f *os.File, token string) error {
	if err := f.Truncate(0); err != nil {
		return err
	}

	// With this process's start time, where the platform tells it
	// (startTime), so a PID reused after this process has gone is not taken
	// for it (Verify).
	line := fmt.Sprintf("%d %s", os.Getpid(), token)
	if st, ok := startTime(os.Getpid()); ok {
		line += " " + st
	}

	_, err := f.WriteAt([]byte(line+"\n"), 0)

	return err
}

// Holder reads the record at path: the holder's PID and token, or "" for
// each when there is none.
func Holder(path string) (pid, token string) {
	pid, token, _ = record(path)

	return pid, token
}

// record reads the record at path: PID, token and, where it was known, the
// holder's start time.
func record(path string) (pid, token, start string) {
	b, err := vfs.ReadRegular(path)
	if err != nil {
		return "", "", ""
	}

	f := strings.Fields(string(b))
	switch len(f) {
	case 2:
		return f[0], f[1], ""
	case 3:
		return f[0], f[1], f[2]
	}

	return "", "", ""
}

// Orphaned reports whether token is the run whose holder has ended: the
// record still holds it (a holder clears its record as it exits, so it was
// killed) and no process is that holder. A process with that token belongs
// to a run nothing holds the lock for, and must stop, not take the lock
// afresh and carry on. A token leaked from a run that ended normally matches
// no record, so is not this.
//
// Gone is told two ways: no process is that holder, or the lock is free (a
// live holder always holds it). The second covers a holder killed but not
// yet reaped, whose PID still looks alive while its lock is already gone.
func Orphaned(path, token string) bool {
	_, got, _ := record(path)
	if token == "" || got != token {
		return false
	}

	return HolderGone(path) || !lockedByAnyone(path)
}

// HolderGone reports whether the record at path names a holder that has
// ended (killed before it could clear it): while the lock is held all the
// same, some process it started (the generator, passed the descriptor)
// holds it now.
func HolderGone(path string) bool {
	pidText, _, start := record(path)

	pid, err := strconv.Atoi(pidText)

	return err == nil && !holderAlive(pid, start)
}

// holderAlive: a process has pid, and, where the record and the platform
// both give a start time, it is the one that wrote the record, not a later
// process given the same PID.
func holderAlive(pid int, start string) bool {
	if !alive(pid) {
		return false
	}

	if start == "" {
		return true
	}

	now, ok := startTime(pid)

	return !ok || now == start
}

// Verify reports whether token names the run holding the lock on path: the
// record there holds token, and some process holds the lock, which this
// tells by failing to take it. A file deleted and created again since the
// holder locked it holds no record, and a token leaked from a run that has
// ended matches none (a holder clears its record as it exits:
// scripts/docslock.go). The checks run cheapest and most telling first: the
// token, then that the record's holder is alive (by PID, and by start time
// where the platform gives one, so a reused PID is not taken for it), and
// last the probe of the lock itself. The probe runs only for a record naming
// a live holder, so it nearly always finds the lock held; it can find it
// free only where no start time is kept (Windows, the BSDs) and a dead
// holder's PID has been reused, and then it takes the free lock for an
// instant, so a run starting at that instant is refused once.
func Verify(path, token string) bool {
	if token == "" {
		return false
	}

	// The record's holder must be alive too: a new holder that has taken the
	// lock but not yet written its record (an instant) leaves a killed
	// holder's record in place, and that record's token would otherwise
	// pass, against the new holder's lock, for the killed run's scripts.
	pidText, got, start := record(path)
	if got != token {
		return false
	}

	if pid, err := strconv.Atoi(pidText); err != nil || !holderAlive(pid, start) {
		return false
	}

	return lockedByAnyone(path)
}

// lockedByAnyone reports whether some process holds the lock on path, by
// failing to take it on an open of its own.
func lockedByAnyone(path string) bool {
	f, err := os.OpenFile(path, os.O_RDWR, 0) //nolint:gosec // the lock file the caller names, opened only to test its lock
	if err != nil {
		return false
	}

	defer func() { _ = f.Close() }()

	free, err := TryLock(f)
	if err == nil && free {
		Unlock(f) // released at once, not on the close (Windows delays that)
	}

	return err == nil && !free
}

// Holds reports whether f, a descriptor for the lock inherited from its
// holder, is an open of the file at path as it is now on disk, and the record
// there holds token: so the lock taken on f still guards that file, and the
// run is the one that took it. A file deleted and created again since is not
// it. scripts/generate_docs.go checks the descriptor docslock passed it this
// way; holding it, it keeps the lock even should docslock be killed.
func Holds(f *os.File, path, token string) bool {
	held, err := f.Stat()
	if err != nil {
		return false
	}

	onDisk, err := os.Stat(path)
	if err != nil || !os.SameFile(held, onDisk) {
		return false
	}

	_, got := Holder(path)

	// And the lock is held: an open of the file with the right record is not
	// proof by itself that the lock taken on it still is.
	return token != "" && got == token && lockedByAnyone(path)
}

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
// kill, a hang-up. The one list docslock catches, Raise re-raises and the
// generator treats as an interrupt.
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

// SignalOf reports the signal that ended the process err (from a Cmd's Run,
// Output or Wait) is about, if a signal ended it.
func SignalOf(err error) (os.Signal, bool) {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return nil, false
	}

	ws, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !ws.Signaled() {
		return nil, false
	}

	return ws.Signal(), true
}
