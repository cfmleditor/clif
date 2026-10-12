package path

import (
	"path/filepath"

	"github.com/cfmleditor/clif/internal/parser"
)

// AssumedProjectMappings makes a project's own declared library namespace
// available to its nested source files. The documented assumption is that
// those library components run in this standard project's host context, even
// when their physical directory has an otherwise empty/restrictive Application.
// It does not import lifecycle methods, beans or shared variables from that host.
func AssumedProjectMappings(dir string, read func(string) ([]byte, error)) map[string]string {
	out := map[string]string{}
	seen := map[string]bool{}

	for range 32 {
		for _, name := range []string{"server.json", ".cfconfig.json"} {
			if _, err := read(filepath.Join(dir, name)); err == nil {
				return out
			}
		}

		if !seen["admin"] {
			if admin := assumedLuceeAdmin(dir, read); admin != "" {
				seen["admin"] = true
				out["admin"] = admin
			}
		}

		for _, profile := range []struct {
			namespace, source, target string
			markers                   []string
		}{
			{"testbox", "tests/Application.cfc", "", []string{"system/BaseSpec.cfc", "system/Assertion.cfc"}},
			{"coldbox", "tests/Application.cfc", "", []string{"system/Bootstrap.cfc", "system/web/Controller.cfc"}},
			{"framework", "Application.cfc", "framework", []string{"framework/one.cfc", "framework/ioc.cfc"}},
			{"mura", "Application.cfc", "core/mura", []string{"core/mura/configBean.cfc", "core/mura/bean/beanFactory.cfc", "core/appcfc/applicationSettings.cfm"}},
			{"murawrm", "Application.cfc", "", []string{"core/mura/configBean.cfc", "core/mura/bean/beanFactory.cfc", "core/appcfc/applicationSettings.cfm"}},
			{"contentbox", "Application.cfc", "modules/contentbox", []string{"modules/contentbox/ModuleConfig.cfc", "modules/contentbox/models/content/ContentService.cfc"}},
		} {
			if seen[profile.namespace] {
				continue
			}

			known := true

			for _, marker := range profile.markers {
				if _, err := read(filepath.Join(dir, marker)); err != nil {
					known = false

					break
				}
			}

			if !known {
				continue
			}
			// A recognized nearer layout owns this namespace even if its own
			// mapping is different or cannot be evaluated.
			seen[profile.namespace] = true
			// Read the declaring application's own mapping expressions, without
			// calling bootstrapMappings and recursively manufacturing defaults.
			mapping := parser.MappingSources(filepath.Join(dir, profile.source), nil, read)[profile.namespace]
			if mapping == filepath.Join(dir, profile.target) {
				out[profile.namespace] = mapping
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return out
}
