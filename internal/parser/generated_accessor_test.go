package parser

import "testing"

func TestGeneratedAccessorsAreDistinguishedFromSourceFunctions(t *testing.T) {
	for _, source := range []string{
		`component accessors="true" {
 property name="rows" type="query";
 property name="items" type="array";
 array function getItems(){return [];}
 }`,
		`<cfcomponent accessors="true">
 <cfproperty name="rows" type="query">
 <cfproperty name="items" type="array">
 <cffunction name="getItems" returntype="array"><cfreturn []></cffunction>
 </cfcomponent>`,
	} {
		pr := Parse(testURI, source)
		for _, name := range []string{"getRows", "setRows", "setItems"} {
			if !collectionFunction(t, pr, name).Generated {
				t.Errorf("%s lost synthetic ownership", name)
			}
		}

		if collectionFunction(t, pr, "getItems").Generated {
			t.Error("explicit override marked as generated")
		}
	}
}
