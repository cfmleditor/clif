// Package flock takes an exclusive, non-blocking lock on an open file: the
// operating system's own (flock on Unix, LockFileEx on Windows), so it is
// released however the holder ends, kill -9 and a power cut included, with no
// owner to guess at. It also has the process-group helpers the holder needs.
// scripts/docslock.go holds docs/.flock with it.
package flock

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cfmleditor/clif/internal/vfs"
)

// ErrUnsupported is returned where the platform has no file lock (wasm).
var ErrUnsupported = errors.ErrUnsupported

// Record writes the holder's record into f, which this process has locked:
// its PID, for a refused run to name, and token, which the processes it
// starts are given to prove they run under it (Verify). Nothing else is
// passed to them, no descriptor included, so nothing they leave behind can
// keep the lock.
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
// ended matches none. The token is compared first, and a holder clears its
// record as it exits (scripts/docslock.go), so the probe, which takes a free
// lock for a moment (refusing a run that starts just then), runs only on a
// record a killed holder left behind.
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

	f, err := os.OpenFile(path, os.O_RDWR, 0) //nolint:gosec // the lock file the caller names, opened only to test its lock
	if err != nil {
		return false
	}

	defer func() { _ = f.Close() }()

	free, err := TryLock(f)

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

	return token != "" && got == token
}
