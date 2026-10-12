package resolve

import (
	"math/rand"
	"regexp"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

// The legacy expression is only a test oracle; runtime lookup uses the scanner.
func TestAssignmentScannerMatchesLegacyGrammar(t *testing.T) {
	legacy := regexp.MustCompile(`(?im)^\s*(?:<cfset\s+)?(?:var\s+)?(?:local\.|variables\.)?(\w+)\s*=\s*([^=].*?)\s*/?>?\s*;?\s*$`)
	check := func(line string) {
		t.Helper()

		old := legacy.FindStringSubmatch(line)

		name, rhs, ok := matchSourceAssignment(line)
		if ok != (old != nil) || ok && (name != old[1] || rhs != old[2]) {
			t.Fatalf("%q: scanner (%q,%q,%v), legacy %q", line, name, rhs, ok, old)
		}
	}

	for _, prefix := range []string{"", "var ", "VAR\t", "local.", "variables.", "var local.", "<cfset ", "<CFSET var variables.", "<cfſet ", "variableſ."} {
		for _, name := range []string{"x", "var", "local", "variables", "_item", "9", "x.y", "$x"} {
			for _, rhs := range []string{"", " ", "=", " ==x", "value", "value;", "value / >;", "value/>;", "'Unicode Kſ'", "fn(\"(\");", ">", "/>", ";;"} {
				check(prefix + name + " =" + rhs)
			}
		}
	}

	random := rand.New(rand.NewSource(42))
	alphabet := "abcVARlocal$<>=/; \t\r\f_'()."

	for range 5000 {
		var line strings.Builder
		for range random.Intn(70) {
			line.WriteByte(alphabet[random.Intn(len(alphabet))])
		}

		check(line.String())
	}
}

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
