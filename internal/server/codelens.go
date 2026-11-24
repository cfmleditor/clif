package server

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"
	"time"

	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/refs"
	"go.lsp.dev/protocol"
)

func (s *Server) codeLensProvider() *protocol.CodeLensOptions {
	if !s.References {
		return nil
	}

	return &protocol.CodeLensOptions{}
}

// handleCodeLens answers textDocument/codeLens with usage counts above each
// function declaration in the document that has known calls. All functions in
// the document share one workspace reference pass via refs.FindCallsMany.
func (s *Server) handleCodeLens(_ context.Context, rawParams []byte) (any, error) {
	if !s.References {
		return nil, nil
	}

	var params protocol.CodeLensParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI
	start := time.Now()

	defer s.lockDoc(docURI)()

	s.mu.RLock()
	pr := s.parseResults[docURI]
	s.mu.RUnlock()

	if pr == nil {
		content, ok := s.getDocument(docURI)
		if !ok {
			return nil, nil
		}

		pr = parser.ParseWithOptions(docURI, content, &parser.ParseOptions{Shallow: true})
	}

	funcs := make([]parser.FunctionDef, 0, len(pr.Funcs))
	names := make([]string, 0, len(pr.Funcs))

	for i := range pr.Funcs {
		d := &pr.Funcs[i]
		if d.Generated || d.Name == "" {
			continue
		}

		funcs = append(funcs, *d)
		names = append(names, d.Name)
	}

	if len(funcs) == 0 {
		return []protocol.CodeLens{}, nil
	}

	sourceFile := cfpath.FromURI(string(docURI))
	related := s.relatedSourcesFor(sourceFile)
	also := make(map[string][]string, len(names))

	for _, name := range names {
		also[strings.ToLower(name)] = related(name)
	}

	byName := refs.FindCallsMany(s.refsFS(), s.searchRoots(), names, sourceFile, also, s.getResolver())

	lenses := make([]protocol.CodeLens, 0, len(funcs))

	for i := range funcs {
		d := &funcs[i]
		count := len(byName[strings.ToLower(d.Name)])

		if count == 0 {
			continue
		}

		lenses = append(lenses, protocol.CodeLens{
			Range: funcDefRange(d),
			Command: protocol.Command{
				Title:   usageLensTitle(count),
				Command: "clif.findRefs",
				Arguments: []protocol.LSPAny{
					mustMarshal(d.Name),
					mustMarshal(string(docURI)),
				},
			},
		})
	}

	s.log.Debug("codeLens", cflog.Int("funcs", len(funcs)), cflog.Int("lenses", len(lenses)), cflog.Any("dur", time.Since(start)))

	return lenses, nil
}

// handleCodeLensResolve answers codeLens/resolve. Lenses are already populated
// in one shared pass during textDocument/codeLens, so resolve returns the lens
// unchanged.
func (s *Server) handleCodeLensResolve(_ context.Context, rawParams []byte) (any, error) {
	var lens protocol.CodeLens
	if err := json.Unmarshal(rawParams, &lens); err != nil {
		return nil, err
	}

	return lens, nil
}

func usageLensTitle(count int) string {
	if count == 1 {
		return "1 usage"
	}

	return fmt.Sprintf("%d usages", count)
}
