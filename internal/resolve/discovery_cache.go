package resolve

import (
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

// DiscoveryDependencies records sources actually read during bean/DI discovery.
// It is shared with the server so ordinary saves need not repeat discovery.
type DiscoveryDependencies struct {
	mu    sync.RWMutex
	files map[string]bool
	dirs  map[string]bool
}

func (d *DiscoveryDependencies) add(file string) {
	if d == nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.files == nil {
		d.files = map[string]bool{}
	}

	d.files[pathKey(file)] = true
}

func (d *DiscoveryDependencies) contains(file string) bool {
	if d == nil {
		return false
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.files[pathKey(file)] {
		return true
	}

	for dir := range d.dirs {
		rel, err := filepath.Rel(dir, pathKey(file))
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

type discoveryFS struct {
	vfs.FS
	dependencies *DiscoveryDependencies
}

func (f discoveryFS) ReadFile(path string) ([]byte, error) {
	f.dependencies.add(path)

	return f.FS.ReadFile(path)
}

func (f discoveryFS) ReadDir(path string) ([]fs.DirEntry, error) {
	if d := f.dependencies; d != nil {
		d.mu.Lock()
		if d.dirs == nil {
			d.dirs = map[string]bool{}
		}

		d.dirs[pathKey(path)] = true
		d.mu.Unlock()
	}

	return f.FS.ReadDir(path)
}

func (f discoveryFS) Stat(path string) (fs.FileInfo, error) {
	f.dependencies.add(path)

	return f.FS.Stat(path)
}

// TrackDiscoveryFS preserves the filesystem while recording discovery reads,
// including failed reads so a newly created dependency invalidates the cache.
func TrackDiscoveryFS(sourceFS vfs.FS, d *DiscoveryDependencies) vfs.FS {
	if sourceFS == nil {
		sourceFS = vfs.OS{}
	}

	return discoveryFS{sourceFS, d}
}

// DiscoveryAffected reports source dependencies and files inside managed roots.
func (r *Resolver) DiscoveryAffected(file string) bool {
	r.configurationOnce.Do(r.recordConfigurationSources)

	if r.Discovery.contains(file) {
		return true
	}

	for _, root := range r.managedBeanPaths() {
		rel, err := filepath.Rel(root, file)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

// InvalidatePaths drops source/path caches while retaining immutable discovery
// policy and bean identities. Configuration changes replace the whole resolver.
func (r *Resolver) InvalidatePaths() {
	r.mu.Lock()
	views := r.contextViews
	r.contextViews = nil
	r.resolveCache = nil
	r.includeCache = nil
	r.interfaceCache = nil
	r.dirCache = nil
	r.appRootCache = nil
	r.slugCache = nil
	r.incGraph = nil
	r.implicitCache = nil
	r.helpers = nil
	r.wb = nil
	r.startupCache = nil
	r.wheelsSources = nil
	r.ctlPathCache = nil
	r.includerCache = nil
	r.callerIdx = nil
	r.argCache = nil
	r.requestWriterIdx = nil
	r.wrapperHosts = nil
	r.wrapperCalls = nil
	r.lazyGetters = nil
	r.lazyFiles = nil
	r.viewArgCache = nil
	r.handoffs = handoffIndex{}
	r.handlerCache = nil
	r.fw1ActionCache = nil
	r.boundEntityCache = nil
	r.extraIncludeHosts = nil
	r.muraEventFiles = nil
	r.returnCache = returnCache{}
	r.funcCache = funcCache{}
	r.computedIncludes = nil
	r.fw1Scopes = nil
	r.mu.Unlock()

	for _, view := range views {
		view.InvalidatePaths()
	}
}

func (r *Resolver) recordConfigurationSources() {
	if r.Discovery == nil {
		return
	}

	read := func(file string) ([]byte, error) {
		r.Discovery.add(file)

		return r.fs().ReadFile(file)
	}

	for _, root := range r.WorkspaceFolders {
		if app := r.FindApplicationRoot(root); app != "" {
			for _, name := range []string{"Application.cfc", "Application.cfm"} {
				cfpath.ApplicationMappingSources(filepath.Join(app, name), r.Mappings, read)
			}
		}
	}

	seen := map[string]bool{}

	for _, file := range r.StartupFiles {
		r.Discovery.add(file)
	}

	queue := r.configuredStartupFiles()
	for len(queue) > 0 && len(seen) < maxStartupTemplates {
		file := queue[0]
		queue = queue[1:]

		if seen[pathKey(file)] {
			continue
		}

		seen[pathKey(file)] = true

		data, err := read(file)
		if err != nil {
			continue
		}

		for _, include := range parser.ExtractIncludes(string(data)) {
			r.Discovery.add(filepath.Join(filepath.Dir(file), filepath.FromSlash(include)))

			for _, root := range r.WorkspaceFolders {
				r.Discovery.add(filepath.Join(root, strings.TrimLeft(filepath.FromSlash(include), string(filepath.Separator))))
			}

			for prefix, root := range r.effectiveMappings(filepath.Dir(file)) {
				virtual := strings.TrimLeft(filepath.ToSlash(include), "/")
				if strings.HasPrefix(strings.ToLower(virtual), strings.ToLower(strings.Trim(prefix, "/"))+"/") {
					r.Discovery.add(filepath.Join(root, filepath.FromSlash(virtual[len(strings.Trim(prefix, "/"))+1:])))
				}
			}

			if path := r.IncludePath(include, file); path != "" {
				queue = append(queue, path)
			}
		}
	}
}
