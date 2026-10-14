package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/codemap"
	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/daemon"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/route"
	"github.com/cfmleditor/clif/internal/vfs"
)

// configSet resolves each file against the .clif.json that actually governs
// it, rather than against one config chosen at the start of the scan.
//
// A TASS-shaped workspace is a dozen applications side by side, each with its own
// config: kiosk declares 52 componentResolvers, tassreporting six, and tassweb its
// own set again. Every one of them lists the others in workspacePaths, so a scan
// rooted anywhere reads all of them — under whichever single config was found
// first. The other applications' resolvers then never fire, their calls resolve to
// nothing, and their functions come out of the map with no edges: 11,387 of 11,682
// functions in the sibling applications had no caller, which reads as a dead
// codebase rather than as a scan configured for somebody else's code.
//
// Nothing about this is TASS-specific. Any monorepo with per-package settings has
// the same shape.
//
// The index is deliberately shared across every config. Function signatures are a
// property of the workspace and do not change with whose resolvers you read them
// under; duplicating it per config would multiply the memory and let the same file
// resolve differently depending on which application asked.
type configSet struct {
	fsys   vfs.FS
	shared *index.Index

	mu       sync.RWMutex
	byDir    map[string]codemap.FileConfig // directory → governing config
	byPath   map[string]codemap.FileConfig // config file path → built environment
	fallback codemap.FileConfig
	seen     map[string]string // config path → content hash, for the fingerprint
}

func newConfigSet(fsys vfs.FS, shared *index.Index, fallback *codemap.FileConfig) *configSet {
	return &configSet{
		fsys:     fsys,
		shared:   shared,
		byDir:    make(map[string]codemap.FileConfig),
		byPath:   make(map[string]codemap.FileConfig),
		fallback: *fallback,
		seen:     make(map[string]string),
	}
}

// use makes cfg the environment for files the config at path governs, in place
// of building another from it. The scan's own resolver was built from that
// config already, and building a second left every file read under a copy with
// empty caches while the first went unused.
func (cs *configSet) use(path string, cfg codemap.FileConfig) {
	hash := ""

	if data, err := vfs.ReadRegular(path); err == nil {
		sum := sha256.Sum256(data)
		hash = hex.EncodeToString(sum[:])
	}

	cs.mu.Lock()
	cs.byPath[path] = cfg
	cs.seen[path] = hash
	cs.mu.Unlock()
}

// For returns the resolution environment for one file, memoised per directory.
// It is called once per file from the scan's worker goroutines.
func (cs *configSet) For(file string) codemap.FileConfig {
	dir := filepath.Dir(file)

	cs.mu.RLock()
	cfg, ok := cs.byDir[dir]
	cs.mu.RUnlock()

	if ok {
		return cfg
	}

	cfg = cs.build(dir)

	cs.mu.Lock()
	cs.byDir[dir] = cfg
	cs.mu.Unlock()

	return cfg
}

func (cs *configSet) build(dir string) codemap.FileConfig {
	found, _ := daemon.FindConfig(dir)
	if found == nil {
		return cs.fallback
	}

	cs.mu.RLock()
	cfg, ok := cs.byPath[found.Path]
	cs.mu.RUnlock()

	if ok {
		return cfg
	}

	lookupRoots := []string{filepath.Dir(found.Path)}
	if cs.fallback.Resolver != nil && len(cs.fallback.Resolver.WorkspaceFolders) > 0 {
		lookupRoots = cs.fallback.Resolver.WorkspaceFolders
	}

	bs, _ := batchSettings(cs.fsys, found, cliWorkspaceFolders(cs.fsys, found, lookupRoots))
	resolver := resolve.NewBatchResolver(cs.fsys, cs.shared, &bs)

	cfg = codemap.FileConfig{Resolver: resolver, Routes: routeResolver(cs.fsys, resolver, found)}

	hash := ""

	if data, err := vfs.ReadRegular(found.Path); err == nil {
		sum := sha256.Sum256(data)
		hash = hex.EncodeToString(sum[:])
	}

	cs.mu.Lock()
	cs.byPath[found.Path] = cfg
	cs.seen[found.Path] = hash
	cs.mu.Unlock()

	return cfg
}

// Fingerprint identifies every config used so far, so a cached edge set is not
// served after the resolvers that produced it change. Call it after the scan.
func (cs *configSet) Fingerprint() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	paths := make([]string, 0, len(cs.seen))
	for p := range cs.seen {
		paths = append(paths, p)
	}

	sort.Strings(paths)

	var b strings.Builder
	for _, p := range paths {
		fmt.Fprintf(&b, "%s=%s\n", p, cs.seen[p])
	}

	return b.String()
}

// Configs names the config files in play, for the scan's own report.
func (cs *configSet) Configs() []string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	paths := make([]string, 0, len(cs.seen))
	for p := range cs.seen {
		paths = append(paths, p)
	}

	sort.Strings(paths)

	return paths
}

// preload builds the environment for every directory that holds a .clif.json
// under the scan roots, before the scan starts.
//
// Two reasons it is not left to happen lazily. The fingerprint has to cover every
// config the build *could* use, and one discovered late would not be in the
// fingerprint that keyed the cache writes already made. And discovering them up
// front is what lets the command tell you which configs it is about to use, which
// is the difference between a surprising result and an explained one.
func (cs *configSet) preload(roots []string) {
	for _, root := range roots {
		// An unreadable entry is skipped, not fatal: a preload that aborted on
		// one permission error would silently fall back to a single config for
		// the whole scan, which is the bug this exists to prevent.
		_ = cs.fsys.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && config.IsFileName(filepath.Base(path)) {
				cs.For(filepath.Join(filepath.Dir(path), "x.cfc"))
			}

			return nil
		})
	}
}

// routeResolver wires the "routes" config to the workspace.
//
// The lookups are closures over the resolver rather than a dependency from
// internal/route onto internal/resolve, because the same convention has to be
// applied from the code-map builder, from the LSP handlers and from tests, and
// each knows the workspace differently.
func routeResolver(fsys vfs.FS, resolver *resolve.Resolver, cfg *daemon.Config) *route.Resolver {
	rc := cfg.Routes()
	if !rc.Enabled() {
		return nil
	}

	root := filepath.Dir(cfg.Path)

	// Paths are resolved against the config's own directory first and the
	// workspace folders after, so a route written in a shared view resolves the
	// same way whichever application's page included it.
	roots := append([]string{root}, cfg.WorkspaceFolders()...)

	var (
		mu    sync.Mutex
		stats = map[string]bool{}
	)

	return &route.Resolver{
		Config: rc,
		Lookups: route.Lookups{
			ComponentPath: func(component string) string {
				return resolver.ComponentPath(component, root)
			},
			HasMethod: func(cfcPath, method string) bool {
				for _, fn := range resolver.EnsureIndexed(cfcPath) {
					if strings.EqualFold(fn.Name, method) {
						return true
					}
				}

				return false
			},
			DirExists: func(rel string) bool {
				mu.Lock()
				defer mu.Unlock()

				key := "d\x00" + rel
				if v, ok := stats[key]; ok {
					return v
				}

				found := false

				for _, r := range roots {
					if info, err := fsys.Stat(filepath.Join(r, filepath.FromSlash(rel))); err == nil && info.IsDir() {
						found = true

						break
					}
				}

				stats[key] = found

				return found
			},
			FindFile: func(rel string) string {
				for _, r := range roots {
					p := filepath.Join(r, filepath.FromSlash(rel))
					if info, err := fsys.Stat(p); err == nil && !info.IsDir() {
						return p
					}
				}

				return ""
			},
		},
	}
}

// seedRootConfig gives the files the root's own config governs the resolver
// routeWorkspace built from it, with that config's routes, rather than a second
// resolver configSet would build from the same file. The two read files alike
// (one index, one bean map), so the copy was only a second set of empty caches;
// unresolved stopped building it for the same reason.
func seedRootConfig(fsys vfs.FS, configs *configSet, root string, fallback *codemap.FileConfig) {
	cfg, _ := daemon.FindConfig(root)
	if cfg == nil || cfg.Path == "" || fallback.Resolver == nil {
		return
	}

	configs.use(cfg.Path, codemap.FileConfig{Resolver: fallback.Resolver, Routes: routeResolver(fsys, fallback.Resolver, cfg)})
}
