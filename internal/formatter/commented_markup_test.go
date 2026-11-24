package formatter

import "testing"

// A comment that holds tags or HTML is code someone switched off, and is meant to be
// switched back on: the editor's toggle-comment shortcut removes the markers and the
// block should be exactly where it was. The formatter moved the markers onto lines of
// their own and re-indented the inside, and because the first line sits after
// `<!---` while the rest sit at their own depth, the common indent it stripped came
// out of the first line's alone and left the rest ragged.
func TestCommentedOutMarkupIsLeftAsWritten(t *testing.T) {
	for name, block := range map[string]string{
		"html in a list":    "<!--- <li>\n        Purple\n    </li> --->",
		"html at the top":   "<!--- <p>\n    Purple\n</p> --->",
		"cf tag in a div":   "<!--- <cfif color EQ 'purple'>\n        Purple\n    </cfif> --->",
		"ragged on purpose": "<!---\n<div>\n          deep\n  </div>\n--->",
	} {
		t.Run(name, func(t *testing.T) {
			src := "<ul>\n    " + block + "\n    <li>\n        Blue\n    </li>\n</ul>\n"

			out := formatStable(t, src)
			assertContains(t, out, block)
		})
	}
}

func TestCommentedOutMarkupSurvivesNesting(t *testing.T) {
	block := "<!--- <cfif color EQ 'purple'>\n        Purple\n    </cfif> --->"
	src := "<div>\n    <div>\n        <div>\n            " + block + "\n        </div>\n    </div>\n</div>\n"

	assertContains(t, formatStable(t, src), block)
}

// Only a comment that holds markup is left alone. Prose is still tidied: its text is
// indented one level under the markers, whatever depth it was typed at.
func TestMultiLineProseCommentIsStillNormalised(t *testing.T) {
	src := "<ul>\n    <!---\n  first line of prose\n    second line\n    --->\n    <li>Blue</li>\n</ul>\n"
	want := "<ul>\n    <!---\n        first line of prose\n          second line\n    --->\n    <li>Blue</li>\n</ul>\n"

	if got := formatStable(t, src); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// Text that merely contains a less-than sign is not markup.
func TestLessThanInProseIsNotMarkup(t *testing.T) {
	src := "<ul>\n    <!---\n  when a < b the\n    second line\n    --->\n    <li>Blue</li>\n</ul>\n"
	want := "<ul>\n    <!---\n        when a < b the\n          second line\n    --->\n    <li>Blue</li>\n</ul>\n"

	if got := formatStable(t, src); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// Prose that starts on the marker's line. The first line sits after `<!---` at a depth
// of its own, so it is not part of the indentation the rest of the comment shares:
// counting it left the common indent at the one space after the marker and the other
// lines came out as deep as they were typed, ragged against the first.
func TestProseStartingOnTheMarkerLineIsAligned(t *testing.T) {
	for name, tc := range map[string]struct{ src, want string }{
		"text lined up under the first": {
			"<ul>\n    <!--- first line of prose\n             second line\n    --->\n    <li>Blue</li>\n</ul>\n",
			"<ul>\n    <!---\n        first line of prose\n        second line\n    --->\n    <li>Blue</li>\n</ul>\n",
		},
		"closing marker on the last line": {
			"<ul>\n    <!--- first line of prose\n             second line --->\n    <li>Blue</li>\n</ul>\n",
			"<ul>\n    <!---\n        first line of prose\n        second line\n    --->\n    <li>Blue</li>\n</ul>\n",
		},
		"indentation among the rest is kept": {
			"<div>\n    <!--- Items:\n            - one\n              continued\n            - two\n    --->\n</div>\n",
			"<div>\n    <!---\n        Items:\n        - one\n          continued\n        - two\n    --->\n</div>\n",
		},
		"less-than sign in the prose": {
			"<ul>\n    <!--- when a < b the\n          second line\n    --->\n    <li>Blue</li>\n</ul>\n",
			"<ul>\n    <!---\n        when a < b the\n        second line\n    --->\n    <li>Blue</li>\n</ul>\n",
		},
		"at the top of a document": {
			"<!--- first\n      second\n--->\n<p>x</p>\n",
			"<!---\n    first\n    second\n--->\n<p>x</p>\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := formatStable(t, tc.src); got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}
