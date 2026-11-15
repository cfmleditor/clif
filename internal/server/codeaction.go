package server

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
)

func (s *Server) handleCodeAction(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.CodeActionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	if _, open := s.getDocument(params.TextDocument.URI); !open {
		return nil, nil
	}

	defer s.lockDocOnly(params.TextDocument.URI)()

	content, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	line := int(params.Range.Start.Line)
	char := byteCol(content, line, params.Range.Start.Character)
	docURI := string(params.TextDocument.URI)

	var actions []protocol.CodeAction

	// An Application.cfc is offered the application events it does not declare.
	// They are refactorings: a client asking only for quick fixes has no use
	// for the scan that finds them.
	if wantsKind(params.Context.Only, protocol.CodeActionKindRefactor) {
		u := params.TextDocument.URI
		for _, action := range s.lifecycleActions(u, content, params.Range) {
			action.Edit = s.documentEdit(u, action.Edit.Changes[u])
			actions = append(actions, action)
		}
	}

	// Offered only on a line that looks as if it holds a call, so it is not
	// noise on every line of the file; the cursor need not be on the call
	// itself. See lineMayHoldCall for why this is not a parse.
	if lineMayHoldCall(content, line) {
		title := fmt.Sprintf("Explain call resolution on line %d", line+1)
		actions = append(actions, protocol.CodeAction{
			Title: title,

			Command: protocol.Command{
				Title:     title,
				Command:   "clif.explainCall",
				Arguments: lspAnyArgs(docURI, line),
			},
		})
	}

	// The file and workspace actions come last, and wherever the cursor is:
	// they are about the file or the project, not the word under the cursor. A
	// client with no way to run a server command of its own, such as Zed
	// before its LSP command picker, reaches them only from here.
	general := append(fileActions(docURI), workspaceActions()...)

	word := parser.WordAtPosition(content, line, char)
	if word == "" {
		return append(actions, general...), nil
	}

	// If on a function name with a qualifier, offer component-level actions
	if qualifier := parser.QualifierBeforeWord(content, line, char); qualifier != "" {
		actions = append(actions, protocol.CodeAction{
			Title: "Find all references to " + word,

			Command: protocol.Command{
				Title:     "Find all references to " + word,
				Command:   "clif.findRefs",
				Arguments: lspAnyArgs(word, docURI),
			},
		}, protocol.CodeAction{
			Title: "Export dependency graph for " + qualifier + "." + word,

			Command: protocol.Command{
				Title:     "Export dependency graph for " + qualifier + "." + word,
				Command:   "clif.exportDeps",
				Arguments: lspAnyArgs(docURI, word),
			},
		})
	} else {
		// Unqualified — offer function-level actions
		actions = append(actions, protocol.CodeAction{
			Title: "Find all calls to " + word,

			Command: protocol.Command{
				Title:     "Find all calls to " + word,
				Command:   "clif.findRefs",
				Arguments: lspAnyArgs(word, docURI),
			},
		}, protocol.CodeAction{
			Title: "Export dependency graph for " + word,

			Command: protocol.Command{
				Title:     "Export dependency graph for " + word,
				Command:   "clif.exportDeps",
				Arguments: lspAnyArgs(docURI, word),
			},
		})
	}

	// Writing the report is its own action, because the two above are ordinary
	// editor gestures and this one puts files in the user's source tree. It is
	// also the only way to reach the export from a client that cannot pass
	// command arguments of its own.
	actions = append(actions, protocol.CodeAction{
		Title: "Export references to " + word + " to a file",

		Command: protocol.Command{
			Title:     "Export references to " + word + " to a file",
			Command:   "clif.findRefs",
			Arguments: lspAnyArgs(word, docURI, true),
		},
	})

	return append(actions, general...), nil
}

// fileActions offers what is about the whole file. The dependency graph here
// is the file-level form of clif.exportDeps, which takes the document
// URI alone; the word actions above pass a function name as well.
func fileActions(docURI string) []protocol.CodeAction {
	title := "Export dependency graph for " + filepath.Base(cfpath.FromURI(docURI))

	return []protocol.CodeAction{{
		Title: title,

		Command: protocol.Command{
			Title:     title,
			Command:   "clif.exportDeps",
			Arguments: lspAnyArgs(docURI),
		},
	}}
}

// workspaceActions offers the commands that walk the whole workspace: the
// parse-error scan, which publishes its findings as diagnostics, and the
// generated known-issues reports, which write their files beside
// .clif.json (see handleExport).
func workspaceActions() []protocol.CodeAction {
	cmds := []struct{ title, command string }{
		{"Scan workspace for parse errors", "clif.scanWorkspace"},
		{"Export unresolved calls report for the workspace", "clif.exportUnresolved"},
		{"Export CFLint report for the workspace", "clif.exportCFLint"},
	}

	out := make([]protocol.CodeAction, 0, len(cmds))

	for _, r := range cmds {
		out = append(out, protocol.CodeAction{
			Title:   r.title,
			Command: protocol.Command{Title: r.title, Command: r.command},
		})
	}

	return out
}

// lspAnyArgs marshals each argument to a protocol.LSPAny for use in a Command's Arguments field.
func lspAnyArgs(args ...any) []protocol.LSPAny {
	out := make([]protocol.LSPAny, len(args))

	for i, a := range args {
		b, _ := json.Marshal(a)
		out[i] = protocol.LSPAny(b)
	}

	return out
}

// wantsKind reports whether a request limited to only (every kind when empty)
// can be answered with an action of kind: it is one of them, or one of them
// is its parent, `refactor` for `refactor.rewrite`.
func wantsKind(only []protocol.CodeActionKind, kind protocol.CodeActionKind) bool {
	if len(only) == 0 {
		return true
	}

	for _, o := range only {
		if o == "" || o == kind || strings.HasPrefix(string(kind), string(o)+".") {
			return true
		}
	}

	return false
}
