package main

import (
	"os"
	"path/filepath"

	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/daemon"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
)

// batchSettings reduces a config to what a resolver is built from, for every
// command that resolves calls in bulk: unresolved, explain and graph.
//
// With no config the project is read for the framework it is (assumed), and
// that framework's resolvers, implicit bases, helper scope and bundled API
// apply as though a config had named it. graph did not do that, so on a project
// with no .clif.json it resolved a fraction of what unresolved did.
//
// The second result names the assumed frameworks, for the command to report.
func batchSettings(fsys vfs.FS, cfg *daemon.Config, folders []string) (resolve.Settings, []string) {
	if cfg == nil {
		assumed := config.Assumed(folders, fsys.ReadFile, fsys.ReadDir)

		return resolve.SettingsFromConfig(assumed, folders), assumed.Frameworks
	}

	return resolve.SettingsFromConfig(cfg.Resolved(), folders), nil
}

// workspaceResolver is the resolver the bulk commands that take paths (deps,
// refs and the MCP tools over them) resolve with, its index filled from files:
// the config found above the first path, or the framework the project is
// assumed to be, through batchSettings, and the components indexed as a scan
// indexes them. refs read a config of its own, one directory up and no further,
// and ran with its resolvers alone.
func workspaceResolver(args, files []string) *resolve.Resolver {
	fsys := vfs.OS{}

	searchDir := "."

	if len(args) > 0 {
		if info, err := os.Stat(args[0]); err == nil && info.IsDir() {
			searchDir = args[0]
		} else {
			searchDir = filepath.Dir(args[0])
		}
	}

	cfg, _ := daemon.FindConfig(searchDir)
	settings, _ := batchSettings(fsys, cfg, cliWorkspaceFolders(fsys, cfg, args))

	resolver := resolve.NewBatchResolver(fsys, index.New(), &settings)
	resolver.LoadBeans(settings.BeanPaths)
	resolver.IndexFiles(files, nil)

	return resolver
}
