// Package daemon manages multi-session daemon mode and configuration.
package daemon

import (
	"encoding/json"
	"path/filepath"

	"github.com/cfmleditor/clif/internal/vfs"

	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/server"

	"github.com/cfmleditor/clif/internal/route"
)

// configJSON is the on-disk shape of .clif.json.
type configJSON = config.JSON

// Config represents a .clif.json file.
type Config struct {
	Path string // absolute path to the config file itself
	Name string // project name used to derive the daemon socket
}

// FindConfig looks for .clif.json starting from dir and walking up to the
// filesystem root. It returns nil when there is none: no config file means no
// project to run a daemon for, and the caller runs a standalone session.
//
// It used to return a Config with an empty Path and the base name of dir as its
// Name, which made "no config found" indistinguishable from "config found" at
// every call site — all of which already test for nil. Two consequences, both
// bad. The standalone branch in runServer became unreachable, so every session
// started a daemon whatever it found. And the socket path is a hash of Name, so
// with no config the daemon was keyed on the *base name of the working
// directory*: two unrelated projects in folders both called "app" shared one
// daemon and one index, and an editor that starts the server with no working
// directory of its own — the IntelliJ plugin does — gave every project on the
// machine the same name, so all of them did. A shared index means one project's
// symbols answering another's go-to-definition and workspace-symbol queries.
func FindConfig(dir string) (*Config, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	d := abs

	for {
		// .clif.json first, then the legacy .cfmleditor.json (config.FileNames).
		for _, p := range config.CandidatesIn(d) {
			data, err := vfs.ReadRegular(p)
			if err != nil {
				continue
			}

			var raw configJSON
			if json.Unmarshal(data, &raw) == nil {
				if raw.WorkspaceName == "" {
					raw.WorkspaceName = filepath.Dir(p)
				}

				return &Config{Path: p, Name: raw.WorkspaceName}, nil
			}
		}

		parent := filepath.Dir(d)
		if parent == d {
			break
		}

		d = parent
	}

	return nil, nil
}

func (c *Config) raw() *configJSON {
	data, err := vfs.ReadRegular(c.Path)
	if err != nil {
		return nil
	}

	var raw configJSON
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}

	return &raw
}

// Resolved is the configuration the file at c.Path describes, read once.
//
// It is [config.ResolveFile] over the file as it is now, so the command line
// reads a .clif.json exactly as the standalone editor does. The accessors below
// each re-derived their setting from the raw file, a second reading of one
// schema, and the two had drifted apart on the folder-name mapping and on an
// absent formatting block. A file that cannot be read resolves like an empty
// one (and gets no implied mapping, having no folder to name).
func (c *Config) Resolved() *config.Resolved {
	raw := c.raw()
	if raw == nil {
		return config.Resolve(&config.JSON{}, filepath.Dir(c.Path))
	}

	return config.ResolveFile(raw, c.Path)
}

// WorkspaceFolders returns the resolved absolute paths of the project folders.
func (c *Config) WorkspaceFolders() []string { return c.Resolved().WorkspaceFolders }

// IndexGlobs returns absolute glob patterns for workspace indexing, the
// config's workspaceIndexGlobs resolved against its workspace folders.
func (c *Config) IndexGlobs() []string { return c.Resolved().IndexGlobs }

// Debug returns whether debug logging is enabled in config.
func (c *Config) Debug() bool {
	raw := c.raw()

	return raw != nil && raw.Debug
}

// The accessors below are the resolved configuration, one setting at a time, for
// the callers that want one. They re-read the file on each call, as they always
// have; a caller that wants several calls Resolved once.

// Mappings returns component path mappings with values resolved to absolute
// paths, plus the one the folder holding the config implies.
func (c *Config) Mappings() map[string]string { return c.Resolved().Mappings }

// StartupFiles returns the configured startup templates as absolute paths.
func (c *Config) StartupFiles() []string { return c.Resolved().StartupFiles }

// ExpressionMappings returns expression-to-value substitutions from config.
func (c *Config) ExpressionMappings() map[string]string { return c.Resolved().ExpressionMappings }

// ServicePropertyResolvers returns the "@serviceproperty" annotation kind →
// dot-path template map from config.
func (c *Config) ServicePropertyResolvers() map[string]string {
	return c.Resolved().ServicePropertyResolvers
}

// Routes returns the framework routing convention from config.
func (c *Config) Routes() route.Config { return c.Resolved().Routes }

// CodeMap returns the code-map settings from config.
func (c *Config) CodeMap() config.CodeMap { return c.Resolved().CodeMap }

// KnownIssues returns the known-issues files in config, made absolute against
// the directory the config file is in.
func (c *Config) KnownIssues() []config.KnownIssues { return c.Resolved().KnownIssues }

// ResolvedFeatures returns the feature switches with their defaults applied.
func (c *Config) ResolvedFeatures() config.ResolvedFeatures { return c.Resolved().Features }

// Linting returns whether linting (cflint) is enabled in config.
func (c *Config) Linting() bool { return c.Resolved().Linting }

// LintMinSeverity returns the configured CFLint severity floor, or "" for none.
func (c *Config) LintMinSeverity() string { return c.Resolved().LintMinSeverity }

// References returns whether textDocument/references is enabled in config.
func (c *Config) References() bool { return c.Resolved().References }

// ComponentResolvers returns the configured component resolvers, then the Java
// stub resolver and the frameworks' presets.
func (c *Config) ComponentResolvers() []config.Resolver { return c.Resolved().ComponentResolvers }

// Frameworks returns the framework presets the config names.
func (c *Config) Frameworks() []string { return c.Resolved().Frameworks }

// PropertyResolvers returns the configured property resolvers.
func (c *Config) PropertyResolvers() []config.PropResolver { return c.Resolved().PropertyResolvers }

// BeanPaths returns the resolved bean namespace → absolute directory mapping.
func (c *Config) BeanPaths() map[string]string { return c.Resolved().BeanPaths }

// ResolvedCompletions returns the completion settings with defaults applied.
func (c *Config) ResolvedCompletions() config.ResolvedCompletions {
	r := c.Resolved()

	return config.ResolvedCompletions{
		TagSnippets:              r.TagSnippets,
		FunctionSnippets:         r.FunctionSnippets,
		GlobalFunctionResolution: r.GlobalFunctionResolution,
	}
}

// ResolvedDocBlock returns the doc block settings with defaults applied.
func (c *Config) ResolvedDocBlock() config.ResolvedDocBlock { return c.Resolved().DocBlock }

// ResolvedFormatting returns the formatting settings with defaults applied, for
// the daemon startup path and the `format` subcommand.
func (c *Config) ResolvedFormatting() config.ResolvedFormatting { return c.Resolved().Formatting }

// CallerInferenceLimits returns the optional batch caller inference policy.
func (c *Config) CallerInferenceLimits() config.CallerInferenceLimits {
	return c.Resolved().CallerInferenceLimits
}

// SettingsFrom builds the per-session settings for every daemon session from a
// config.
//
// It lives here rather than at the call site because the call site is package
// main, which nothing can test. A key read from config but never copied into
// Settings reaches no daemon session at all, and that is not a visible
// failure: it looks exactly like the key not being set. `completions` was
// missing for exactly that reason. The settings are now the resolved
// configuration itself (server.Settings embeds it), so a new key reaches every
// session by being in config.Resolved.
func SettingsFrom(c *Config) server.Settings {
	return server.Settings{Resolved: *c.Resolved(), ConfigPath: c.Path}
}
