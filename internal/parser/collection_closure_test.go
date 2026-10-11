package parser

import "testing"

func TestCollectionClosuresRespectNamedFunctionLocals(t *testing.T) {
	for _, tc := range []struct {
		name, other, body string
		valid             bool
	}{
		{"unrelated anonymous", `function other() { var cb=function() { return false; }; }`, "", true},
		{"unrelated arrow", `function other() { var cb=()=>false; }`, "", true},
		{"same local name", `function other() { var cb=function() { var items={}; items[key]=false; }; }`, "", true},
		{"top level closure", `variables.cb=function() { return false; };`, "", true},
		{"same function capture", "", `var cb=function() { items[key]=false; };`, false},
		{"same function shadow", "", `var cb=function() { var items={}; items[key]=new models.Site(); };`, false},
		{"shared alias", `function other() { var cb=()=>false; }`, `variables.shared=items;`, false},
		{"nested named capture", "", `function nested() {
   var cb=function(){ items[key]=false; };
  }`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pr := ParseWithOptions(testURI, `component {
 function one() {
  var items={}; items[key]=new models.Site();
  `+tc.body+`
  return items[key];
 }
 function all() {
  var items={}; items[key]=new models.Site();
  `+tc.body+`
  return items;
 }
 `+tc.other+`
}`, &ParseOptions{ExtractCalls: true})

			one := collectionFunction(t, pr, "one")
			all := collectionFunction(t, pr, "all")

			if (one.ReturnComponent == "models.Site") != tc.valid {
				t.Fatalf("indexed return=%q, valid=%v", one.ReturnComponent, tc.valid)
			}

			if (all.CollectionValues != nil) != tc.valid {
				t.Fatalf("whole collection=%+v, valid=%v", all.CollectionValues, tc.valid)
			}
		})
	}
}
