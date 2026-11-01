package path

import (
	"path/filepath"
	"regexp"
	"strings"
)

// A Mura plugin is deployed as plugins/<directory>, and Mura's plugin manager
// writes plugins/mappings.cfm for it: the plugin's package maps to its
// directory, and each <mapping name directory> its plugin/config.xml declares
// maps to that directory inside it. Every Mura application includes that file,
// so the plugin's own components name each other through the package
// (Slatwall's `extends="Slatwall.com.utility.BaseObject"`). A checkout of a
// plugin is that directory, and nothing in it spells the mapping as CFML.

// MuraPlugin is a Mura plugin recognised at Root.
type MuraPlugin struct {
	// Root is the plugin's directory, holding plugin/config.xml(.cfm).
	Root string
	// Package is the plugin's <package>.
	Package string
	// Mappings are the mappings Mura gives the plugin, by name: the package
	// and each declared <mapping>, to absolute directories.
	Mappings map[string]string
}

var (
	muraPluginTagRe     = regexp.MustCompile(`(?i)<plugin\s*>`)
	muraPluginPackageRe = regexp.MustCompile(`(?i)<package>\s*([A-Za-z_][\w-]*)\s*</package>`)
	muraPluginMappingRe = regexp.MustCompile(`(?i)<mapping\s+name\s*=\s*["']([A-Za-z_][\w-]*)["']\s+directory\s*=\s*["']([^"'#<>]+)["']`)
)

// AssumedMuraPlugin is the Mura plugin whose plugin/config.xml.cfm or
// plugin/config.xml is in dir, if one is. It needs the <plugin> element and a
// literal <package>: a Mura module's config.xml.cfm has neither.
func AssumedMuraPlugin(dir string, read func(string) ([]byte, error)) (MuraPlugin, bool) {
	for _, name := range []string{"config.xml.cfm", "config.xml"} {
		data, err := read(filepath.Join(dir, "plugin", name))
		if err != nil {
			continue
		}

		if !muraPluginTagRe.Match(data) {
			return MuraPlugin{}, false
		}

		m := muraPluginPackageRe.FindSubmatch(data)
		if m == nil {
			return MuraPlugin{}, false
		}

		p := MuraPlugin{Root: dir, Package: string(m[1]), Mappings: map[string]string{string(m[1]): dir}}

		for _, mm := range muraPluginMappingRe.FindAllSubmatch(data, -1) {
			target := filepath.Clean(filepath.Join(dir, filepath.FromSlash(string(mm[2]))))
			// A mapping is a directory inside the plugin.
			if rel, err := filepath.Rel(dir, target); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				continue
			}

			if _, dup := p.Mappings[string(mm[1])]; !dup {
				p.Mappings[string(mm[1])] = target
			}
		}

		return p, true
	}

	return MuraPlugin{}, false
}

// NearestMuraPlugin is the Mura plugin dir is in: AssumedMuraPlugin of dir or
// the nearest directory above it that is one.
func NearestMuraPlugin(dir string, read func(string) ([]byte, error)) (MuraPlugin, bool) {
	for range 32 {
		if p, ok := AssumedMuraPlugin(dir, read); ok {
			return p, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return MuraPlugin{}, false
}
