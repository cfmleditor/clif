package server

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

// Adapted from IntelliLucee's testFindUsagesVariable at
// 83e5e802d98d06845b220dfdd72db23da47c8be7, with another function as a control.
func TestIntelliLuceeVariableReferences(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, filepath.Join(root, ".clif.json"), `{"references":{"enabled":true}}`)

	for _, tc := range []struct {
		name, source string
		declLine     int
		reads        map[int]string
	}{
		{"counter.cfm", `<cfset count = 0>
<cfset count = count + 1>
<cfoutput>#count#</cfoutput>
<!--- count --->
<cfoutput>count</cfoutput>`, 0, map[int]string{1: "<cfset count = ", 2: "<cfoutput>#"}},
		{"Counter.cfc", `component {
  function tally() {
    var count = 0;
    count = count + 1;
    return "#count#";
  }
  function other() {
    var count = 10;
    return count;
  }
}`, 2, map[int]string{3: "    count = ", 4: "    return \"#"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := startWorkflow(t, root, false)
			doc := client.open(filepath.Join(root, tc.name), tc.source)
			lines := strings.Split(tc.source, "\n")
			col := uint32(strings.Index(lines[tc.declLine], "count"))

			var result []protocol.Location
			client.call(protocol.MethodTextDocumentReferences, protocol.ReferenceParams{
				TextDocument: doc,
				Position:     protocol.Position{Line: uint32(tc.declLine), Character: col + 1},
				Context:      protocol.ReferenceContext{IncludeDeclaration: false},
			}, &result)

			for line, prefix := range tc.reads {
				start := protocol.Position{Line: uint32(line), Character: uint32(len(prefix))}

				want := protocol.Range{Start: start, End: protocol.Position{Line: start.Line, Character: start.Character + 5}}
				if !slices.Contains(result, protocol.Location{URI: doc.URI, Range: want}) {
					t.Errorf("missing reference at %v in %v", want, result)
				}
			}

			for _, loc := range result {
				if _, inBinding := tc.reads[int(loc.Range.Start.Line)]; loc.URI != doc.URI || !inBinding {
					t.Errorf("returned a declaration or an unrelated occurrence: %+v", loc)
				}
			}
		})
	}
}
