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
