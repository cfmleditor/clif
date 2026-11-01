package path

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAMuraPluginCheckoutHasTheMappingsMuraGivesIt: Slatwall is a Mura
// plugin, and its components extend "Slatwall.com.utility.BaseObject" through
// the mapping Mura's plugin manager writes for its <package>, and reach
// ValidateThis through the <mapping> its plugin/config.xml.cfm declares.
// Nothing in the checkout spells either as CFML, so every base was a
// component that did not exist. A Mura module's config.xml.cfm, which has no
// <plugin>, maps nothing, and a mapping leaving the plugin is not followed.
func TestAMuraPluginCheckoutHasTheMappingsMuraGivesIt(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"plugin/config.xml.cfm": `<cfsilent><cfset v = "1"></cfsilent>
<plugin>
<name>Slatwall</name>
<package>Slatwall</package>
<directoryFormat>packageOnly</directoryFormat>
<mappings>
	<mapping name="ValidateThis" directory="org/ValidateThis"/>
	<mapping name="Outside" directory="../elsewhere"/>
</mappings>
</plugin>`,
		"Application.cfc":              `component {}`,
		"com/utility/BaseObject.cfc":   `component {}`,
		"admin/Application.cfc":        `component {}`,
		"module/plugin/config.xml.cfm": `<mura name="Plugin" contenttypes="*"/>`,
		"module/Application.cfc":       `component {}`,
	}

	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	InvalidateAppMappingsCache()
	t.Cleanup(InvalidateAppMappingsCache)

	got := LoadAppMappings(filepath.Join(root, "admin"))
	if got["Slatwall"] != root {
		t.Errorf("Slatwall maps to %q, want %q", got["Slatwall"], root)
	}

	if want := filepath.Join(root, "org", "ValidateThis"); got["ValidateThis"] != want {
		t.Errorf("ValidateThis maps to %q, want %q", got["ValidateThis"], want)
	}

	if _, ok := got["Outside"]; ok {
		t.Errorf("a mapping leaving the plugin was followed: %q", got["Outside"])
	}

	if _, ok := AssumedMuraPlugin(filepath.Join(root, "module"), os.ReadFile); ok {
		t.Errorf("a Mura module's config.xml.cfm was read as a plugin")
	}
}
