package parser

import (
	"slices"
	"testing"
)

// A string in a <cfscript> may hold the closing tag, and Lucee's own tests do:
//
//	var content = "<cfscript>a=1;</cfscript>";
//
// All three places that look for the end of the block took the first
// `</cfscript>`, which closed it inside the string, so the rest of the script
// was read as markup and its calls were lost.
func TestACloseTagInAStringDoesNotEndTheScript(t *testing.T) {
	for _, c := range []struct {
		name string
		src  string
		want []string
	}{
		{
			"a double-quoted string",
			"<cfscript>\nvar a = \"<cfscript>x=1;</cfscript>\";\nfirst();\n</cfscript>\n<cfoutput>#second()#</cfoutput>\n",
			[]string{"first", "second"},
		},
		{
			"a single-quoted string",
			"<cfscript>\nvar a = '</cfscript>';\nfirst();\n</cfscript>\n",
			[]string{"first"},
		},
		{
			"several strings, in either case",
			"<cfscript>\n// a comment\nvar a = \"</cfscript>\";\nvar b = '</CFSCRIPT>';\nfirst();\n</cfscript>\n<cfset x = second()>\n",
			[]string{"first", "second"},
		},
		{
			"in a script component wrapped in a cfscript",
			"<cfscript>\ncomponent {\n\tfunction f() {\n\t\tvar c = \"\n<cfscript>\na=1;\n</cfscript>\";\n\t\tfirst();\n\t}\n}\n</cfscript>\n",
			[]string{"first"},
		},
		{
			// Two strings that each hold a whole pair, as Lucee's LDEV0869 does:
			// the second opener sits between the first two closers.
			"two strings that each hold a pair",
			"<cfscript>\ncomponent {\n\tfunction f() {\n\t\tvar a = \"\n<cfscript>\nx=1;\n</cfscript>\";\n\t\tvar b = \"\n<cfscript>\ny=2;\n</cfscript>\";\n\t\tfirst();\n\t}\n}\n</cfscript>\n",
			[]string{"first"},
		},
		{
			// Ordinary files: a closer, text, an opener, a closer.
			"two blocks",
			"<cfscript>\nfirst();\n</cfscript>\ntext\n<cfscript>\nsecond();\n</cfscript>\n",
			[]string{"first", "second"},
		},
		{
			// A stray second closer with no string behind it: the first is the
			// end, as it always was.
			"a stray closer after the block",
			"<cfscript>\nfirst();\n</cfscript>\nstray </cfscript>\n<cfset x = second()>\n",
			[]string{"first", "second"},
		},
		{
			// One closer, and a string that never closes: nothing to choose.
			"an unclosed string and one closer",
			"<cfscript>\nvar a = \"oops;\nfirst();\n</cfscript>\n<cfset x = second()>\n",
			[]string{"second"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := tagFileCalls(t, c.src)
			if !slices.Equal(got, c.want) {
				t.Errorf("got %v want %v", got, c.want)
			}
		})
	}
}
