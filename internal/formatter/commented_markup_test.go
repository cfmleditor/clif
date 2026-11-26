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

// A commented-out block is returned as written, but the comment as a whole still sits where
// the formatter puts it. When that depth is not the one it was typed at, the first line moves
// and the lines after it must move with it, or the block comes out ragged: toggling the
// comment off then leaves its tags misaligned, which is what returning it as written was
// meant to avoid.
func TestCommentedOutMarkupMovesAsAUnit(t *testing.T) {
	// The block is typed at column 0 and lands one level in; the same block typed two levels
	// deep and brought out to one level lands the same way, whatever its inside was indented by.
	want := "<div>\n    <p>x</p>\n    <!--- <li>\n            Purple\n        </li> --->\n</div>\n"

	for name, src := range map[string]string{
		"typed shallower than it lands": "<div>\n<p>x</p>\n<!--- <li>\n        Purple\n    </li> --->\n</div>\n",
		"typed deeper than it lands":    "<div>\n        <p>x</p>\n        <!--- <li>\n                Purple\n            </li> --->\n</div>\n",
	} {
		t.Run(name, func(t *testing.T) {
			if got := formatStable(t, src); got != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestCommentedOutMarkupKeepsBlankLinesAndShorterLines(t *testing.T) {
	// A blank line stays empty rather than gaining the indent, and a line indented less than the
	// comment's own line is not one that could move with it, so it is left as written.
	src := "<div>\n        <!--- <li>\n\nPurple\n    </li> --->\n        <p>x</p>\n</div>\n"
	want := "<div>\n    <!--- <li>\n\nPurple\n    </li> --->\n    <p>x</p>\n</div>\n"

	if got := formatStable(t, src); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestCommentedOutMarkupInsideANestedTag(t *testing.T) {
	// Each block moves by what its own first line moves by.
	src := "<div>\n<div>\n<!--- <b>\n  x\n</b> --->\n</div>\n<!--- <i>\n  y\n</i> --->\n</div>\n"
	want := "<div>\n    <div>\n        <!--- <b>\n          x\n        </b> --->\n    </div>\n    <!--- <i>\n      y\n    </i> --->\n</div>\n"

	if got := formatStable(t, src); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// The same, with the tab indentation the formatter uses by default.
func TestCommentedOutMarkupMovesAsAUnitWithTabs(t *testing.T) {
	opts := testOpts()
	opts.UseTabs = true

	src := "<div>\n<p>x</p>\n<!--- <li>\n        Purple\n    </li> --->\n</div>\n"
	want := "<div>\n\t<p>x</p>\n\t<!--- <li>\n\t        Purple\n\t    </li> --->\n</div>\n"

	got := mustFormatWith(t, src, &opts)
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}

	if again := mustFormatWith(t, got, &opts); again != got {
		t.Errorf("not idempotent:\n%q\n%q", got, again)
	}
}

// The comments inside a <cfquery> body, when query formatting is on. A comment inside an
// expression, and one between an operator and its operand, are written by different code.
func TestCommentedOutMarkupInAQueryMovesAsAUnit(t *testing.T) {
	for name, tc := range map[string]struct{ src, want string }{
		"inside an expression": {
			"<div>\n<cfquery name=\"q\">\nSELECT * FROM t WHERE a = <!--- <b>\n    x\n  </b> ---> 1\n</cfquery>\n</div>\n",
			"\n            <!--- <b>\n                x\n              </b> --->\n",
		},
		"after an operator": {
			"<div>\n<cfquery name=\"q\">\nSELECT * FROM t WHERE a + <!--- <b>\n    x\n  </b> ---> 1 > 0\n</cfquery>\n</div>\n",
			"\n        <!--- <b>\n            x\n          </b> --->\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assertContains(t, formatStable(t, tc.src), tc.want)
		})
	}
}

// A comment in a query that holds no markup is written as it always was.
func TestProseCommentInAQueryIsNotMoved(t *testing.T) {
	src := "<div>\n<cfquery name=\"q\">\nSELECT * FROM t WHERE a + <!--- first\n    second\n  third ---> 1 > 0\n</cfquery>\n</div>\n"

	assertContains(t, formatStable(t, src), "\n        <!--- first\n    second\n  third --->\n")
}
