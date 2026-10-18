package resolve

import (
	"testing"
)

// TestAChainOnACreatedComponentTypesTheConstruction: an inline
// createObject( "component", "Field" ) that the parser could not bind — a
// bare statement or a tag-template expression, where an assignment would
// have recorded the ref — names the calling directory's component, so its
// chain continues on the component rather than as a builtin's dynamic
// value. Lucee's admin helper `field()` UDFs are the corpus shape. A
// construction whose path names no indexed component keeps the dynamic
// acceptance it had.
func TestAChainOnACreatedComponentTypesTheConstruction(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Field.cfc": `component {
	function getName() { return ""; }
}`,
		"Driver.cfc": `component {
	function bare() {
		createObject("component", "Field").getName();
		createObject("component", "Field").notADeclaration();
		createObject("component", "missing.Driver").getLongName();
	}
}`,
		"TagDriver.cfc": `<cfcomponent>
	<cffunction name="make">
		<cfreturn createObject("component","Field").getName()>
	</cffunction>
	<cffunction name="makeBroken">
		<cfreturn createObject("component","Field").describe()>
	</cffunction>
</cfcomponent>`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Driver.cfc"), map[string]string{
		"createObject.getName":         "",
		"createObject.notADeclaration": "method 'notADeclaration' not found in Field",
		// No indexed component under the constructed path: still dynamic.
		"createObject.getLongName": "",
	})

	tagReasons := reasonsWith(t, &Resolver{}, dir, "TagDriver.cfc")
	if tagReasons["createObject.getName"] != "" {
		t.Errorf("TagDriver createObject.getName: got %q", tagReasons["createObject.getName"])
	}

	if want := "method 'describe' not found in Field"; tagReasons["createObject.describe"] != want {
		t.Errorf("TagDriver createObject.describe: got %q, want %q", tagReasons["createObject.describe"], want)
	}
}
