package server

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

func TestVariableReferencesAdditionalCases(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, filepath.Join(root, ".clif.json"), `{"references":{"enabled":true}}`)

	t.Run("includeDeclaration true includes declaration and from-use lookup works", func(t *testing.T) {
		client := startWorkflow(t, root, false)
		src := "component {\n  function tally() {\n    var count = 0;\n    count = count + 1;\n    return local.count;\n  }\n}"
		doc := client.open(filepath.Join(root, "IncludeDecl.cfc"), src)

		// Ask from `local.count` on line 4.
		col := uint32(strings.Index(strings.Split(src, "\n")[4], "count"))

		var withDecl []protocol.Location
		client.call(protocol.MethodTextDocumentReferences, protocol.ReferenceParams{
			TextDocument: doc,
			Position:     protocol.Position{Line: 4, Character: col + 1},
			Context:      protocol.ReferenceContext{IncludeDeclaration: true},
		}, &withDecl)

		if len(withDecl) != 4 {
			t.Fatalf("withDecl = %+v, want 4 locations (decl, lhs, rhs, local.count)", withDecl)
		}

		var withoutDecl []protocol.Location
		client.call(protocol.MethodTextDocumentReferences, protocol.ReferenceParams{
			TextDocument: doc,
			Position:     protocol.Position{Line: 4, Character: col + 1},
			Context:      protocol.ReferenceContext{IncludeDeclaration: false},
		}, &withoutDecl)

		if len(withoutDecl) != 3 {
			t.Fatalf("withoutDecl = %+v, want 3 locations", withoutDecl)
		}

		if slices.ContainsFunc(withoutDecl, func(l protocol.Location) bool { return l.Range.Start.Line == 2 }) {
			t.Errorf("withoutDecl unexpectedly contained declaration line 2: %+v", withoutDecl)
		}
	})

	t.Run("variables scope respects local shadowing while matching explicit variables.count", func(t *testing.T) {
		client := startWorkflow(t, root, false)
		src := strings.Join([]string{
			"component {",
			"  variables.count = 0;",
			"  function usePage() {",
			"    return count;",
			"  }",
			"  function shadowed() {",
			"    var count = 5;",
			"    return count + variables.count;",
			"  }",
			"}",
		}, "\n")
		doc := client.open(filepath.Join(root, "Shadow.cfc"), src)

		col := uint32(strings.Index(strings.Split(src, "\n")[1], "count"))

		var locs []protocol.Location
		client.call(protocol.MethodTextDocumentReferences, protocol.ReferenceParams{
			TextDocument: doc,
			Position:     protocol.Position{Line: 1, Character: col + 1},
			Context:      protocol.ReferenceContext{IncludeDeclaration: false},
		}, &locs)

		var gotLines []uint32
		for _, l := range locs {
			gotLines = append(gotLines, l.Range.Start.Line)
		}

		// Should match line 3 (`return count`) and line 7 (`variables.count`),
		// NOT line 6 (`var count`) or bare `count` on line 7.
		wantLines := []uint32{3, 7}
		if !slices.Equal(gotLines, wantLines) {
			t.Errorf("lines = %v, want %v (locs=%+v)", gotLines, wantLines, locs)
		}
	})

	t.Run("tag function local ignores comments, markup, and sibling function", func(t *testing.T) {
		client := startWorkflow(t, root, false)
		src := strings.Join([]string{
			"<cfcomponent>",
			"<cffunction name=\"first\">",
			"  <cfset var total = 1>",
			"  <cfoutput>#total# total</cfoutput>",
			"  <!--- total --->",
			"  <cfreturn total>",
			"</cffunction>",
			"<cffunction name=\"second\">",
			"  <cfset var total = 2>",
			"  <cfreturn total>",
			"</cffunction>",
			"</cfcomponent>",
		}, "\n")
		doc := client.open(filepath.Join(root, "TagFunc.cfc"), src)

		col := uint32(strings.Index(strings.Split(src, "\n")[2], "total"))

		var locs []protocol.Location
		client.call(protocol.MethodTextDocumentReferences, protocol.ReferenceParams{
			TextDocument: doc,
			Position:     protocol.Position{Line: 2, Character: col + 1},
			Context:      protocol.ReferenceContext{IncludeDeclaration: false},
		}, &locs)

		var gotLines []uint32
		for _, l := range locs {
			gotLines = append(gotLines, l.Range.Start.Line)
		}

		wantLines := []uint32{3, 5}
		if !slices.Equal(gotLines, wantLines) {
			t.Errorf("lines = %v, want %v (locs=%+v)", gotLines, wantLines, locs)
		}
	})

	t.Run("ignores struct keys and other object members", func(t *testing.T) {
		client := startWorkflow(t, root, false)
		src := strings.Join([]string{
			"component {",
			"  function run(o) {",
			"    var count = 1;",
			"    var s = { count: count };",
			"    o.count = count;",
			"    return count;",
			"  }",
			"}",
		}, "\n")
		doc := client.open(filepath.Join(root, "Members.cfc"), src)

		col := uint32(strings.Index(strings.Split(src, "\n")[2], "count"))

		var locs []protocol.Location
		client.call(protocol.MethodTextDocumentReferences, protocol.ReferenceParams{
			TextDocument: doc,
			Position:     protocol.Position{Line: 2, Character: col + 1},
			Context:      protocol.ReferenceContext{IncludeDeclaration: false},
		}, &locs)

		if len(locs) != 3 {
			t.Fatalf("got %d locs (%+v), want 3 (struct value, rhs of o.count, return)", len(locs), locs)
		}
	})
}
