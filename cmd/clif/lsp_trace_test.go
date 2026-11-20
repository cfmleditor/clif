package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cfpath "github.com/cfmleditor/clif/internal/path"
)

func TestLSPTraceProcess(_ *testing.T) {
	if os.Getenv("CLIF_TEST_LSP_TRACE") != "1" {
		return
	}

	runServer()
	os.Exit(0)
}

// Launch outside the project to exercise the standalone server's workspace
// config. Separate processes let each run open its own CLIF_LOG.
func TestLSPTraceRecordsResultsOnlyInDebugMode(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	for _, debug := range []bool{false, true} {
		t.Run(fmt.Sprintf("debug=%t", debug), func(t *testing.T) {
			dir := t.TempDir()
			workspace := filepath.Join(dir, "project")
			writeTestFile(t, filepath.Join(workspace, ".clif.json"), fmt.Sprintf(`{"debug":%t}`, debug))

			logPath := filepath.Join(dir, "clif.log")

			root, err := json.Marshal(string(cfpath.ToURI(workspace)))
			if err != nil {
				t.Fatal(err)
			}

			var input strings.Builder

			for _, body := range []string{
				fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"rootUri":%s,"capabilities":{}}}`, root),
				`{"jsonrpc":"2.0","id":"result","method":"workspace/executeCommand","params":{"command":"clif.searchDocs","arguments":["arraylen","function",1]}}`,
				`{"jsonrpc":"2.0","id":"error","method":"workspace/executeCommand","params":{"command":"clif.format","arguments":[]}}`,
			} {
				fmt.Fprintf(&input, "Content-Length: %d\r\n\r\n%s", len(body), body)
			}

			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, exe, "-test.run=^TestLSPTraceProcess$")
			cmd.Dir = dir

			cmd.Env = append(os.Environ(), "CLIF_TEST_LSP_TRACE=1", "CLIF_LOG="+logPath)
			cmd.Stdin = strings.NewReader(input.String())

			var stderr bytes.Buffer

			cmd.Stderr = &stderr

			stdout, err := cmd.Output()
			if err != nil {
				t.Fatalf("clif exited: %v\n%s", err, &stderr)
			}

			for _, want := range []string{`"id":"result"`, `"name":"arraylen"`, `"id":"error"`, "requires a document URI argument"} {
				if !strings.Contains(strings.ToLower(string(stdout)), strings.ToLower(want)) {
					t.Errorf("LSP output missing %q:\n%s", want, stdout)
				}
			}

			if bytes.Contains(stdout, []byte("LSP response")) {
				t.Error("trace payloads were forwarded over the LSP connection")
			}

			log, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}

			for _, want := range []string{"LSP response", "clif.searchDocs", "arraylen", "clif.format", "requires a document URI argument"} {
				if present := strings.Contains(strings.ToLower(string(log)), strings.ToLower(want)); present != debug {
					t.Errorf("trace contains %q = %t, want %t:\n%s", want, present, debug, log)
				}
			}
		})
	}
}
