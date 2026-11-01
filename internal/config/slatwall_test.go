package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// TestASlatwallCheckoutIsAMuraPluginOnFW1WithItsStoredValues: Slatwall is a
// Mura plugin whose Application.cfc is an FW/1 application, and nothing
// recognised it, so no preset was assumed. Recognised by its
// plugin/config.xml.cfm, it assumes the Mura and FW/1 presets and types what
// its Application.cfc stores: $.slatwall (the Mura scope key it sets), getFW(),
// getService( "x" ), and the ColdSpring-autowired service properties. Another
// Mura plugin gets the Mura preset alone.
func TestASlatwallCheckoutIsAMuraPluginOnFW1WithItsStoredValues(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		filepath.Join(root, "plugin", "config.xml.cfm"):            `<plugin><package>Slatwall</package></plugin>`,
		filepath.Join(root, "com", "service", "BaseService.cfc"):   `component {}`,
		filepath.Join(root, "com", "utility", "SlatwallScope.cfc"): `component {}`,
	}

	read := func(p string) ([]byte, error) {
		if s, ok := files[p]; ok {
			return []byte(s), nil
		}

		return nil, os.ErrNotExist
	}

	r := Assumed([]string{root}, read)
	if !slices.Equal(r.Frameworks, []string{"fw1", "mura"}) {
		t.Fatalf("frameworks = %v, want [fw1 mura]", r.Frameworks)
	}

	resolvers := ParserResolvers(r.ComponentResolvers)
	for expr, want := range map[string]string{
		"$.slatwall":                 slatwallScope,
		"rc.$.Slatwall":              slatwallScope,
		"getFW()":                    slatwallApp,
		`getService("orderService")`: "Slatwall.com.service.orderService",
		"cart()":                     "Slatwall.com.entity.Order",
		"getPluginConfig()":          "mura.plugin.pluginConfig",
	} {
		if got, _, _ := parser.ResolveFromCallMatch(expr, resolvers); got != want {
			t.Errorf("%s resolves to %q, want %q", expr, got, want)
		}
	}

	props := ParserPropertyResolvers(r.PropertyResolvers)
	for name, want := range map[string]string{
		"orderService": "Slatwall.com.service.orderService",
		"orderDAO":     "Slatwall.com.dao.orderDAO",
		"DAO":          "",
	} {
		if got := parser.ResolveProperty(map[string]string{"name": name}, props); got != want {
			t.Errorf("property %s resolves to %q, want %q", name, got, want)
		}
	}

	files[filepath.Join(root, "plugin", "config.xml.cfm")] = `<plugin><package>Other</package></plugin>`

	if got := Assumed([]string{root}, read); !slices.Equal(got.Frameworks, []string{"mura"}) || len(got.PropertyResolvers) != 0 {
		t.Errorf("another Mura plugin: frameworks = %v, property resolvers %v", got.Frameworks, got.PropertyResolvers)
	}
}
