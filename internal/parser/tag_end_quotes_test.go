package parser

import (
	"strings"
	"testing"
)

// A '<' inside a quoted attribute value is the value's text, not a tag
// opening: a hint holding "<br />" or an output holding "</cfif>" ended the
// tag there, and every attribute after it was lost.
func TestAQuotedLessThanDoesNotEndATag(t *testing.T) {
	for src, wantAttr := range map[string]string{
		`<cfargument name="m" hint="one <br /> two" type="Array">`: "type",
		`<cffile action="append" output="</cfif>" file="f.txt">`:   "file",
		`<cfset x="a<b"><cfif a>x</cfif>`:                          "",
	} {
		ends := newTagEnds(src)
		k := strings.IndexAny(src, " \t")
		end := tagEnd(&ends, k)

		if wantAttr == "" {
			if got := src[:end]; got != `<cfset x="a<b">` {
				t.Errorf("%q ends at %q", src, got)
			}

			continue
		}

		if end != len(src) {
			t.Errorf("%q ends at %d, want the whole tag (%d)", src, end, len(src))
		}

		if got := getAttr(src[:end], wantAttr); got == "" {
			t.Errorf("%q lost %s", src, wantAttr)
		}
	}

	// A CF tag written inside an HTML one still ends it where it opens.
	src := `<input <cfif a>checked</cfif>>`

	ends := newTagEnds(src)
	if end := tagEnd(&ends, len("<input")); src[:end] != "<input " {
		t.Errorf("%q ends at %q, want <input ", src, src[:end])
	}
}
