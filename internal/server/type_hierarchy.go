package server

import (
	"context"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// handlePrepareTypeHierarchy answers textDocument/prepareTypeHierarchy.
// On-demand only (zero idle or per-keystroke overhead): resolves a component
// reference under the cursor when present, otherwise returns the current .cfc.
func (s *Server) handlePrepareTypeHierarchy(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.TypeHierarchyPrepareParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI
	if _, open := s.getDocument(docURI); !open {
		return nil, nil
	}

	defer s.lockDoc(docURI)()

	content, ok := s.getDocument(docURI)
	if !ok {
		return nil, nil
	}

	file := cfpath.FromURI(string(docURI))
	baseDir := filepath.Dir(file)

	// If the cursor sits on a component reference (e.g. a variable holding a
	// CFC, a typed parameter, or a dotted path), prepare the hierarchy for that
	// target component first.
	if target := s.typeHierarchyTargetAt(docURI, content, params.Position, baseDir); target != "" {
		if item, ok := s.typeHierarchyItemForFile(target); ok {
			return []protocol.TypeHierarchyItem{item}, nil
		}
	}

	if !strings.EqualFold(filepath.Ext(file), ".cfc") {
		return nil, nil
	}

	item, ok := s.typeHierarchyItemForFile(file)
	if !ok {
		return nil, nil
	}

	return []protocol.TypeHierarchyItem{item}, nil
}

// handleTypeHierarchySupertypes answers typeHierarchy/supertypes by resolving
// the component's `extends` parent via the index and resolver.
func (s *Server) handleTypeHierarchySupertypes(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.TypeHierarchySupertypesParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	file := cfpath.FromURI(string(params.Item.URI))
	if file == "" {
		return []protocol.TypeHierarchyItem{}, nil
	}

	extends := s.extendsForFile(params.Item.URI, file)
	if extends == "" {
		return []protocol.TypeHierarchyItem{}, nil
	}

	parent := s.getResolver().ComponentPath(extends, filepath.Dir(file))
	if parent == "" || cfpath.SamePath(parent, file) {
		return []protocol.TypeHierarchyItem{}, nil
	}

	item, ok := s.typeHierarchyItemForFile(parent)
	if !ok {
		return []protocol.TypeHierarchyItem{}, nil
	}

	return []protocol.TypeHierarchyItem{item}, nil
}

// handleTypeHierarchySubtypes answers typeHierarchy/subtypes in O(candidates)
// using the index's ShapeGeneration-cached ExtendersByName map and verifying
// each candidate's resolved extends target against the requested component.
func (s *Server) handleTypeHierarchySubtypes(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.TypeHierarchySubtypesParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	file := cfpath.FromURI(string(params.Item.URI))
	if file == "" {
		return []protocol.TypeHierarchyItem{}, nil
	}

	base := strings.ToLower(strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)))
	if base == "" {
		return []protocol.TypeHierarchyItem{}, nil
	}

	candidates := s.index.ExtendersByName()[base]
	if len(candidates) == 0 {
		return []protocol.TypeHierarchyItem{}, nil
	}

	r := s.getResolver()
	out := make([]protocol.TypeHierarchyItem, 0, len(candidates))

	for _, cand := range candidates {
		candFile := cfpath.FromURI(cand.URI)
		if candFile == "" || cfpath.SamePath(candFile, file) {
			continue
		}

		parent := r.ComponentPath(cand.Extends, filepath.Dir(candFile))
		if parent == "" || !cfpath.SamePath(parent, file) {
			continue
		}

		if item, ok := s.typeHierarchyItemForFile(candFile); ok {
			out = append(out, item)
		}
	}

	slices.SortFunc(out, func(a, b protocol.TypeHierarchyItem) int {
		return strings.Compare(string(a.URI), string(b.URI))
	})

	return out, nil
}

func (s *Server) typeHierarchyTargetAt(docURI uri.URI, content string, pos protocol.Position, baseDir string) string {
	lines := parser.NewLineTable(content)
	off := positionToByteOffset(content, lines, pos)

	// Dotted component path under cursor (e.g. `extends="models.Base"` or `new models.User()`).
	if dotted := dottedIdentAt(content, off); dotted != "" {
		if target := s.getResolver().ComponentPath(dotted, baseDir); target != "" && strings.EqualFold(filepath.Ext(target), ".cfc") {
			return target
		}
	}

	// Variable holding a component (reuses ComponentOf).
	if wStart, wEnd, ok := wordSpanAt(content, off); ok {
		word := content[wStart:wEnd]
		pr := s.parseResultOf(docURI)

		if comp := s.getResolver().ComponentOf(word, pos.Line, pr, baseDir); comp != "" {
			if target := s.getResolver().ComponentPath(comp, baseDir); target != "" && strings.EqualFold(filepath.Ext(target), ".cfc") {
				return target
			}
		}
	}

	return ""
}

func dottedIdentAt(content string, off int) string {
	if content == "" {
		return ""
	}

	i := off
	if i == len(content) || (!isWordByte(content[i]) && content[i] != '.') {
		if i > 0 && (isWordByte(content[i-1]) || content[i-1] == '.') {
			i--
		} else {
			return ""
		}
	}

	start := i
	for start > 0 && (isWordByte(content[start-1]) || content[start-1] == '.') {
		start--
	}

	end := i + 1
	for end < len(content) && (isWordByte(content[end]) || content[end] == '.') {
		end++
	}

	s := strings.Trim(content[start:end], ".")
	if s == "" || !strings.Contains(s, ".") {
		return ""
	}

	return s
}

func (s *Server) extendsForFile(docURI uri.URI, file string) string {
	if ext, ok := s.index.ExtendsForFile(docURI); ok {
		return ext
	}

	if pr := s.parseResultOf(docURI); pr != nil {
		return pr.Extends
	}

	raw, err := os.ReadFile(file) //nolint:gosec // workspace CFC file resolved by ComponentPath
	if err != nil {
		return ""
	}

	pr := parser.ParseWithOptions(docURI, lex.NormalizeLineBreaks(string(raw)), &parser.ParseOptions{Shallow: true})
	s.index.SetExtends(docURI, pr.Extends)

	return pr.Extends
}

func (s *Server) typeHierarchyItemForFile(file string) (protocol.TypeHierarchyItem, bool) {
	if file == "" {
		return protocol.TypeHierarchyItem{}, false
	}

	docURI := uri.File(file)
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

	if name == "" {
		return protocol.TypeHierarchyItem{}, false
	}

	content, open := s.getDocument(docURI)
	if !open {
		if raw, err := os.ReadFile(file); err == nil { //nolint:gosec // workspace CFC file resolved by ComponentPath
			content = lex.NormalizeLineBreaks(string(raw))
		}
	}

	kind := protocol.SymbolKindClass
	declRange := protocol.Range{}

	if content != "" {
		if _, end, ok := parser.ComponentDeclarationBounds(content); ok {
			line := strings.Count(content[:end], "\n")
			lineStart := strings.LastIndexByte(content[:end], '\n') + 1
			endCol := lex.UTF16Len(content[lineStart:end])
			declRange = protocol.Range{
				Start: protocol.Position{Line: conv.Uint32(line), Character: 0},
				End:   protocol.Position{Line: conv.Uint32(line), Character: conv.Uint32(endCol)},
			}

			header := strings.ToLower(content[lineStart:end])
			if strings.Contains(header, "interface") {
				kind = protocol.SymbolKindInterface
			}
		}
	}

	var detail *string
	if dotPath, _ := s.fileToPackageIn(file); dotPath != "" {
		detail = &dotPath
	}

	return protocol.TypeHierarchyItem{
		Name:           name,
		Kind:           kind,
		Detail:         detail,
		URI:            docURI,
		Range:          declRange,
		SelectionRange: declRange,
	}, true
}
