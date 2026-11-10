package resolve

import (
	"slices"
	"strings"
	"testing"
)

// reachingScript is a script function with blocks, strings, comments and an
// interpolated string holding braces, repeated n times.
func reachingScript(n int) string {
	var b strings.Builder

	b.WriteString("component {\n\tfunction run( flag ) {\n\t\tvar x = new A();\n")

	for range n {
		b.WriteString("\t\tif ( flag ) {\n\t\t\tx = new B(); // a { in a comment\n\t\t} else {\n")
		b.WriteString("\t\t\tx = \"#f( '{' )# }\";\n\t\t\t/* } */\n\t\t}\n")
		b.WriteString("\t\twhile ( flag ) { x.go(); }\n")
	}

	b.WriteString("\t\treturn x;\n\t}\n}\n")

	return b.String()
}

// reachingTemplate is a tag template with branching tags, a comment and a
// <cfscript> island, repeated n times.
func reachingTemplate(n int) string {
	var b strings.Builder

	b.WriteString("<cfset x = new A()>\n")

	for range n {
		b.WriteString("<cfif flag>\n\t<cfset x = new B()>\n<cfelseif other>\n\t<cfset x = 1>\n<cfelse>\n")
		b.WriteString("\t<!--- <cfif> --->\n\t<cfloop from=\"1\" to=\"2\" index=\"i\"><cfset x.go()></cfloop>\n</cfif>\n")
		b.WriteString("<cfscript>\nif ( flag ) {\n\tx = new C();\n}\n</cfscript>\n")
	}

	b.WriteString("<cfset x.go()>\n")

	return b.String()
}

// The events kept for a scope, cut where a use's line ends, are the events a
// scan of the text up to that line gives: blockPaths reads the scope once and
// stops where it wants, and it is right only if stopping is the same as having
// scanned the shorter text.
func TestScopeBlockEventsMatchAScanOfTheShorterText(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		tag     bool
	}{
		{"script", reachingScript(4), false},
		{"template", reachingTemplate(4), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lines := strings.Count(tc.content, "\n")

			for _, scope := range []int{0, 1, 2} {
				from, ok := lineOffset(tc.content, scope)
				if !ok {
					t.Fatalf("no line %d", scope)
				}

				all := scopeBlockEvents(tc.content, from, tc.tag)

				for use := scope; use < lines; use++ {
					end, ok := lineOffset(tc.content, use+1)
					if !ok {
						end = len(tc.content)
					}

					want := blockEvents(tc.content[from:end], tc.tag)

					var got []blockEvent

					for _, ev := range all {
						if ev.offset >= end-from {
							break
						}

						got = append(got, ev)
					}

					if !slices.EqualFunc(got, want, func(a, b blockEvent) bool {
						return a.offset == b.offset && a.op == b.op && a.kind == b.kind && slices.Equal(a.ends, b.ends)
					}) {
						t.Fatalf("scope %d, use %d: kept events %v, a scan to the line gives %v", scope, use, got, want)
					}
				}
			}
		})
	}
}

// Placing assignments in their blocks costs what the use asks about, not the
// length of the scope it is in: reachingUnion is asked for each lookup of a
// name assigned more than once, and reading a long template from its start
// each time made a scan quadratic in the template's length.
func TestBlockPathsDoesNotRescanTheScope(t *testing.T) {
	allocs := func(content string, tag bool) float64 {
		lines := strings.Count(content, "\n")
		use := lines - 1
		at := []int{use - 2, 0}

		blockPaths(content, 0, use, "x", at, tag) // the scope's first use reads it

		return testing.AllocsPerRun(20, func() { blockPaths(content, 0, use, "x", at, tag) })
	}

	for _, tc := range []struct {
		name string
		text func(int) string
		tag  bool
	}{
		{"script", reachingScript, false},
		{"template", reachingTemplate, true},
	} {
		small, large := allocs(tc.text(50), tc.tag), allocs(tc.text(400), tc.tag)
		if large > small*2+4 {
			t.Errorf("%s: %.0f allocations for a scope eight times longer, against %.0f", tc.name, large, small)
		}
	}
}
