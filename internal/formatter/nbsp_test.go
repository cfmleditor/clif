package formatter

import (
	"strings"
	"testing"
)

// U+00A0 is content: it renders, and Go's strings.TrimSpace treats it as
// whitespace. The formatter trimmed it from the edge of a text run, so output
// lost a character and the whitespace-only guard refused the file. Galaxie
// Blog's Udf.cfc (a comment line ending in no-break spaces) and the corpus's
// includeWhitespace.cfm were both refused for it.
func TestNoBreakSpacesAtTheEdgeOfTextAreKept(t *testing.T) {
	const nbsp = " "

	for name, src := range map[string]string{
		"text before a newline": "<p>text" + nbsp + "\n</p>\n",
		"after a comment":       "<cfcomponent>\n<cffunction name=\"f\">\n\t<!--- a ---> " + nbsp + nbsp + "\n\t<cfset x = 1>\n</cffunction>\n</cfcomponent>\n",
	} {
		t.Run(name, func(t *testing.T) {
			out := mustFormat(t, src)

			if got, want := strings.Count(out, nbsp), strings.Count(src, nbsp); got != want {
				t.Fatalf("formatting kept %d of %d no-break spaces:\n%s", got, want, out)
			}
		})
	}
}

func TestTrimSpaceTrimsOnlyASCII(t *testing.T) {
	if got := trimSpace(" \t\r\n x  \n"); got != " x " {
		t.Fatalf("trimSpace = %q", got)
	}

	if got := string(trimSpaceBytes([]byte("  y  "))); got != " y " {
		t.Fatalf("trimSpaceBytes = %q", got)
	}
}
