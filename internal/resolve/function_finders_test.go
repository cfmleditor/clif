package resolve

import (
	"slices"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// The finders that count a component's braces read its strings as the parse
// does. In ColdBox's Router.cfc a #...# holds a quoted "{": read with a plain
// scanner, the string ends at that quote, the { counts as a brace, and every
// function after it was lost to all three.
func TestFunctionFindersReadStringsAsTheParseDoes(t *testing.T) {
	src := `component {
	function first() {
		thisRegex = listFirst( thisRegex, "{" ) & "{#listLast( thisPattern, "{" )#)";
		return thisRegex;
	}

	function after() {
		return 1;
	}
}
`
	if names := parser.ApplicationMethodNames(src); !slices.Contains(names, "after") {
		t.Errorf("ApplicationMethodNames = %v, want after", names)
	}

	if _, ok := wheelsMethods(src)["after"]; !ok {
		t.Errorf("wheelsMethods lost after")
	}

	if newProducerPlans(src).get("after") == nil {
		t.Errorf("producer plans lost after")
	}
}
