package formatter

import (
	"strings"
	"testing"
)

// An escaped hash ("##") is its own node in the grammar. isInlineNode left it
// out, so it counted as block-level and ended the run of text around it: text
// containing `##` was broken onto separate lines, and the continuation was
// written with no indent. At the top of a document, where each sibling was
// formatted on its own rather than as a run, a lone `#` split the text the same
// way, taking `#color#` apart into "#color" and "#.". Only whitespace moved, so
// the whitespace-only guard let it through.
func TestHashesInTextStayOnTheLine(t *testing.T) {
	const prose = "This is a ##test of hashtags. #color#."

	for name, tc := range map[string]struct{ src, want string }{
		"at the top of a document": {prose + "\n", prose + "\n"},
		"in cfoutput":              {"<cfoutput>" + prose + "</cfoutput>\n", "<cfoutput>\n    " + prose + "\n</cfoutput>\n"},
		"in an element":            {"<p>" + prose + "</p>\n", "<p>" + prose + "</p>\n"},
		"in a link":                {`<a href="#asdf">` + prose + "</a>\n", `<a href="#asdf">` + prose + "</a>\n"},
		"in a nested element": {
			"<cfoutput>\n    <p>\n        " + prose + "\n    </p>\n</cfoutput>\n",
			"<cfoutput>\n    <p>\n        " + prose + "\n    </p>\n</cfoutput>\n",
		},
		"in a cfif": {
			"<cfif flavor NEQ ''>\n    " + prose + "\n</cfif>\n",
			"<cfif flavor NEQ ''>\n\n    " + prose + "\n\n</cfif>\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := formatStable(t, tc.src); got != tc.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tc.want)
			}
		})
	}
}

func TestEscapedHashAloneStaysInline(t *testing.T) {
	for name, src := range map[string]string{
		"leading":  "##tag line\n",
		"trailing": "line ending in ##\n",
		"only":     "##\n",
	} {
		t.Run(name, func(t *testing.T) {
			if got := formatStable(t, src); got != src {
				t.Errorf("got %q, want %q", got, src)
			}
		})
	}
}

// Text at the top of a document is still one run per stretch of text: a tag
// between two lines of text keeps them apart, and a blank line is not invented.
func TestTextAroundATagStaysSeparate(t *testing.T) {
	src := "First line #a#.\n<p>block</p>\nSecond line ##b.\n"

	if got := formatStable(t, src); got != src {
		t.Errorf("got %q, want %q", got, src)
	}
}

// A long run is still wrapped to the line width, and the wrap never lands inside
// a #expression#.
func TestLongTextWithHashesWrapsOutsideExpressions(t *testing.T) {
	src := "<p>" + strings.Repeat("word ##x #some.value# more words here ", 6) + "</p>\n"
	out := formatStable(t, src)

	for line := range strings.SplitSeq(out, "\n") {
		if strings.Count(strings.ReplaceAll(line, "##", ""), "#")%2 != 0 {
			t.Errorf("a line splits an expression:\n%s\nin:\n%s", line, out)
		}
	}
}
