// Command docpipe owns the docs lock and every mutation in a docs run.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/cfmleditor/clif/internal/flock"
	"github.com/cfmleditor/clif/internal/vfs"
)

var commandRunner = commandOutput

var runCommand = func(name string, args ...string) ([]byte, error) {
	return commandRunner(context.Background(), name, args...)
}
var checkStopped = func() {}

type exitStatus int

type usageError string

func (e usageError) Error() string { return string(e) }

// exit unwinds generator refusals through the same cleanup as other failures.
func exit(code int) { panic(exitStatus(code)) }

func main() {
	code, by := run(os.Args[1:])
	if by != nil {
		fmt.Fprintf(os.Stderr, "%v: docs run stopped\n", by)
		flock.Exit(by)
	}

	os.Exit(code)
}

func run(args []string) (code int, by os.Signal) {
	ctx, stop := watchStops()
	runCommand = func(name string, args ...string) ([]byte, error) { return commandRunner(ctx, name, args...) }
	checkStopped = func() {
		if ctx.Err() != nil {
			exit(1)
		}
	}

	defer func() { by = stop() }()
	defer func() {
		if r := recover(); r != nil {
			if status, ok := r.(exitStatus); ok {
				code = int(status)
			} else {
				panic(r)
			}
		}
	}()

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: docpipe docs|docs-cfdocs|assemble|bump|generate|generate-only|fetch-cfdocs|fetch-lucee|pin [args]")

		return 2, nil
	}

	p := newPipeline()
	// Read-only pin queries need no lock; the pin is replaced atomically.
	readOnly := args[0] == "pin" && len(args) > 1 && (args[1] == "get" || args[1] == "check")
	if !readOnly {
		held, err := takeLock()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)

			return 1, nil
		}
		defer func() { flock.Unlock(held); _ = held.Close() }()

		if err := p.recover(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)

			return 1, nil
		}
	}

	checkStopped()

	if err := p.dispatch(ctx, args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)

		if _, ok := errors.AsType[usageError](err); ok {
			return 2, nil
		}

		return 1, nil
	}

	checkStopped()

	return 0, nil
}

func takeLock() (*os.File, error) {
	if err := os.MkdirAll("docs", 0o755); err != nil { //nolint:gosec // docs sources, generated files and their lock retain their public repository modes
		return nil, err
	}

	f, err := os.OpenFile("docs/.flock", os.O_CREATE|os.O_RDWR, 0o644) //nolint:gosec // the public repository lock contains only an informational PID
	if err != nil {
		return nil, err
	}

	ok, err := flock.TryLock(f)
	if err != nil || !ok {
		_ = f.Close()

		if err != nil {
			return nil, err
		}

		return nil, errors.New("another docs run holds docs/.flock; wait for it to finish")
	}
	// Informational only. No child or step uses this record to prove ownership.
	if err := f.Truncate(0); err == nil {
		_, _ = fmt.Fprintln(f, os.Getpid())
	}

	if info, err := os.Lstat("docs/.lock"); err == nil && info.Mode()&os.ModeSymlink != 0 {
		_ = os.Remove("docs/.lock")
	}

	return f, nil
}

// watchStops cancels external commands promptly. Filesystem steps are synchronous:
// they complete and clean up while the lock is held, then main re-raises the stop.
func watchStops() (context.Context, func() os.Signal) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 3)
	flock.NotifyStops(ch)

	done, ended := make(chan struct{}), make(chan struct{})

	var (
		mu    sync.Mutex
		first os.Signal
	)

	record := func(s os.Signal) {
		mu.Lock()
		if first == nil {
			first = s
		}
		mu.Unlock()
		cancel()
	}

	go func() {
		defer close(ended)

		select {
		case s := <-ch:
			record(s)
		case <-done:
		}
	}()

	return ctx, func() os.Signal {
		signal.Stop(ch)
		close(done)
		<-ended

		for {
			select {
			case s := <-ch:
				record(s)
			default:
				cancel()
				mu.Lock()
				result := first
				mu.Unlock()

				return result
			}
		}
	}
}

func commandOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	// Only read-only repository queries, isolated git fetches, and curl downloads
	// are children. They never receive the lock or write committed docs paths.
	if name != "git" && name != "curl" {
		return nil, fmt.Errorf("unsupported docs subprocess %q", name)
	}

	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // fixed executables and arguments chosen by the pipeline

	var stderr strings.Builder

	cmd.Stderr = &stderr
	cmd.WaitDelay = time.Second

	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}

	return out, nil
}

// readFile is bounded and refuses devices and FIFOs, as every repository read does.
func readFile(path string) ([]byte, error) { return vfs.ReadRegular(path) }
