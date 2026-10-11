package config

import (
	"io/fs"
	"path/filepath"
	"slices"

	cfpath "github.com/cfmleditor/clif/internal/path"
)

// AssumedFrameworks enables existing preset conventions for one recognized
// source project when no editor config selected the policy. Multiple distinct
// project owners with different deployment contexts acquire no global presets.
// These are documented runtime conventions, not inferred variable declarations.
func AssumedFrameworks(roots []string, read func(string) ([]byte, error), directories ...func(string) ([]fs.DirEntry, error)) []string {
	owners := map[string]bool{}
	presets := map[string]bool{}

	for _, root := range roots {
		for namespace, target := range cfpath.AssumedProjectMappings(root, read) {
			owner := target

			switch namespace {
			case "testbox", "coldbox":
				presets[namespace] = true
			case "framework":
				owner = filepath.Dir(target)
				presets["fw1"] = true
			case "contentbox":
				owner = filepath.Dir(filepath.Dir(target))
				presets["contentbox"], presets["coldbox"] = true, true
			case "mura":
				owner = filepath.Dir(filepath.Dir(target))
				presets["mura"] = true
			default:
				continue
			}

			owners[owner] = true
		}

		if public := cfpath.AssumedWheelsRoot(root, read); public != "" {
			owners[filepath.Dir(public)] = true
			presets["wheels"] = true
		}
	}

	if len(owners) != 1 {
		return nil
	}

	if len(directories) > 0 && directories[0] != nil {
		for owner := range owners {
			for _, name := range assumedSourceTools(owner, read, directories[0]) {
				presets[name] = true
			}
		}
	}

	out := make([]string, 0, len(presets))
	for name := range presets {
		out = append(out, name)
	}

	slices.Sort(out)

	return out
}
