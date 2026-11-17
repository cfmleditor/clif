//go:build ignore
// +build ignore

// Runs a command holding the docs lock, so no two docs runs (a fetch, a bump,
// a generation) interleave:
//
//	target/docs-gen/docslock-<hash> <command> [args...]
//
// lib/underlock.sh builds it, named for a hash of its sources, and runs it;
// never `go run`, which turns every failing exit of its child into exit 1, a
// stop by a signal with it.
//
// The lock is the operating system's (internal/flock) on docs/.flock, so it
// is released however this process ends, kill -9 and a power cut included:
// there is no stale lock to judge or take over. It is held by this process
// alone. The command is given a fresh token (DOCS_LOCK_TOKEN), recorded in
// docs/.flock beside this PID, and a process proves it runs under the lock by
// `docslock --held`: the record holds its token and the lock is held
// (flock.Verify). No descriptor is passed down, so nothing the command leaves
// behind (git's background gc, the go command's telemetry) can keep the lock.
// So this waits for the command, and passes it every signal that stops a run.
//
// docs/.lock is the previous version's lock, a symlink naming its owner as
// "<pid>@<start time>". This keeps one there naming itself while it runs, so a
// run of that version (another checkout, a branch not yet rebased) finds a
// live owner and refuses, instead of taking the lock as stale.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cfmleditor/clif/internal/flock"
)

const (
	lockPath   = "docs/.flock"
	markerPath = "docs/.lock"
)

// marker is the docs/.lock symlink this run made, removed on the way out.
var marker string

// held is docs/.flock while this holds its lock; its record is cleared on the
// way out, so the run's token matches nothing once the run is over, and
// flock.Verify never has to probe a free lock for a token leaked from it.
var held *os.File

func exit(code int) {
	cleanup()
	os.Exit(code)
}

// cleanup clears the record and removes the marker, on every way out.
func cleanup() {
	if held != nil {
		_ = held.Truncate(0)
	}

	if marker != "" {
		if got, err := os.Readlink(markerPath); err == nil && got == marker {
			_ = os.Remove(markerPath)
		}
	}
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	exit(1)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: docslock <command> [args...]  |  docslock --held")
		os.Exit(2)
	}

	// docslock --held: exit 0 if the caller runs under the lock (its
	// DOCS_LOCK_TOKEN is the holder's). lib/underlock.sh asks, so a script
	// started under the lock carries on and any other takes it.
	if os.Args[1] == "--held" {
		if flock.Verify(lockPath, os.Getenv("DOCS_LOCK_TOKEN")) {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if err := os.MkdirAll("docs", 0o755); err != nil {
		fail("%v", err)
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		fail("cannot open %s: %v", lockPath, err)
	}

	// The token is made before the lock is taken, so that once it is, the
	// record is the very next thing written: until then the file holds the
	// previous holder's, which a run whose docslock was killed could pass
	// flock.Verify with against this lock.
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		fail("%v", err)
	}
	token := hex.EncodeToString(b)

	// Retried for a moment before refusing: flock.Verify, testing a record
	// a killed holder left, takes a free lock for an instant, and a run
	// starting just then must not be turned away for it.
	var ok bool
	for try := 0; try < 10; try++ {
		if ok, err = flock.TryLock(f); ok || err != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		fail("cannot lock %s: %v", lockPath, err)
	}

	if !ok {
		pid, _ := flock.Holder(lockPath)
		if pid == "" {
			pid = "?"
		}
		fail("another docs run (PID %s) holds %s; wait for it to finish", pid, lockPath)
	}

	held = f
	if err := flock.Record(f, token); err != nil {
		fail("cannot write %s: %v", lockPath, err)
	}

	takeMarker()

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), "DOCS_LOCK_TOKEN="+token)
	// The copy the run's scripts ask (docslock --held), whatever is checked
	// out or built meanwhile.
	if self, err := os.Executable(); err == nil {
		cmd.Env = append(cmd.Env, "DOCS_LOCK_BIN="+self)
	}

	// The command runs as a job of its own (flock.Group): a process group,
	// given the terminal while it runs if this process has it, so a Ctrl-C
	// there reaches its processes once, and a Ctrl-Z stops it and this
	// process's job together, for the shell to resume. A signal sent to this
	// process (a kill, a CI runner's stop, a hang-up) is passed to that whole
	// group: to bash and to what it is running alike, so nothing carries on
	// behind a run reported stopped. The docs scripts trap these
	// (lib/underlock.sh), so bash waits for what it is running to stop before
	// it cleans up. On Windows the job ends with this process, however it
	// ends. This exits as the command did.
	group := flock.NewGroup(cmd)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	if err := cmd.Start(); err != nil {
		fail("%v", err)
	}
	group.Started(cmd.Process)

	go func() {
		for s := range sigs {
			_ = flock.SignalGroup(cmd.Process, s)
		}
	}()

	code, err := group.Wait()
	if err != nil {
		fail("%v", err)
	}

	// Ended by a signal, the command's end is passed on as one (a Ctrl-C to
	// the make that ran this too), after the cleanup, so a shell loop or
	// `make -k` sees an interrupted run, not just an exit code.
	cleanup()
	group.Reraise()
	os.Exit(code)
}

// takeMarker puts a symlink at docs/.lock naming this process as the
// previous version's scripts name an owner, "<pid>@<start time>" (the start
// time as `ps -o lstart=` gives it, spaces as underscores), so a run of that
// version finds a live owner and refuses. One already there is that
// version's, or a docslock's killed before it could remove its own: it is
// replaced once its owner is gone (no process has its PID with its start
// time), and refused while it lives. Anything else there that is not a
// symlink (an earlier commit's lock file) is replaced. Windows, where those
// scripts' PIDs are MSYS's, is left alone.
func takeMarker() {
	if runtime.GOOS == "windows" {
		return
	}

	self := owner(os.Getpid())
	if self == "" {
		return // no ps: a run of that version could not have named an owner either
	}

	for range 3 {
		if err := os.Symlink(self, markerPath); err == nil {
			marker = self

			return
		}

		info, err := os.Lstat(markerPath)
		if err != nil {
			continue // gone meanwhile: try again
		}

		var stale string // the link's content, judged stale
		if info.Mode()&os.ModeSymlink != 0 {
			stale, _ = os.Readlink(markerPath)
			pid, err := strconv.Atoi(strings.SplitN(stale, "@", 2)[0])
			if err == nil && owner(pid) == stale {
				fail("a docs run of the previous version of these scripts (PID %d) holds %s; wait for it to finish", pid, markerPath)
			}
		}

		// Moved aside, then checked to be the one judged stale, before it
		// is deleted: a run of the previous version may have replaced it
		// with its own, live, in between (it removes a stale lock and makes
		// its own the same way). One that has changed is put back, and the
		// next try finds it live.
		aside := fmt.Sprintf("%s.stale.%d", markerPath, os.Getpid())
		if err := os.Rename(markerPath, aside); err != nil {
			continue // gone meanwhile: look again
		}

		if now, err := os.Readlink(aside); err == nil && now != stale {
			_ = os.Symlink(now, markerPath) // put back; fails only if another is there already
			_ = os.Remove(aside)

			continue
		}

		if err := os.RemoveAll(aside); err != nil {
			fail("cannot remove a stale %s: %v", markerPath, err)
		}
	}

	fail("cannot create %s", markerPath)
}

// owner is pid's owner string, "<pid>@<start time>", or "" if no process has
// it (or ps cannot say).
func owner(pid int) string {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "lstart=").Output()
	start := strings.Join(strings.Fields(string(out)), "_")
	if err != nil || start == "" {
		return ""
	}

	return strconv.Itoa(pid) + "@" + start
}
