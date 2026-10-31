package parser

import (
	"slices"
	"testing"
)

// A string in a <cfif>, <cfelseif> or <cfreturn> can hold a `#…#` span, and the
// span scan over the whole tag records the call in it before the tag's handler
// scans the expression, which finds the same call again. <cfset> subtracts what
// is recorded; these tags did not, so each such call was reported twice.
//
// The subtraction is by source span, not by name: two calls of one name are two
// calls, in a span or out of one, and must both stay.
func TestACallInAStringInATagConditionIsRecordedOnce(t *testing.T) {
	for _, c := range []struct {
		name string
		src  string
		want []string
	}{
		{"cfif", "<cfif x eq \"#f( 1 )#\">y</cfif>", []string{"f"}},
		{"cfelseif", "<cfif a>b<cfelseif x eq \"#f( 1 )#\">c</cfif>", []string{"f"}},
		{"cfreturn", "<cffunction name=\"m\"><cfreturn \"a #f( 1 )# b\"></cffunction>", []string{"f"}},
		{"two spans on one line", "<cfif x eq \"#f( \"a\" )# #f( \"b\" )#\">y</cfif>", []string{"f", "f"}},
		{"a call beside a span of the same name", "<cfif f( \"a\" ) eq \"#f( \"b\" )#\">y</cfif>", []string{"f", "f"}},
		{"two calls in a condition", "<cfif f( \"a\" ) and f( \"b\" )>y</cfif>", []string{"f", "f"}},
		{"a span nested in an argument", "<cfif g( h( \"#f( 1 )#\" ) )>y</cfif>", []string{"f", "g", "h"}},
		{"on the line after the tag", "<cfif\n\tx eq \"#f( 1 )#\">y</cfif>", []string{"f"}},
		{"a call in the attributes of another tag", "<cfoutput><cfset x = \"#f( 1 )#\"></cfoutput>", []string{"f"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			pr := ParseWithOptions(testURI, c.src, &ParseOptions{ExtractCalls: true})

			var got []string

			cs := pr.AllCalls()
			for i := range cs {
				got = append(got, cs[i].FuncName)
			}

			slices.Sort(got)

			if !slices.Equal(got, c.want) {
				t.Fatalf("calls = %v, want %v", got, c.want)
			}
		})
	}
}
