//go:build ignore
// +build ignore

// Runs a command holding docs/.lock, so no two docs runs (a fetch, a bump, a
// generation) interleave:
//
//	go build -o target/docs-gen/docslock scripts/docslock.go
//	target/docs-gen/docslock <command> [args...]
//
// It is built and run, never `go run`, which turns every failing exit of its
// child into exit 1, and a stop by a signal with it (lib/underlock.sh).
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
		fmt.Fprintln(os.Stderr, "usage: docslock <command> [args...]")
		os.Exit(2)
	}

	// docslock --held: exit 0 if this process was started under docs/.lock
	// (lib/underlock.sh asks, so a script started under the lock carries on,
	// and one that only inherited the DOCS_LOCK_HELD flag takes it). It is
	// the check the generator makes: fd 3 is docs/.lock as it is on disk.
	if os.Args[1] == "--held" {
		if os.Getenv("DOCS_LOCK_HELD") != "1" {
			os.Exit(1)
		}
		var fd3 *os.File
		if runtime.GOOS != "windows" {
			if os.Getenv("DOCS_LOCK_FD") != "3" {
				os.Exit(1)
			}
			fd3 = os.NewFile(3, lockPath)
		}
		if !flock.Holds(fd3, lockPath) {
			os.Exit(1)
		}
		os.Exit(0)
	}

	if err := os.MkdirAll("docs", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	replaceOldLock()

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

	// The command leads a process group of its own (flock.NewGroup), and
	// every signal that stops a run is passed to that whole group, once: to
	// bash and to what it is running alike (the generator, git, curl), so
	// nothing carries on behind a run reported stopped. A terminal's Ctrl-C
	// or hang-up reaches only this process's group, never the command's, so
	// it arrives there once, not twice (a second could cut a script's
	// cleanup short). The docs scripts trap these signals (lib/underlock.sh),
	// so bash waits for what it is running to stop before it cleans up.
	// Either way this exits as the command did.
	flock.NewGroup(cmd)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	go func() {
		for s := range sigs {
			_ = flock.SignalGroup(cmd.Process, s)
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

// replaceOldLock: the previous version of these scripts left docs/.lock as a
// symlink naming its owner, "<pid>@<start time>". It is no lock the OS holds,
// and opening it would follow the link. It is replaced once no process has
// its PID; otherwise a run of that version may still be going (another
// checkout, a branch not yet rebased), and this refuses. Anything else that
// is not a regular file is refused too: nothing here made it.
func replaceOldLock() {
	info, err := os.Lstat(lockPath)
	if err != nil || info.Mode().IsRegular() {
		return
	}

	owner := "?"
	gone := false
	if info.Mode()&os.ModeSymlink != 0 {
		owner, _ = os.Readlink(lockPath)
		pid, err := strconv.Atoi(strings.SplitN(owner, "@", 2)[0])
		gone = err == nil && flock.ProcessGone(pid)
	}

	if !gone {
		if owner == "?" {
			fmt.Fprintf(os.Stderr, "error: %s is not a lock file.\n", lockPath)
		} else {
			fmt.Fprintf(os.Stderr, "error: %s is an earlier version's lock, held by %q, which may still be running.\n", lockPath, owner)
		}
		fmt.Fprintf(os.Stderr, "       If no docs run is going, delete it: rm -rf %s\n", lockPath)
		os.Exit(1)
	}

	if err := os.Remove(lockPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot replace an earlier version's %s: %v\n", lockPath, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "note: replaced an earlier version's %s (its owner, %s, is gone)\n", lockPath, owner)
}
