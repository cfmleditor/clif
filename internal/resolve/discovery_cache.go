package resolve

import (
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/confine"

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
		if confine.Within(pathKey(file), []string{dir}) {
			return true
		}
	}

	return false
}

// reads reports whether file was read or probed as a file, or is an entry of a
// directory that was listed.
func (d *DiscoveryDependencies) reads(file string) bool {
	if d == nil {
		return false
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.files[pathKey(file)] || d.dirs[pathKey(filepath.Dir(file))]
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

	return confine.Within(file, slices.Collect(maps.Values(r.managedBeanPaths())))
}

// DiscoveryReadsFile reports whether discovery read this file itself, or listed
// the directory it is in: the files a project's markers are made of, which
// are not CFML (a JSON config, a plugin's config.xml). It is exact where
// DiscoveryAffected is broad, which a file that is no source cannot afford:
// the whole of a directory discovery listed counts for DiscoveryAffected, so an
// unrelated JSON file anywhere under the workspace root would drop the
// resolver, and a file is created or removed in a listed directory only to
// change that listing.
func (r *Resolver) DiscoveryReadsFile(file string) bool {
	r.configurationOnce.Do(r.recordConfigurationSources)

	// What discovery reads that is not CFML it probes by exact path: the
	// JSON configs up the directory chain and plugin/config.xml. Its walks of
	// a project read .cfc files, which are sources (DiscoveryAffected). A
	// managed bean root is configuration, so anything under it counts.
	return r.Discovery.reads(file) || confine.Within(file, slices.Collect(maps.Values(r.managedBeanPaths())))
}

// InvalidatePaths drops source/path caches while retaining immutable discovery
// policy and bean identities. Configuration changes replace the whole resolver.
func (r *Resolver) InvalidatePaths() {
	r.mu.Lock()
	views := r.contextViews
	r.contextViews = nil
	r.pathCacheEpoch++
	r.pathCaches = pathCaches{}
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
