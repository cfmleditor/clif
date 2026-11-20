package server

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

// Adapted from IntelliLucee's script-page, for-in and catch rename scenarios
// at 83e5e802d98d06845b220dfdd72db23da47c8be7.
func TestIntelliLuceeRenameAdditionalBindings(t *testing.T) {
	for _, tc := range []struct {
		name, source, oldName, newName string
		line                           int
	}{
		{"page.cfs", "var caption = \"Hello\";\nwriteOutput(caption);", "caption", "title", 0},
		{"Loop.cfc", "component {\n  function run(items) {\n    for (item in items) {\n      writeOutput(item);\n    }\n  }\n}", "item", "element", 2},
		{"Trap.cfc", "component {\n  function run() {\n    try {\n      work();\n    } catch (any problem) {\n      writeOutput(problem.message);\n    }\n  }\n}", "problem", "failure", 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeCFC(t, filepath.Join(root, ".clif.json"), `{"features":{"rename":true}}`)
			client := startWorkflow(t, root, false)
			doc := client.open(filepath.Join(root, tc.name), tc.source)
			col := uint32(strings.Index(strings.Split(tc.source, "\n")[tc.line], tc.oldName))
			pos := protocol.Position{Line: uint32(tc.line), Character: col + 1}

			var edit protocol.WorkspaceEdit
			client.call(protocol.MethodTextDocumentRename, protocol.RenameParams{
				TextDocument: doc, Position: pos, NewName: tc.newName,
			}, &edit)

			edits := edit.Changes[doc.URI]
			slices.SortFunc(edits, func(a, b protocol.TextEdit) int {
				if d := cmp.Compare(b.Range.Start.Line, a.Range.Start.Line); d != 0 {
					return d
				}

				return cmp.Compare(b.Range.Start.Character, a.Range.Start.Character)
			})

			got := tc.source
			for _, change := range edits {
				got = applyEdit(got, change.Range, change.NewText)
			}

			// Only the binding and its references change. In particular,
			// "items" stays "items" when the loop's "item" is renamed.
			want := tc.source
			for _, line := range []int{tc.line, tc.line + 1} {
				lines := strings.Split(want, "\n")
				lines[line] = strings.Replace(lines[line], tc.oldName, tc.newName, 1)
				want = strings.Join(lines, "\n")
			}

			if got != want {
				t.Errorf("renamed source:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}
