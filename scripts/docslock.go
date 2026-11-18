//go:build ignore
// +build ignore

// Runs a command holding the docs lock, so no two docs runs (a fetch, a bump,
// a generation) interleave:
//
//	target/docs-gen/docslock-<hash> [--then] <command> [args...]
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
// program run by --then is, and it passes it to nothing it runs.
//
// Signals: the command runs in this process's own process group, so a
// terminal's Ctrl-C, Ctrl-Z or hang-up reaches it, and everything it runs,
// directly, once, and job control is the shell's as for any command. This
// process only catches those, to wait for the command. A SIGTERM, sent to
// this process alone (a kill, a CI runner's stop), is passed on to the
// command (bash, which traps it: lib/underlock.sh). A command that a signal
// ends, this ends by the same signal (flock.Raise), so what ran this sees an
// interrupted run.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cfmleditor/clif/internal/flock"
	"github.com/cfmleditor/clif/internal/vfs"
)

const lockPath = "docs/.flock"

// held is docs/.flock while this holds its lock; its record is cleared on the
// way out, so the run's token matches nothing once the run is over.
var held *os.File

// cleanup, holding the lock, clears what killed runs left (clearLeftovers),
// then the record, and releases the lock, on every way out (by now
// nothing this ran is still running, the generator included). Released
// explicitly: Windows releases one held at exit only after a delay.
func cleanup() {
	if held != nil {
		clearLeftovers()
		_ = held.Truncate(0)
		flock.Unlock(held)
	}
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	cleanup()
	os.Exit(1)
}

// current is the command running, for the signal forwarder, and the first
// stopping signal sent to this process.
var current struct {
	sync.Mutex
	p       *os.Process
	got     os.Signal
	started time.Time // when p was about to be started
}

// sigs relays the stopping signals sent to this process.
var sigs = make(chan os.Signal, 1)

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	// docslock --held: exit 0 if the caller runs under the lock (its
	// DOCS_LOCK_TOKEN is the holder's), 3 if its run's docslock has ended
	// (flock.Orphaned: killed), 1 otherwise. lib/underlock.sh asks: a script
	// under the lock carries on, one whose run has ended stops, and any
	// other (a token leaked from a run that ended normally, or none) takes
	// the lock.
	if os.Args[1] == "--held" {
		token := os.Getenv("DOCS_LOCK_TOKEN")
		switch {
		case flock.Verify(lockPath, token):
			os.Exit(0)
		case flock.Orphaned(lockPath, token):
			os.Exit(3)
		}
		os.Exit(1)
	}

	// --then: once the command has succeeded, run the program it names,
	// still under the lock, and pass it the lock's descriptor (fd 3), so it
	// holds the lock itself: should this process be killed while it runs, the
	// lock stays held until it ends. `make generate` runs the generator so
	// (scripts/generate.sh), the one step that must not lose the lock part
	// way through its write. The command names it by writing its path to the
	// file DOCS_LOCK_THEN_FILE gives, once it has built it, under the lock:
	// so the path is the one it built, whatever its sources were when the
	// run began, and the file's name holds this run's token, so a value
	// leaked to another run names nothing there.
	args := os.Args[1:]
	then := false
	if args[0] == "--then" {
		if len(args) < 2 {
			usage()
		}
		then, args = true, args[1:]
	}

	// Caught from the start, so none ends this process between taking the
	// lock and clearing its record. Passed on to the command: a SIGTERM, and
	// a Ctrl-C or hang-up that cannot have come from the terminal (this
	// process not in its foreground: flock.InForeground). One that did has
	// reached the command already, and a second copy could cut its cleanup
	// short. One that comes while no command runs stops the run before the
	// next would start (run checks for one and starts the command under
	// current's lock, so none falls between the two), and any that came
	// ends this process by it.
	//
	// A signal this process was started with ignored (nohup's SIGHUP, a
	// background job's SIGINT) is left so: caught, it would be passed on, and
	// the command, which inherits it ignored otherwise, would die of it. A
	// hang-up is judged by whether this process had the terminal at the
	// start: by the time it arrives the terminal is often gone, and asking
	// then would take a hang-up the terminal sent the whole group for one
	// sent here alone, and deliver it to the command a second time.
	fgAtStart := flock.InForeground()
	flock.NotifyStops(sigs)
	go func() {
		for s := range sigs {
			at := time.Now() // as it arrived: the wait for the lock below may outlast a start
			pass := s == syscall.SIGTERM ||
				(s == os.Interrupt && !flock.InForeground()) ||
				(s == syscall.SIGHUP && !fgAtStart)

			current.Lock()
			if current.got == nil {
				current.got = s
			}
			// One that arrived before the running command was started never
			// reached it from the terminal either: it is passed on as well.
			if current.p != nil && (pass || at.Before(current.started)) {
				_ = current.p.Signal(s)
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
	// record is the very next thing written.
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		fail("%v", err)
	}
	token := hex.EncodeToString(b)

	ok, err := flock.TryLock(f)
	if err != nil {
		fail("cannot lock %s: %v", lockPath, err)
	}

	// An empty record with the lock held is a run in its last instant: it
	// clears its record just before it releases the lock. Tried again for a
	// moment, rather than refuse a run that would start a breath later.
	// So is a record naming a holder that has ended, with the lock held:
	// a new holder may be in its first instant, not yet having written its
	// own (or a killed run's generator holds it; then the retries run out).
	for try := 0; !ok && try < 10; try++ {
		if pid, _ := flock.Holder(lockPath); pid != "" && !flock.HolderGone(lockPath) {
			break
		}
		time.Sleep(20 * time.Millisecond)
		if ok, err = flock.TryLock(f); err != nil {
			fail("cannot lock %s: %v", lockPath, err)
		}
	}

	if !ok {
		pid, _ := flock.Holder(lockPath)
		if pid == "" {
			pid = "?"
		}
		if flock.HolderGone(lockPath) {
			fail("another docs run holds %s: its docslock (PID %s) has ended, but a program it started (the generator)\n"+
				"       still holds the lock until its write is done; wait for it to finish", lockPath, pid)
		}
		fail("another docs run (PID %s) holds %s; wait for it to finish", pid, lockPath)
	}

	held = f
	if err := flock.Record(f, token); err != nil {
		fail("cannot write %s: %v", lockPath, err)
	}

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

	// On Windows, everything the run starts is in a job that ends with this
	// process (flock.JoinJob); on Linux the command dies with it
	// (flock.DieWithParent). Elsewhere the scripts' own lock checks stop them.
	flock.JoinJob()

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = env
	thenFile := ""
	if then {
		// With forward slashes, which Windows accepts too: the script that
		// writes it may run under MSYS bash, which might take backslashes
		// in a relative name literally.
		thenFile = "target/docs-gen/then-" + token
		cmd.Env = append(cmd.Env, "DOCS_LOCK_THEN_FILE="+thenFile)
	}
	flock.DieWithParent(cmd)
	code, by := run(cmd, nil)

	program := ""
	if thenFile != "" && (code != 0 || by != nil) {
		// Failed or stopped: its own end is the run's, whatever the file
		// says, and the file is not read.
		_ = os.Remove(thenFile)
	} else if thenFile != "" {
		program = thenProgram(thenFile)
		// Succeeded without naming the program it was to hand on: the
		// handoff was lost, and the run did not do its last step. Never
		// a quiet success (CI's docs check would pass without the
		// generator having run).
		if code == 0 && program == "" {
			fail("--then: %s named no program to run (its file %s was not written, was empty, or could not be read)", args[0], thenFile)
		}
	}

	if code == 0 && program != "" {
		cmd = exec.Command(program)
		cmd.Env = env
		if runtime.GOOS != "windows" { // which passes a child no descriptor
			cmd.ExtraFiles = []*os.File{f}
			cmd.Env = append(cmd.Env, "DOCS_LOCK_FD=3")
		}
		// Not killed with this process: holding the lock itself (Unix), it
		// is better left to finish its write than cut off between its two
		// renames. On Windows it breaks away from the job for the same
		// reason; holding no descriptor there, its own check of the lock
		// before writing is what stops it should this process be gone. A
		// job around this process that forbids breaking away refuses the
		// start: then it starts again in the job, as everything else.
		if flock.Breakaway(cmd) {
			code, by = run(cmd, func(c *exec.Cmd) { flock.NoBreakaway(c) })
		} else {
			code, by = run(cmd, nil)
		}
	}

	// At the end, still holding the lock (cleanup clears what killed runs
	// left, then the record). Signals are still caught meanwhile, so none
	// ends this process before its record is cleared.
	cleanup()
	signal.Stop(sigs)

	// Ended by a signal, this ends by it too; and by one sent to this
	// process that the command took and survived (or one that came as it
	// ended), so the run still reads as stopped, not as done.
	if by == nil {
		current.Lock()
		by = current.got
		current.Unlock()
	}
	if by != nil {
		flock.Exit(by)
	}
	os.Exit(code)
}

// run runs cmd with this process's stdio and waits for it: its exit code, and
// the signal that ended it, if one did (128 + its number then, as a shell
// reports it). retry, if given, adjusts a copy of cmd to start again should
// the first start be refused.
//
// A stopping signal that came while no command ran (before this one, or
// between two) reached nothing: the run stops by it instead of starting cmd.
// That check, the start and the recording of the process are one step under
// current's lock, so the forwarder sees either no process (and the check
// sees its signal) or this one: none falls between, unseen or sent twice.
func run(cmd *exec.Cmd, retry func(*exec.Cmd)) (int, os.Signal) {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	current.Lock()
	if got := current.got; got != nil {
		current.Unlock()
		stopBy(got)
	}

	current.started = time.Now()
	err := cmd.Start()
	if err != nil && retry != nil && flock.BreakawayRefused(err) {
		// A Cmd cannot be started twice: a fresh one, alike.
		again := exec.Command(cmd.Path, cmd.Args[1:]...)
		again.Env, again.ExtraFiles, again.SysProcAttr = cmd.Env, cmd.ExtraFiles, cmd.SysProcAttr
		again.Stdin, again.Stdout, again.Stderr = cmd.Stdin, cmd.Stdout, cmd.Stderr
		retry(again)

		cmd, err = again, again.Start()
	}

	if err != nil {
		current.Unlock()
		fail("%v", err)
	}

	current.p = cmd.Process
	current.Unlock()

	err = cmd.Wait()

	current.Lock()
	current.p = nil
	current.Unlock()

	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0, nil
	case errors.As(err, &exitErr):
		if sig, ok := flock.SignalOf(err); ok {
			if n, ok := sig.(syscall.Signal); ok {
				return 128 + int(n), sig
			}
		}

		return exitErr.ExitCode(), nil
	default:
		// Not knowing how the command ended, it may still be running:
		// stopped before the lock goes, so nothing it does is unlocked.
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		fail("%v", err)

		return 1, nil
	}
}

// stopBy ends the run by sig, a stopping signal sent to this process before
// a command could start: cleaned up, then by the signal itself where it can,
// else with the shell's code for it (flock.Exit).
func stopBy(sig os.Signal) {
	cleanup()
	signal.Stop(sigs)
	flock.Exit(sig)
}

// thenProgram reads and removes the --then file: the program the command
// named, or "" if it named none. Only a program under target/docs-gen, where
// the docs scripts build theirs, is taken: the descriptor is passed to
// nothing else.
func thenProgram(file string) string {
	b, err := vfs.ReadRegular(file) // never a FIFO, a device: this run holds the lock
	_ = os.Remove(file)
	if err != nil {
		return ""
	}

	// Empty (the script truncated it but wrote nothing): it named none, which
	// the caller reports as such.
	raw := strings.TrimSpace(string(b))
	if raw == "" {
		return ""
	}

	p := filepath.Clean(raw)
	if filepath.IsAbs(p) || filepath.Dir(p) != buildDir {
		fail("--then: %q is not a program the docs scripts built (under target/docs-gen)", p)
	}

	return p
}

// oldLockPath is the earlier scripts' lock, retired (clearLeftovers).
const oldLockPath = "docs/.lock"

// buildDir is where lib/hashbuild.sh builds the docs programs.
var buildDir = filepath.Join("target", "docs-gen")

// clearLeftovers removes from target/docs-gen what killed runs left that
// nothing ever executes: --then files (then-*) and temporary builds (*.tmp),
// once ten minutes old, so a build in progress now is not one of them. Here,
// holding the lock, as a run ends. Builds themselves (docslock-<hash>,
// generate_docs-<hash>, check_vendored-<hash>) are never removed: any run may
// be about to execute one, and deleting it is a race with that run's start;
// each is a few MB, a new one appears only when its program's code changes,
// and `make clean` removes them.
func clearLeftovers() {
	// docs/.lock, the earlier scripts' lock: a symlink naming its owner, which
	// nothing reads any more. Only a symlink, the one form it ever took.
	if info, err := os.Lstat(oldLockPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		_ = os.Remove(oldLockPath)
	}

	entries, err := os.ReadDir(buildDir)
	if err != nil {
		return
	}

	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "then-") && !strings.HasSuffix(name, ".tmp") {
			continue
		}

		if info, err := e.Info(); err == nil && time.Since(info.ModTime()) >= 10*time.Minute {
			_ = os.Remove(filepath.Join(buildDir, name))
		}
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: docslock [--then] <command> [args...]  |  docslock --held")
	os.Exit(2)
}
