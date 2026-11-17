//go:build ignore
// +build ignore

// Runs a command holding docs/.lock, so no two docs runs (a fetch, a bump, a
// generation) interleave:
//
//	go run scripts/docslock.go <command> [args...]
//
// The lock is the operating system's (internal/flock), so it is released
// however its holders end, kill -9 and a power cut included: there is no
// owner to record, and no stale lock to judge or take over. The command, and
// everything it runs, inherits the lock's file descriptor as fd 3 (Unix), so
// the lock lasts while any of them is alive, and DOCS_LOCK_HELD=1 tells a
// nested docs script not to take it again. scripts/generate_docs.go checks
// fd 3 is the lock before it reads or writes anything.
//
// Every docs script runs itself through this when started directly, from the
// repository root; signals reach the command, and this exits as it did.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/cfmleditor/clif/internal/flock"
)

const lockPath = "docs/.lock"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run scripts/docslock.go <command> [args...]")
		os.Exit(2)
	}

	if err := os.MkdirAll("docs", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	// An earlier version of these scripts left docs/.lock as a symlink naming
	// its owner, or as a directory holding a pid file. Neither is a lock the
	// OS holds, and opening one would follow the link (creating a file named
	// for its owner) or fail on the directory: replace it with this file.
	if info, err := os.Lstat(lockPath); err == nil && !info.Mode().IsRegular() {
		if err := os.RemoveAll(lockPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot replace an earlier version's %s: %v\n", lockPath, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "note: replaced an earlier version's %s\n", lockPath)
	}
	_ = os.RemoveAll(lockPath + ".takeover") // the earlier version's, if left

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot open %s: %v\n", lockPath, err)
		os.Exit(1)
	}

	ok, err := flock.TryLock(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot lock %s: %v\n", lockPath, err)
		os.Exit(1)
	}

	if !ok {
		b, _ := os.ReadFile(lockPath)
		holder := strings.TrimSpace(string(b))
		if holder == "" {
			holder = "?"
		}
		fmt.Fprintf(os.Stderr, "error: another docs run (PID %s) holds %s; wait for it to finish\n", holder, lockPath)
		os.Exit(1)
	}

	// For the message above, in a run that finds the lock taken. Nothing
	// decides anything by it: the lock itself is the OS's.
	_ = f.Truncate(0)
	_, _ = f.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0)

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), "DOCS_LOCK_HELD=1")
	if runtime.GOOS != "windows" {
		// fd 3 in the command and all it runs: the lock outlives this
		// process, should it be killed, until they are done too.
		cmd.ExtraFiles = []*os.File{f}
		cmd.Env = append(cmd.Env, "DOCS_LOCK_FD=3")
	}

	// A Ctrl-C reaches the whole process group, the command included; a
	// signal sent to this process alone is passed on. Either way this waits
	// for the command, and exits as it did.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	go func() {
		for s := range sigs {
			_ = cmd.Process.Signal(s)
		}
	}()

	err = cmd.Wait()

	var exitErr *exec.ExitError
	switch {
	case err == nil:
		os.Exit(0)
	case errors.As(err, &exitErr):
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			os.Exit(128 + int(ws.Signal()))
		}
		os.Exit(exitErr.ExitCode())
	default:
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
