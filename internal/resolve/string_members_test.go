package resolve

import "testing"

// TestAStringsMemberFunctionsAreItsMethods: a CFML member function the doc
// sources record for strings is a method of a string value; one they do not
// is still reported. lucee-docs calls `cleanLanguage.reFindNoCase( … )` on a
// string literal.
func TestAStringsMemberFunctionsAreItsMethods(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Use.cfc": `component {
function run() {
	var s = "javascript";
	s.reFindNoCase( "^java" );
	s.listRest();
	s.buildAdminLink();
}
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Use.cfc"), map[string]string{
		"s.reFindNoCase":   "",
		"s.listRest":       "",
		"s.buildAdminLink": "method 'buildAdminLink' not found on builtin native.string",
	})
}
