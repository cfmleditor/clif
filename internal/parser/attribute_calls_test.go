package parser

import (
	"slices"
	"testing"
)

// A call written in `#...#` inside a component's or a property's attribute is a
// call: Lucee computes a property's default and a component's hint that way,
//
//	property name="uuid" default="#createUUID()#";
//	component hint="prefix-#now()#" { }
//
// Both readers consumed the attribute tokens to learn the name, type and
// extends, and never looked inside a string, so each call was recorded nowhere
// and a function only ever called from a default read as unreachable. Lucee's
// own tests are written this way in 12 files.
func TestACallInAComponentOrPropertyAttributeIsRecorded(t *testing.T) {
	for _, c := range []struct {
		name string
		src  string
		want []string
	}{
		{"a property default", `component { property name="u" default="#createUUID()#"; }`, []string{"createUUID"}},
		{"a single-quoted default", `component { property name="u" type="any" default='#{"ts":now(),"n":1}#'; }`, []string{"now"}},
		{"a default with text around it", `component { property name="u" default="pre-#repeatString( 'x', 2 )#-post"; }`, []string{"repeatString"}},
		{"an unquoted default", `component { property name="u" default=#repeatString( 'u', 2 )#; }`, []string{"repeatString"}},
		{"two properties", `component { property name="a" default="#now()#"; property name="b" default="#createUUID()#"; }`, []string{"createUUID", "now"}},
		{"a positional property", `component { property string u default="#now()#"; }`, []string{"now"}},
		{"a component hint", `component hint="#now()#" { function f() {} }`, []string{"now"}},
		{"a hint with text around it", `component hint="prefix-#now()#-suffix" extends="a.B" { }`, []string{"now"}},
		{"an extends that is computed", `component extends="#base()#" { }`, []string{"base"}},
		{"an interface", `interface hint="#now()#" { }`, []string{"now"}},
		{
			// Nothing in a plain default, a value with no call, or a hash that
			// is only a variable is a call.
			"no call",
			`component hint="plain" { property name="a" default="#user.name#"; property name="b" default="literal"; property name="c"; }`,
			[]string{},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			pr := ParseWithOptions("file:///t.cfc", c.src, &ParseOptions{ExtractCalls: true})

			got := []string{}
			calls := pr.AllCalls()

			for i := range calls {
				got = append(got, calls[i].FuncName)
			}

			slices.Sort(got)

			if !slices.Equal(got, c.want) {
				t.Errorf("got %v want %v", got, c.want)
			}
		})
	}
}

// `property` declares a property only where a name or a type follows it. Lucee's
// admin writes `for( property in defaultLang ) { … }`, and the declaration reader
// took every `property` for a declaration and swallowed tokens up to the next
// semicolon, which here is inside the loop's first statement: the call in its
// condition and the variable it assigns were lost, in both variable scans.
func TestPropertyAsAVariableIsNotADeclaration(t *testing.T) {
	loop := "for( property in defaultLang ) {\n\tif( !structKeyExists( langs, property ) ) { total = 1; }\n}\n"

	for _, c := range []struct{ name, src string }{
		{"in a component body", "component {\n" + loop + "}\n"},
		{"in a cfscript of a template", "<cfscript>\n" + loop + "</cfscript>\n"},
		{"compared", "component {\n\tif( property == other ) { total = 1; structKeyExists( a, b ); }\n}\n"},
		{"as an argument", "component {\n\tcheck( property ); total = 1; structKeyExists( a, b );\n}\n"},
		{"assigned", "component {\n\tproperty = 1; structKeyExists( a, b ); total = 1;\n}\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			pr := ParseWithOptions("file:///t.cfc", c.src, &ParseOptions{ExtractCalls: true})

			found := false
			calls := pr.AllCalls()

			for i := range calls {
				found = found || calls[i].FuncName == "structKeyExists"
			}

			if !found {
				t.Errorf("structKeyExists was lost")
			}

			for scan, vars := range map[string][]string{"VariablesVars": pr.VariablesVars(), "AllVars": allVarNames(pr)} {
				if !slices.Contains(vars, "total") {
					t.Errorf("%s: total was lost, got %v", scan, vars)
				}
			}
		})
	}

	// A declaration is still one.
	pr := Parse(testURI, `component { property name="a" type="string"; property string b; property c; }`)
	if got := len(pr.Properties); got != 3 {
		t.Errorf("declared %d properties, want 3", got)
	}
}

func allVarNames(pr *ParseResult) []string {
	names := []string{}
	vars := pr.AllVars()

	for i := range vars {
		names = append(names, vars[i].Name)
	}

	return names
}
