package server

import (
	"encoding/json"
	"path/filepath"
	"strings"

	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/resolve"
)

// Mapping configuration includes named CFConfig files, which need not have a
// conventional basename. Cache selections until server/config invalidation.
func (s *Server) isMappingJSON(path string) bool {
	if !strings.EqualFold(filepath.Ext(path), ".json") {
		return false
	}

	base := filepath.Base(path)
	if strings.EqualFold(base, "server.json") || strings.EqualFold(base, ".cfconfig.json") {
		return true
	}

	s.resolverMu.Lock()
	defer s.resolverMu.Unlock()

	if s.mappingJSONFiles == nil {
		s.mappingJSONFiles = map[string]bool{}
	}

	if s.mappingJSONDirs == nil {
		s.mappingJSONDirs = map[string]bool{}
	}

	seen := s.mappingJSONDirs
	for _, root := range append(s.searchRoots(), filepath.Dir(path)) {
		for dir, n := filepath.Clean(root), 0; n < 32; n++ {
			if seen[dir] {
				break
			}

			seen[dir] = true

			data, err := s.FS.ReadFile(filepath.Join(dir, "server.json"))
			if err == nil {
				var config struct {
					CFConfig struct {
						File string `json:"file"`
					} `json:"cfconfig"`
				}
				if json.Unmarshal(data, &config) == nil && config.CFConfig.File != "" && !strings.ContainsAny(config.CFConfig.File, "#$") {
					file := filepath.FromSlash(config.CFConfig.File)
					if !filepath.IsAbs(file) {
						file = filepath.Join(dir, file)
					}

					s.mappingJSONFiles[filepath.Clean(file)] = true
				}
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}

			dir = parent
		}
	}

	return s.mappingJSONFiles[filepath.Clean(path)]
}

// peekResolver is the session's resolver, or nil when none is built; it never
// builds one. For dropping what a resolver holds: with none, it holds nothing.
func (s *Server) peekResolver() *resolve.Resolver {
	s.resolverMu.Lock()
	defer s.resolverMu.Unlock()

	return s.resolver
}

// invalidateNonSourceCaches drops what a changed file that is no source can
// have made stale, short of what discovery found (the callers drop the
// resolver for that): what the mapping walk kept of a marker it probed, which
// is held process-wide and outlives the resolver, and the resolver's path
// caches, which hold answers made from those mappings (a session with a config
// file never runs the discovery that would record the marker) and the slugs
// read from box.json and module.json.
func (s *Server) invalidateNonSourceCaches(file string) {
	probed := cfpath.WasProbed(file)
	if probed {
		cfpath.InvalidateAppMappingsCache()
	}

	if r := s.peekResolver(); r != nil && (probed || isSlugFile(file)) {
		r.InvalidatePaths()
		s.invalidateRoutes()
	}
}

// isSlugFile reports whether file is one the resolver reads a slug from: a
// box.json's slug or a LuCLI module.json's name (boxSlug, lucliModuleName).
// Judged by name, not by what the cache holds: a lookup that has read the file
// and not yet stored its answer leaves nothing to find, and the change would
// be missed. They are rarely written, so dropping the path caches for each is
// cheap.
func isSlugFile(file string) bool {
	base := filepath.Base(file)

	return strings.EqualFold(base, "box.json") || strings.EqualFold(base, "module.json")
}

// discoveryResolver is the resolver that holds what discovery found, or nil
// when the session holds none: no resolver and no resolver set kept empty. A
// saved or changed file that is no source has nothing to invalidate then, and
// asking would build one, and read the project to do it. A resolver set kept
// empty has no resolver of its own, and gets the one it is answered by.
func (s *Server) discoveryResolver() *resolve.Resolver {
	s.resolverMu.Lock()
	r, kept := s.resolver, s.resolversBuilt
	s.resolverMu.Unlock()

	if r == nil && kept {
		return s.getResolver()
	}

	return r
}

func (s *Server) invalidateSourceFile(file string) {
	if !cfpath.IsCFMLFile(file) && !s.isMappingJSON(file) {
		// Not source, but possibly a marker discovery read (a JSON config, a
		// plugin's config.xml): what the session resolves with is kept until
		// something it read changes, and that includes what it found missing.
		if r := s.discoveryResolver(); r != nil && r.DiscoveryReadsFile(file) {
			s.invalidateResolver()
			cfpath.InvalidateAppMappingsCache()
		} else {
			s.invalidateNonSourceCaches(file)
		}

		return
	}

	r := s.getResolver()
	if isApplicationFile(file) || s.isMappingJSON(file) || r.DiscoveryAffected(file) {
		s.invalidateResolver()
	} else {
		r.InvalidatePaths()
		s.invalidateRoutes()
	}

	if cfpath.IsCFMLFile(file) || s.isMappingJSON(file) {
		cfpath.InvalidateAppMappingsCache()
	}
}
