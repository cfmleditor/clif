//go:build passcount

package parser

import "testing"

// The file references are what the parse's walk recorded: asking a parse for
// them reads no token and runs no parse. Each was its own walk of the
// regions, and document links took three passes of a script component's
// tokens after the parse had read them.
func TestFileRefsReadNothingAgain(t *testing.T) {
	src := `component {
	include "top.cfm";
	function f() {
		cfmodule( template = "m.cfm" );
		cf_box( title = "x" );
		include "inner.cfm";
	}
}
`
	links := ParseWithOptions("file:///t/a.cfc", src, &ParseOptions{ExtractLinks: true})
	plain := ParseWithOptions("file:///t/a.cfc", src, nil)

	ResetScanCounts()

	if n := len(links.DocumentLinks()); n != 3 {
		t.Errorf("DocumentLinks: %d links, want 3", n)
	}

	if n := len(links.CustomTagCalls(nil)); n != 2 {
		t.Errorf("CustomTagCalls: %d calls, want 2", n)
	}

	if n := len(links.IncludeSites()); n != 2 {
		t.Errorf("IncludeSites of the links parse: %d, want 2", n)
	}

	if n := len(plain.IncludeSites()); n != 2 {
		t.Errorf("IncludeSites of a plain parse: %d, want 2", n)
	}

	if tokens, parses := ScanCounts(); tokens != 0 || parses != 0 {
		t.Errorf("answering read %d tokens and ran %d parses, want none", tokens, parses)
	}
}
