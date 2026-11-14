package server

import (
	"context"
	"encoding/json/v2"
	"errors"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// renameProvider is what the server advertises for rename: nothing unless
// features.rename is on, and then with the check-first request, so an editor
// asks whether the name can be renamed before it offers a box to type in.
func (s *Server) renameProvider() protocol.RenameProvider {
	if !s.Features.Rename {
		return protocol.Boolean(false)
	}

	prepare := true

	return &protocol.RenameOptions{PrepareProvider: &prepare}
}

// reservedNames are what a variable may not be called: CFML's scopes and the
// words its statements and literals are written with.
var reservedNames = map[string]bool{
	"var": true, "function": true, "return": true, "if": true, "else": true, "for": true, "while": true,
	"do": true, "switch": true, "case": true, "default": true, "break": true, "continue": true, "try": true,
	"catch": true, "finally": true, "throw": true, "in": true, "new": true, "true": true, "false": true,
	"null": true, "import": true, "include": true, "component": true, "interface": true, "property": true,
	"local": true, "arguments": true, "variables": true, "this": true, "super": true, "application": true,
	"session": true, "request": true, "server": true, "url": true, "form": true, "cgi": true, "cookie": true,
	"client": true, "cfthread": true, "thread": true, "and": true, "or": true, "not": true, "xor": true,
	"eq": true, "neq": true, "is": true, "gt": true, "lt": true, "gte": true, "lte": true, "mod": true,
	"contains": true, "eqv": true, "imp": true,
}

// renameAt is the plan for renaming the variable at the position of params in
// docURI, with the text and what is needed to turn the plan into positions.
func (s *Server) renameAt(docURI uri.URI, pos protocol.Position) (*parser.LocalRename, string, error) {
	if !s.Features.Rename {
		return nil, "", errors.New("rename is not enabled: set features.rename")
	}

	content, ok := s.getDocument(docURI)
	if !ok {
		return nil, "", errors.New("the document is not open")
	}

	lines := parser.NewLineTable(content)

	start, ok := lines.Start(int(pos.Line))
	if !ok {
		return nil, "", errors.New("the position is outside the document")
	}

	at := start + byteCol(content, int(pos.Line), pos.Character)

	spans := parser.FunctionSpansOf(cfpath.FromURI(string(docURI)), content)

	plan, err := parser.PlanLocalRename(content, spans, at)

	return plan, content, err
}

// handlePrepareRename answers textDocument/prepareRename: the variable under
// the cursor and its name, or why it cannot be renamed.
func (s *Server) handlePrepareRename(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.PrepareRenameParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	defer s.lockDoc(params.TextDocument.URI)()

	plan, content, err := s.renameAt(params.TextDocument.URI, params.Position)
	if err != nil {
		return nil, err
	}

	// The range of the word under the cursor: the plan holds each place the
	// variable is written, and the cursor is on one of them.
	pos := rangePositioner(content)
	lines := parser.NewLineTable(content)
	start, _ := lines.Start(int(params.Position.Line))
	at := start + byteCol(content, int(params.Position.Line), params.Position.Character)

	for _, r := range plan.Ranges {
		if at >= r[0] && at <= r[1] {
			return protocol.PrepareRenamePlaceholder{
				Range:       protocol.Range{Start: pos(r[0]), End: pos(r[1])},
				Placeholder: content[r[0]:r[1]],
			}, nil
		}
	}

	return nil, errors.New("cannot rename: the cursor is not on the variable")
}

// handleRename answers textDocument/rename for a local variable: an edit for
// each place the function writes it, in this document alone.
func (s *Server) handleRename(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.RenameParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI

	defer s.lockDoc(docURI)()

	plan, content, err := s.renameAt(docURI, params.Position)
	if err != nil {
		return nil, err
	}

	if err := checkNewName(plan, params.NewName); err != nil {
		return nil, err
	}

	pos := rangePositioner(content)
	edits := make([]protocol.TextEdit, 0, len(plan.Ranges))

	for _, r := range plan.Ranges {
		edits = append(edits, protocol.TextEdit{
			Range:   protocol.Range{Start: pos(r[0]), End: pos(r[1])},
			NewText: params.NewName,
		})
	}

	return &protocol.WorkspaceEdit{Changes: map[uri.URI][]protocol.TextEdit{docURI: edits}}, nil
}

// checkNewName reports why name cannot be the new name of the variable plan
// renames: not a name, a word CFML reserves, or one the function already uses.
func checkNewName(plan *parser.LocalRename, name string) error {
	if name == "" || !lex.IsWordStart(name[0]) && name[0] != '_' && name[0] != '$' {
		return errors.New("'" + name + "' is not a valid variable name")
	}

	for i := 1; i < len(name); i++ {
		if !lex.IsWord(name[i]) && name[i] != '$' {
			return errors.New("'" + name + "' is not a valid variable name")
		}
	}

	lower := strings.ToLower(name)

	if reservedNames[lower] {
		return errors.New("'" + name + "' is reserved")
	}

	// Changing only the case of the variable's own name is a rename; any other
	// name the function declares is taken.
	if !strings.EqualFold(lower, plan.Name) && plan.Taken[lower] {
		return errors.New("the function already has an argument or variable named '" + name + "'")
	}

	return nil
}

// rangePositioner turns byte offsets of content into LSP positions, in the
// UTF-16 columns of the editor.
func rangePositioner(content string) func(int) protocol.Position {
	lines := parser.NewLineTable(content)
	cols := newColMapper(content)

	return func(offset int) protocol.Position {
		line := lines.LineOf(offset)
		start, _ := lines.Start(line)

		return protocol.Position{
			Line:      conv.Uint32(line),
			Character: cols.col(conv.Uint32(line), conv.Uint32(offset-start)),
		}
	}
}
