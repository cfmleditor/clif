package server

import (
	"cmp"
	"container/heap"
	"context"
	"encoding/json/v2"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/frameworkapi"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// The symbol picker sends each prefix while the user types. Keep those broad,
// transient one- and two-character responses bounded; once the query is
// specific enough, return the complete match set again.
const workspaceSymbolShortQueryLimit = 1000

func (s *Server) handleDocumentSymbol(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.DocumentSymbolParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI

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

	return buildDocumentSymbols(docURI, pr), nil
}

func buildDocumentSymbols(docURI uri.URI, pr *parser.ParseResult) []protocol.DocumentSymbol {
	content := pr.Content
	lines := parser.NewLineTable(content)
	header, isComp := parser.ComponentOutline(content)

	ctorName := "init"
	if header.InitMethod != "" {
		ctorName = header.InitMethod
	}

	methods := make([]protocol.DocumentSymbol, 0, len(pr.Funcs))

	for i := range pr.Funcs {
		d := &pr.Funcs[i]
		if d.Generated {
			continue
		}

		selRange := funcDefRange(d)
		fullRange := functionOutlineRange(content, lines, d, pr.Scopes, selRange)

		kind := protocol.SymbolKindFunction

		if isComp {
			if !header.IsInterface && strings.EqualFold(d.Name, ctorName) {
				kind = protocol.SymbolKindConstructor
			} else {
				kind = protocol.SymbolKindMethod
			}
		}

		methods = append(methods, protocol.DocumentSymbol{
			Name:           d.Name,
			Detail:         strPtr(d.ReturnType),
			Kind:           kind,
			Range:          fullRange,
			SelectionRange: selRange,
		})
	}

	if !isComp {
		return methods
	}

	props := pr.PropertySymbols()
	children := make([]protocol.DocumentSymbol, 0, len(props)+len(methods))

	for i := range props {
		p := &props[i]
		selRange := propertySelectionRange(p)
		fullRange := containRange(protocol.Range{
			Start: protocol.Position{Line: p.Line, Character: p.Col},
			End:   protocol.Position{Line: p.EndLine, Character: p.EndCol},
		}, selRange)

		children = append(children, protocol.DocumentSymbol{
			Name:           p.Name,
			Detail:         strPtr(p.Type),
			Kind:           protocol.SymbolKindProperty,
			Range:          fullRange,
			SelectionRange: selRange,
		})
	}

	children = append(children, methods...)

	kind := protocol.SymbolKindClass
	if header.IsInterface {
		kind = protocol.SymbolKindInterface
	}

	selRange := offsetRange(content, lines, header.NameStart, header.NameEnd)
	fullRange := containRange(offsetRange(content, lines, header.Start, header.End), selRange)

	return []protocol.DocumentSymbol{{
		Name:           componentSymbolName(docURI, header.IsInterface),
		Kind:           kind,
		Range:          fullRange,
		SelectionRange: selRange,
		Children:       children,
	}}
}

func componentSymbolName(docURI uri.URI, isInterface bool) string {
	base := filepath.Base(cfpath.FromURI(string(docURI)))
	if ext := filepath.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}

	if base != "" && base != "." && base != string(filepath.Separator) {
		return base
	}

	if isInterface {
		return "interface"
	}

	return "component"
}

func propertySelectionRange(p *parser.PropertySymbol) protocol.Range {
	if p.NameEndCol > p.NameCol {
		return protocol.Range{
			Start: protocol.Position{Line: p.NameLine, Character: p.NameCol},
			End:   protocol.Position{Line: p.NameLine, Character: p.NameEndCol},
		}
	}

	return protocol.Range{
		Start: protocol.Position{Line: p.Line, Character: p.Col},
		End:   protocol.Position{Line: p.Line, Character: p.Col},
	}
}

func functionOutlineRange(content string, lines parser.LineTable, d *parser.FunctionDef, scopes []parser.FuncScope, sel protocol.Range) protocol.Range {
	endLine := int(d.Line)
	if d.HasNameRange() && int(d.NameLine) > endLine {
		endLine = int(d.NameLine)
	}

	for _, sc := range scopes {
		if sc.Start == int(d.Line) && strings.EqualFold(sc.Name, d.Name) {
			if sc.End > endLine {
				endLine = sc.End
			}

			break
		}
	}

	startCol := firstNonSpaceCol(content, lines, int(d.Line))
	endCol := lineUTF16Len(content, lines, endLine)

	return containRange(protocol.Range{
		Start: protocol.Position{Line: d.Line, Character: startCol},
		End:   protocol.Position{Line: conv.Uint32(endLine), Character: endCol},
	}, sel)
}

func offsetRange(content string, lines parser.LineTable, start, end int) protocol.Range {
	return protocol.Range{
		Start: offsetPosition(content, lines, start),
		End:   offsetPosition(content, lines, max(start, end)),
	}
}

func offsetPosition(content string, lines parser.LineTable, offset int) protocol.Position {
	offset = min(max(offset, 0), len(content))
	line := lines.LineOf(offset)

	lineStart, ok := lines.Start(line)
	if !ok {
		return protocol.Position{}
	}

	return protocol.Position{
		Line:      conv.Uint32(line),
		Character: conv.Uint32(lex.UTF16Len(content[lineStart:offset])),
	}
}

func firstNonSpaceCol(content string, lines parser.LineTable, line int) uint32 {
	text := lineTextFromTable(content, lines, line)
	i := 0

	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}

	return conv.Uint32(lex.UTF16Len(text[:i]))
}

func lineUTF16Len(content string, lines parser.LineTable, line int) uint32 {
	return conv.Uint32(lex.UTF16Len(lineTextFromTable(content, lines, line)))
}

func lineTextFromTable(content string, lines parser.LineTable, line int) string {
	start, ok := lines.Start(line)
	if !ok {
		return ""
	}

	end := len(content)
	if next, ok := lines.Start(line + 1); ok {
		end = next - 1
	}

	if end > start && content[end-1] == '\r' {
		end--
	}

	return content[start:end]
}

func containRange(outer, inner protocol.Range) protocol.Range {
	if posBefore(inner.Start, outer.Start) {
		outer.Start = inner.Start
	}

	if posBefore(outer.End, inner.End) {
		outer.End = inner.End
	}

	return outer
}

func posBefore(a, b protocol.Position) bool {
	return a.Line < b.Line || (a.Line == b.Line && a.Character < b.Character)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

func (s *Server) workspaceSymbolProvider() protocol.WorkspaceSymbolProvider {
	if s.workspaceSymbolDefer {
		resolveProvider := true

		return &protocol.WorkspaceSymbolOptions{ResolveProvider: &resolveProvider}
	}

	return protocol.Boolean(true)
}

func clientDefersWorkspaceSymbol(caps protocol.ClientCapabilities) bool {
	w := caps.Workspace
	if w == nil || w.Symbol == nil {
		return false
	}

	return slices.Contains(w.Symbol.ResolveSupport.Properties, "location.range")
}

func (s *Server) handleWorkspaceSymbol(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.WorkspaceSymbolParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	query := strings.ToLower(params.Query)

	match := func(name string) bool {
		return query == "" || containsFoldStr(name, query)
	}

	var defs []*parser.FunctionDef

	if len(query) < 3 {
		selection := workspaceSymbolHeap{query: query}
		if !s.index.VisitMatchingFunctionBatches(match, selection.visit) {
			// Function definitions changed between batches. Retry from a full
			// snapshot so the results still all come from one generation.
			selection = workspaceSymbolHeap{query: query}
			selection.visit(s.index.FunctionsMatching(match))
		}

		defs = selection.results()
	} else {
		defs = browsableWorkspaceSymbols(s.index.FunctionsMatching(match))
	}

	if s.workspaceSymbolDefer {
		symbols := make([]protocol.WorkspaceSymbol, len(defs))
		locs := make([]protocol.LocationUriOnly, len(defs))

		for i, d := range defs {
			locs[i].URI = d.URI
			symbols[i].Name = d.Name
			symbols[i].Kind = protocol.SymbolKindFunction
			symbols[i].Location = &locs[i]
		}

		return symbols, nil
	}

	symbols := make([]protocol.SymbolInformation, 0, len(defs))

	for _, d := range defs {
		symbols = append(symbols, protocol.SymbolInformation{
			Name: d.Name,
			Kind: protocol.SymbolKindFunction,
			Location: protocol.Location{
				URI:   d.URI,
				Range: funcDefRange(d),
			},
		})
	}

	return symbols, nil
}

func (s *Server) handleWorkspaceSymbolResolve(_ context.Context, rawParams []byte) (any, error) {
	var sym protocol.WorkspaceSymbol
	if err := json.Unmarshal(rawParams, &sym); err != nil {
		return nil, err
	}

	var docURI uri.URI

	switch loc := sym.Location.(type) {
	case *protocol.LocationUriOnly:
		docURI = loc.URI
	case *protocol.Location:
		return sym, nil
	default:
		return sym, nil
	}

	if docURI == "" {
		return sym, nil
	}

	for _, d := range s.index.FunctionsForFile(docURI) {
		if strings.EqualFold(d.Name, sym.Name) {
			loc := funcDefLocation(d)
			sym.Location = &loc

			return sym, nil
		}
	}

	sym.Location = &protocol.Location{URI: docURI}

	return sym, nil
}

func compareWorkspaceSymbols(a, b *parser.FunctionDef, query string) int {
	if d := cmp.Compare(symbolMatchRank(a.Name, query), symbolMatchRank(b.Name, query)); d != 0 {
		return d
	}

	if d := strings.Compare(a.Name, b.Name); d != 0 {
		return d
	}

	if d := cmp.Compare(a.URI, b.URI); d != 0 {
		return d
	}

	return cmp.Compare(a.Line, b.Line)
}

type workspaceSymbolHeap struct {
	defs   []*parser.FunctionDef
	query  string
	capped bool
}

func (h *workspaceSymbolHeap) Len() int { return len(h.defs) }
func (h *workspaceSymbolHeap) Less(i, j int) bool {
	return compareWorkspaceSymbols(h.defs[i], h.defs[j], h.query) > 0
}
func (h *workspaceSymbolHeap) Swap(i, j int) { h.defs[i], h.defs[j] = h.defs[j], h.defs[i] }
func (h *workspaceSymbolHeap) Push(value any) {
	def, ok := value.(*parser.FunctionDef)
	if !ok {
		panic(fmt.Sprintf("workspace symbol heap received %T", value))
	}

	h.defs = append(h.defs, def)
}

func (h *workspaceSymbolHeap) Pop() any {
	last := len(h.defs) - 1
	value := h.defs[last]
	h.defs = h.defs[:last]

	return value
}

func (h *workspaceSymbolHeap) visit(defs []*parser.FunctionDef) {
	for _, def := range browsableWorkspaceSymbols(defs) {
		h.consider(def)
	}
}

func (h *workspaceSymbolHeap) consider(def *parser.FunctionDef) {
	if len(h.defs) < workspaceSymbolShortQueryLimit {
		if len(h.defs) == cap(h.defs) {
			capacity := min(max(8, 2*len(h.defs)), workspaceSymbolShortQueryLimit)
			h.defs = append(make([]*parser.FunctionDef, 0, capacity), h.defs...)
		}

		h.defs = append(h.defs, def)

		return
	}

	if !h.capped {
		h.capped = true
		heap.Init(h)
	}

	if compareWorkspaceSymbols(def, h.defs[0], h.query) < 0 {
		h.defs[0] = def
		heap.Fix(h, 0)
	}
}

func (h *workspaceSymbolHeap) results() []*parser.FunctionDef {
	if h.capped {
		slices.SortFunc(h.defs, func(a, b *parser.FunctionDef) int {
			return compareWorkspaceSymbols(a, b, h.query)
		})
	}

	return h.defs
}

func browsableWorkspaceSymbols(defs []*parser.FunctionDef) []*parser.FunctionDef {
	out := defs[:0]

	for _, d := range defs {
		// A framework's bundled API is looked up, not browsed: it is in the
		// index only because a call reached it, and it has nowhere to open.
		if !frameworkapi.IsStubURI(string(d.URI)) {
			out = append(out, d)
		}
	}

	return out
}

func symbolMatchRank(name, query string) int {
	switch {
	case strings.EqualFold(name, query):
		return 0
	case len(name) >= len(query) && strings.EqualFold(name[:len(query)], query):
		return 1
	default:
		return 2
	}
}

func containsFoldStr(s, substr string) bool {
	n := len(substr)
	if n == 0 {
		return true
	}

	end := len(s) - n
	for i := 0; i <= end; i++ {
		match := true

		for j := range n {
			// Both sides go through the same fold, and the fold only touches
			// A-Z. The old code folded the haystack with |0x20 and compared
			// against a raw needle byte, so anything outside a-z never matched
			// its own fold: '_' is 0x5F and 0x5F|0x20 is 0x7F, which meant any
			// workspace-symbol query containing an underscore silently returned
			// nothing. Folding both sides with |0x20 would fix that but make
			// '@' match '`' and '[' match '{', so the fold is explicit.
			if lex.LowerASCII(s[i+j]) != lex.LowerASCII(substr[j]) {
				match = false

				break
			}
		}

		if match {
			return true
		}
	}

	return false
}
