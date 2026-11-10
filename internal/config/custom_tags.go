package config

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
)

// CustomTagPath is one customTagPaths entry: a directory the engine searches
// for custom tags, as a Lucee or ColdFusion administrator's custom tag paths
// are. Source can only name the paths an Application.cfc sets
// (this.customTagPaths, which is read without configuration); the server's
// own are written here.
//
//	"customTagPaths": [
//	  "shared/customtags",
//	  {"directory": "lib/tags", "recursive": true},
//	  {"directory": "lib/bridge", "prefix": "cfa"}
//	]
//
// A bare string is a directory. Without a prefix a directory answers
// <cf_name>, and <cfmodule name="a.b"> as a/b.cfm under it. With one it is a
// tag library: <cfabridge> runs bridge.cfm in it, for prefix "cfa". A Lucee
// library directory, whose abridge.cfm is <cfabridge>, is prefix "cf".
type CustomTagPath struct {
	Directory string `json:"directory"`
	// Prefix is the start of the tag names the directory answers, "cf"
	// included; empty for an ordinary custom tag path. Compared without case.
	Prefix string `json:"prefix,omitempty"`
	// Recursive searches the directory's subdirectories too, as ColdFusion
	// always does and Lucee does with "search subdirectories" switched on.
	Recursive bool `json:"recursive,omitempty"`
}

// UnmarshalJSON accepts a bare directory for the common case.
func (p *CustomTagPath) UnmarshalJSON(b []byte) error {
	var dir string
	if json.Unmarshal(b, &dir) == nil {
		*p = CustomTagPath{Directory: dir}

		return nil
	}

	type plain CustomTagPath

	var v plain
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	*p = CustomTagPath(v)

	return nil
}

// ResolveCustomTagPaths makes each entry's directory absolute against
// baseDir, lowercases its prefix, and drops an entry naming no directory.
func ResolveCustomTagPaths(raw []CustomTagPath, baseDir string) []CustomTagPath {
	if len(raw) == 0 {
		return nil
	}

	out := make([]CustomTagPath, 0, len(raw))

	for _, p := range raw {
		if strings.TrimSpace(p.Directory) == "" {
			continue
		}

		dir := filepath.FromSlash(p.Directory)
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(baseDir, dir)
		}

		out = append(out, CustomTagPath{Directory: filepath.Clean(dir), Prefix: strings.ToLower(strings.TrimSpace(p.Prefix)), Recursive: p.Recursive})
	}

	return out
}

// CustomTagPrefixes are the tag library prefixes paths declare, lowercased,
// each once.
func CustomTagPrefixes(paths []CustomTagPath) []string {
	var out []string

	for _, p := range paths {
		if p.Prefix != "" && !slices.Contains(out, p.Prefix) {
			out = append(out, p.Prefix)
		}
	}

	return out
}
