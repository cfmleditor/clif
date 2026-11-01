package config

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

// AssumedApp is the configuration a file is read as when no config file governs
// it and its project, read as a whole, assumes no framework (AssumedFrameworks):
// the conventions of the nearest ColdBox application or module above dir. A
// repository of several applications (ColdBox's samples), or a module that does
// not ship ColdBox (every coldbox-modules checkout and its test harness), has no
// single owner for AssumedFrameworks to name, yet each application declares
// itself: `config/ColdBox.cfc` is an application's configuration and
// `ModuleConfig.cfc` a module's. That root's source tools (a CommandBox
// `build/` task runner, migrations) join it, as they do for a project owner.
//
// root is the directory the conventions were read from, which callers key a
// shared environment on; nil and "" when no root declares ColdBox.
func AssumedApp(dir string, read func(string) ([]byte, error), list func(string) ([]fs.DirEntry, error)) (r *Resolved, root string) {
	for range 32 {
		if coldBoxRoot(dir, list) {
			frameworks := append([]string{"coldbox"}, assumedSourceTools(dir, read, list)...)
			slices.Sort(frameworks)

			r = Resolve(&JSON{}, "")
			r.Frameworks = slices.Compact(frameworks)
			r.ComponentResolvers = append(r.ComponentResolvers, FrameworkResolvers(r.Frameworks)...)

			return r, dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return nil, ""
}

// coldBoxRoot reports whether dir is a ColdBox application's root (it holds
// config/ColdBox.cfc) or a module's (ModuleConfig.cfc). Names are compared
// without case: ColdBox's own samples spell the first Coldbox.cfc.
func coldBoxRoot(dir string, list func(string) ([]fs.DirEntry, error)) bool {
	has := func(dir, name string) bool {
		entries, err := list(dir)
		if err != nil {
			return false
		}

		return slices.ContainsFunc(entries, func(e fs.DirEntry) bool { return !e.IsDir() && strings.EqualFold(e.Name(), name) })
	}

	if has(dir, "ModuleConfig.cfc") {
		return true
	}

	entries, err := list(dir)
	if err != nil {
		return false
	}

	for _, e := range entries {
		if e.IsDir() && strings.EqualFold(e.Name(), "config") && has(filepath.Join(dir, e.Name()), "ColdBox.cfc") {
			return true
		}
	}

	return false
}
