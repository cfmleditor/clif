package resolve

import (
	"regexp"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

// TestSourceFactsWindowsAndRefresh: the last assignment and the query words
// read from a file's tables answer what the line reader and the expression
// they replaced did, for every window, and a changed text is read afresh.
func TestSourceFactsWindowsAndRefresh(t *testing.T) {
	r := &Resolver{}
	text := "var x = fn(\n value\n);\n// QUERY cfdbinfo cfquery; notquery query_name $query\nlocal.x = [];\nx.go();\n"
	pr := &parser.ParseResult{URI: uri.URI("file:///sample.cfc"), Content: text}
	legacyWords := regexp.MustCompile(`(?i)\b(?:query|cfquery|cfdbinfo)\b`)

	lines := strings.Split(text, "\n")
	for start := range lines {
		for header := start; header <= len(lines); header++ {
			want, at, ok := localAssignmentAt(text, "x", start, header)

			got, gotAt, gotOK := r.localAssignmentAt(pr, "x", start, header)
			if got != want || gotAt != at || gotOK != ok {
				t.Fatalf("assignment window %d:%d changed", start, header)
			}

			if header < len(lines) && header > start {
				from, to := r.sourceLineStarts(pr)[start], r.sourceLineStarts(pr)[header]-1
				if r.hasQueryOutputWord(pr, from, to) != legacyWords.MatchString(text[from:to]) {
					t.Fatalf("query window %d:%d changed", start, header)
				}
			}
		}
	}

	first := r.lexicalSource(pr)

	if allocs := testing.AllocsPerRun(10, func() { r.localAssignmentAt(pr, "x", 0, len(lines)) }); allocs > 1 {
		t.Fatalf("repeated lookup allocated %g times", allocs)
	}

	if r.lexicalSource(pr) != first {
		t.Fatal("unchanged source was scanned again")
	}

	changed := &parser.ParseResult{URI: pr.URI, Content: "x = 2;"}
	if got, _, ok := r.localAssignmentAt(changed, "x", 0, 1); !ok || got != "2" {
		t.Fatalf("stale assignment: %q", got)
	}

	if r.hasQueryOutputWord(changed, 0, len(changed.Content)) {
		t.Fatal("stale query word")
	}
}
