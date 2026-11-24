//go:build !wasip1

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/daemon"
	"github.com/cfmleditor/clif/internal/index"
	cflog "github.com/cfmleditor/clif/internal/log"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/server"
)

func TestMCPDaemonClientSharesLiveIndex(t *testing.T) {
	root := t.TempDir()
	svc := filepath.Join(root, "OrderService.cfc")
	content := strings.Join([]string{
		`component {`,
		`	public numeric function calculateTotal(required numeric subtotal) {`,
		`		return subtotal * 1.1;`,
		`	}`,
		`}`,
	}, "\n")

	if err := os.WriteFile(svc, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	idx := index.New()
	idx.IndexFile(cfpath.ToURI(svc), content)

	// macOS Unix socket paths have a 104-byte limit; t.TempDir() under /var/folders
	// can exceed that with long test names.
	sockDir, err := os.MkdirTemp("/tmp", "clif-mcp-*") //nolint:usetesting // t.TempDir() exceeds macOS 104-byte Unix socket path limit
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(sockDir) })

	sockPath := filepath.Join(sockDir, "d.sock")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settings := &server.Settings{
		WorkspaceFolders: []string{root},
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- daemon.Serve(ctx, sockPath, cflog.NewLogger(false), idx, nil, settings)
	}()

	// Wait for the Unix socket to appear.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(sockPath); err == nil {
			break
		}

		time.Sleep(15 * time.Millisecond)
	}

	client, err := connectMCPDaemon(root, sockPath)
	if err != nil {
		t.Fatalf("connectMCPDaemon: %v", err)
	}

	defer func() { _ = client.Close() }()

	// Query symbols from the daemon's shared live index.
	syms, err := client.WorkspaceSymbols("calculateTotal")
	if err != nil {
		t.Fatalf("WorkspaceSymbols: %v", err)
	}

	if len(syms) == 0 || syms[0].Name != "calculateTotal" {
		t.Fatalf("WorkspaceSymbols = %+v, want calculateTotal", syms)
	}

	// LiveExplore via daemon should return mode=daemon-symbol + verbatim snippet.
	res, err := client.LiveExplore("calculateTotal", true)
	if err != nil {
		t.Fatalf("LiveExplore: %v", err)
	}

	m := mustMap(t, res)
	if m["mode"] != "daemon-symbol" {
		t.Errorf("mode = %v, want daemon-symbol", m["mode"])
	}

	defs, ok := m["definitions"].([]map[string]any)
	if !ok || len(defs) == 0 {
		t.Fatalf("definitions = %+v, want non-empty", m["definitions"])
	}

	if src, _ := defs[0]["source"].(string); !strings.Contains(src, "calculateTotal") {
		t.Errorf("expected source snippet containing calculateTotal, got %q", src)
	}

	cancel()
	<-errCh
}
