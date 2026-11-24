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

// collectImplementers finds the components whose implements attribute names
// the interface at ifaceFile, from the index's record of it: the candidates
// are the files sharing the interface's name, and each is kept only when one of
// its names resolves to ifaceFile.
func (s *Server) collectImplementers(ifaceFile string) []string {
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(ifaceFile), filepath.Ext(ifaceFile)))
	cands := s.index.ImplementersByName()[base]

	if len(cands) == 0 {
		return nil
	}

	res := s.getResolver()

	var out []string

	for _, c := range cands {
		path := cfpath.FromURI(c.URI)
		if path == "" || cfpath.SamePath(path, ifaceFile) {
			continue
		}

		for _, name := range c.Interfaces {
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

	return append(out, s.walkExtenders(queue, visited)...)
}

// walkExtenders is the breadth-first walk down `extends` from the files in
// queue, skipping those in visited (which it updates), and returns the
// components it reaches. A child counts only when its extends resolves to the
// file it extends from: a bare name shared by several components is not
// evidence.
func (s *Server) walkExtenders(queue []string, visited map[string]bool) []string {
	extenders := s.index.ExtendersByName()
	res := s.getResolver()

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

// relatedSources are the files whose calls to name are references to the same
// function as file's, along the lines a call can dispatch down or a method
// overrides up:
//   - below file: its subtypes, through `extends` and through `implements`
//     (a component implementing file when it is an interface), whether or not
//     they redeclare name. A call typed as one of them reaches this method, or
//     the override of it;
//   - above file: the ancestors, through `extends`, and the interfaces it or
//     an ancestor implements, that declare name. A call typed as one of them
//     may reach this method.
//
// Siblings are not included: a call typed as a sibling never reaches file.
func (s *Server) relatedSources(file, name string) []string {
	res := s.getResolver()
	declares := func(path string) bool {
		return slices.ContainsFunc(res.EnsureIndexed(path), func(fn *parser.FunctionDef) bool {
			return strings.EqualFold(fn.Name, name)
		})
	}

	out := s.collectTransitiveSubtypes(file, true)
	seen := map[string]bool{strings.ToLower(file): true}

	for _, sub := range out {
		seen[strings.ToLower(sub)] = true
	}

	// Up: the extends chain, and the interfaces each file on it implements
	// (an interface's own `extends` is the same chain). A file is walked
	// whether or not it declares name: a grandparent may.
	pending := []string{file}

	visit := func(path string) {
		if path == "" || seen[strings.ToLower(path)] {
			return
		}

		seen[strings.ToLower(path)] = true

		if declares(path) {
			out = append(out, path)
		}

		pending = append(pending, path)
	}

	for hops := 0; len(pending) > 0 && hops < 64; hops++ {
		cur := pending[0]
		pending = pending[1:]

		curURI := cfpath.ToURI(cur)

		if ext, _ := s.index.ExtendsForFile(curURI); ext != "" {
			visit(res.ComponentPath(ext, filepath.Dir(cur)))
		}

		for _, iface := range s.index.ImplementsForFile(curURI) {
			visit(res.ComponentPath(iface, filepath.Dir(cur)))
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
