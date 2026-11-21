package server

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
)

type componentPathKind uint8

const (
	componentPathNone componentPathKind = iota
	componentPathCFC
	componentPathOther
)

type componentPathCtx struct {
	kind       componentPathKind
	prefix     string
	quoteStart int // byte column on the cursor line where the prefix starts, or -1
}

type callArgInfo struct {
	name      string
	quotedVal string
	isQuoted  bool
}

// componentPathContext reports whether the cursor is inside a quoted component
// path string (such as <cfobject component="..."> or createObject("component", "...")).
func componentPathContext(content string, line, char int, cc *completionContext) componentPathCtx {
	if ctx := createObjectComponentContext(content, line, char); ctx.kind != componentPathNone {
		return ctx
	}

	if cc.inAttrValue {
		return tagComponentPathContext(content, line, char, cc.tagName)
	}

	return componentPathCtx{}
}

func tagComponentPathContext(content string, line, char int, tagName string) componentPathCtx {
	attrName := parser.FindCurrentAttr(content, line, char)
	if attrName == "" || tagName == "" {
		return componentPathCtx{}
	}

	textBefore := parser.TextBeforeCursor(content, line, char)

	lastOpen := strings.LastIndexByte(textBefore, '<')
	if lastOpen < 0 {
		return componentPathCtx{}
	}

	afterOpen := textBefore[lastOpen:]
	kind := tagAttrComponentKind(tagName, attrName, afterOpen)

	if kind == componentPathNone {
		return componentPathCtx{}
	}

	prefix, ok := openAttrQuotePrefix(afterOpen)
	if !ok || strings.ContainsRune(prefix, '#') {
		return componentPathCtx{}
	}

	if attrName == "implements" {
		if comma := strings.LastIndexByte(prefix, ','); comma >= 0 {
			prefix = strings.TrimLeft(prefix[comma+1:], " \t")
		}
	}

	quoteStart := -1
	if !strings.ContainsRune(prefix, '\n') {
		quoteStart = char - len(prefix)
	}

	return componentPathCtx{
		kind:       kind,
		prefix:     prefix,
		quoteStart: quoteStart,
	}
}

func tagAttrComponentKind(tagName, attrName, afterOpen string) componentPathKind {
	switch tagName {
	case "cfobject":
		typeVal, hasType := findOpenTagAttr(afterOpen, "type")

		switch attrName {
		case "component":
			if !hasType || typeVal == "" || strings.EqualFold(typeVal, "component") {
				return componentPathCFC
			}

			return componentPathOther
		case "class":
			if strings.EqualFold(typeVal, "component") {
				return componentPathCFC
			}

			if hasType && typeVal != "" {
				return componentPathOther
			}
		}
	case "cfinvoke":
		if attrName == "component" {
			return componentPathCFC
		}
	case "cfcomponent", "cfinterface":
		if attrName == "extends" || attrName == "implements" {
			return componentPathCFC
		}
	}

	return componentPathNone
}

// openAttrQuotePrefix returns the unclosed quoted attribute value prefix at the
// end of afterOpen (starting at '<').
func openAttrQuotePrefix(afterOpen string) (string, bool) {
	var quote byte

	openIdx := -1

	for i := 0; i < len(afterOpen); i++ {
		ch := afterOpen[i]
		if quote == 0 {
			if ch == '"' || ch == '\'' {
				quote = ch
				openIdx = i
			}

			continue
		}

		if ch == quote {
			if i+1 < len(afterOpen) && afterOpen[i+1] == quote {
				i++

				continue
			}

			quote = 0
			openIdx = -1
		}
	}

	if quote == 0 || openIdx < 0 {
		return "", false
	}

	return afterOpen[openIdx+1:], true
}

// findOpenTagAttr extracts a completed attribute's value from an open tag prefix
// (starting at '<').
func findOpenTagAttr(afterOpen, target string) (string, bool) {
	i := 1
	for i < len(afterOpen) && (afterOpen[i] == ' ' || afterOpen[i] == '\t') {
		i++
	}

	for i < len(afterOpen) && !isTagDelim(afterOpen[i]) {
		i++
	}

	for i < len(afterOpen) {
		for i < len(afterOpen) && isSpaceByte(afterOpen[i]) {
			i++
		}

		nameStart := i
		for i < len(afterOpen) && !isTagDelim(afterOpen[i]) && afterOpen[i] != '=' {
			i++
		}

		name := afterOpen[nameStart:i]
		if name == "" {
			break
		}

		for i < len(afterOpen) && isSpaceByte(afterOpen[i]) {
			i++
		}

		if i >= len(afterOpen) || afterOpen[i] != '=' {
			continue
		}

		i++
		for i < len(afterOpen) && isSpaceByte(afterOpen[i]) {
			i++
		}

		if i >= len(afterOpen) {
			break
		}

		val, next, closed := readAttrVal(afterOpen, i)
		i = next

		if closed && strings.EqualFold(name, target) {
			return val, true
		}
	}

	return "", false
}

func readAttrVal(s string, i int) (val string, next int, closed bool) {
	if s[i] == '"' || s[i] == '\'' {
		q := s[i]
		start := i + 1
		i++

		for i < len(s) {
			if s[i] == q {
				if i+1 < len(s) && s[i+1] == q {
					i += 2

					continue
				}

				return s[start:i], i + 1, true
			}

			i++
		}

		return "", i, false
	}

	start := i
	for i < len(s) && !isTagDelim(s[i]) {
		i++
	}

	return s[start:i], i, true
}

func isTagDelim(b byte) bool {
	return isSpaceByte(b) || b == '>' || b == '/'
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

// createObjectComponentContext detects whether the cursor is inside the class
// or component path string argument of a createObject(...) call.
func createObjectComponentContext(content string, line, char int) componentPathCtx {
	off := min(lineStartOffset(content, line)+char, len(content))

	const maxLookback = 2048

	winStart := max(0, off-maxLookback)
	window := content[winStart:off]

	if parser.IndexFold(window, "createobject") < 0 {
		return componentPathCtx{}
	}

	// Scan createObject occurrences from innermost (rightmost) backwards.
	searchEnd := len(window)
	for searchEnd > 0 {
		idx := lastIndexFold(window[:searchEnd], "createobject")
		if idx < 0 {
			break
		}

		searchEnd = idx

		absIdx := winStart + idx
		if !isStandaloneCallIdent(content, absIdx, len("createobject")) {
			continue
		}

		parenPos := absIdx + len("createobject")
		for parenPos < off && isSpaceByte(content[parenPos]) {
			parenPos++
		}

		if parenPos >= off || content[parenPos] != '(' {
			continue
		}

		if ctx, enclosing := inspectCreateObjectCall(content, parenPos, off, char); enclosing {
			return ctx
		}
	}

	return componentPathCtx{}
}

func isStandaloneCallIdent(content string, start, length int) bool {
	if start > 0 {
		prev := content[start-1]
		if parser.IsWordChar(prev) || prev == '.' {
			return false
		}
	}

	after := start + length
	if after < len(content) && parser.IsWordChar(content[after]) {
		return false
	}

	return true
}

func lastIndexFold(s, needle string) int {
	pos := 0
	last := -1

	for pos < len(s) {
		idx := parser.IndexFold(s[pos:], needle)
		if idx < 0 {
			break
		}

		last = pos + idx
		pos = last + 1
	}

	return last
}

// inspectCreateObjectCall walks from the '(' at parenPos up to cursor offset
// off. enclosing is true when the call is still open at off.
func inspectCreateObjectCall(content string, parenPos, off, char int) (componentPathCtx, bool) {
	var (
		args      []callArgInfo
		quote     byte
		openQuote = -1
		paren     = 1
		bracket   = 0
		brace     = 0
		argStart  = parenPos + 1
	)

	for i := parenPos + 1; i < off; i++ {
		ch := content[i]
		if quote != 0 {
			if ch == quote {
				if i+1 < off && content[i+1] == quote {
					i++

					continue
				}

				quote = 0
				openQuote = -1
			}

			continue
		}

		switch ch {
		case '"', '\'':
			quote = ch
			openQuote = i
		case '(':
			paren++
		case ')':
			paren--
			if paren == 0 {
				return componentPathCtx{}, false
			}
		case '[':
			bracket++
		case ']':
			bracket--
		case '{':
			brace++
		case '}':
			brace--
		case ',':
			if paren == 1 && bracket == 0 && brace == 0 {
				args = append(args, parseCallArg(content[argStart:i]))
				argStart = i + 1
			}
		}
	}

	if paren != 1 || bracket != 0 || brace != 0 || quote == 0 || openQuote < 0 {
		return componentPathCtx{}, true
	}

	prefix := content[openQuote+1 : off]
	if strings.ContainsRune(prefix, '#') {
		return componentPathCtx{}, true
	}

	activeName := parseActiveArgName(content[argStart:openQuote])
	kind := classifyCreateObjectArg(args, activeName)

	if kind == componentPathNone {
		return componentPathCtx{}, true
	}

	quoteStart := -1
	if !strings.ContainsRune(prefix, '\n') {
		quoteStart = char - len(prefix)
	}

	return componentPathCtx{
		kind:       kind,
		prefix:     prefix,
		quoteStart: quoteStart,
	}, true
}

func parseCallArg(raw string) callArgInfo {
	s := strings.TrimSpace(raw)
	info := callArgInfo{}

	if name, rest, ok := splitNamedArg(s); ok {
		info.name = name
		s = strings.TrimSpace(rest)
	}

	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		info.isQuoted = true
		info.quotedVal = s[1 : len(s)-1]
	}

	return info
}

func parseActiveArgName(beforeQuote string) string {
	s := strings.TrimSpace(beforeQuote)
	if !strings.HasSuffix(s, "=") {
		return ""
	}

	name := strings.TrimSpace(s[:len(s)-1])
	if name == "" {
		return ""
	}

	for i := range len(name) {
		if !parser.IsWordChar(name[i]) {
			return ""
		}
	}

	return strings.ToLower(name)
}

func splitNamedArg(s string) (name, rest string, ok bool) {
	eq := strings.IndexByte(s, '=')
	if eq <= 0 {
		return "", "", false
	}

	lhs := strings.TrimSpace(s[:eq])
	if lhs == "" {
		return "", "", false
	}

	for i := range len(lhs) {
		if !parser.IsWordChar(lhs[i]) {
			return "", "", false
		}
	}

	return strings.ToLower(lhs), s[eq+1:], true
}

func classifyCreateObjectArg(prevArgs []callArgInfo, activeName string) componentPathKind {
	typeVal, hasType := createObjectTypeArg(prevArgs)

	switch activeName {
	case "component":
		if !hasType || strings.EqualFold(typeVal, "component") {
			return componentPathCFC
		}

		return componentPathOther
	case "class":
		if !hasType {
			return componentPathNone
		}

		if strings.EqualFold(typeVal, "component") {
			return componentPathCFC
		}

		return componentPathOther
	case "":
		if len(prevArgs) != 1 || !hasType {
			return componentPathNone
		}

		if strings.EqualFold(typeVal, "component") {
			return componentPathCFC
		}

		return componentPathOther
	default:
		return componentPathNone
	}
}

func createObjectTypeArg(prevArgs []callArgInfo) (string, bool) {
	for _, a := range prevArgs {
		if a.name == "type" && a.isQuoted {
			return a.quotedVal, true
		}
	}

	if len(prevArgs) > 0 && prevArgs[0].name == "" && prevArgs[0].isQuoted {
		return prevArgs[0].quotedVal, true
	}

	return "", false
}

// componentPathItems builds completion items for indexed workspace CFCs whose
// dot-path matches ctx.prefix.
func (s *Server) componentPathItems(pos protocol.Position, content string, ctx componentPathCtx) []protocol.CompletionItem {
	files := s.index.ComponentFiles()
	if len(files) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(files))

	var paths []string

	addPath := func(dotPath string) {
		if dotPath == "" || !lex.HasPrefixFold(dotPath, ctx.prefix) {
			return
		}

		key := strings.ToLower(dotPath)
		if _, exists := seen[key]; exists {
			return
		}

		seen[key] = struct{}{}

		paths = append(paths, dotPath)
	}

	for _, u := range files {
		filePath := cfpath.FromURI(string(u))
		if strings.EqualFold(filepath.Base(filePath), "Application.cfc") {
			continue
		}

		if dotPath, ok := s.fileToPackageIn(filePath); ok {
			addPath(dotPath)
		}

		for _, dotPath := range s.mappingDotPaths(filePath) {
			addPath(dotPath)
		}
	}

	if len(paths) == 0 {
		return nil
	}

	slices.Sort(paths)

	var replaceRange *protocol.Range

	if ctx.quoteStart >= 0 {
		lineText := parser.LineTextAt(content, int(pos.Line))
		startCol := lineCol(lineText, ctx.quoteStart)
		replaceRange = &protocol.Range{
			Start: protocol.Position{Line: pos.Line, Character: startCol},
			End:   pos,
		}
	}

	items := make([]protocol.CompletionItem, 0, len(paths))
	for _, dotPath := range paths {
		item := protocol.CompletionItem{
			Label:    dotPath,
			Kind:     protocol.CompletionItemKindClass,
			Detail:   optStr("component"),
			SortText: optStr(SortProperties + dotPath),
		}
		if replaceRange != nil {
			item.TextEdit = &protocol.TextEdit{
				Range:   *replaceRange,
				NewText: dotPath,
			}
		}

		items = append(items, item)
	}

	return items
}

func (s *Server) mappingDotPaths(filePath string) []string {
	if len(s.Mappings) == 0 {
		return nil
	}

	slashFile := filepath.ToSlash(filePath)

	var out []string

	for name, target := range s.Mappings {
		cleanTarget := strings.TrimRight(filepath.ToSlash(target), "/")
		if cleanTarget == "" || !strings.HasPrefix(slashFile, cleanTarget+"/") {
			continue
		}

		rel := slashFile[len(cleanTarget)+1:]
		rel = strings.TrimSuffix(rel, filepath.Ext(rel))

		prefix := strings.Trim(strings.ReplaceAll(name, "\\", "/"), "/")
		if prefix != "" {
			rel = prefix + "/" + rel
		}

		out = append(out, strings.ReplaceAll(rel, "/", "."))
	}

	return out
}
