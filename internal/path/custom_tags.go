package path

import (
	"encoding/json"
	"strings"
)

// TagRoot is a directory custom tags are searched in, and whether its
// subdirectories are searched too.
type TagRoot struct {
	Dir       string
	Recursive bool
}

// appTagPathsCache is LoadAppCustomTagPaths' answers, keyed as
// appMappingsCache is and filled beside it, under appMappingsCacheMu.
var appTagPathsCache = map[string][]string{}

// LoadAppCustomTagPaths is the this.customTagPaths the Application file in
// appDir sets, each directory absolute. It is read in the walk that reads the
// file's mappings (LoadAppMappingsWithInitial), and kept with them, so the
// file is not read again for it.
func LoadAppCustomTagPaths(appDir string, initial map[string]string) []string {
	LoadAppMappingsWithInitial(appDir, initial)

	appMappingsCacheMu.RLock()
	defer appMappingsCacheMu.RUnlock()

	return appTagPathsCache[appCacheKey(appDir, initial)]
}

// ServerCustomTagRoots is the custom tag directories the server's CFConfig
// file (see serverConfig) declares in customTagMappings (Lucee) or
// customTagPaths (CommandBox cfconfig), and whether it sets
// customTagDeepSearch, which has every custom tag path searched recursively,
// an application's too. Lucee writes them as
//
//	"customTagMappings": [{"physical": "{lucee-config}/customtags/", "inspectTemplate": "never"}],
//	"customTagDeepSearch": "true"
//
// A directory written with a placeholder or a variable is left out: it
// names no folder the source can see.
func ServerCustomTagRoots(appDir string) (roots []TagRoot, deep bool) {
	f := serverFactsFor(appDir)

	return f.tagRoots, f.deep
}

// jsonTrue reads a CFConfig switch, which Lucee writes as a boolean or as the
// string "true" or "yes".
func jsonTrue(raw json.RawMessage) bool {
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return b
	}

	var s string
	if json.Unmarshal(raw, &s) == nil {
		s = strings.ToLower(strings.TrimSpace(s))

		return s == "true" || s == "yes"
	}

	return false
}
