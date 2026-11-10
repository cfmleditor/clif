package path

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// Server mappings are application defaults. Application declarations and
// explicit editor configuration override them; no environment values are read.
// The caller may change the map it is given.
func loadServerMappings(appDir string) map[string]string {
	return maps.Clone(serverFactsFor(appDir).mappings)
}

// serverFacts is what the server's CFConfig file (serverConfig) says that a
// lookup reads: its mappings, its custom tag paths, and whether they are
// searched recursively. The file is decoded once per application directory
// for all of them (serverFactsFor).
type serverFacts struct {
	mappings map[string]string
	tagRoots []TagRoot
	deep     bool
}

// serverFactsCache is serverFactsFor's answers by application directory,
// dropped with the application mappings (InvalidateAppMappingsCache).
var serverFactsCache sync.Map

// serverFactsFor is appDir's serverFacts, decoded once.
func serverFactsFor(appDir string) *serverFacts {
	if held, ok := serverFactsCache.Load(appDir); ok {
		f, _ := held.(*serverFacts)

		return f
	}

	held, _ := serverFactsCache.LoadOrStore(appDir, decodeServerFacts(appDir))
	f, _ := held.(*serverFacts)

	return f
}

// customTagEntry is one custom tag directory in a CFConfig file.
type customTagEntry struct {
	Physical string `json:"physical"`
	Primary  string `json:"primary"`
}

func decodeServerFacts(appDir string) *serverFacts {
	f := &serverFacts{mappings: map[string]string{}}

	config, data, ok := serverConfig(appDir)
	if !ok {
		return f
	}

	var source struct {
		Mappings map[string]json.RawMessage `json:"mappings"`
		// Lucee's own .CFConfig.json says customTagMappings; CommandBox's
		// cfconfig format says customTagPaths. Both are read.
		CustomTagMappings []customTagEntry `json:"customTagMappings"`
		CustomTagPaths    []customTagEntry `json:"customTagPaths"`
		DeepSearch        json.RawMessage  `json:"customTagDeepSearch"`
	}
	if json.Unmarshal(data, &source) != nil {
		return f
	}

	for key, raw := range source.Mappings {
		physical := ""
		if json.Unmarshal(raw, &physical) != nil {
			var entry struct {
				Physical string `json:"physical"`
				Primary  string `json:"primary"`
			}
			if json.Unmarshal(raw, &entry) != nil || strings.EqualFold(entry.Primary, "archive") {
				continue
			}

			physical = entry.Physical
		}

		key = strings.ToLower(strings.Trim(key, "/"))

		if !staticConfigPath(physical) {
			continue
		}

		if !filepath.IsAbs(physical) {
			physical = filepath.Join(filepath.Dir(config), filepath.FromSlash(physical))
		}

		f.mappings[key] = filepath.Clean(physical)
	}

	f.deep = jsonTrue(source.DeepSearch)

	for _, m := range slices.Concat(source.CustomTagMappings, source.CustomTagPaths) {
		physical := m.Physical
		if strings.EqualFold(m.Primary, "archive") || !staticConfigPath(physical) || strings.ContainsAny(physical, "{}") {
			continue
		}

		physical = filepath.FromSlash(physical)
		if !filepath.IsAbs(physical) {
			physical = filepath.Join(filepath.Dir(config), physical)
		}

		f.tagRoots = append(f.tagRoots, TagRoot{Dir: filepath.Clean(physical), Recursive: f.deep})
	}

	return f
}

// serverConfig is the CFConfig file that governs appDir, and its content: the
// one a server.json names in cfconfig.file, else a .cfconfig.json, in appDir
// or the nearest folder above it holding either. ok is false when there is
// none, or the one named is computed or cannot be read.
func serverConfig(appDir string) (path string, data []byte, ok bool) {
	for dir, n := filepath.Clean(appDir), 0; n < 32; n++ {
		selected := false

		config := filepath.Join(dir, ".cfconfig.json")
		if data, err := probeRead(filepath.Join(dir, "server.json")); err == nil {
			var server struct {
				CFConfig struct {
					File string `json:"file"`
				} `json:"cfconfig"`
			}
			if json.Unmarshal(data, &server) == nil && server.CFConfig.File != "" {
				if !staticConfigPath(server.CFConfig.File) {
					return "", nil, false
				}

				selected = true

				config = server.CFConfig.File
				if !filepath.IsAbs(config) {
					config = filepath.Join(dir, filepath.FromSlash(config))
				}
			}
		}

		if data, err := probeRead(config); err == nil {
			return config, data, true
		}

		if selected {
			return "", nil, false
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil, false
		}

		dir = parent
	}

	return "", nil, false
}

func staticConfigPath(value string) bool {
	return value != "" && !strings.ContainsAny(value, "#$") && !strings.Contains(value, "://") && !strings.HasPrefix(value, "~")
}
