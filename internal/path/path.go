// Package path resolves CFML component dot-paths to filesystem paths.
package path

import (
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cfmleditor/clif/internal/parser"
	"github.com/cfmleditor/clif/internal/vfs"
)

// DefaultFS is the filesystem used by this package. Override for testing or WASM.
var DefaultFS vfs.FS = vfs.OS{}

var (
	appMappingsCache   = make(map[string]map[string]string)
	appMappingsCacheMu sync.RWMutex
)

// ParseApplicationMappings delegates to parser.ParseApplicationMappings.
func ParseApplicationMappings(content string, appDir string) map[string]string {
	return parser.ParseApplicationMappings(content, appDir)
}

// readApplicationFile reads Application.cfc or Application.cfm from appDir.
func readApplicationFile(appDir string) []byte {
	for _, name := range []string{"Application.cfc", "Application.cfm"} {
		data, err := DefaultFS.ReadFile(filepath.Join(appDir, name))
		if err == nil {
			return data
		}
	}

	return nil
}

// LoadAppMappings returns cached Application.cfc mappings for appDir,
// loading and parsing them on first access.
func LoadAppMappings(appDir string) map[string]string {
	return LoadAppMappingsWithInitial(appDir, nil)
}

// LoadAppMappingsWithInitial resolves parent constructors from frozen server
// and editor mappings. The cache key includes these bootstrap inputs, so two
// callers with different explicit mappings cannot share inherited facts.
func LoadAppMappingsWithInitial(appDir string, initial map[string]string) map[string]string {
	keys := make([]string, 0, len(initial))
	for key := range initial {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	cacheKey := appDir

	var cacheKeySb58 strings.Builder
	for _, key := range keys {
		cacheKeySb58.WriteString("\x00" + key + "\x00" + initial[key])
	}

	cacheKey += cacheKeySb58.String()

	appMappingsCacheMu.RLock()

	if m, ok := appMappingsCache[cacheKey]; ok {
		appMappingsCacheMu.RUnlock()

		return m
	}

	appMappingsCacheMu.RUnlock()

	m := bootstrapMappings(appDir, initial, DefaultFS.ReadFile)

	app := ApplicationMappingSources(filepath.Join(appDir, "Application.cfc"), initial, DefaultFS.ReadFile)
	for _, name := range []string{"Application.cfc", "Application.cfm"} {
		file := filepath.Join(appDir, name)
		if _, err := DefaultFS.Stat(file); err != nil {
			continue
		}

		if name != "Application.cfc" {
			app = ApplicationMappingSources(file, initial, DefaultFS.ReadFile)
		}

		maps.Copy(m, app)

		break
	}

	appMappingsCacheMu.Lock()
	appMappingsCache[cacheKey] = m
	appMappingsCacheMu.Unlock()

	return m
}

// ApplicationMappingSources uses only server/editor configuration to bootstrap
// ancestry. Reads include missing physical parent candidates for invalidation.
func ApplicationMappingSources(file string, initial map[string]string, read func(string) ([]byte, error)) map[string]string {
	bootstrap, parent := ApplicationParentLookup(filepath.Dir(file), initial, read)

	return parser.MappingSourcesWithParent(file, bootstrap, read, parent)
}

// ApplicationParentLookup freezes the inputs used to resolve application
// ancestry before constructor mappings are available.
func ApplicationParentLookup(appDir string, initial map[string]string, read func(string) ([]byte, error)) (map[string]string, func(string, string) string) {
	bootstrap := bootstrapMappings(appDir, initial, read)
	for key, value := range initial {
		for existing := range bootstrap {
			if strings.EqualFold(existing, key) {
				delete(bootstrap, existing)
			}
		}

		bootstrap[key] = value
	}

	return bootstrap, func(name, dir string) string {
		if IsRelativeComponentPath(name) {
			candidate := strings.ReplaceAll(name, "\\", "/")
			if !strings.HasSuffix(strings.ToLower(candidate), ".cfc") {
				candidate += ".cfc"
			}

			_, _ = read(filepath.Join(dir, filepath.FromSlash(candidate)))

			return ResolvePath(name, dir, nil)
		}

		_, _ = read(filepath.Join(dir, filepath.FromSlash(strings.ReplaceAll(name, ".", "/"))+".cfc"))
		for key, root := range bootstrap {
			prefix := strings.ReplaceAll(strings.Trim(key, "/"), "/", ".")
			if prefix == "" || len(name) > len(prefix) && strings.EqualFold(name[:len(prefix)], prefix) && name[len(prefix)] == '.' {
				rest := name
				if prefix != "" {
					rest = name[len(prefix)+1:]
				}

				_, _ = read(filepath.Join(root, filepath.FromSlash(strings.ReplaceAll(rest, ".", "/"))+".cfc"))
			}
		}

		return ResolvePath(name, dir, bootstrap)
	}
}

// InvalidateAppMappingsCache clears the cached Application.cfc mappings.
func InvalidateAppMappingsCache() {
	appMappingsCacheMu.Lock()
	appMappingsCache = make(map[string]map[string]string)
	appMappingsCacheMu.Unlock()
}

// ParseAppBeanPaths delegates to parser.ParseAppBeanPaths.
func ParseAppBeanPaths(content string, appDir string) map[string]string {
	return parser.ParseAppBeanPaths(content, appDir)
}

// LoadAppBeanPaths returns cached Application.cfc bean paths for appDir.
func LoadAppBeanPaths(appDir string) map[string]string {
	appMappingsCacheMu.RLock()

	key := appDir + "|beans"
	if m, ok := appMappingsCache[key]; ok {
		appMappingsCacheMu.RUnlock()

		return m
	}

	appMappingsCacheMu.RUnlock()

	content := readApplicationFile(appDir)
	if content == nil {
		return nil
	}

	m := parser.ParseAppBeanPaths(string(content), appDir)

	appMappingsCacheMu.Lock()
	appMappingsCache[key] = m
	appMappingsCacheMu.Unlock()

	return m
}

// ParseOrmLocations delegates to parser.ParseOrmLocations.
func ParseOrmLocations(content string, appDir string) []string {
	return parser.ParseOrmLocations(content, appDir)
}

// LoadOrmLocations returns cached ORM cfcLocation paths for appDir.
func LoadOrmLocations(appDir string) []string {
	appMappingsCacheMu.RLock()

	key := appDir + "|orm"
	if m, ok := appMappingsCache[key]; ok {
		appMappingsCacheMu.RUnlock()

		out := make([]string, 0, len(m))
		for _, v := range m {
			out = append(out, v)
		}

		return out
	}

	appMappingsCacheMu.RUnlock()

	content := readApplicationFile(appDir)
	if content == nil {
		return nil
	}

	locs := parser.ParseOrmLocations(string(content), appDir)

	m := make(map[string]string, len(locs))
	for i, l := range locs {
		m[string(rune('0'+i))] = l
	}

	appMappingsCacheMu.Lock()
	appMappingsCache[key] = m
	appMappingsCacheMu.Unlock()

	return locs
}

// ResolvePath resolves a CFML dot-path (e.g. "models.User") to an absolute
// .cfc file path. It checks mappings first (matching the first segment, case-insensitively),
// then falls back to resolving relative to baseDir. CFML component paths are conventionally
// case-insensitive, so every path segment is matched without regard to case. Returns empty
// string if not found.
func ResolvePath(dotPath string, baseDir string, mappings map[string]string) string {
	return ResolvePathCached(dotPath, baseDir, mappings, nil)
}

// ResolvePathCached is ResolvePath with a DirCache. A nil cache lists every
// directory afresh, which is what ResolvePath does.
func ResolvePathCached(dotPath string, baseDir string, mappings map[string]string, cache *DirCache) string {
	if IsRelativeComponentPath(dotPath) {
		return resolveRelativeComponentPath(dotPath, baseDir, cache)
	}
	// Normalize CF slash-paths (e.g. /tassweb/packages/foo) to dot-paths
	if strings.Contains(dotPath, "/") {
		dotPath = strings.TrimPrefix(dotPath, "/")
		dotPath = strings.ReplaceAll(dotPath, "/", ".")
	}

	if abs := resolveNestedMapping(dotPath, mappings, cache); abs != "" {
		return abs
	}

	parts := strings.SplitN(dotPath, ".", 2)
	if mappings != nil {
		if mapped, ok := lookupFold(mappings, parts[0]); ok {
			var segments []string
			if len(parts) == 2 {
				segments = strings.Split(parts[1], ".")
			} else {
				segments = []string{parts[0]}
			}

			segments[len(segments)-1] += ".cfc"

			if abs := resolveSegmentsCached(mapped, segments, cache); abs != "" {
				return abs
			}
		}
	}

	segments := strings.Split(dotPath, ".")
	segments[len(segments)-1] += ".cfc"

	if abs := resolveSegmentsCached(baseDir, segments, cache); abs != "" {
		return abs
	}

	// The empty key represents a configured "/" default. Named mappings and
	// components beside the caller retain precedence.
	if root, ok := lookupFold(mappings, ""); ok && root != "" {
		return resolveSegmentsCached(root, segments, cache)
	}

	return ""
}

// IsRelativeComponentPath distinguishes an explicit declaring-relative path
// from a mapped slash path or dotted component name. Its dots are navigation
// or filename characters, rather than package separators.
func IsRelativeComponentPath(component string) bool {
	component = strings.ReplaceAll(component, "\\", "/")

	return strings.HasPrefix(component, "./") || strings.HasPrefix(component, "../")
}

func resolveRelativeComponentPath(component, baseDir string, cache *DirCache) string {
	component = strings.ReplaceAll(component, "\\", "/")
	if strings.ContainsAny(component, "#\x00") || strings.HasSuffix(component, "/") {
		return ""
	}

	segments := strings.Split(component, "/")

	last := len(segments) - 1
	if !strings.HasSuffix(strings.ToLower(segments[last]), ".cfc") {
		segments[last] += ".cfc"
	}

	dir := baseDir

	for _, segment := range segments {
		switch segment {
		case ".":
		case "..":
			dir = filepath.Dir(dir)
		default:
			if segment == "" {
				return ""
			}

			dir = resolveSegmentsCached(dir, []string{segment}, cache)
			if dir == "" {
				return ""
			}
		}
	}

	if info, err := DefaultFS.Stat(dir); err == nil && !info.IsDir() {
		return dir
	}

	return ""
}

// InFolderNamed returns where rel lands when its first segment names one of
// roots: the rest of rel joined onto each root whose base name is that segment,
// case-insensitively. rel is a slash path, with or without a leading slash
// ("/tassweb/includes/x.cfm"). It returns nothing when rel has one segment.
//
// It is the path form of the rule behind Resolver's folder-named fallback: a
// workspace folder called tassweb stands for a mapping of "tassweb" onto
// itself, so a project whose mappings only ever named its own folders needs
// none. Callers try it last, after every configured and relative lookup, so it
// only answers a path nothing else did.
func InFolderNamed(roots []string, rel string) []string {
	seg, rest, ok := strings.Cut(strings.TrimPrefix(filepath.ToSlash(rel), "/"), "/")
	if !ok || seg == "" || rest == "" {
		return nil
	}

	var out []string

	for _, root := range roots {
		if strings.EqualFold(filepath.Base(root), seg) {
			out = append(out, filepath.Join(root, filepath.FromSlash(rest)))
		}
	}

	return out
}

// lookupFold returns the value for the first key in m that matches key
// case-insensitively, and whether one was found.
// resolveNestedMapping resolves dotPath through a mapping whose name has more
// than one segment — `this.mappings[ "/modules/wheels" ]` beside
// `this.mappings[ "/modules" ]` — which the first-segment lookup below cannot
// see. The engine takes the longest mapping that prefixes the path, so the
// candidates are tried longest first; a path none of them holds falls through
// to the single-segment lookup.
func resolveNestedMapping(dotPath string, mappings map[string]string, cache *DirCache) string {
	var keys []string

	for k := range mappings {
		if norm := nestedMappingKey(k); norm != "" && len(dotPath) > len(norm) &&
			dotPath[len(norm)] == '.' && strings.EqualFold(dotPath[:len(norm)], norm) {
			keys = append(keys, k)
		}
	}

	slices.SortFunc(keys, func(a, b string) int {
		if d := len(nestedMappingKey(b)) - len(nestedMappingKey(a)); d != 0 {
			return d
		}

		return strings.Compare(a, b)
	})

	for _, k := range keys {
		segments := strings.Split(dotPath[len(nestedMappingKey(k))+1:], ".")
		segments[len(segments)-1] += ".cfc"

		if abs := resolveSegmentsCached(mappings[k], segments, cache); abs != "" {
			return abs
		}
	}

	return ""
}

// nestedMappingKey is a mapping name with more than one segment spelled as a
// dot-path prefix (`modules/wheels` → `modules.wheels`), or "" for a
// single-segment one, which the first-segment lookup already answers.
func nestedMappingKey(k string) string {
	k = strings.ReplaceAll(strings.Trim(k, "/"), "/", ".")
	if !strings.Contains(k, ".") {
		return ""
	}

	return k
}

func lookupFold(m map[string]string, key string) (string, bool) {
	if v, ok := m[key]; ok {
		return v, true
	}

	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}

	return "", false
}

// DirCache remembers the names in a directory, so resolving a family of
// dot-paths lists each of their shared parents once instead of once per path.
//
// The listing above is per path *segment*, so resolving
// app.packages.core.mod1.A, .mod1.B and .mod2.C lists app, packages and core
// again for every one of them. Measured over 1,000 components under a shared
// four-segment prefix: 5,000 ReadDir calls and 68,000 entries walked, against
// 44 and 1,043 with this. 31.7ms against 1.2ms.
//
// It holds names rather than fs.DirEntry values because names are all
// resolveSegments reads, and a name is a string where a DirEntry is an
// interface over a stat result.
//
// The cache belongs to a resolve.Resolver rather than to this package. That is
// what gives it the right lifetime for nothing: the Resolver is already dropped
// wherever the server decides a path answer may have gone stale — a save, a
// watched-file change, a reindex — and dropping it drops this with it. A
// package-level cache would need that plumbing built again, would be shared
// between daemon sessions, and would quietly serve one test's filesystem to the
// next, since DefaultFS is swappable.
type DirCache struct {
	mu    sync.RWMutex
	names map[string][]string

	// recent holds every listing read for MissTTL, misses included: one
	// parse asks the same directories again and again, and a lookup that
	// failed in a directory listed it each time — 491 listings, a quarter of
	// opening a 22,000-line component in the editor. keep's reason for not
	// remembering a miss is a component written later, which this denies
	// for at most MissTTL.
	recent map[string]recentListing

	// keepAll keeps every listing read, misses included: right only where
	// nothing is written while the cache lives (a batch scan).
	keepAll bool
}

type recentListing struct {
	names []string
	at    time.Time
}

// MissTTL is how long a listing a lookup did not find its segment in is
// trusted. A variable so tests elsewhere can shorten it.
var MissTTL = time.Second

// NewDirCache returns an empty cache.
func NewDirCache() *DirCache { return &DirCache{} }

// NewSnapshotDirCache returns an empty cache for a run over files that do not
// change while it lasts, which keeps every listing it reads, so a directory a
// lookup failed in is not listed again. keep's reason for leaving those out
// is a component written while the cache lives, which a batch scan has not.
// Re-listing them was 0.85 GB of a tassweb scan's allocation.
func NewSnapshotDirCache() *DirCache { return &DirCache{keepAll: true} }

// namesIn returns the entry names in dir and whether they came from the cache.
// It does not populate it — see keep.
func (c *DirCache) namesIn(dir string) (names []string, cached bool, err error) {
	if c != nil {
		c.mu.RLock()
		names, cached = c.names[dir]
		c.mu.RUnlock()

		if cached {
			return names, true, nil
		}
	}

	if c != nil {
		c.mu.RLock()
		r, ok := c.recent[dir]
		c.mu.RUnlock()

		if ok && time.Since(r.at) < MissTTL {
			return r.names, false, nil
		}
	}

	names, err = readDirNames(dir)
	if c != nil && c.keepAll && err == nil {
		c.keep(dir, names)

		return names, true, nil
	}

	if c != nil && err == nil {
		c.mu.Lock()
		if c.recent == nil || len(c.recent) >= 4096 {
			c.recent = map[string]recentListing{}
		}

		c.recent[dir] = recentListing{names: names, at: time.Now()}
		c.mu.Unlock()
	}

	return names, false, err
}

// keep records a directory's listing, and is called only where the segment
// being looked for was found in it.
//
// Caching a listing that did not hold what was wanted is what makes a cache
// wrong rather than merely stale: those are exactly the directories a newly
// written component appears in, and remembering "not here" would go on denying
// it for the resolver's life — including for sibling components that were never
// looked up. A directory a lookup walked *through* is a directory that exists
// and is being used; one it failed in gets listed again next time. The win is
// in the shared prefixes, and those are all hits.
func (c *DirCache) keep(dir string, names []string) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.names == nil {
		c.names = make(map[string][]string)
	}

	c.names[dir] = names
}

func readDirNames(dir string) ([]string, error) {
	entries, err := DefaultFS.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}

	return names, nil
}

// resolveSegmentsCached walks from baseDir through each path segment (the last segment being the
// filename, already suffixed with .cfc), returning the real, case-correct absolute path if
// found, or "" if any segment is missing.
//
// Each segment is resolved via a directory listing rather than a direct Stat: on a
// case-insensitive filesystem (APFS, NTFS default), Stat succeeds for any case variant of an
// existing name, which would silently accept whatever case the dot-path happened to use
// instead of the real on-disk case. Listing the directory and matching entries — exact case
// first, then case-insensitively — is the only way to recover the true name on every platform.
func resolveSegmentsCached(baseDir string, segments []string, cache *DirCache) string {
	dir := baseDir

	for _, seg := range segments {
		names, cached, err := cache.namesIn(dir)
		if err != nil {
			return ""
		}

		found := ""

		for _, name := range names {
			if name == seg {
				found = name

				break
			}
		}

		if found == "" {
			for _, name := range names {
				if strings.EqualFold(name, seg) {
					found = name

					break
				}
			}
		}

		if found == "" {
			return ""
		}

		if !cached {
			cache.keep(dir, names)
		}

		dir = filepath.Join(dir, found)
	}

	return dir
}

// ResolveFrom resolves a path written in a config file: an absolute path is
// kept as it is, and a relative one is taken from baseDir. Joining an absolute
// path onto baseDir unconditionally puts it under baseDir, where it names
// nothing.
func ResolveFrom(baseDir, p string) string {
	if filepath.IsAbs(p) {
		return p
	}

	return filepath.Join(baseDir, p)
}

// ResolveMappings resolves relative paths in a map to absolute using baseDir.
func ResolveMappings(raw map[string]string, baseDir string) map[string]string {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]string, len(raw))

	for k, v := range raw {
		out[k] = ResolveFrom(baseDir, v)
	}

	return out
}

// matchGlobSuffix reports whether the trailing segments of path match the
// segment pattern suffix — the part of a glob that follows "**".
//
// Both callers used to match suffix against filepath.Base(path) alone, which
// works for the common "**/*.cfc" and silently matches nothing the moment the
// suffix has more than one segment: "src/**/models/*.cfc" compares
// "models/*.cfc" against "User.cfc" and never fires, so a workspaceIndexGlobs
// entry written that way indexes no files and reports no error.
//
// filepath.Match is applied per segment, so a "*" still does not cross a
// separator; it is only the "**" that spans depth.
func matchGlobSuffix(suffix, path string) bool {
	want := strings.Split(suffix, string(filepath.Separator))
	got := strings.Split(path, string(filepath.Separator))

	if len(got) < len(want) {
		return false
	}

	got = got[len(got)-len(want):]

	for i, w := range want {
		if matched, _ := filepath.Match(w, got[i]); !matched {
			return false
		}
	}

	return true
}

// ExpandGlob expands a glob pattern, supporting ** for recursive matching.
func ExpandGlob(pattern string) []string {
	if !strings.Contains(pattern, "**") {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil
		}

		return matches
	}

	before, after, _ := strings.Cut(pattern, "**")
	base := filepath.Clean(before)
	suffix := after
	suffix = strings.TrimPrefix(suffix, string(filepath.Separator))

	var out []string

	_ = DefaultFS.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if suffix == "" {
			out = append(out, path)

			return nil
		}

		if matchGlobSuffix(suffix, path) {
			out = append(out, path)
		}

		return nil
	})

	return out
}

// IsCFMLFile returns true if the path/URI refers to a CFML file (.cfc, .cfm, .cfml, .cfs).
func IsCFMLFile(path string) bool {
	if len(path) < 4 {
		return false
	}

	end := path[len(path)-1]
	switch end | 0x20 {
	case 'c': // .cfc
		return len(path) > 4 && path[len(path)-4] == '.' &&
			(path[len(path)-3]|0x20) == 'c' && (path[len(path)-2]|0x20) == 'f'
	case 'm': // .cfm
		return path[len(path)-4] == '.' && (path[len(path)-3]|0x20) == 'c' && (path[len(path)-2]|0x20) == 'f'
	case 'l': // .cfml
		return len(path) > 5 && path[len(path)-5] == '.' && (path[len(path)-4]|0x20) == 'c' &&
			(path[len(path)-3]|0x20) == 'f' && (path[len(path)-2]|0x20) == 'm'
	case 's': // .cfs
		return path[len(path)-4] == '.' && (path[len(path)-3]|0x20) == 'c' && (path[len(path)-2]|0x20) == 'f'
	}

	return false
}

// SkipScanDir reports whether a directory is one no scan of a workspace should
// descend into.
//
// Every dot-directory is skipped, not a hand-kept list of them. The list was
// ".git", ".svn", "node_modules", "target" and "vendor", and on a real workspace
// it let in ".claude/worktrees" — git worktrees living inside the project, each a
// complete second copy of the codebase. A whole-project map built over that counts
// every component twice, reports every function as having a mysterious duplicate,
// and inflates the unreferenced list with thousands of entries from a checkout
// nobody is editing. The same applies to any other tool's dot-directory, which is
// why this is a rule rather than another name on a list.
//
// The root itself is exempt (callers test path != root), so pointing a scan at a
// dot-directory, or at a vendored package, still works.
func SkipScanDir(name string) bool {
	if strings.HasPrefix(name, ".") && name != "." && name != ".." {
		return true
	}

	switch name {
	case "node_modules", "target", "vendor", "bower_components":
		return true
	default:
		return false
	}
}

// CollectCFML walks the roots for CFML files, skipping the directories
// [SkipScanDir] names. The unresolved scan, the code map, the editor's scans and
// find-references each walked on their own, with two copies of the rule and a
// third walker that skipped nothing.
//
// An unreadable entry is skipped, not fatal: aborting a walk on one permission
// error would silently scan a fraction of the workspace and report it as the
// whole thing.
func CollectCFML(fsys vfs.FS, roots []string) []string {
	return Collect(fsys, roots, IsCFMLFile)
}

// CollectCFC is [CollectCFML] for components alone, which is what the editor's
// index holds.
func CollectCFC(fsys vfs.FS, roots []string) []string {
	return Collect(fsys, roots, IsCFCFile)
}

// Collect is the walk behind both: the files under roots that keep accepts,
// outside the directories [SkipScanDir] names.
func Collect(fsys vfs.FS, roots []string, keep func(path string) bool) []string {
	var files []string

	for _, root := range roots {
		_ = fsys.Walk(root, func(path string, info os.FileInfo, err error) error {
			// An entry that could not be read, or a root that cannot be stat'ed
			// (reported with a nil FileInfo), is neither descended into nor
			// collected.
			readable := err == nil && info != nil

			switch {
			case readable && info.IsDir() && path != root && SkipScanDir(info.Name()):
				return filepath.SkipDir
			case readable && !info.IsDir() && keep(path):
				files = append(files, path)
			}

			return nil
		})
	}

	return files
}

// IsCFCFile returns true if the path/URI refers to a CFC file.
func IsCFCFile(path string) bool {
	return len(path) > 4 && path[len(path)-4] == '.' &&
		(path[len(path)-3]|0x20) == 'c' && (path[len(path)-2]|0x20) == 'f' && (path[len(path)-1]|0x20) == 'c'
}

// IsBinary returns true if data appears to be binary (contains null bytes in the first 512 bytes).
func IsBinary[T string | []byte](data T) bool {
	n := min(len(data), 512)

	for i := range n {
		if data[i] == 0 {
			return true
		}
	}

	return false
}

// MatchesGlob checks whether a file path matches any of the given glob patterns.
func MatchesGlob(filePath string, globs []string) bool {
	for _, g := range globs {
		if !strings.Contains(g, "**") {
			if matched, _ := filepath.Match(g, filePath); matched {
				return true
			}

			if strings.HasPrefix(filePath, g+"/") || filePath == g {
				return true
			}

			continue
		}

		before, after, _ := strings.Cut(g, "**")
		base := filepath.Clean(before)
		suffix := after
		suffix = strings.TrimPrefix(suffix, string(filepath.Separator))

		if !strings.HasPrefix(filePath, base+"/") && filePath != base {
			continue
		}

		if suffix == "" {
			return true
		}

		if matchGlobSuffix(suffix, filePath) {
			return true
		}
	}

	return false
}

// SamePath reports whether two filesystem paths refer to the same file,
// tolerating case differences (common on APFS/NTFS) and cosmetic
// differences like trailing slashes or "./" segments.
func SamePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}

	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// CfcNameFromURI extracts the CFC filename without extension from a URI.
func CfcNameFromURI(fileURI string) string {
	path := strings.TrimPrefix(fileURI, "file://")
	if decoded, err := url.PathUnescape(path); err == nil {
		path = decoded
	}

	base := filepath.Base(path)

	return strings.TrimSuffix(base, filepath.Ext(base))
}
