package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

type fileStamp struct {
	size    int64
	modTime time.Time
}

// mcpMapWatcher watches a workspace for CFML file edits and incrementally
// rebuilds codemap.db using buildGraph's SQLite file cache (WAL mode allows
// concurrent read-only queries while the sync commits).
type mcpMapWatcher struct {
	root         string
	dbPath       string
	pollInterval time.Duration
	debounce     time.Duration

	mu         sync.Mutex
	stamps     map[string]fileStamp
	pending    map[string]struct{}
	syncing    bool
	lastSynced time.Time
	syncCount  int
	lastErr    string
	dirtySince time.Time
	cancel     context.CancelFunc
	done       chan struct{}
}

func newMCPMapWatcher(root, dbPath string) *mcpMapWatcher {
	return &mcpMapWatcher{
		root:         root,
		dbPath:       dbPath,
		pollInterval: 1 * time.Second,
		debounce:     2 * time.Second,
		pending:      make(map[string]struct{}),
	}
}

// Start takes an initial snapshot of the workspace and launches the background
// watch loop. Call Stop to shut it down cleanly.
func (w *mcpMapWatcher) Start(parent context.Context) {
	w.mu.Lock()
	w.stamps = snapshotCFMLStamps(w.root)
	ctx, cancel := context.WithCancel(parent)
	w.cancel = cancel
	w.done = make(chan struct{})
	w.mu.Unlock()

	go w.loop(ctx)
}

// Stop cancels the watch loop and waits for any in-flight sync to finish.
func (w *mcpMapWatcher) Stop() {
	w.mu.Lock()
	cancel := w.cancel
	done := w.done
	w.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if done != nil {
		<-done
	}
}

func (w *mcpMapWatcher) loop(ctx context.Context) {
	defer close(w.done)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *mcpMapWatcher) tick(ctx context.Context) {
	now := time.Now()
	current := snapshotCFMLStamps(w.root)

	w.mu.Lock()

	changed := diffStamps(w.stamps, current)
	if len(changed) > 0 {
		w.stamps = current
		for _, f := range changed {
			w.pending[f] = struct{}{}
		}

		w.dirtySince = now
	}

	shouldSync := len(w.pending) > 0 && !w.syncing && !w.dirtySince.IsZero() && now.Sub(w.dirtySince) >= w.debounce
	if shouldSync {
		w.syncing = true
	}

	w.mu.Unlock()

	if !shouldSync || ctx.Err() != nil {
		return
	}

	err := w.rebuild()

	w.mu.Lock()
	w.syncing = false

	if err != nil {
		w.lastErr = err.Error()
		fmt.Fprintf(os.Stderr, "clif mcp --watch: sync failed: %v\n", err)
	} else {
		w.lastErr = ""
		w.lastSynced = time.Now()
		w.syncCount++

		clear(w.pending)

		w.dirtySince = time.Time{}
	}

	w.mu.Unlock()
}

func (w *mcpMapWatcher) rebuild() error {
	flags := graphFlags{
		db:    w.dbPath,
		quiet: true,
		level: "function",
		paths: []string{w.root},
	}

	_, err := buildGraph(&flags)

	return err
}

func (w *mcpMapWatcher) status() map[string]any {
	w.mu.Lock()
	defer w.mu.Unlock()

	pending := make([]string, 0, len(w.pending))
	for f := range w.pending {
		rel, err := filepath.Rel(w.root, f)
		if err == nil {
			pending = append(pending, filepath.ToSlash(rel))
		} else {
			pending = append(pending, f)
		}
	}

	slices.Sort(pending)

	out := map[string]any{
		"enabled":      true,
		"syncing":      w.syncing,
		"syncCount":    w.syncCount,
		"pendingFiles": pending,
	}

	if !w.lastSynced.IsZero() {
		out["lastSynced"] = w.lastSynced.UTC().Format(time.RFC3339)
	}

	if w.lastErr != "" {
		out["lastError"] = w.lastErr
	}

	return out
}

func snapshotCFMLStamps(root string) map[string]fileStamp {
	files := cfpath.CollectCFML(vfs.OS{}, []string{root})
	out := make(map[string]fileStamp, len(files)+1)

	for _, f := range files {
		if info, err := os.Stat(f); err == nil {
			out[f] = fileStamp{size: info.Size(), modTime: info.ModTime()}
		}
	}

	cfgPath := filepath.Join(root, ".clif.json")
	if info, err := os.Stat(cfgPath); err == nil {
		out[cfgPath] = fileStamp{size: info.Size(), modTime: info.ModTime()}
	}

	return out
}

func diffStamps(prev, curr map[string]fileStamp) []string {
	var changed []string

	for f, st := range curr {
		if old, ok := prev[f]; !ok || old.size != st.size || !old.modTime.Equal(st.modTime) {
			changed = append(changed, f)
		}
	}

	for f := range prev {
		if _, ok := curr[f]; !ok {
			changed = append(changed, f)
		}
	}

	return changed
}
