package server

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/refs"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type crossFileTargetKind uint8

const (
	crossFileMethod crossFileTargetKind = iota + 1
	crossFileProperty
)

type crossFileRenameTarget struct {
	kind     crossFileTargetKind
	name     string
	declRng  protocol.Range
	pr       *parser.ParseResult
	content  string
	filePath string
}

func (s *Server) crossFileTargetAt(docURI uri.URI, pos protocol.Position) (*crossFileRenameTarget, bool) {
	file := cfpath.FromURI(string(docURI))
	if !strings.EqualFold(filepath.Ext(file), ".cfc") {
		return nil, false
	}

	content, ok := s.getDocument(docURI)
	if !ok {
		return nil, false
	}

	pr := s.parseResultOf(docURI)
	if pr == nil || pr.Content != content {
		pr = parser.Parse(docURI, content)
	}

	// 1. Check declared method names.
	for i := range pr.Funcs {
		fn := &pr.Funcs[i]
		if fn.Generated || !fn.HasNameRange() {
			continue
		}

		if pos.Line == fn.NameLine && pos.Character >= fn.NameCol && pos.Character <= fn.NameEndCol {
			return &crossFileRenameTarget{
				kind: crossFileMethod,
				name: fn.Name,
				declRng: protocol.Range{
					Start: protocol.Position{Line: fn.NameLine, Character: fn.NameCol},
					End:   protocol.Position{Line: fn.NameLine, Character: fn.NameEndCol},
				},
				pr:       pr,
				content:  content,
				filePath: file,
			}, true
		}
	}

	// 2. Check declared property names.
	lines := strings.Split(content, "\n")

	for _, p := range pr.PropertySymbols() {
		if p.Line != pos.Line || int(p.Line) >= len(lines) {
			continue
		}

		if startCol, endCol, found := findPropNameCols(lines[p.Line], p.Name); found {
			if pos.Character >= startCol && pos.Character <= endCol {
				return &crossFileRenameTarget{
					kind: crossFileProperty,
					name: p.Name,
					declRng: protocol.Range{
						Start: protocol.Position{Line: p.Line, Character: startCol},
						End:   protocol.Position{Line: p.Line, Character: endCol},
					},
					pr:       pr,
					content:  content,
					filePath: file,
				}, true
			}
		}
	}

	return nil, false
}

func findPropNameCols(lineText, name string) (startCol, endCol uint32, ok bool) {
	if name == "" {
		return 0, 0, false
	}

	// 1. Prefer explicit `name="foo"` or `name='foo'` attribute value.
	if s, e, found := findTagAttrValue(lineText, 0, len(lineText), "name"); found && strings.EqualFold(lineText[s:e], name) {
		return conv.Uint32(lex.UTF16Len(lineText[:s])), conv.Uint32(lex.UTF16Len(lineText[:e])), true
	}

	// 2. Shorthand `property [type] <name> [= default];`: take the last whole-word
	// match before any `=`, `;`, or `{`, so `property user user;` selects the
	// property name rather than the type.
	limit := len(lineText)
	if idx := strings.IndexAny(lineText, "=;{"); idx >= 0 {
		limit = idx
	}

	lowerLine := strings.ToLower(lineText[:limit])
	lowerName := strings.ToLower(name)

	lastPos := -1
	from := 0

	for {
		idx := strings.Index(lowerLine[from:], lowerName)
		if idx < 0 {
			break
		}

		pos := from + idx
		end := pos + len(name)

		beforeOK := pos == 0 || !isSemIdentByte(lineText[pos-1])
		afterOK := end >= limit || !isSemIdentByte(lineText[end])

		if beforeOK && afterOK {
			lastPos = pos
		}

		from = pos + 1
	}

	if lastPos < 0 {
		return 0, 0, false
	}

	return conv.Uint32(lex.UTF16Len(lineText[:lastPos])), conv.Uint32(lex.UTF16Len(lineText[:lastPos+len(name)])), true
}

func (s *Server) prepareCrossFileRename(docURI uri.URI, pos protocol.Position) (protocol.PrepareRenamePlaceholder, bool) {
	t, ok := s.crossFileTargetAt(docURI, pos)
	if !ok {
		return protocol.PrepareRenamePlaceholder{}, false
	}

	return protocol.PrepareRenamePlaceholder{
		Range:       t.declRng,
		Placeholder: t.name,
	}, true
}

func (s *Server) performCrossFileRename(docURI uri.URI, pos protocol.Position, newName string) (*protocol.WorkspaceEdit, error) {
	t, ok := s.crossFileTargetAt(docURI, pos)
	if !ok {
		return nil, parser.ErrNoVariable
	}

	if err := checkCrossFileIdentifier(newName); err != nil {
		return nil, err
	}

	switch t.kind {
	case crossFileMethod:
		return s.renameComponentMethod(docURI, t, newName)
	case crossFileProperty:
		return s.renameComponentProperty(docURI, t, newName)
	default:
		return nil, parser.ErrNoVariable
	}
}

func checkCrossFileIdentifier(name string) error {
	if name == "" || (!lex.IsWordStart(name[0]) && name[0] != '_' && name[0] != '$') {
		return errors.New("'" + name + "' is not a valid CFML identifier")
	}

	for i := 1; i < len(name); i++ {
		if !lex.IsWord(name[i]) && name[i] != '$' {
			return errors.New("'" + name + "' is not a valid CFML identifier")
		}
	}

	if parser.ClassifyVariableName(name) == parser.NameReserved {
		return errors.New("'" + name + "' is reserved")
	}

	return nil
}

func (s *Server) renameComponentMethod(docURI uri.URI, t *crossFileRenameTarget, newName string) (*protocol.WorkspaceEdit, error) {
	// Refuse collision with an existing method in the same CFC (unless case-only change).
	if !strings.EqualFold(t.name, newName) {
		for i := range t.pr.Funcs {
			if strings.EqualFold(t.pr.Funcs[i].Name, newName) {
				return nil, fmt.Errorf("component already declares a method named '%s'", newName)
			}
		}
	}

	res := s.getResolver()

	// Refuse if any ancestor component (parent, grandparent, ...) or direct
	// subtype declares the same method (renaming one side would silently break
	// polymorphism).
	visited := map[string]bool{strings.ToLower(t.filePath): true}
	curDir := filepath.Dir(t.filePath)
	curExtends := t.pr.Extends

	for curExtends != "" {
		parentPath := res.ComponentPath(curExtends, curDir)
		if parentPath == "" {
			break
		}

		key := strings.ToLower(parentPath)
		if visited[key] {
			break
		}

		visited[key] = true

		for _, fn := range res.EnsureIndexed(parentPath) {
			if strings.EqualFold(fn.Name, t.name) {
				return nil, fmt.Errorf("cannot rename '%s': overrides method in ancestor component '%s'", t.name, curExtends)
			}
		}

		parentExtends, _ := s.index.ExtendsForFile(cfpath.ToURI(parentPath))
		curDir = filepath.Dir(parentPath)
		curExtends = parentExtends
	}

	compName := strings.TrimSuffix(filepath.Base(t.filePath), filepath.Ext(t.filePath))
	for _, child := range s.index.ExtendersByName()[strings.ToLower(compName)] {
		childPath := cfpath.FromURI(child.URI)
		if strings.EqualFold(childPath, t.filePath) {
			continue
		}

		for _, fn := range res.EnsureIndexed(childPath) {
			if strings.EqualFold(fn.Name, t.name) {
				return nil, fmt.Errorf("cannot rename '%s': overridden in child component '%s'", t.name, filepath.Base(childPath))
			}
		}
	}

	if err := s.implementsRenameConflict(t); err != nil {
		return nil, err
	}

	editsByURI := map[uri.URI][]protocol.TextEdit{
		docURI: {{Range: t.declRng, NewText: newName}},
	}

	// Collect workspace-wide call sites targeting this component's method using
	// refs.Trace (full parse, not ByResolversOnly) so local `new` / `createObject`
	// receivers resolve accurately.
	entries := refs.Trace(s.refsFS(), s.searchRoots(), &refs.Options{
		FuncName:   t.name,
		SourceFile: t.filePath,
		Resolver:   res,
		Explain:    true,
	})

	for i := range entries {
		e := &entries[i]
		if !e.Resolved && e.Component == "" && e.Reason != "" {
			return nil, fmt.Errorf("cannot safely rename '%s': unresolved/ambiguous call at %s:%d (%s)", t.name, filepath.Base(e.File), e.Line+1, e.Reason)
		}

		callerURI := cfpath.ToURI(e.File)
		callerContent, ok := s.getDocument(callerURI)

		if !ok {
			data, rerr := s.FS.ReadFile(e.File)
			if rerr != nil {
				continue
			}

			callerContent = string(data)
		}

		rngs := callNameRangesOnLine(callerContent, e.Line, t.name, e.Variable)
		if len(rngs) > 1 && e.Variable == "" {
			return nil, fmt.Errorf("cannot safely rename '%s': multiple calls on %s:%d", t.name, filepath.Base(e.File), e.Line+1)
		}

		for _, rng := range rngs {
			editsByURI[callerURI] = append(editsByURI[callerURI], protocol.TextEdit{
				Range:   rng,
				NewText: newName,
			})
		}
	}

	return s.multiDocumentEdit(editsByURI), nil
}

func (s *Server) renameComponentProperty(docURI uri.URI, t *crossFileRenameTarget, newName string) (*protocol.WorkspaceEdit, error) {
	if t.pr.Accessors {
		return nil, fmt.Errorf("cannot safely rename property '%s': component has accessors=true (generated getter/setter callers would break)", t.name)
	}

	if !strings.EqualFold(t.name, newName) {
		for _, p := range t.pr.PropertySymbols() {
			if strings.EqualFold(p.Name, newName) {
				return nil, fmt.Errorf("component already declares a property named '%s'", newName)
			}
		}
	}

	edits := []protocol.TextEdit{{Range: t.declRng, NewText: newName}}
	lines := parser.NewLineTable(t.content)
	posOf := rangePositioner(t.content, lines)
	lowerProp := strings.ToLower(t.name)

	// Find all `this.<prop>` and `variables.<prop>` references in script regions.
	// If a tag region also mentions the property name outside the declaration,
	// refuse rather than silently missing tag-based references.
	for _, r := range parser.ClassifyRegions(t.content) {
		if r.Kind == parser.RegionTag {
			if strings.Contains(strings.ToLower(r.Text), "."+lowerProp) {
				return nil, fmt.Errorf("cannot safely rename property '%s': referenced inside tag markup", t.name)
			}

			continue
		}

		if r.Kind != parser.RegionScript {
			continue
		}

		sc := parser.NewSourceScanner(r.Text)

		var prev2, prev1 parser.Token

		for {
			cur := sc.NextSkipComments()
			if cur.Kind == parser.TokEOF {
				break
			}

			if cur.Kind == parser.TokIdent && strings.EqualFold(cur.Value, t.name) &&
				prev1.Kind == parser.TokDot &&
				prev2.Kind == parser.TokIdent &&
				(strings.EqualFold(prev2.Value, "this") || strings.EqualFold(prev2.Value, "variables")) {
				start := r.Offset + cur.Offset
				end := start + len(cur.Value)
				edits = append(edits, protocol.TextEdit{
					Range:   protocol.Range{Start: posOf(start), End: posOf(end)},
					NewText: newName,
				})
			}

			prev2 = prev1
			prev1 = cur
		}
	}

	return s.multiDocumentEdit(map[uri.URI][]protocol.TextEdit{docURI: edits}), nil
}

// callNameRangesOnLine finds every call identifier matching targetName followed
// by '(' on the 0-based line in content. When receiver is non-empty, only calls
// qualified by `<receiver>.<targetName>(` are matched; when receiver is empty,
// only bare (non-member) calls are matched.
func callNameRangesOnLine(content string, targetLine uint32, targetName, receiver string) []protocol.Range {
	lines := strings.Split(content, "\n")
	if int(targetLine) >= len(lines) {
		return nil
	}

	lineText := lines[targetLine]
	sc := parser.NewSourceScanner(lineText)

	var (
		prev2, prev1 parser.Token
		out          []protocol.Range
	)

	cur := sc.NextSkipComments()
	for cur.Kind != parser.TokEOF {
		next := sc.NextSkipComments()

		if cur.Kind == parser.TokIdent && strings.EqualFold(cur.Value, targetName) && next.Kind == parser.TokLParen {
			isDecl := prev1.Kind == parser.TokIdent && strings.EqualFold(prev1.Value, "function")
			match := false

			if !isDecl {
				if receiver != "" {
					match = prev1.Kind == parser.TokDot && prev2.Kind == parser.TokIdent && strings.EqualFold(prev2.Value, receiver)
				} else {
					match = true
				}
			}

			if match {
				sCol := conv.Uint32(lex.UTF16Len(lineText[:cur.Offset]))
				eCol := conv.Uint32(lex.UTF16Len(lineText[:cur.Offset+len(cur.Value)]))
				out = append(out, protocol.Range{
					Start: protocol.Position{Line: targetLine, Character: sCol},
					End:   protocol.Position{Line: targetLine, Character: eCol},
				})
			}
		}

		prev2 = prev1
		prev1 = cur
		cur = next
	}

	return out
}

func (s *Server) multiDocumentEdit(editsByURI map[uri.URI][]protocol.TextEdit) *protocol.WorkspaceEdit {
	// Deduplicate and sort edits per URI.
	uris := make([]uri.URI, 0, len(editsByURI))
	for u, list := range editsByURI {
		if len(list) == 0 {
			continue
		}

		editsByURI[u] = dedupSortEdits(list)
		uris = append(uris, u)
	}

	slices.Sort(uris)

	s.mu.RLock()
	versioned := s.clientDocChanges
	versions := make(map[uri.URI]*int32, len(uris))

	for _, u := range uris {
		if v, known := s.docVersions[u]; known {
			vc := v
			versions[u] = &vc
		}
	}

	s.mu.RUnlock()

	if !versioned {
		changes := make(map[uri.URI][]protocol.TextEdit, len(uris))
		for _, u := range uris {
			changes[u] = editsByURI[u]
		}

		return &protocol.WorkspaceEdit{Changes: changes}
	}

	docChanges := make([]protocol.DocumentChange, 0, len(uris))
	for _, u := range uris {
		list := editsByURI[u]
		elems := make([]protocol.TextDocumentEditElement, len(list))

		for i := range list {
			elems[i] = &list[i]
		}

		docChanges = append(docChanges, &protocol.TextDocumentEdit{
			TextDocument: protocol.OptionalVersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: u},
				Version:                versions[u],
			},
			Edits: elems,
		})
	}

	return &protocol.WorkspaceEdit{DocumentChanges: docChanges}
}

func dedupSortEdits(list []protocol.TextEdit) []protocol.TextEdit {
	slices.SortFunc(list, func(a, b protocol.TextEdit) int {
		if a.Range.Start.Line != b.Range.Start.Line {
			if a.Range.Start.Line < b.Range.Start.Line {
				return -1
			}

			return 1
		}

		if a.Range.Start.Character < b.Range.Start.Character {
			return -1
		}

		if a.Range.Start.Character > b.Range.Start.Character {
			return 1
		}

		return 0
	})

	out := make([]protocol.TextEdit, 0, len(list))

	for _, e := range list {
		if len(out) > 0 && out[len(out)-1].Range == e.Range {
			continue
		}

		out = append(out, e)
	}

	return out
}

// implementsRenameConflict refuses a rename across `implements` (including
// through sub-interfaces and ancestor components): an interface's method and
// the method that satisfies it are renamed together or not at all.
func (s *Server) implementsRenameConflict(t *crossFileRenameTarget) error {
	res := s.getResolver()

	type ancestor struct {
		path     string
		viaIface bool
	}

	seen := map[string]bool{strings.ToLower(t.filePath): true}
	pending := []ancestor{{path: t.filePath}}

	enqueue := func(path string, viaIface bool) {
		if path == "" || seen[strings.ToLower(path)] {
			return
		}

		seen[strings.ToLower(path)] = true
		pending = append(pending, ancestor{path: path, viaIface: viaIface})
	}

	for hops := 0; len(pending) > 0 && hops < 64; hops++ {
		cur := pending[0]
		pending = pending[1:]

		if cur.viaIface {
			for _, fn := range res.EnsureIndexed(cur.path) {
				if strings.EqualFold(fn.Name, t.name) {
					return fmt.Errorf("cannot rename '%s': implements method in interface '%s'", t.name, filepath.Base(cur.path))
				}
			}
		}

		curURI := cfpath.ToURI(cur.path)
		for _, iface := range s.index.ImplementsForFile(curURI) {
			enqueue(res.ComponentPath(iface, filepath.Dir(cur.path)), true)
		}

		if ext, _ := s.index.ExtendsForFile(curURI); ext != "" {
			enqueue(res.ComponentPath(ext, filepath.Dir(cur.path)), cur.viaIface)
		}
	}

	for _, implPath := range s.collectTransitiveImplementers(t.filePath) {
		for _, fn := range res.EnsureIndexed(implPath) {
			if strings.EqualFold(fn.Name, t.name) {
				return fmt.Errorf("cannot rename '%s': implemented in component '%s'", t.name, filepath.Base(implPath))
			}
		}
	}

	return nil
}
