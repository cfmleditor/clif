// Component dot-path to file resolution: mappings, workspace folders, slugs and the engine components.

package resolve

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/frameworkapi"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

// describeResolver names the resolver at idx for trace output, so a wrong component can be
// traced back to the exact componentResolvers entry that produced it rather than just to
// "a componentResolver".
func (r *Resolver) describeResolver(idx int) string {
	if idx < 0 || idx >= len(r.Resolvers) {
		return "unknown resolver"
	}

	return r.Resolvers[idx].Describe()
}

// ComponentPath resolves a component dot-path to an absolute .cfc file path
// using the standard fallback chain: baseDir → Application.cfc root → workspace folders.
// If component contains pipe characters, each alternative is tried left-to-right.
func (r *Resolver) ComponentPath(component, baseDir string) string {
	// Apply expression mappings (replace runtime expressions with static values).
	// A key may list multiple pipe-delimited alternatives that all map to the same value.
	for _, key := range r.expressionKeys() {
		value := r.ExpressionMappings[key]

		for expr := range strings.SplitSeq(key, "|") {
			if expr != "" && strings.Contains(component, expr) {
				component = strings.ReplaceAll(component, expr, value)
			}
		}
	}

	component = strings.TrimPrefix(component, parser.MockPrefix)

	key := component + "\t" + baseDir

	r.mu.RLock()

	if r.resolveCache != nil {
		if p, ok := r.resolveCache[key]; ok {
			r.mu.RUnlock()

			return p
		}
	}

	r.mu.RUnlock()

	var result string

	if strings.Contains(component, "|") {
		for alt := range strings.SplitSeq(component, "|") {
			if p := r.componentPathUncached(alt, baseDir); p != "" {
				result = p

				break
			}
		}
	} else {
		result = r.componentPathUncached(component, baseDir)
	}

	r.mu.Lock()
	if r.resolveCache == nil {
		r.resolveCache = make(map[string]string)
	}

	r.resolveCache[key] = result
	r.mu.Unlock()

	return result
}

// expressionKeys returns ExpressionMappings' keys in the order they apply
// (parser.ExpressionMappingOrder), worked out once: the resolver is dropped
// wherever its configuration changes.
func (r *Resolver) expressionKeys() []string {
	if len(r.ExpressionMappings) == 0 {
		return nil
	}

	r.mu.RLock()
	keys := r.exprKeys
	r.mu.RUnlock()

	if keys != nil {
		return keys
	}

	keys = parser.ExpressionMappingOrder(r.ExpressionMappings)

	r.mu.Lock()
	r.exprKeys = keys
	r.mu.Unlock()

	return keys
}

// dirs returns the resolver's directory-listing cache, creating it on first
// use. Its lifetime is the resolver's: the server drops the whole Resolver
// wherever it decides a path answer may have gone stale, which is what the
// cache needs and what resolveCache beside it already relies on.
func (r *Resolver) dirs() *cfpath.DirCache {
	// A listing is the filesystem's, not a view's, so views share their
	// owner's rather than each listing the same directories again.
	r = r.owner()

	r.mu.RLock()
	c := r.dirCache
	r.mu.RUnlock()

	if c != nil {
		return c
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.dirCache == nil {
		r.dirCache = cfpath.NewDirCache()
		if r.SnapshotScan {
			r.dirCache = cfpath.NewSnapshotDirCache()
		}
	}

	return r.dirCache
}

// fs is the resolver's filesystem, with the framework stubs mounted: a
// preset's, and the ones a fully namespaced path reaches without one
// (frameworkapi.Namespaced).
// readText is the text of the file at path, shared with the batch content
// cache rather than copied out of it (vfs.ReadString).
func (r *Resolver) readText(path string) (string, error) {
	return vfs.ReadString(r.fs(), path)
}

func (r *Resolver) fs() vfs.FS {
	r.mu.RLock()
	f := r.stubFS
	r.mu.RUnlock()

	if f != nil {
		return f
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stubFS == nil {
		r.stubFS = frameworkapi.Wrap(r.FS)
	}

	return r.stubFS
}

func (r *Resolver) componentPathUncached(component, baseDir string) string {
	// Explicit relative component paths retain declaring-directory identity.
	// A missing parent must not borrow another application's same-named file.
	if cfpath.IsRelativeComponentPath(component) {
		return cfpath.ResolvePathCached(component, baseDir, nil, r.dirs())
	}
	// A component already named by its file: what a function returning
	// `this` returns.
	if filepath.IsAbs(component) && strings.HasSuffix(strings.ToLower(component), ".cfc") {
		if info, err := r.fs().Stat(component); err == nil && !info.IsDir() {
			return component
		}
	}

	// An ORM entity named by entityNew()/entityLoad(): the entity of that
	// name, else the name read as a path, as it was before it was marked.
	if name, ok := strings.CutPrefix(component, parser.EntityPrefix); ok {
		if p := r.nearestEntity(name, baseDir); p != "" {
			return p
		}

		return r.ComponentPath(name, baseDir)
	}

	// A WireBox id with its module: `Name@module`.
	if strings.Contains(component, "@") {
		p, _ := r.wireboxID(component, baseDir)

		return p
	}

	mappings := r.effectiveMappings(baseDir)
	dirs := r.dirs()

	if p := cfpath.ResolvePathCached(component, baseDir, mappings, dirs); p != "" {
		return p
	}

	if appDir := r.FindApplicationRoot(baseDir); appDir != "" {
		if p := cfpath.ResolvePathCached(component, appDir, mappings, dirs); p != "" {
			return p
		}
	}

	for _, root := range r.WorkspaceFolders {
		if p := cfpath.ResolvePathCached(component, root, mappings, dirs); p != "" {
			return p
		}
	}

	if p := r.inFolderNamed(component, dirs); p != "" {
		return p
	}

	// A module's cfmapping is an application mapping ColdBox registers when
	// it loads the module.
	if p := r.inModuleMapping(component); p != "" {
		return p
	}

	if root, rest := r.slugRoot(component, baseDir); root != "" {
		if p := cfpath.ResolvePathCached(rest, root, nil, dirs); p != "" {
			return p
		}
	}

	if root, rest := r.lucliModuleRoot(component, baseDir); root != "" {
		if p := cfpath.ResolvePathCached(rest, root, nil, dirs); p != "" {
			return p
		}
	}

	// A framework component nothing on disk answers: its API, from the stubs.
	// Last of the dot-path lookups, so a workspace with the framework checked
	// out resolves to the real file exactly as it did.
	if p := r.Stubs.Path(component); p != "" {
		return p
	}

	// A path in a framework's own namespace names that framework whether or
	// not a preset does: `extends="testbox.system.compat.framework.TestCase"`
	// is TestBox, and Lucee's test suite extends it in 1,885 files with no
	// preset named.
	if p := frameworkapi.Namespaced(component); p != "" {
		return p
	}

	// An id a WireBox binder maps, `map( "Mailer" ).to( … )`: asked only
	// once no path answered, since reading the binders lists the index.
	if p, ok := r.wireboxID(component, baseDir); ok && p != "" {
		return p
	}

	return r.lastResortPath(component, baseDir)
}

// lastResortPath is componentPathUncached once no path has answered: an
// engine component, a file of the name nearest baseDir, an ORM entity, a
// bean, or a class a framework's WireBox maps by file name.
func (r *Resolver) lastResortPath(component, baseDir string) string {
	// A component the engine ships is the engine's, not whatever file of that
	// name the workspace happens to hold, so the file-name search below never
	// answers for one.
	if qualified, ok := engineComponent(component); ok {
		return r.engineSource(qualified)
	}

	// For bare names (no dots/slashes), search the index by filename as a last resort.
	// This handles `extends="BaseAssertionsTest"` where the file isn't in the same
	// directory or workspace root but is somewhere in the indexed workspace.
	if !strings.Contains(component, ".") && !strings.Contains(component, "/") && r.Index != nil {
		candidates := r.Index.FindFilesByBasename(component)
		// Lazy indexing of a namespaced stub must not make its bare name
		// visible to the workspace. Otherwise a parallel scan's answer depends
		// on whether another file has already used that framework component.
		// Preset ids still resolve through IDPackages below.
		candidates = slices.DeleteFunc(candidates, frameworkapi.IsStub)

		if len(candidates) == 1 {
			return candidates[0]
		}

		if len(candidates) > 1 {
			// Multiple matches — pick the one with the shortest relative path
			// from baseDir, and the lowest path among equals, as
			// Index.LookupPreferred does for a function. Several copies of a
			// component in sibling directories — ContentBox keeps one
			// RailoDBInfo.cfc per patch — are all equally near a sibling that
			// has none, and taking whichever came first made the answer, and
			// every code-map edge through it, change from run to run.
			best := ""
			bestDist := -1

			for _, c := range candidates {
				rel, err := filepath.Rel(baseDir, c)
				if err != nil {
					continue
				}

				dist := strings.Count(rel, string(filepath.Separator))

				if bestDist < 0 || dist < bestDist || dist == bestDist && c < best {
					best = c
					bestDist = dist
				}
			}

			if best != "" {
				return best
			}
		}
	}

	// An ORM entity by its entityname: entityNew( "cbAuthor" ) is the
	// Author.cfc that declares that name.
	if !strings.ContainsAny(component, "./") && r.Index != nil {
		named := r.Index.NamedEntities(component)
		if p := r.pickEntity(named, named, baseDir); p != "" {
			return p
		}
	}

	// A bean by name: DI/1's getBean( "userService" ) and the beanPaths a
	// configuration names, aliases included (cfpath.BuildBeanMap).
	if !strings.ContainsAny(component, "./") && r.Index != nil {
		if p := r.Index.LookupBean(component); p != "" && filepath.IsAbs(p) {
			return p
		}
	}

	// A bare id a framework's WireBox maps by file name: a CommandBox
	// module's `inject="FileSystem"` is commandbox.system.util.FileSystem.
	// After the workspace's own files, so a project's FileSystem.cfc wins.
	if !strings.ContainsAny(component, "./") {
		for _, pkg := range r.Stubs.IDPackages() {
			if p := r.ComponentPath(pkg+"."+component, baseDir); p != "" {
				return p
			}
		}
	}

	return ""
}

// slugRoot finds the package a dot-path names itself by: the nearest
// directory at or above baseDir whose box.json slug is the path's first
// segment. It returns that directory and the rest of the path.
//
// A CommandBox package is installed under its slug, so its own code spells
// its components that way: coldbox-platform's handlers extend
// `coldbox.system.EventHandler`, which is its own system/EventHandler.cfc. A
// checkout of the package has no mapping for it, and every such path was a
// component that did not exist — over ten thousand calls between
// coldbox-platform and TestBox. A configured mapping still comes first.
func (r *Resolver) slugRoot(component, baseDir string) (root, rest string) {
	first, rest, ok := strings.Cut(component, ".")
	if !ok || first == "" || rest == "" || r.FS == nil {
		return "", ""
	}

	for dir := baseDir; ; {
		if slug := r.boxSlug(dir); slug != "" && strings.EqualFold(slug, first) {
			return dir, rest
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ""
		}

		dir = parent
	}
}

// lucliModuleRoot is the directory a LuCLI module is installed from, when
// component is `modules.<name>.rest` and a directory above baseDir holds the
// module.json naming it: LuCLI installs a module at modules/<name>, and the
// module addresses its own components that way. cfwheels' CLI, in cli/lucli,
// writes `new modules.wheels.services.deploy.config.ConfigLoader()`.
func (r *Resolver) lucliModuleRoot(component, baseDir string) (root, rest string) {
	first, rest, ok := strings.Cut(component, ".")
	if !ok || !strings.EqualFold(first, "modules") || r.FS == nil {
		return "", ""
	}

	name, rest, ok := strings.Cut(rest, ".")
	if !ok || name == "" || rest == "" {
		return "", ""
	}

	for dir := baseDir; ; {
		if module := r.lucliModuleName(dir); module != "" && strings.EqualFold(module, name) {
			return dir, rest
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ""
		}

		dir = parent
	}
}

// lucliModuleName is the name dir's module.json gives a LuCLI module, "" when
// it has none or names no main component. It shares boxSlug's cache.
func (r *Resolver) lucliModuleName(dir string) string {
	key := "module.json\x00" + dir

	r.mu.RLock()
	name, ok := r.slugCache[key]
	r.mu.RUnlock()

	if ok {
		return name
	}

	if data, err := r.fs().ReadFile(filepath.Join(dir, "module.json")); err == nil {
		var module struct {
			Name string `json:"name"`
			Main string `json:"main"`
		}

		if json.Unmarshal(data, &module) == nil && strings.TrimSpace(module.Main) != "" {
			name = strings.TrimSpace(module.Name)
		}
	}

	r.mu.Lock()
	if r.slugCache == nil {
		r.slugCache = make(map[string]string)
	}

	r.slugCache[key] = name
	r.mu.Unlock()

	return name
}

// boxSlug is the slug of dir's box.json, "" when it has none; remembered per
// directory, since every unresolved path walks the same ancestors.
func (r *Resolver) boxSlug(dir string) string {
	r.mu.RLock()
	slug, ok := r.slugCache[dir]
	r.mu.RUnlock()

	if ok {
		return slug
	}

	if data, err := r.fs().ReadFile(filepath.Join(dir, "box.json")); err == nil {
		var box struct {
			Slug string `json:"slug"`
		}

		if json.Unmarshal(data, &box) == nil {
			slug = strings.TrimSpace(box.Slug)
		}
	}

	r.mu.Lock()
	if r.slugCache == nil {
		r.slugCache = make(map[string]string)
	}

	r.slugCache[dir] = slug
	r.mu.Unlock()

	return slug
}

// engineImports are the components an engine imports implicitly, so that
// `new Query()` and `new http()` need no path: Lucee's org.lucee.cfml.* and
// Adobe ColdFusion's com.adobe.coldfusion.*, which script-tag components
// share names across.
var engineImports = map[string]bool{
	"dbinfo":     true,
	"feed":       true,
	"ftp":        true,
	"http":       true,
	"ldap":       true,
	"mail":       true,
	"pop":        true,
	"query":      true,
	"storedproc": true,
}

const (
	luceeComponents = "org.lucee.cfml."
	adobeComponents = "com.adobe.coldfusion."
	// adobeAdminAPI is Adobe ColdFusion's administrator API, which the engine
	// ships in its CFIDE directory: Mura's database setup creates datasources
	// through cfide.adminapi.datasource on Adobe ColdFusion.
	adobeAdminAPI = "cfide.adminapi."
)

// engineComponent reports whether component names one the engine provides,
// and how Lucee's source tree spells it: bare `Query` is
// org.lucee.cfml.Query. An Adobe component has no source to find, so its
// spelling is "".
//
// It is asked only once nothing configured resolved the path, so a project's
// own Query.cfc beside the caller, or a mapping for org.lucee.cfml, still
// wins, as it does at runtime.
func engineComponent(component string) (qualified string, ok bool) {
	lower := strings.ToLower(component)

	switch {
	case engineImports[lower]:
		return luceeComponents + component, true
	case strings.HasPrefix(lower, luceeComponents) && len(lower) > len(luceeComponents):
		return component, true
	case strings.HasPrefix(lower, adobeComponents), strings.HasPrefix(lower, adobeAdminAPI):
		return "", true
	}

	return "", false
}

// engineSource finds an engine component's own source in the workspace: an
// indexed file whose path ends in the component's path, as a Lucee checkout
// keeps org/lucee/cfml/Query.cfc and the org/lucee/cfml/test/LuceeTestCase
// its tests extend. Calls on it are then checked against the real methods.
// Without the source the component is still the engine's, and canResolveCall
// accepts calls on it as dynamic (engineComponent) rather than reporting a
// component that does not exist.
func (r *Resolver) engineSource(qualified string) string {
	if qualified == "" || r.Index == nil {
		return ""
	}

	candidates := r.Index.FindFilesByBasename(strings.ReplaceAll(qualified, ".", "/"))
	if len(candidates) == 0 {
		return ""
	}

	return candidates[0]
}

// inFolderNamed resolves a dot-path whose first segment names a workspace
// folder, inside that folder: with ../tassweb among the workspace folders,
// tassweb.packages.tass.core.kernel2 is <tassweb>/packages/tass/core/kernel2.cfc.
//
// That is what a mapping from a folder's name to the folder says, and it was
// the only kind of mapping tassweb's config held: all three of its mappings,
// tassweb, tassreporting and tassdoc, named a workspace folder by its own name.
// Without them 250,000 of its calls stopped resolving. A workspace folder now
// implies that mapping, so a project needs one only where the name differs
// from the folder.
//
// It comes after every configured lookup and before slugRoot, so an explicit mapping,
// an Application.cfc mapping and a path relative to the file, the application
// or a workspace folder all come first, and nothing that resolved before
// resolves differently now. The folders are the resolver's, which are the
// config's workspacePaths, or the editor's folders when the config names none.
func (r *Resolver) inFolderNamed(component string, dirs *cfpath.DirCache) string {
	dotted := strings.ReplaceAll(strings.TrimPrefix(component, "/"), "/", ".")

	seg, rest, ok := strings.Cut(dotted, ".")
	if !ok || seg == "" || rest == "" {
		return ""
	}

	for _, root := range r.WorkspaceFolders {
		if !strings.EqualFold(filepath.Base(root), seg) {
			continue
		}

		if p := cfpath.ResolvePathCached(rest, root, nil, dirs); p != "" {
			return p
		}
	}

	return ""
}
