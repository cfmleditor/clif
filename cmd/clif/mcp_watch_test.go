package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/codemap/store"
)

func TestMCPMapWatcherAutoSyncsCodemapDB(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "codemap.db")

	first := filepath.Join(root, "Alpha.cfc")
	if err := os.WriteFile(first, []byte("component {\n\tfunction alphaFunc() {}\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Build initial codemap.db.
	initFlags := graphFlags{db: dbPath, quiet: true, level: "function", paths: []string{root}}
	if _, err := buildGraph(&initFlags); err != nil {
		t.Fatalf("initial buildGraph: %v", err)
	}

	// Open read-only handle as clif mcp does.
	ro, err := store.OpenReadOnly(dbPath)
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}

	defer func() { _ = ro.Close() }()

	w := newMCPMapWatcher(root, dbPath)
	w.pollInterval = 20 * time.Millisecond
	w.debounce = 40 * time.Millisecond

	w.Start(context.Background())
	defer w.Stop()

	// Write a brand-new CFC file on disk while MCP is running.
	second := filepath.Join(root, "Beta.cfc")
	if err := os.WriteFile(second, []byte("component {\n\tfunction betaAddedLive() {}\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		st := w.status()
		if cnt, _ := st["syncCount"].(int); cnt >= 1 {
			break
		}

		time.Sleep(25 * time.Millisecond)
	}

	if cnt, _ := w.status()["syncCount"].(int); cnt < 1 {
		t.Fatalf("watcher did not sync within timeout: status=%+v", w.status())
	}

	// The already-open read-only store must immediately see the new symbol via WAL.
	syms, err := ro.SearchSymbols(&store.SearchOptions{Query: "betaAddedLive", Limit: 10})
	if err != nil {
		t.Fatalf("SearchSymbols after watch sync: %v", err)
	}

	if len(syms) == 0 {
		t.Fatal("expected betaAddedLive to appear in codemap.db after --watch sync")
	}
}
