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

// renameRequest is a rename's plan, with the document it was made in and
// where the cursor stood, so the checks and the edits are made from one reading.
type renameRequest struct {
	plan    *parser.LocalRename
	content string
	at      int
	pos     func(int) protocol.Position
}

// renameAt plans the rename of the variable at pos in docURI.
func (s *Server) renameAt(docURI uri.URI, pos protocol.Position) (*renameRequest, error) {
	if !s.Features.Rename {
		return nil, errors.New("rename is not enabled: set features.rename")
	}

	content, ok := s.getDocument(docURI)
	if !ok {
		return nil, errors.New("the document is not open")
	}

	lines := parser.NewLineTable(content)

	start, ok := lines.Start(int(pos.Line))
	if !ok {
		return nil, errors.New("the position is outside the document")
	}

	at := start + byteCol(content, int(pos.Line), pos.Character)

	spans := parser.FunctionSpansOf(cfpath.FromURI(string(docURI)), content)

	plan, err := parser.PlanLocalRename(content, spans, at)
	if err != nil {
		return nil, err
	}

	// The plan is the variable of the word at the cursor, and a word spelled
	// like it, a struct key or another object's member, is not one of its
	// places: that is a different thing than the one pointed at.
	if _, _, onVariable := plan.RangeAt(at); !onVariable {
		return nil, errors.New("cannot rename: the cursor is on a name spelled like the variable, not on the variable")
	}

	return &renameRequest{plan: plan, content: content, at: at, pos: rangePositioner(content, lines)}, nil
}

// handlePrepareRename answers textDocument/prepareRename: the variable under
// the cursor and its name, or why it cannot be renamed.
func (s *Server) handlePrepareRename(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.PrepareRenameParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	defer s.lockDoc(params.TextDocument.URI)()

	req, err := s.renameAt(params.TextDocument.URI, params.Position)
	if err != nil {
		return nil, err
	}

	start, end, _ := req.plan.RangeAt(req.at)

	return protocol.PrepareRenamePlaceholder{
		Range:       protocol.Range{Start: req.pos(start), End: req.pos(end)},
		Placeholder: req.content[start:end],
	}, nil
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

	req, err := s.renameAt(docURI, params.Position)
	if err != nil {
		return nil, err
	}

	if err := checkNewName(req.plan, params.NewName); err != nil {
		return nil, err
	}

	edits := make([]protocol.TextEdit, 0, len(req.plan.Ranges))

	for _, r := range req.plan.Ranges {
		edits = append(edits, protocol.TextEdit{
			Range:   protocol.Range{Start: req.pos(r[0]), End: req.pos(r[1])},
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

	if parser.ReservedVariableNames[lower] {
		return errors.New("'" + name + "' is reserved")
	}

	// Changing only the case of the variable's own name is a rename; any other
	// name the function writes is taken, whether it declares it or reads it
	// from outside, since the variable would shadow it.
	if !strings.EqualFold(lower, plan.Name) && plan.Taken[lower] {
		return errors.New("the function already uses '" + name + "', which the variable would shadow or be mistaken for")
	}

	return nil
}

// rangePositioner turns byte offsets of content, whose line table is lines,
// into LSP positions, in the UTF-16 columns of the editor.
func rangePositioner(content string, lines parser.LineTable) func(int) protocol.Position {
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
