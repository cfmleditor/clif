package server

import (
	"context"
	"encoding/json/v2"
	"path/filepath"
	"regexp"
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

	// The lock covers only the parse and the cursor test: collecting subtypes
	// below reads other files and must not hold up edits to this one.
	unlock := s.lockDoc(docURI)

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
	isInterface := false

	if header, ok := parser.ComponentOutline(content); ok {
		isInterface = header.IsInterface
	}

	if methodName == "" {
		onComponentHeader = cursorInDeclaration(content, int(params.Position.Line))
	}

	unlock()

	if methodName == "" && !onComponentHeader {
		return []protocol.Location{}, nil
	}

	subtypes := s.collectTransitiveSubtypes(file, isInterface)
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

// cursorInDeclaration reports whether line falls within the component or
// interface declaration, from its keyword to the `{` or `>` that ends it, so a
// header split over several lines answers on each of them.
func cursorInDeclaration(content string, line int) bool {
	openOff, _, found := parser.ComponentDeclarationBounds(content)
	if !found {
		return false
	}

	lines := parser.NewLineTable(content)
	first := lines.LineOf(openOff)

	if header, ok := parser.ComponentOutline(content); ok && header.NameStart <= openOff {
		first = lines.LineOf(header.NameStart)
	}

	return line >= first && line <= lines.LineOf(openOff)
}

var implementsAttr = regexp.MustCompile(`(?i)\bimplements\s*=\s*(?:"([^"]*)"|'([^']*)'|([\w.,]+))`)

// declaredImplements lists the interfaces a component's header names in its
// implements attribute.
func declaredImplements(content string) []string {
	openOff, _, found := parser.ComponentDeclarationBounds(content)
	if !found || openOff > len(content) {
		return nil
	}

	start := 0
	if header, ok := parser.ComponentOutline(content); ok && header.NameStart <= openOff {
		start = header.NameStart
	}

	m := implementsAttr.FindStringSubmatch(content[start:openOff])
	if m == nil {
		return nil
	}

	var out []string

	for name := range strings.SplitSeq(m[1]+m[2]+m[3], ",") {
		if name = strings.TrimSpace(name); name != "" {
			out = append(out, name)
		}
	}

	return out
}

// collectImplementers finds the components whose implements attribute names
// the interface at ifaceFile. The index records extends but not implements,
// so this reads each indexed component whose text mentions the word; it runs
// only for an interface.
func (s *Server) collectImplementers(ifaceFile string) []string {
	res := s.getResolver()

	var out []string

	for _, u := range s.index.ComponentFiles() {
		path := cfpath.FromURI(string(u))
		if path == "" || cfpath.SamePath(path, ifaceFile) {
			continue
		}

		content, ok := s.getDocument(u)
		if !ok {
			data, err := s.FS.ReadFile(path)
			if err != nil {
				continue
			}

			content = string(data)
		}

		if !strings.Contains(strings.ToLower(content), "implements") {
			continue
		}

		for _, name := range declaredImplements(content) {
			if target := res.ComponentPath(name, filepath.Dir(path)); target != "" && cfpath.SamePath(target, ifaceFile) {
				out = append(out, path)

				break
			}
		}
	}

	return out
}

// collectTransitiveSubtypes performs a cycle-guarded BFS over Index.ExtendersByName()
// to find all direct and transitive subtypes of rootFile. For an interface the
// components implementing it are the first level. A child counts only when its
// extends resolves to the file it extends from: a bare name shared by several
// components is not evidence.
func (s *Server) collectTransitiveSubtypes(rootFile string, isInterface bool) []string {
	extenders := s.index.ExtendersByName()

	res := s.getResolver()
	visited := map[string]bool{strings.ToLower(rootFile): true}
	queue := []string{rootFile}

	var out []string

	if isInterface {
		for _, impl := range s.collectImplementers(rootFile) {
			key := strings.ToLower(impl)
			if visited[key] {
				continue
			}

			visited[key] = true

			out = append(out, impl)
			queue = append(queue, impl)
		}
	}

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

			resolvedParent := res.ComponentPath(child.Extends, filepath.Dir(childPath))
			if resolvedParent == "" || !cfpath.SamePath(resolvedParent, cur) {
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
