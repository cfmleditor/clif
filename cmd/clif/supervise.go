//go:build !wasip1

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/cfmleditor/clif/internal/supervise"
	"github.com/cfmleditor/clif/internal/vfs"
	"go.lsp.dev/jsonrpc2"
)

// supervisedEnv marks the language server a supervisor started, which serves
// instead of supervising again. noSupervisorEnv turns supervision off.
const (
	supervisedEnv   = "CLIF_SUPERVISED"
	noSupervisorEnv = "CLIF_NO_SUPERVISOR"
)

// serveLanguage runs the language server, under a supervisor that restarts it
// after a crash and replays the editor's session to it (internal/supervise),
// unless this process is the supervised server or supervision is switched off.
func serveLanguage() {
	if os.Getenv(supervisedEnv) != "" || os.Getenv(noSupervisorEnv) != "" {
		runServer()

		return
	}

	code := supervise.Run(context.Background(), supervise.Options{
		Start:  startServerChild,
		Editor: jsonrpc2.NewStream(vfs.Stdio()),
		Log: func(msg string) {
			fmt.Fprintf(os.Stderr, "clif supervisor: %s\n", msg)
		},
	})

	os.Exit(code)
}

// execChild is a language server started as a child process.
type execChild struct {
	cmd    *exec.Cmd
	stream jsonrpc2.Stream
}

func startServerChild() (supervise.Child, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(context.Background(), exe) //nolint:gosec // this binary, run again as the server it supervises

	cmd.Env = append(os.Environ(), supervisedEnv+"=1")
	cmd.Stderr = os.Stderr

	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &execChild{cmd: cmd, stream: jsonrpc2.NewStream(childPipes{out, in})}, nil
}

func (c *execChild) Stream() jsonrpc2.Stream { return c.stream }

func (c *execChild) Wait() int {
	err := c.cmd.Wait()
	if err == nil {
		return 0
	}

	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() >= 0 {
		return exit.ExitCode()
	}

	return 1
}

// childPipes is a child's stdout and stdin as one connection. Closing it
// closes both, and the child sees its stdin end.
type childPipes struct {
	io.ReadCloser
	w io.WriteCloser
}

func (p childPipes) Write(b []byte) (int, error) { return p.w.Write(b) }

func (p childPipes) Close() error {
	return errors.Join(p.w.Close(), p.ReadCloser.Close())
}
