//go:build ignore
// +build ignore

// Runs a command holding the docs lock, so no two docs runs (a fetch, a bump,
// a generation) interleave:
//
//	target/docs-gen/docslock-<hash> [--then <program>] <command> [args...]
//
// lib/underlock.sh builds it, named for a hash of its sources, and runs it;
// never `go run`, which turns every failing exit of its child into exit 1, a
// stop by a signal with it.
//
// The lock is the operating system's (internal/flock) on docs/.flock, so it
// is released however this process ends, kill -9 and a power cut included:
// there is no stale lock to judge or take over. The command is given a fresh
// token (DOCS_LOCK_TOKEN), recorded in docs/.flock beside this PID, and a
// process proves it runs under the lock by `docslock --held`: the record
// holds its token, its holder is alive and the lock is held (flock.Verify).
// The command is passed no descriptor, so nothing it leaves behind (git's
// background gc, the go command's telemetry) can keep the lock; only a
// program named by --then is, and it passes it to nothing it runs. So this
// waits for the command, and passes it every signal that stops a run.
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
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
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
		usage()
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

	// --then <program>: once the command has succeeded, run program too,
	// still under the lock, and pass it the lock's descriptor (fd 3), so it
	// holds the lock itself: should this process be killed while it runs, the
	// lock stays held until it ends. `make generate` runs the generator so
	// (scripts/generate.sh), the one step that must not lose the lock part
	// way through its write. Only that program gets the descriptor, and it
	// passes it to nothing it runs.
	args := os.Args[1:]
	then := ""
	if args[0] == "--then" {
		if len(args) < 3 {
			usage()
		}
		then, args = args[1], args[2:]
	}

	// A signal sent to this process (a kill, a CI runner's stop, a hang-up)
	// is passed to the whole of the command running, its job (flock.Group);
	// one that comes while none is running is kept, and the run stops by it
	// before the next would start (stopIfSignalled). Caught from before the
	// lock is taken, so none can end this process between taking it and
	// starting a command, before the cleanup could clear the record and the
	// marker.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for s := range sigs {
			current.Lock()
			if current.got == nil {
				current.got = s
			}
			if current.p != nil {
				_ = flock.SignalGroup(current.p, s)
			}
			current.Unlock()
		}
	}()

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

	env := append(os.Environ(), "DOCS_LOCK_TOKEN="+token)
	// The copy the run's scripts ask (docslock --held), whatever is checked
	// out or built meanwhile: always set, as every docs script under the
	// lock needs it. Where the OS cannot say (no /proc), the path this was
	// started by, made absolute.
	self, err := os.Executable()
	if err != nil {
		if self, err = filepath.Abs(os.Args[0]); err != nil {
			fail("cannot find this program's own path: %v", err)
		}
	}
	env = append(env, "DOCS_LOCK_BIN="+self)

	stopIfSignalled(sigs)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = env
	if then != "" {
		// Tells scripts/generate.sh that the program comes next, run here.
		cmd.Env = append(cmd.Env, "DOCS_LOCK_THEN_SET="+token)
	}
	code, group := run(cmd, true)

	if code == 0 && then != "" {
		// A stopping signal that came while no command was running (between
		// the two, as the first ended) reached nothing: the run stops here,
		// by it, rather than start the program anyway.
		stopIfSignalled(sigs)

		cmd = exec.Command(then)
		cmd.Env = env
		if runtime.GOOS != "windows" { // which passes a child no descriptor
			cmd.ExtraFiles = []*os.File{f}
			cmd.Env = append(cmd.Env, "DOCS_LOCK_FD=3")
		}
		// Not killed with this process (Linux's Pdeathsig): it holds the
		// lock itself, and is better left to finish its write than cut off
		// part way. On Windows it is in this process's job, so it is killed
		// with it, and a kill between its two renames can leave the
		// generated files out of step (TestGeneratedSourcesMatchTheDocs
		// then fails until a regeneration).
		code, group = run(cmd, false)
	}

	// Ended by a signal, the command's end is passed on as one (a Ctrl-C to
	// the make that ran this too), after the cleanup, so a shell loop or
	// `make -k` sees an interrupted run, not just an exit code.
	signal.Stop(sigs)
	current.Lock()
	sentHere := current.got != nil
	current.Unlock()
	cleanup()
	group.Reraise(sentHere)
	os.Exit(code)
}

// current is the command running, for the signal forwarder.
var current struct {
	sync.Mutex
	p   *os.Process
	got os.Signal // the first stopping signal sent to this process, if any
}

// run runs cmd as a job of its own (flock.Group): a process group, given the
// terminal while it runs if this process has it, so a Ctrl-C there reaches
// its processes once, and a Ctrl-Z stops it and this process's job together,
// for the shell to resume. The docs scripts trap the stopping signals
// (lib/underlock.sh), so bash waits for what it is running to stop before it
// cleans up. withParent: killed should this process die (Linux), as nothing
// it runs holds the lock but through this process. On Windows the whole run
// is in a job that ends with this process.
func run(cmd *exec.Cmd, withParent bool) (int, *flock.Group) {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	group := flock.NewGroup(cmd, withParent)
	if err := cmd.Start(); err != nil {
		fail("%v", err)
	}

	// Recorded for the forwarder, and any signal it kept meanwhile (one that
	// came between the start and now, which reached nothing) sent on at once.
	current.Lock()
	current.p = cmd.Process
	if current.got != nil {
		_ = flock.SignalGroup(current.p, current.got)
	}
	current.Unlock()

	group.Started(cmd.Process)

	code, err := group.Wait()
	if err != nil {
		fail("%v", err)
	}

	current.Lock()
	current.p = nil
	current.Unlock()

	return code, group
}

// stopIfSignalled ends the run by the stopping signal sent to this process,
// if one came, before another command starts: cleaned up, then by the
// signal itself where it can (flock.Raise), else with the shell's code for it.
func stopIfSignalled(sigs chan os.Signal) {
	current.Lock()
	got := current.got
	current.Unlock()

	if got == nil {
		return
	}

	signal.Stop(sigs)
	cleanup()
	flock.Raise(got)

	if n, ok := got.(syscall.Signal); ok {
		os.Exit(128 + int(n))
	}

	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: docslock [--then <program>] <command> [args...]  |  docslock --held")
	os.Exit(2)
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
