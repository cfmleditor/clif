package server

import (
	"context"
	"encoding/json/v2"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// handleImplementation answers textDocument/implementation ("Go to Implementation"):
//   - On a CFC/interface method declaration: jumps to every subtype method that
//     overrides/implements it.
//   - On a CFC/interface declaration header: jumps to every subtype CFC.
func (s *Server) handleImplementation(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.ImplementationParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI
	file := cfpath.FromURI(string(docURI))

	if !strings.EqualFold(filepath.Ext(file), ".cfc") {
		return []protocol.Location{}, nil
	}

	content, ok := s.getDocument(docURI)
	if !ok {
		return []protocol.Location{}, nil
	}

	defer s.lockDoc(docURI)()

	pr := s.parseResultOf(docURI)
	if pr == nil || pr.Content != content {
		pr = parser.Parse(docURI, content)
	}

	// Check if cursor is on a declared method name.
	methodName := ""

	for i := range pr.Funcs {
		fn := &pr.Funcs[i]
		if fn.Generated || !fn.HasNameRange() {
			continue
		}

		if params.Position.Line == fn.NameLine &&
			params.Position.Character >= fn.NameCol &&
			params.Position.Character <= fn.NameEndCol {
			methodName = fn.Name

			break
		}
	}

	// If not on a method name, check if cursor is on the component/interface header.
	onComponentHeader := false

	if methodName == "" {
		if openOff, _, found := parser.ComponentDeclarationBounds(content); found {
			lines := parser.NewLineTable(content)
			if lines.LineOf(openOff) == int(params.Position.Line) {
				onComponentHeader = true
			}
		}
	}

	if methodName == "" && !onComponentHeader {
		return []protocol.Location{}, nil
	}

	subtypes := s.collectTransitiveSubtypes(file)
	if len(subtypes) == 0 {
		return []protocol.Location{}, nil
	}

	res := s.getResolver()
	locs := make([]protocol.Location, 0, len(subtypes))

	for _, subPath := range subtypes {
		subURI := cfpath.ToURI(subPath)

		if methodName == "" {
			locs = append(locs, protocol.Location{
				URI:   subURI,
				Range: s.componentDeclRange(subURI, subPath),
			})

			continue
		}

		for _, fn := range res.EnsureIndexed(subPath) {
			if fn.Generated || !strings.EqualFold(fn.Name, methodName) {
				continue
			}

			rng := protocol.Range{
				Start: protocol.Position{Line: fn.Line, Character: 0},
				End:   protocol.Position{Line: fn.Line, Character: 0},
			}
			if fn.HasNameRange() {
				rng = protocol.Range{
					Start: protocol.Position{Line: fn.NameLine, Character: fn.NameCol},
					End:   protocol.Position{Line: fn.NameLine, Character: fn.NameEndCol},
				}
			}

			locs = append(locs, protocol.Location{URI: subURI, Range: rng})

			break
		}
	}

	slices.SortFunc(locs, func(a, b protocol.Location) int {
		if c := strings.Compare(string(a.URI), string(b.URI)); c != 0 {
			return c
		}

		if a.Range.Start.Line != b.Range.Start.Line {
			return int(a.Range.Start.Line) - int(b.Range.Start.Line)
		}

		return int(a.Range.Start.Character) - int(b.Range.Start.Character)
	})

	return locs, nil
}

// collectTransitiveSubtypes performs a cycle-guarded BFS over Index.ExtendersByName()
// to find all direct and transitive subtypes of rootFile.
func (s *Server) collectTransitiveSubtypes(rootFile string) []string {
	extenders := s.index.ExtendersByName()
	if len(extenders) == 0 {
		return nil
	}

	res := s.getResolver()
	visited := map[string]bool{strings.ToLower(rootFile): true}
	queue := []string{rootFile}

	var out []string

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		compName := strings.TrimSuffix(filepath.Base(cur), filepath.Ext(cur))
		for _, child := range extenders[strings.ToLower(compName)] {
			childPath := cfpath.FromURI(child.URI)
			key := strings.ToLower(childPath)

			if visited[key] {
				continue
			}

			// Verify child's extends actually resolves to cur when resolvable.
			resolvedParent := res.ComponentPath(child.Extends, filepath.Dir(childPath))
			if resolvedParent != "" && !strings.EqualFold(resolvedParent, cur) {
				continue
			}

			visited[key] = true

			out = append(out, childPath)
			queue = append(queue, childPath)
		}
	}

	return out
}

func (s *Server) componentDeclRange(docURI uri.URI, file string) protocol.Range {
	content, ok := s.getDocument(docURI)
	if !ok {
		if data, err := s.FS.ReadFile(file); err == nil {
			content = string(data)
		}
	}

	if openOff, _, found := parser.ComponentDeclarationBounds(content); found {
		lines := parser.NewLineTable(content)
		uLine := conv.Uint32(lines.LineOf(openOff))

		return protocol.Range{
			Start: protocol.Position{Line: uLine, Character: 0},
			End:   protocol.Position{Line: uLine, Character: 0},
		}
	}

	return protocol.Range{}
}
