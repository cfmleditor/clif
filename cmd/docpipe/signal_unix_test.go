//go:build unix

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestStopCancelsFetchAndReleasesLock(t *testing.T) {
	root := fixture(t)
	if err := os.Remove(filepath.Join(root, "docs", ".sha-cfdocs")); err != nil {
		t.Fatal(err)
	}

	ready := filepath.Join(root, "ready")
	cmd := driverCommand(t, root, []string{"DOCPIPE_TEST_BLOCK=" + ready}, "docs")

	var output bytes.Buffer

	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})

	deadline := time.Now().Add(5 * time.Second)

	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}

		if time.Now().After(deadline) {
			t.Fatal("driver did not start its fetch")
		}

		time.Sleep(10 * time.Millisecond)
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Wait(); err == nil {
		t.Fatal("stopped driver reported success")
	}

	status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGTERM {
		t.Fatalf("driver exit = %v; output: %s", cmd.ProcessState, &output)
	}

	if out, code := runDriver(t, root, nil, "pin", "set", "cfdocs", fixturePin); code != 0 {
		t.Fatalf("stop kept the lock: exit %d: %s", code, out)
	}
}
