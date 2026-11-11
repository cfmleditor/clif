package parser

import (
	"strings"
	"testing"
)

// The comment rule is the grammar's: comments nest, an opener is "<!" and
// three or more dashes, a close three or more dashes and ">". Every reader
// goes through CFMLCommentEnd, and the scanner's comment token is the same
// span.
func TestCFMLCommentEndIsTheGrammarsRule(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		end       string // the text left after the comment
		closed    bool
	}{
		{"plain", `<!--- a ---> b`, " b", true},
		{"nested", `<!--- a <!--- b ---> <cfset x = 1> ---> c`, " c", true},
		{"html close does not close", `<!--- a --> b ---> c`, " c", true},
		{"longer dashes close", `<!--- a ----> b`, " b", true},
		{"longer dashes open", `<!---- a <!----- b ---> c ---> d`, " d", true},
		{"an opener's dashes are its own", `<!------> a ---> b`, " b", true},
		{"two dashes do not open", `<!--- a <!-- b ---> c`, " c", true},
		{"unclosed", `<!--- a <!--- b --->`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			end, closed := CFMLCommentEnd(tc.src, 0)
			if tc.src[end:] != tc.end || closed != tc.closed {
				t.Fatalf("left %q, closed %v; want %q, %v", tc.src[end:], closed, tc.end, tc.closed)
			}

			sc := NewScanner(tc.src)
			if tok := sc.Next(); tok.Kind != TokCFComment || tok.Value != tc.src[:end] {
				t.Fatalf("scanner token %v %q, want the comment %q", tok.Kind, tok.Value, tc.src[:end])
			}
		})
	}
}

// A tag after a nested comment's close is still commented out, for the tag
// walks as for the parse.
func TestATagInsideANestedCommentIsNotRead(t *testing.T) {
	src := "<!--- <!--- inner ---> <cfinclude template=\"no.cfm\"> --->\n<cfinclude template=\"yes.cfm\">"

	if got := ExtractIncludes(src); len(got) != 1 || got[0] != "yes.cfm" {
		t.Fatalf("includes %q", got)
	}

	if got := NewScanner(src).Next(); !strings.HasSuffix(got.Value, "--->") || strings.Contains(src[len(got.Value):], "no.cfm") {
		t.Fatalf("scanner comment %q", got.Value)
	}
}
