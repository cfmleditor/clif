package parser

import (
	"strings"
	"testing"
)

func TestBuiltinProducerCannotTypeShadowedCallable(t *testing.T) {
	for _, source := range []string{
		`component {function run(listToArray){var parts=listToArray();}}`,
		`component {function run(){var listToArray=unknown;var parts=listToArray();}}`,
		`component {function run(){each([],function(listToArray){var parts=listToArray();});}}`,
		`<cfcomponent><cffunction name="run"><cfargument name="listToArray"><cfscript>var parts=listToArray();</cfscript></cffunction></cfcomponent>`,
	} {
		pr := ParseWithOptions(testURI, source, &ParseOptions{BuiltinReturnLookup: func(name string) string {
			if strings.EqualFold(name, "listToArray") {
				return NativeTypePrefix + "array"
			}

			return ""
		}})
		if len(pr.Scopes) == 0 {
			t.Fatalf("fixture has no function scope: %s", source)
		}

		for _, scope := range pr.Scopes {
			for _, ref := range pr.FuncComponentRefsAt(scope.Start, scope.End, uint32(scope.End)) {
				if ref.Variable == "parts" && ref.Component == NativeTypePrefix+"array" {
					t.Errorf("shadowed callable inherited builtin array: %+v in %s", ref, source)
				}
			}
		}
	}
}
