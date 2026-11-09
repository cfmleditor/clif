// Completion inside and around tags: attribute values, closing tags, tag names and the > that closes one.

package server

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// attrValueItems offers the documented values of the attribute the cursor is
// inside.
func attrValueItems(content string, line, char int, tagName string) []protocol.CompletionItem {
	attrName := parser.FindCurrentAttr(content, line, char)
	if attrName == "" || tagName == "" {
		return nil
	}

	attrs := docs.TagParams(tagName)
	if attrs == nil {
		attrs = docs.HTMLTagParams(tagName)
	}

	var items []protocol.CompletionItem

	for i := range attrs {
		if !strings.EqualFold(attrs[i].Name, attrName) { // attrName is lowercase ASCII already
			continue
		}

		for _, v := range attrs[i].ParamValues() {
			items = append(items, protocol.CompletionItem{
				SortText: optStr(SortProperties),
				Label:    v,
				Kind:     protocol.CompletionItemKindValue,
				Detail:   optStr(attrName + " value"),
			})
		}

		break
	}

	return items
}

// trailingGtAt finds a '>' after the cursor with only whitespace before it,
// which a close-tag completion replaces rather than doubles. It returns the
// byte column just past it and the same column as an LSP character, or -1.
func trailingGtAt(content string, line uint32, char int) (int, uint32) {
	lineStart := 0
	for range line {
		idx := strings.IndexByte(content[lineStart:], '\n')
		if idx < 0 {
			lineStart = len(content)

			break
		}

		lineStart += idx + 1
	}

	lineEnd := strings.IndexByte(content[lineStart:], '\n')
	if lineEnd < 0 {
		lineEnd = len(content) - lineStart
	}

	lineText := content[lineStart : lineStart+lineEnd]
	if char >= len(lineText) {
		return -1, 0
	}

	after := lineText[char:]
	if idx := strings.IndexByte(after, '>'); idx != -1 && strings.TrimSpace(after[:idx]) == "" {
		gt := char + idx + 1

		return gt, lineCol(lineText, gt)
	}

	return -1, 0
}

// closeTagItems offers to close each tag still open at the cursor.
func (s *Server) closeTagItems(content string, hasDoc bool, docURI uri.URI, pos protocol.Position, char int) []protocol.CompletionItem {
	trailingGt := -1

	var trailingGtCol uint32 // trailingGt as an LSP character

	if hasDoc {
		trailingGt, trailingGtCol = trailingGtAt(content, pos.Line, char)
	}

	var items []protocol.CompletionItem

	tags := make(map[string]int)

	for i, tag := range s.findUnclosedTagsScoped(content, docURI, int(pos.Line), char) {
		if _, ok := tags[tag]; ok {
			continue
		}

		item := protocol.CompletionItem{
			Label:  tag,
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: optStr("Close tag"),

			SortText: optStr(fmt.Sprintf("%04d", i)),
		}
		if trailingGt >= 0 {
			item.TextEdit = &protocol.TextEdit{
				Range: protocol.Range{
					Start: pos,
					End:   protocol.Position{Line: pos.Line, Character: trailingGtCol},
				},
				NewText: tag + ">",
			}
		} else {
			item.InsertText = optStr(tag + ">")
		}

		items = append(items, item)
		tags[tag] = len(items)
	}

	return items
}

// tagAttributeItems offers the attributes of the tag the cursor is inside.
func tagAttributeItems(tagName string) []protocol.CompletionItem {
	attrs := docs.TagParams(tagName)
	if attrs == nil {
		attrs = docs.HTMLTagParams(tagName)
	}

	items := make([]protocol.CompletionItem, 0, len(attrs))

	for i := range attrs {
		p := &attrs[i]

		items = append(items, protocol.CompletionItem{
			SortText: optStr(SortProperties),
			Label:    p.Name,
			Kind:     protocol.CompletionItemKindProperty,
			Detail:   optStr(p.Description),

			InsertText: optStr(p.Name + `="$1"`),

			InsertTextFormat: protocol.InsertTextFormatSnippet,
		})
	}

	return items
}

// cfelseifItem offers to turn a `<cfelse` being typed into `<cfelseif`.
func cfelseifItem(content string, pos protocol.Position, char int) protocol.CompletionItem {
	// Replace from the "<" that opened the tag to the cursor. The "<" is not
	// necessarily on the cursor's line — `<cfelse` with the cursor on the
	// following line is an ordinary mid-edit state — so its position has to
	// be computed rather than derived by subtracting a length from the
	// cursor's column. Doing the latter underflowed uint32 and produced a
	// range starting at character 4294967288, after its own end.
	textBefore := parser.TextBeforeCursor(content, int(pos.Line), char)
	openLine, openChar := pos.Line, pos.Character

	if open := strings.LastIndex(textBefore, "<"); open >= 0 {
		l, c := parser.PositionAt(content, open)
		openLine, openChar = conv.Uint32(l), lineCol(parser.LineTextAt(content, l), c)
	}

	return protocol.CompletionItem{
		SortText: optStr(SortProperties),
		Label:    "if",
		Kind:     protocol.CompletionItemKindKeyword,
		Detail:   optStr("Convert to cfelseif"),

		FilterText: optStr("if"),

		InsertTextFormat: protocol.InsertTextFormatSnippet,
		TextEdit: &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: openLine, Character: openChar},
				End:   pos,
			},
			NewText: "<cfelseif $1",
		},
	}
}

// tagNameItems offers every CFML tag, as a snippet, and every HTML tag.
func tagNameItems() []protocol.CompletionItem {
	cfTags, tags := docs.AllTags(), docs.HTMLTags()
	items := make([]protocol.CompletionItem, 0, len(cfTags)+len(tags))

	for _, tag := range cfTags {
		items = append(items, protocol.CompletionItem{
			Label:    tag.Name,
			SortText: optStr(SortTags + tag.Name),

			Kind:   protocol.CompletionItemKindKeyword,
			Detail: optStr(tag.Description),

			InsertText: optStr(buildTagSnippet(tag)),

			InsertTextFormat: protocol.InsertTextFormatSnippet,
		})
	}

	for i := range tags {
		tag := &tags[i]

		items = append(items, protocol.CompletionItem{
			Label:    tag.Name,
			SortText: optStr(SortTags + tag.Name),

			Kind:   protocol.CompletionItemKindKeyword,
			Detail: optStr(tag.Description),
		})
	}

	return items
}

// findUnclosedTagsScoped scans for unclosed tags within the enclosing function body,
// falling back to the full file if the cursor is outside any function.
func (s *Server) findUnclosedTagsScoped(content string, docURI uri.URI, line, char int) []string {
	s.mu.RLock()
	funcs := s.funcRanges[docURI]
	s.mu.RUnlock()

	startLine := 0
	idx := sort.Search(len(funcs), func(i int) bool {
		return funcs[i].End >= line
	})

	if idx < len(funcs) && line >= funcs[idx].Start {
		startLine = funcs[idx].Start
	}

	return parser.FindUnclosedTags(content, startLine, line, char)
}

// duplicateGtCompletion offers to remove a duplicate '>' when the user types
// '>' immediately after an existing tag-closing '>'.
func duplicateGtCompletion(content string, line, char int) (protocol.CompletionItem, bool) {
	if char < 2 {
		return protocol.CompletionItem{}, false
	}

	// One line, taken directly. Splitting the document to read a single row of
	// it costs a string header per line — 516KB on a 32,000-line component, on
	// a path that runs every time a '>' is typed, alongside two more doing the
	// same thing. A line past the end comes back empty, which the length guard
	// below rejects as it would an empty line.
	lineText := parser.LineTextAt(content, line)
	if char > len(lineText) || lineText[char-2] != '>' {
		return protocol.CompletionItem{}, false
	}
	// Verify the previous '>' closes a tag.
	before := lineText[:char-1]

	openIdx := strings.LastIndexByte(before, '<')
	if openIdx == -1 {
		return protocol.CompletionItem{}, false
	}

	if strings.ContainsRune(before[openIdx:len(before)-1], '>') {
		return protocol.CompletionItem{}, false
	}

	return protocol.CompletionItem{
		Label:  ">",
		Kind:   protocol.CompletionItemKindKeyword,
		Detail: optStr("Remove duplicate >"),

		SortText: optStr(SortCloseTags),

		FilterText: optStr(">"),

		TextEdit: &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: conv.Uint32(line), Character: lineCol(lineText, char-1)},
				End:   protocol.Position{Line: conv.Uint32(line), Character: lineCol(lineText, char)},
			},
			NewText: "",
		},
	}, true
}

// closeTagCompletion returns a completion item when '>' is typed mid-tag
// with non-whitespace content between the typed '>' and the tag's existing '>'.
// The completion moves the content before the '>' and removes the duplicate.
func closeTagCompletion(content string, line, char int) (protocol.CompletionItem, bool) {
	// One line, for the reason duplicateGtCompletion gives.
	lineText := parser.LineTextAt(content, line)
	if char >= len(lineText) {
		return protocol.CompletionItem{}, false
	}

	// Find next '>' after cursor.
	rest := lineText[char:]

	idx := strings.IndexByte(rest, '>')
	if idx == -1 {
		return protocol.CompletionItem{}, false
	}

	middle := rest[:idx]

	// Only offer completion when there's non-whitespace content.
	if strings.TrimSpace(middle) == "" {
		return protocol.CompletionItem{}, false
	}

	// Must not contain '<' (would mean we left the tag).
	if strings.ContainsRune(middle, '<') {
		return protocol.CompletionItem{}, false
	}

	// Verify we're inside a tag.
	before := lineText[:char]

	openIdx := strings.LastIndexByte(before, '<')
	if openIdx == -1 {
		return protocol.CompletionItem{}, false
	}

	if strings.ContainsRune(lineText[openIdx:char-1], '>') {
		return protocol.CompletionItem{}, false
	}

	endChar := char + idx + 1

	return protocol.CompletionItem{
		Label:  ">",
		Kind:   protocol.CompletionItemKindKeyword,
		Detail: optStr("Close tag"),

		SortText: optStr(SortCloseTags),

		FilterText: optStr(">"),

		InsertTextFormat: protocol.InsertTextFormatSnippet,
		TextEdit: &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: conv.Uint32(line), Character: lineCol(lineText, char-1)},
				End:   protocol.Position{Line: conv.Uint32(line), Character: lineCol(lineText, endChar)},
			},
			NewText: middle + ">",
		},
	}, true
}
