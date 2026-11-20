package server

import (
	"context"
	"encoding/json/v2"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func (s *Server) handleDocumentLink(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.DocumentLinkParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	dt := newDefTimer()

	defer func() { dt.report(s.log, "documentLink", string(params.TextDocument.URI), 0, 0) }()

	docURI := params.TextDocument.URI

	defer s.lockDoc(docURI)()

	dt.mark("lockDoc")

	content, ok := s.getDocument(docURI)
	if !ok {
		return nil, nil
	}

	dt.mark("getDocument")

	// Use cached parse result for global-scope links; scan function bodies on demand.
	s.mu.RLock()
	pr := s.parseResults[docURI]
	s.mu.RUnlock()

	dt.mark("parseResult")

	var docLinks []parser.DocumentLink
	if pr != nil {
		// The document cache's parse records includes, not links: they are
		// read here, once per content, by a parse that reads the file's
		// references alone.
		docLinks = pr.DocumentLinks()
	} else {
		// Fallback: full scan
		docLinks = parser.ExtractLinks(content)
	}

	dt.mark("funcLinks")

	var links []protocol.DocumentLink

	cols := newColMapper(content)

	for _, dl := range docLinks {
		tip := dl.Path
		data, _ := json.Marshal(map[string]string{
			"path": dl.Path,
			"uri":  string(params.TextDocument.URI),
		})
		links = append(links, protocol.DocumentLink{
			Range: protocol.Range{
				Start: protocol.Position{Line: dl.Line, Character: cols.col(dl.Line, dl.Start)},
				End:   protocol.Position{Line: dl.Line, Character: cols.col(dl.Line, dl.End)},
			},
			Tooltip: &tip,
			Data:    protocol.LSPAny(data),
		})
	}

	// Route links are appended rather than merged into docLinks: they already
	// carry a resolved Target, so they need no resolve round trip, and a route is
	// not a file path the parser could have produced.
	dt.mark("buildLinks")

	links = append(links, s.routeLinks(content)...)

	dt.mark("routeLinks")

	// Custom tag links carry their Target too: the template depends on the
	// custom tag roots, which a link's path alone cannot say.
	links = append(links, s.customTagLinks(content, docURI)...)

	dt.mark("customTagLinks")

	return links, nil
}

func (s *Server) handleDocumentLinkResolve(_ context.Context, rawParams []byte) (any, error) {
	var link protocol.DocumentLink
	if err := json.Unmarshal(rawParams, &link); err != nil {
		return nil, err
	}

	var data map[string]string

	_ = json.Unmarshal(link.Data, &data)

	if data == nil {
		return link, nil
	}

	filePath := data["path"]
	docURI := data["uri"]

	if filePath == "" || docURI == "" {
		return link, nil
	}

	target := s.resolveTemplatePath(filePath, uri.URI(docURI))
	if target != "" {
		targetURI := cfpath.ToURI(target)
		link.Target = &targetURI
	}

	return link, nil
}

// resolveTemplatePath shares the search order for navigation and document links.
func (s *Server) resolveTemplatePath(filePath string, docURI uri.URI) string {
	baseDir := filepath.Dir(cfpath.FromURI(string(docURI)))

	// Try relative to current file
	candidate := filepath.Join(baseDir, filePath)
	if _, err := s.FS.Stat(candidate); err == nil {
		return candidate
	}

	// Try relative to Application.cfc root
	resolver := s.getResolver()
	if appDir := resolver.FindApplicationRoot(baseDir); appDir != "" {
		candidate = filepath.Join(appDir, filePath)
		if _, err := s.FS.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Try mappings — match the first path segment against mapping keys
	mappings := resolver.EffectiveMappings(baseDir)
	if len(mappings) > 0 {
		clean := strings.TrimPrefix(filePath, "/")
		if seg, rest, _ := strings.Cut(clean, "/"); seg != "" {
			for key, dir := range mappings {
				if strings.EqualFold(seg, key) {
					candidate = filepath.Join(dir, rest)
					if _, err := s.FS.Stat(candidate); err == nil {
						return candidate
					}
				}
			}
		}
	}

	// Try relative to workspace folders
	roots := s.searchRoots()
	for _, root := range roots {
		candidate = filepath.Join(root, filePath)
		if _, err := s.FS.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Last, a workspace folder named by the first segment, which is the
	// mapping a folder implies (resolve's inFolderNamed).
	for _, candidate := range cfpath.InFolderNamed(roots, filePath) {
		if _, err := s.FS.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}
