package server

import (
	"context"
	"encoding/json/v2"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
)

// handleInlayHint answers textDocument/inlayHint with:
//  1. A component's own dot path, `Path: (models.User)`, beside its declaration
//     (for non-Application.cfc `.cfc` files).
//  2. Parameter-name hints (`name:`) before positional call arguments whose
//     callee signature is known (built-in functions, local/component methods,
//     and `new` constructors).
//
// Opt-in (features.inlayHints): an editor that draws its own hints would
// otherwise show each twice.
func (s *Server) handleInlayHint(_ context.Context, rawParams []byte) (any, error) {
	if !s.Features.InlayHints {
		return nil, nil
	}

	var params protocol.InlayHintParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI

	// An unopened document has no text to hint, and no close to take its lock
	// away again.
	if _, open := s.getDocument(docURI); !open {
		return nil, nil
	}

	defer s.lockDoc(docURI)()

	content, ok := s.getDocument(docURI)
	if !ok {
		return nil, nil
	}

	file := cfpath.FromURI(string(docURI))

	var hints []protocol.InlayHint

	if s.Features.InlayHintsComponentPath {
		if h, ok := s.componentPathHint(content, file, params.Range); ok {
			hints = append(hints, h)
		}
	}

	if s.Features.InlayHintsParameterNames {
		hints = append(hints, s.parameterInlayHints(docURI, file, content, params.Range)...)
	}

	if hints == nil {
		return []protocol.InlayHint{}, nil
	}

	return hints, nil
}

func (s *Server) componentPathHint(content, file string, want protocol.Range) (protocol.InlayHint, bool) {
	if !strings.EqualFold(filepath.Ext(file), ".cfc") || strings.EqualFold(filepath.Base(file), "Application.cfc") {
		return protocol.InlayHint{}, false
	}

	open, end, ok := parser.ComponentDeclarationBounds(content)
	if !ok {
		return protocol.InlayHint{}, false
	}

	// A self-closed `<cfcomponent />` has no body to name.
	if content[open] == '>' && open > 0 && content[open-1] == '/' {
		return protocol.InlayHint{}, false
	}

	// One offset: the lines before it counted, and its own line's columns, not a
	// table of the whole file.
	line := strings.Count(content[:end], "\n")
	lineStart := strings.LastIndexByte(content[:end], '\n') + 1
	at := protocol.Position{Line: conv.Uint32(line), Character: conv.Uint32(lex.UTF16Len(content[lineStart:end]))}

	if !positionInRange(at, want) {
		return protocol.InlayHint{}, false
	}

	dotPath, _ := s.fileToPackageIn(file)
	if dotPath == "" {
		return protocol.InlayHint{}, false
	}

	pad := true

	return protocol.InlayHint{
		Position:    at,
		Label:       protocol.String("Path: (" + dotPath + ")"),
		PaddingLeft: &pad,
		Tooltip:     protocol.String("The dot path that names this component"),
	}, true
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
