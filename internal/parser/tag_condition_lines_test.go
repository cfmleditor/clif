package parser

import (
	"slices"
	"testing"
)

// A condition written on the line after the tag name follows a newline or a
// tab, not a space. tagBody looked for a space alone, so
//
//	<cfif
//		f( a )>
//
// had no body and recorded nothing, and
//
//	<cfif
//		( f( a ) and g( b ) )>
//
// was cut at the space before `and`, losing f. What it did find it filed on the
// tag's line rather than its own. Galaxie Blog's device-detection test is the
// second shape: `reFindNoCase` and `left` were reported a line early.
func TestAMultiLineTagConditionKeepsItsCallsAndTheirLines(t *testing.T) {
	// Lines are the parser's own, counted from zero; the report adds one.
	type call struct {
		name string
		line uint32
	}

	for _, c := range []struct {
		name string
		src  string
		want []call
	}{
		{"on one line", "<cfif ( f( a ) and g( b ) )>x</cfif>", []call{{"f", 0}, {"g", 0}}},
		{"under the tag name", "<cfif\n\tf( a )>x</cfif>", []call{{"f", 1}}},
		{"a group under the tag name", "<cfif\n\t( f( a ) and g( b ) )>x</cfif>", []call{{"f", 1}, {"g", 1}}},
		{"a tab after the name", "<cfif\t( f( a ) )>x</cfif>", []call{{"f", 0}}},
		{"under a cfelseif", "<cfif x>a<cfelseif\n\tf( a )>b</cfif>", []call{{"f", 1}}},
		{"under a cfreturn", "<cffunction name=\"m\">\n<cfreturn\n\tf( a )>\n</cffunction>", []call{{"f", 2}}},
		{"a condition spanning lines", "<cfif f( a ) and\n\tg( b )>x</cfif>", []call{{"f", 0}, {"g", 1}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			pr := ParseWithOptions(testURI, c.src, &ParseOptions{ExtractCalls: true})

			var got []call

			cs := pr.AllCalls()
			for i := range cs {
				got = append(got, call{cs[i].FuncName, cs[i].Line})
			}

			slices.SortFunc(got, func(a, b call) int {
				if a.line != b.line {
					return int(a.line) - int(b.line)
				}

				return compareStrings(a.name, b.name)
			})

			if !slices.Equal(got, c.want) {
				t.Fatalf("calls = %v, want %v", got, c.want)
			}
		})
	}
}

func compareStrings(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}

	return 0
}
