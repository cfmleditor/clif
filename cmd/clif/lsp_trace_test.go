package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
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

type traceTestPipes struct {
	io.ReadCloser
	io.WriteCloser
}

func (p traceTestPipes) Close() error {
	return errors.Join(p.ReadCloser.Close(), p.WriteCloser.Close())
}

func TestLSPTraceRecordsRejectedWorkspaceEdit(t *testing.T) {
	dir := t.TempDir()
	workspace := filepath.Join(dir, "project")
	writeTestFile(t, filepath.Join(workspace, ".clif.json"),
		`{"debug":true,"formatting":{"enabled":true,"whitespaceOnly":true},"linting":{"enabled":false}}`)

	logPath := filepath.Join(dir, "clif.log")

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestLSPTraceProcess$")
	cmd.Dir = dir

	cmd.Env = append(os.Environ(), "CLIF_TEST_LSP_TRACE=1", "CLIF_LOG="+logPath)

	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()

		if cmd.ProcessState == nil {
			_ = cmd.Wait()
		}
	})

	const reason = "editor refused the stale document version"

	edits := make(chan []byte, 1)
	leaks := make(chan string, 1)
	rpc := jsonrpc2.NewConn(jsonrpc2.NewStream(traceTestPipes{ReadCloser: out, WriteCloser: in}))
	rpc.Go(ctx, func(_ context.Context, req *jsonrpc2.Request) (any, error) {
		switch req.Method() {
		case protocol.MethodWorkspaceApplyEdit:
			select {
			case edits <- bytes.Clone(req.Params()):
			default:
			}

			return protocol.ApplyWorkspaceEditResult{Applied: false, FailureReason: new(reason)}, nil
		case protocol.MethodWindowLogMessage:
			if bytes.Contains(req.Params(), []byte("LSP client call")) {
				select {
				case leaks <- string(req.Params()):
				default:
				}
			}
		}

		return nil, nil
	})

	call := func(method string, params any) {
		t.Helper()

		if _, err := rpc.Call(ctx, method, params, nil); err != nil {
			t.Fatalf("%s: %v", method, err)
		}
	}
	notify := func(method string, params any) {
		t.Helper()

		if err := rpc.Notify(ctx, method, params); err != nil {
			t.Fatalf("%s: %v", method, err)
		}
	}

	call(protocol.MethodInitialize, map[string]any{
		"workspaceFolders": []protocol.WorkspaceFolder{{URI: cfpath.ToURI(workspace), Name: "project"}},
		"capabilities": map[string]any{"workspace": map[string]any{
			"applyEdit": true, "workspaceEdit": map[string]bool{"documentChanges": true},
		}},
	})

	doc := cfpath.ToURI(filepath.Join(workspace, "Widget.cfc"))
	notify(protocol.MethodTextDocumentDidOpen, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI: doc, LanguageID: "cfml", Version: 7,
			Text: "component {\nfunction value() {\nreturn 1;\n}\n}\n",
		},
	})
	call(protocol.MethodWorkspaceExecuteCommand, map[string]any{
		"command": "clif.format", "arguments": []string{string(doc)},
	})

	select {
	case raw := <-edits:
		if !bytes.Contains(raw, []byte(`"version":7`)) {
			t.Errorf("format edit did not use the buffer version: %s", raw)
		}
	default:
		t.Fatal("clif.format did not send workspace/applyEdit")
	}

	call(protocol.MethodShutdown, nil)
	notify(protocol.MethodExit, nil)

	if err := in.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-rpc.Done():
	case <-ctx.Done():
		t.Fatal("clif did not disconnect")
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("clif exited: %v", err)
	}

	select {
	case leak := <-leaks:
		t.Errorf("private client-call trace reached the editor: %s", leak)
	default:
	}

	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	for line := range strings.SplitSeq(string(log), "\n") {
		if strings.Contains(line, "\tLSP client call\t") && strings.Contains(line, "workspace/applyEdit") &&
			strings.Contains(line, reason) && strings.Contains(line, `\"applied\":false`) {
			return
		}
	}

	t.Fatal("debug log did not record the editor's rejected workspace edit")
}
