package server

import (
	"context"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func quickFixesFor(t *testing.T, s *Server, docURI uri.URI, diags []protocol.Diagnostic) []protocol.CodeAction {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Range:        diags[0].Range,
		Context: protocol.CodeActionContext{
			Diagnostics: diags,
			Only:        []protocol.CodeActionKind{protocol.CodeActionKindQuickFix},
		},
	})

	got, err := s.handleCodeAction(context.Background(), raw)
	if err != nil {
		t.Fatalf("handleCodeAction error: %v", err)
	}

	actions, _ := got.([]protocol.CodeAction)

	return actions
}

func TestCodeActionQuickFixDuplicateArgument(t *testing.T) {
	src := strings.Join([]string{
		`<cfcomponent>`,
		`	<cffunction name="f">`,
		`		<cfargument name="id">`,
		`		<cfargument name="id">`,
		`	</cffunction>`,
		`</cfcomponent>`,
	}, "\n")

	s := newTestServer()
	docURI := uri.File("/workspace/dup.cfc")
	openDoc(t, s, docURI, src)

	diag := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 3, Character: 2},
			End:   protocol.Position{Line: 3, Character: 24},
		},
		Code: protocol.String("duplicate-argument"),
	}

	actions := quickFixesFor(t, s, docURI, []protocol.Diagnostic{diag})
	if len(actions) != 1 || actions[0].Title != "Remove duplicate <cfargument>" {
		t.Fatalf("actions = %+v, want 1 Remove duplicate <cfargument>", actions)
	}

	edits := actions[0].Edit.Changes[docURI]
	if len(edits) != 1 || edits[0].Range.Start.Line != 3 || edits[0].Range.End.Line != 4 || edits[0].NewText != "" {
		t.Fatalf("edits = %+v, want delete line 3", edits)
	}
}

func TestCodeActionQuickFixMissingVarScriptAndTag(t *testing.T) {
	src := "component {\n\tfunction f() {\n\t\tcount = 1;\n\t}\n}\n"
	s := newTestServer()
	docURI := uri.File("/workspace/mv.cfc")
	openDoc(t, s, docURI, src)

	diag := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 7},
		},
		Code: protocol.String("MISSING_VAR"),
	}

	actions := quickFixesFor(t, s, docURI, []protocol.Diagnostic{diag})
	if len(actions) != 2 {
		t.Fatalf("script MISSING_VAR actions = %+v, want 2 (var + local.)", actions)
	}

	if actions[0].Edit.Changes[docURI][0].NewText != "var " || actions[1].Edit.Changes[docURI][0].NewText != "local." {
		t.Fatalf("unexpected edits: %+v", actions)
	}

	// Tag form: `<cfset total = 1>`.
	tagSrc := "<cfset total = 1>\n"
	tagURI := uri.File("/workspace/mv.cfm")
	openDoc(t, s, tagURI, tagSrc)

	tagDiag := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 7},
			End:   protocol.Position{Line: 0, Character: 12},
		},
		Code: protocol.String("MISSING_VAR"),
	}

	tagActions := quickFixesFor(t, s, tagURI, []protocol.Diagnostic{tagDiag})
	if len(tagActions) != 2 || tagActions[0].Edit.Changes[tagURI][0].Range.Start.Character != 7 {
		t.Fatalf("tag MISSING_VAR actions = %+v, want 2 at col 7", tagActions)
	}
}
