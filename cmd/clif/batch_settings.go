package main

import (
	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/daemon"
	"github.com/cfmleditor/clif/internal/frameworkapi"
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
	s := resolve.Settings{WorkspaceFolders: folders}

	if cfg == nil {
		assumed := config.AssumedFrameworks(folders, fsys.ReadFile, fsys.ReadDir)
		for _, rule := range config.FrameworkResolvers(assumed) {
			s.Resolvers = append(s.Resolvers, rule.Parser())
		}

		s.ImplicitExtends = config.ImplicitExtends(assumed)
		s.HelperScope = config.HelperScope(assumed)
		s.Stubs = frameworkapi.For(assumed)

		return s, assumed
	}

	for _, r := range cfg.ComponentResolvers() {
		s.Resolvers = append(s.Resolvers, r.Parser())
	}

	s.CallerInferenceLimits = cfg.CallerInferenceLimits()
	s.Mappings = cfg.Mappings()
	s.StartupFiles = cfg.StartupFiles()
	s.ExpressionMappings = cfg.ExpressionMappings()
	s.BeanPaths = cfg.BeanPaths()
	s.PropertyResolvers = configPropertyResolvers(cfg)
	s.ImplicitExtends = config.ImplicitExtends(cfg.Frameworks())
	s.HelperScope = config.HelperScope(cfg.Frameworks())
	s.Stubs = frameworkapi.For(cfg.Frameworks())

	return s, nil
}
