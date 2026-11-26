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
	"github.com/cfmleditor/clif/internal/refs"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// handlePrepareCallHierarchy answers textDocument/prepareCallHierarchy.
// On-demand only (zero idle or per-keystroke overhead): prepares a
// CallHierarchyItem for the function declaration, called function, or enclosing
// function at the cursor position.
func (s *Server) handlePrepareCallHierarchy(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.CallHierarchyPrepareParams
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

	pr := s.parseResultOf(docURI)
	if pr == nil {
		return nil, nil
	}

	// 1. Check if cursor is on a function declaration in this document.
	for i := range pr.Funcs {
		d := &pr.Funcs[i]
		if d.Generated || d.Name == "" {
			continue
		}

		sel := funcDefRange(d)
		if positionInRange(params.Position, sel) || (params.Position.Line == d.Line && !d.HasNameRange()) {
			return []protocol.CallHierarchyItem{s.callHierarchyItemForFunc(d, pr)}, nil
		}
	}

	// 2. Check if cursor is on a call site (qualified or unqualified).
	lines := parser.NewLineTable(content)
	off := positionToByteOffset(content, lines, params.Position)

	if wStart, wEnd, ok := wordSpanAt(content, off); ok {
		word := content[wStart:wEnd]
		if qual := qualifierBeforeOffset(content, wStart); qual != "" {
			if def := s.resolveUserFunc(qual, word, docURI, params.Position.Line); def != nil {
				return []protocol.CallHierarchyItem{s.callHierarchyItemForFunc(def, nil)}, nil
			}
		} else if def, inFile, total := s.index.LookupPreferred(word, docURI); def != nil && (inFile || total == 1) {
			return []protocol.CallHierarchyItem{s.callHierarchyItemForFunc(def, nil)}, nil
		}
	}

	// 3. Fall back to the enclosing function scope.
	if scope := pr.FuncScopeAt(int(params.Position.Line)); scope.Start >= 0 && scope.Name != "" {
		if def := s.index.FunctionInFile(docURI, scope.Name); def != nil {
			return []protocol.CallHierarchyItem{s.callHierarchyItemForFunc(def, pr)}, nil
		}
	}

	return nil, nil
}

// handleCallHierarchyIncomingCalls answers callHierarchy/incomingCalls using
// the workspace-bounded refs.FindCallsMany scanner, grouped by caller function.
func (s *Server) handleCallHierarchyIncomingCalls(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.CallHierarchyIncomingCallsParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	name := params.Item.Name
	if name == "" {
		return []protocol.CallHierarchyIncomingCall{}, nil
	}

	sourceFile := cfpath.FromURI(string(params.Item.URI))
	also := map[string][]string{strings.ToLower(name): s.relatedSources(sourceFile, name)}
	byName := refs.FindCallsMany(s.refsFS(), s.searchRoots(), []string{name}, sourceFile, also, s.getResolver())
	entries := byName[strings.ToLower(name)]

	if len(entries) == 0 {
		return []protocol.CallHierarchyIncomingCall{}, nil
	}

	type callerKey struct {
		file string
		fn   string
	}

	grouped := make(map[callerKey]*protocol.CallHierarchyIncomingCall)
	order := make([]callerKey, 0, len(entries))

	// A caller file is read and its lines found once, however many calls it
	// holds: read again for each call, 8,000 calls in one file took 871 ms.
	type callerText struct {
		content string
		lines   parser.LineTable
		ok      bool
		file    *entryFile // for the place of each call, from its offset
	}

	callerTexts := map[string]*callerText{}

	rangeOf := func(e *refs.Entry) protocol.Range {
		ct, seen := callerTexts[e.File]
		if !seen {
			ct = &callerText{file: s.entryFile(e.File)}
			ct.content, ct.ok = s.readFileContent(uri.File(e.File), e.File)

			if ct.ok {
				ct.lines = parser.NewLineTable(ct.content)
			}

			callerTexts[e.File] = ct
		}

		// Where this call stands, when the parse knew: a line with two calls
		// has two ranges, not the first one twice.
		if e.HasStart {
			if rng, found := ct.file.spanRange(e.Start, name); found {
				return rng
			}
		}

		if !ct.ok {
			return protocol.Range{
				Start: protocol.Position{Line: e.Line},
				End:   protocol.Position{Line: e.Line},
			}
		}

		return s.callSiteRangeInContent(ct.content, ct.lines, e.Line, name)
	}

	for i := range entries {
		e := &entries[i]
		k := callerKey{file: e.File, fn: strings.ToLower(e.Function)}

		call, exists := grouped[k]
		if !exists {
			from := s.callerHierarchyItem(e.File, e.Function)
			call = &protocol.CallHierarchyIncomingCall{From: from}
			grouped[k] = call
			order = append(order, k)
		}

		call.FromRanges = append(call.FromRanges, rangeOf(e))
	}

	out := make([]protocol.CallHierarchyIncomingCall, 0, len(order))
	for _, k := range order {
		out = append(out, *grouped[k])
	}

	slices.SortFunc(out, func(a, b protocol.CallHierarchyIncomingCall) int {
		if c := strings.Compare(string(a.From.URI), string(b.From.URI)); c != 0 {
			return c
		}

		if a.From.Range.Start.Line != b.From.Range.Start.Line {
			return int(a.From.Range.Start.Line) - int(b.From.Range.Start.Line)
		}

		return strings.Compare(a.From.Name, b.From.Name)
	})

	return out, nil
}

// handleCallHierarchyOutgoingCalls answers callHierarchy/outgoingCalls by
// reading only the caller's own file and resolving each call site in its body.
func (s *Server) handleCallHierarchyOutgoingCalls(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.CallHierarchyOutgoingCallsParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.Item.URI
	file := cfpath.FromURI(string(docURI))

	if file == "" {
		return []protocol.CallHierarchyOutgoingCall{}, nil
	}

	content, ok := s.readFileContent(docURI, file)
	if !ok {
		return []protocol.CallHierarchyOutgoingCall{}, nil
	}

	pr := parser.ParseWithOptions(docURI, content, &parser.ParseOptions{ExtractCalls: true})
	scope := findMatchingScope(pr, params.Item.Name, int(params.Item.SelectionRange.Start.Line))

	if scope.Start < 0 {
		return []protocol.CallHierarchyOutgoingCall{}, nil
	}

	calls := pr.FuncCalls(scope.Start, scope.End)
	if len(calls) == 0 {
		return []protocol.CallHierarchyOutgoingCall{}, nil
	}

	baseDir := filepath.Dir(file)

	type calleeKey struct {
		uri  string
		line uint32
		name string
	}

	grouped := make(map[calleeKey]*protocol.CallHierarchyOutgoingCall)
	order := make([]calleeKey, 0, len(calls))

	// The file's lines once, not once per call in the function.
	contentLines := parser.NewLineTable(content)

	// The calls were found in this text, so their offsets are places in it.
	contentFile := &entryFile{text: content}

	for i := range calls {
		c := &calls[i]
		def := s.resolveOutgoingCallee(c, docURI, file, baseDir)

		if def == nil {
			continue
		}

		k := calleeKey{uri: string(def.URI), line: def.Line, name: strings.ToLower(def.Name)}

		outCall, exists := grouped[k]
		if !exists {
			to := s.callHierarchyItemForFunc(def, nil)
			outCall = &protocol.CallHierarchyOutgoingCall{To: to}
			grouped[k] = outCall
			order = append(order, k)
		}

		// Where this call stands, when the parse knew: a line with two calls
		// has two ranges, not the first one twice.
		rng, placed := protocol.Range{}, false
		if c.HasSpan {
			rng, placed = contentFile.spanRange(int(c.StartByte), c.FuncName)
		}

		if !placed {
			rng = s.callSiteRangeInContent(content, contentLines, c.Line, c.FuncName)
		}

		outCall.FromRanges = append(outCall.FromRanges, rng)
	}

	out := make([]protocol.CallHierarchyOutgoingCall, 0, len(order))
	for _, k := range order {
		out = append(out, *grouped[k])
	}

	slices.SortFunc(out, func(a, b protocol.CallHierarchyOutgoingCall) int {
		if c := strings.Compare(string(a.To.URI), string(b.To.URI)); c != 0 {
			return c
		}

		if a.To.Range.Start.Line != b.To.Range.Start.Line {
			return int(a.To.Range.Start.Line) - int(b.To.Range.Start.Line)
		}

		return strings.Compare(a.To.Name, b.To.Name)
	})

	return out, nil
}

func (s *Server) resolveOutgoingCallee(c *parser.CallSite, docURI uri.URI, file, baseDir string) *parser.FunctionDef {
	if c.Component != "" && !strings.HasPrefix(c.Component, "$") {
		if def := s.getResolver().ResolveFunc(c.Component, c.FuncName, baseDir); def != nil {
			return def
		}
	}

	if c.Variable != "" {
		if def := s.resolveUserFunc(c.Variable, c.FuncName, docURI, c.Line); def != nil {
			return def
		}

		return nil
	}

	if def := s.index.FunctionInFile(docURI, c.FuncName); def != nil {
		return def
	}

	if strings.EqualFold(filepath.Ext(file), ".cfc") {
		if def := s.getResolver().ResolveFunc(file, c.FuncName, baseDir); def != nil {
			return def
		}
	}

	return nil
}

func (s *Server) callHierarchyItemForFunc(def *parser.FunctionDef, pr *parser.ParseResult) protocol.CallHierarchyItem {
	file := cfpath.FromURI(string(def.URI))
	sel := funcDefRange(def)
	full := sel

	if pr == nil {
		pr = s.parseResultOf(def.URI)
	}

	if pr != nil {
		if scope := findMatchingScope(pr, def.Name, int(def.Line)); scope.Start >= 0 {
			full = protocol.Range{
				Start: protocol.Position{Line: conv.Uint32(scope.Start), Character: 0},
				End:   protocol.Position{Line: conv.Uint32(scope.End), Character: 0},
			}
			// SelectionRange must be contained inside Range per LSP spec.
			if posBefore(sel.Start, full.Start) {
				full.Start = sel.Start
			}

			if posBefore(full.End, sel.End) {
				full.End = sel.End
			}
		}
	}

	kind := protocol.SymbolKindFunction

	if strings.EqualFold(filepath.Ext(file), ".cfc") {
		if strings.EqualFold(def.Name, "init") {
			kind = protocol.SymbolKindConstructor
		} else {
			kind = protocol.SymbolKindMethod
		}
	}

	var detail *string
	if dotPath, _ := s.fileToPackageIn(file); dotPath != "" {
		detail = &dotPath
	} else if base := filepath.Base(file); base != "" {
		detail = &base
	}

	return protocol.CallHierarchyItem{
		Name:           def.Name,
		Kind:           kind,
		Detail:         detail,
		URI:            def.URI,
		Range:          full,
		SelectionRange: sel,
	}
}

func (s *Server) callerHierarchyItem(file, fnName string) protocol.CallHierarchyItem {
	docURI := uri.File(file)
	if fnName != "" {
		if def := s.index.FunctionInFile(docURI, fnName); def != nil {
			return s.callHierarchyItemForFunc(def, nil)
		}
	}

	name := fnName
	kind := protocol.SymbolKindFunction

	if name == "" {
		name = filepath.Base(file)
		kind = protocol.SymbolKindFile
	}

	r := protocol.Range{}

	return protocol.CallHierarchyItem{
		Name:           name,
		Kind:           kind,
		URI:            docURI,
		Range:          r,
		SelectionRange: r,
	}
}

func findMatchingScope(pr *parser.ParseResult, name string, line int) parser.FuncScope {
	for _, sc := range pr.Scopes {
		if strings.EqualFold(sc.Name, name) && sc.Start <= line && line <= sc.End {
			return sc
		}
	}

	for _, sc := range pr.Scopes {
		if strings.EqualFold(sc.Name, name) {
			return sc
		}
	}

	return parser.FuncScope{Start: -1, End: -1}
}

func (s *Server) readFileContent(docURI uri.URI, file string) (string, bool) {
	if content, ok := s.getDocument(docURI); ok {
		return content, true
	}

	raw, err := os.ReadFile(file) //nolint:gosec // workspace file resolved by index/resolver
	if err != nil {
		return "", false
	}

	return lex.NormalizeLineBreaks(string(raw)), true
}

func (s *Server) callSiteRangeInContent(content string, lines parser.LineTable, line uint32, funcName string) protocol.Range {
	lineIdx := int(line)

	if lineIdx < 0 || lineIdx >= len(lines) {
		return protocol.Range{
			Start: protocol.Position{Line: line, Character: 0},
			End:   protocol.Position{Line: line, Character: 0},
		}
	}

	lStart := int(lines[lineIdx])
	lEnd := len(content)

	if lineIdx+1 < len(lines) {
		lEnd = int(lines[lineIdx+1])
	}

	lineText := content[lStart:lEnd]
	if idx := parser.IndexFold(lineText, funcName); idx >= 0 {
		startCol := lex.UTF16Len(lineText[:idx])
		endCol := startCol + lex.UTF16Len(lineText[idx:idx+len(funcName)])

		return protocol.Range{
			Start: protocol.Position{Line: line, Character: conv.Uint32(startCol)},
			End:   protocol.Position{Line: line, Character: conv.Uint32(endCol)},
		}
	}

	return protocol.Range{
		Start: protocol.Position{Line: line, Character: 0},
		End:   protocol.Position{Line: line, Character: 0},
	}
}

func qualifierBeforeOffset(content string, wordStart int) string {
	i := wordStart
	for i > 0 && (content[i-1] == ' ' || content[i-1] == '\t') {
		i--
	}

	if i == 0 || content[i-1] != '.' {
		return ""
	}

	end := i - 1
	start := end

	for start > 0 && (isWordByte(content[start-1]) || content[start-1] == '.') {
		start--
	}

	return strings.Trim(content[start:end], ".")
}
