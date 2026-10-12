package path

import "path/filepath"

// A Lucee source checkout's admin helpers run under the engine's /admin
// namespace, despite being physically nested under core/src. This is a
// documented deployment assumption for the recognized source layout only.
// Explicit editor/application mappings and server configuration take precedence
// through the common bootstrap mapping policy.
func assumedLuceeAdmin(root string, read func(string) ([]byte, error)) string {
	for _, marker := range []string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc",
		"core/src/main/cfml/context/admin/services.update.functions.cfm",
		"core/src/main/cfml/context/admin/ext.functions.cfm",
	} {
		if _, err := read(filepath.Join(root, marker)); err != nil {
			return ""
		}
	}

	return filepath.Join(root, "core", "src", "main", "cfml", "context", "admin")
}
