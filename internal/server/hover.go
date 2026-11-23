package server

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func (s *Server) handleHover(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.HoverParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	defer s.lockDoc(params.TextDocument.URI)()

	content, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	line := int(params.Position.Line)
	char := byteCol(content, line, params.Position.Character)

	word := parser.WordAtPosition(content, line, char)
	if word == "" {
		return nil, nil
	}

	// Builtin function
	// User-defined function via qualifier (e.g. service.getMethod) — check first
	docURI := params.TextDocument.URI
	if qualifier := parser.QualifierBeforeWord(content, line, char); qualifier != "" {
		if def := s.resolveUserFunc(qualifier, word, docURI, conv.Uint32(line)); def != nil {
			value := def.FormatHover()
			if d := stubDoc(def); d != nil {
				value += "\n\n" + d.Markdown()
			}

			return &protocol.Hover{
				Contents: &protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: value,
				},
			}, nil
		}

		if !s.GlobalFunctionResolution || isComponentThis(qualifier, docURI) {
			return nil, nil
		}
	}

	// Builtin function
	if e, ok := docs.LookupFunction(word); ok {
		previews := e.FunctionPreviews()
		value := "```cfs\n" + strings.Join(previews, "\n```\n\n```cfs\n") + "\n```\n\n" + e.Doc()

		return &protocol.Hover{
			Contents: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: value,
			},
		}, nil
	}

	// Builtin tag
	if e, ok := docs.LookupTag(word); ok {
		return &protocol.Hover{
			Contents: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: fmt.Sprintf("```cfml\n<%s>\n```\n\n%s", e.Name, e.Doc()),
			},
		}, nil
	}

	// User-defined function in current file or index (unqualified)
	match, inFile, total := s.index.LookupPreferred(word, docURI)
	if total > 0 {
		// Only show if in current file or (global resolution enabled + exactly one match)
		var def *parser.FunctionDef

		if inFile || (s.GlobalFunctionResolution && total == 1) {
			def = match
		}

		if def != nil {
			return &protocol.Hover{
				Contents: &protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: def.FormatHover(),
				},
			}, nil
		}
	}

	return nil, nil
}

func (s *Server) resolveUserFunc(qualifier, funcName string, docURI uri.URI, line uint32) *parser.FunctionDef {
	key := fmt.Sprintf("%s:%d:%s.%s", docURI, line, qualifier, funcName)

	s.mu.RLock()

	if s.lastResolveKey == key {
		def := s.lastResolveDef
		s.mu.RUnlock()

		return def
	}

	s.mu.RUnlock()

	def := s.doResolveUserFunc(qualifier, funcName, docURI, line)

	s.mu.Lock()
	s.lastResolveKey = key
	s.lastResolveDef = def
	s.mu.Unlock()

	return def
}

func isComponentThis(qualifier string, docURI uri.URI) bool {
	return strings.EqualFold(qualifier, "this") && strings.EqualFold(filepath.Ext(docURI.Path()), ".cfc")
}

func (s *Server) doResolveUserFunc(qualifier, funcName string, docURI uri.URI, line uint32) *parser.FunctionDef {
	var comp string

	switch {
	case isComponentThis(qualifier, docURI):
		if def := s.index.FunctionInFile(docURI, funcName); def != nil {
			return def
		}

		comp = cfpath.FromURI(string(docURI))
	case strings.EqualFold(qualifier, "super"):
		// Resolve from parent component
		s.mu.RLock()
		pr := s.parseResults[docURI]
		s.mu.RUnlock()

		if pr == nil || pr.Extends == "" {
			return nil
		}

		comp = pr.Extends
	case strings.HasPrefix(qualifier, "~?"):
		comp = s.cfResolverSet().Resolve(qualifier[2:])
	case strings.HasPrefix(qualifier, "~"):
		comp = qualifier[1:]
	default:
		s.ensureFuncRefsIndexed(docURI, int(line))
		ref := s.index.LookupComponentRefInFile(qualifier, docURI, line)

		if ref == nil {
			if name, _, _ := parser.MemberReceiverName(qualifier); name != "" && name != qualifier {
				ref = s.index.LookupComponentRefInFile(name, docURI, line)
			}
		}

		switch {
		case ref != nil:
			comp = ref.Component
		case s.componentOfAt(docURI, qualifier, line) != "":
			comp = s.componentOfAt(docURI, qualifier, line)
		default:
			comp = s.cfResolverSet().Resolve(qualifier)
		}
	}

	if comp == "" {
		return nil
	}

	currentPath := docURI.Path()
	baseDir := filepath.Dir(currentPath)

	return s.getResolver().ResolveFunc(comp, funcName, baseDir)
}

func (s *Server) componentOfAt(docURI uri.URI, variable string, line uint32) string {
	s.mu.RLock()
	pr := s.parseResults[docURI]
	s.mu.RUnlock()

	if pr == nil {
		return ""
	}

	return s.getResolver().ComponentOf(variable, line, pr, filepath.Dir(docURI.Path()))
}
