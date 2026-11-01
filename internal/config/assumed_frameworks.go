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
	return assumedFrameworks(roots, read, directories...).frameworks
}

// assumption is what a recognised project is read as: its presets, and the
// project rules no preset holds.
type assumption struct {
	frameworks []string
	preside    bool
	// slatwall is set when the project is Slatwall's own checkout.
	slatwall bool
}

// presideRequestContext is what Preside hands every handler, view and
// interceptor as `event`: its config sets ColdBox's requestContextDecorator
// to it, and the methods Preside's code calls on the request
// (buildAdminLink(), addAdminBreadCrumb()) are declared there, not on
// ColdBox's RequestContext.
const presideRequestContext = "preside.system.coldboxModifications.RequestContextDecorator"

func assumedFrameworks(roots []string, read func(string) ([]byte, error), directories ...func(string) ([]fs.DirEntry, error)) assumption {
	owners := map[string]bool{}
	preside, slatwall := false, false
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
			case "preside":
				// Preside runs on ColdBox, which it takes as a dependency
				// rather than shipping; the stubs answer for it.
				presets["coldbox"], preside = true, true
			case "mura":
				owner = filepath.Dir(filepath.Dir(target))
				presets["mura"] = true
			default:
				continue
			}

			owners[owner] = true
		}

		// A Mura plugin runs inside Mura, which hands it the Mura scope.
		if plugin, ok := cfpath.NearestMuraPlugin(root, read); ok {
			owners[plugin.Root] = true
			presets["mura"] = true
			if isSlatwall(plugin, read) {
				// Slatwall's Application.cfc is an FW/1 application (its own
				// copy, org/fw1/framework.cfc), whose views call buildURL()
				// and view() on it.
				slatwall, presets["fw1"] = true, true
			}
		}

		if public := cfpath.AssumedWheelsRoot(root, read); public != "" {
			owners[filepath.Dir(public)] = true
			presets["wheels"] = true
		}
	}

	if len(owners) != 1 {
		return assumption{}
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

	return assumption{frameworks: out, preside: preside, slatwall: slatwall}
}

// Assumed is the configuration a project with no config file is read as: the
// framework its source is, when it is one, with that framework's resolvers.
// Everything else is the defaults. The command line and the editor each did this
// by hand.
func Assumed(roots []string, read func(string) ([]byte, error), directories ...func(string) ([]fs.DirEntry, error)) *Resolved {
	r := Resolve(&JSON{}, "")
	a := assumedFrameworks(roots, read, directories...)
	r.Frameworks = a.frameworks

	if a.preside {
		// Ahead of the ColdBox preset's own `event`, which it replaces.
		r.ComponentResolvers = append(r.ComponentResolvers, variableResolver(presideRequestContext, "event"))
	}

	if a.slatwall {
		r.ComponentResolvers = append(r.ComponentResolvers, slatwallResolvers()...)
		r.PropertyResolvers = append(r.PropertyResolvers, slatwallPropertyResolvers()...)
	}

	r.ComponentResolvers = append(r.ComponentResolvers, FrameworkResolvers(r.Frameworks)...)

	return r
}
