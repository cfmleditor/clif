package server

import (
	"context"
	"encoding/json/v2"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
)

// handleInlayHint answers textDocument/inlayHint: a component's own dot path,
// `Path: (models.User)`, beside its declaration, where an editor can show what
// to write to refer to it. Opt-in (features.inlayHints): it is new, and an
// editor that draws its own hints would show each twice.
//
// Only a .cfc that is not an Application.cfc has one: that is the file
// `new` and `createObject` name, and an application's root has no dot path a
// call would use. The path is the one clif.copyPackage answers.
func (s *Server) handleInlayHint(_ context.Context, rawParams []byte) (any, error) {
	if !s.Features.InlayHints {
		return nil, nil
	}

	var params protocol.InlayHintParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI
	file := cfpath.FromURI(string(docURI))

	if !strings.EqualFold(filepath.Ext(file), ".cfc") || strings.EqualFold(filepath.Base(file), "Application.cfc") {
		return nil, nil
	}

	defer s.lockDocOnly(docURI)()

	content, ok := s.getDocument(docURI)
	if !ok {
		return nil, nil
	}

	end, ok := parser.ComponentDeclarationEnd(content)
	if !ok {
		return nil, nil
	}

	// Before the brace, and before the blanks leading up to it on its line.
	for end > 0 && (content[end-1] == ' ' || content[end-1] == '\t') {
		end--
	}

	lines := parser.NewLineTable(content)
	line := lines.LineOf(end)
	start, _ := lines.Start(line)
	at := protocol.Position{Line: conv.Uint32(line), Character: newColMapper(content).col(conv.Uint32(line), conv.Uint32(end-start))}

	if !positionInRange(at, params.Range) {
		return []protocol.InlayHint{}, nil
	}

	dotPath := s.fileToPackage(file)
	if dotPath == "" {
		return []protocol.InlayHint{}, nil
	}

	pad := true

	return []protocol.InlayHint{{
		Position:    at,
		Label:       protocol.String("Path: (" + dotPath + ")"),
		PaddingLeft: &pad,
		Tooltip:     protocol.String("The dot path that names this component"),
	}}, nil
}

// positionInRange reports whether p lies within r, its ends included.
func positionInRange(p protocol.Position, r protocol.Range) bool {
	if p.Line < r.Start.Line || p.Line > r.End.Line {
		return false
	}

	if p.Line == r.Start.Line && p.Character < r.Start.Character {
		return false
	}

	return p.Line != r.End.Line || p.Character <= r.End.Character
}
