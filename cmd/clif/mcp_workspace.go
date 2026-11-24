package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cfmleditor/clif/internal/daemon"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
	"go.lsp.dev/uri"
)

// mcpWorkspace holds a lazily initialized, incrementally refreshed workspace
// resolver shared across live MCP tools (explain_call, find_references,
// get_type_hierarchy, resolve_route).
//
// Building the index lazily keeps MCP startup instantaneous; refreshing only
// files whose mtime changed (or rebuilding if .clif.json changed) keeps
// subsequent calls fast while staying accurate as files are edited on disk.
type mcpWorkspace struct {
	root string

	mu          sync.Mutex
	env         *explainEnv
	cfg         *daemon.Config
	cfgPath     string
	cfgModTime  time.Time
	fileModTime map[string]time.Time
}

func newMCPWorkspace(root string) *mcpWorkspace {
	return &mcpWorkspace{root: root}
}

func (w *mcpWorkspace) get() (*explainEnv, *daemon.Config, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	absRoot, err := filepath.Abs(w.root)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving workspace root %s: %w", w.root, err)
	}

	cfg, _ := daemon.FindConfig(absRoot)

	var (
		cfgPath string
		cfgMod  time.Time
	)

	if cfg != nil && cfg.Path != "" {
		cfgPath = cfg.Path
		if info, statErr := os.Stat(cfgPath); statErr == nil {
			cfgMod = info.ModTime()
		}
	}

	if w.env == nil || cfgPath != w.cfgPath || !cfgMod.Equal(w.cfgModTime) {
		return w.rebuildLocked(absRoot, cfg, cfgPath, cfgMod)
	}

	w.refreshLocked()

	return w.env, w.cfg, nil
}

func (w *mcpWorkspace) rebuildLocked(absRoot string, cfg *daemon.Config, cfgPath string, cfgMod time.Time) (*explainEnv, *daemon.Config, error) {
	fsys := vfs.OS{}
	env := newExplainEnv(fsys, absRoot, true, &unresolvedFlags{})

	if env == nil || env.resolver == nil {
		return nil, nil, fmt.Errorf("could not index the workspace at %s", w.root)
	}

	files := cfpath.CollectCFML(fsys, env.opt.WorkspaceFolders)
	mtimes := make(map[string]time.Time, len(files))

	for _, f := range files {
		if info, err := os.Stat(f); err == nil {
			mtimes[f] = info.ModTime()
		}
	}

	w.env = env
	w.cfg = cfg
	w.cfgPath = cfgPath
	w.cfgModTime = cfgMod
	w.fileModTime = mtimes

	return w.env, w.cfg, nil
}

func (w *mcpWorkspace) refreshLocked() {
	fsys := vfs.OS{}
	files := cfpath.CollectCFML(fsys, w.env.opt.WorkspaceFolders)
	seen := make(map[string]struct{}, len(files))

	var changed []string

	for _, f := range files {
		seen[f] = struct{}{}

		info, err := os.Stat(f)
		if err != nil {
			continue
		}

		mt := info.ModTime()
		if prev, ok := w.fileModTime[f]; !ok || !mt.Equal(prev) {
			changed = append(changed, f)
			w.fileModTime[f] = mt
		}
	}

	for f := range w.fileModTime {
		if _, ok := seen[f]; !ok {
			w.env.resolver.Index.RemoveFile(uri.File(f))
			delete(w.fileModTime, f)
		}
	}

	if len(changed) > 0 {
		w.env.resolver.IndexFiles(changed, nil)
	}
}
