package server

import (
	"context"
	"encoding/json/v2"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
)

var linkedTagWordPattern = `[a-zA-Z0-9_:-]+`

// handleLinkedEditingRange answers textDocument/linkedEditingRange: when the
// cursor is on the name of a paired CFML or HTML opening/closing tag, returns
// both tag-name ranges so the editor edits them simultaneously.
func (s *Server) handleLinkedEditingRange(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.LinkedEditingRangeParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	content, ok := s.getDocument(params.TextDocument.URI)
	if !ok || content == "" {
		return nil, nil
	}

	lines := parser.NewLineTable(content)

	lineStart, ok := lines.Start(int(params.Position.Line))
	if !ok {
		return nil, nil
	}

	cursor := lineStart + byteCol(content, int(params.Position.Line), params.Position.Character)

	openSpan, closeSpan, found := findLinkedTagPair(content, cursor)
	if !found {
		return nil, nil
	}

	posOf := rangePositioner(content, lines)

	return &protocol.LinkedEditingRanges{
		Ranges: []protocol.Range{
			{Start: posOf(openSpan[0]), End: posOf(openSpan[1])},
			{Start: posOf(closeSpan[0]), End: posOf(closeSpan[1])},
		},
		WordPattern: &linkedTagWordPattern,
	}, nil
}

type linkedTagOpen struct {
	lowerName string
	nameStart int
	nameEnd   int
}

// findLinkedTagPair performs a single-pass O(N) tag match over tag regions in
// content and returns the byte spans [start, end] of the opening and closing
// tag names if cursor lies on either name.
func findLinkedTagPair(content string, cursor int) (openName, closeName [2]int, ok bool) {
	var stack []linkedTagOpen

	openCount := make(map[string]int)

	for _, r := range parser.ClassifyRegions(content) {
		if r.Kind != parser.RegionTag {
			continue
		}

		if o, c, found := scanLinkedTagRegion(r.Text, r.Offset, cursor, &stack, openCount); found {
			return o, c, true
		}
	}

	return [2]int{}, [2]int{}, false
}

func scanLinkedTagRegion(text string, base, cursor int, stack *[]linkedTagOpen, openCount map[string]int) (openName, closeName [2]int, ok bool) {
	for i := 0; i < len(text); {
		lt := strings.IndexByte(text[i:], '<')
		if lt < 0 {
			break
		}

		tagStart := i + lt
		if next, skipped := skipMarkupComment(text, tagStart); skipped {
			if next < 0 {
				break
			}

			i = next

			continue
		}

		if tagStart+1 >= len(text) || text[tagStart+1] == '!' || text[tagStart+1] == '?' {
			i = tagStart + 1

			continue
		}

		isClose := text[tagStart+1] == '/'
		nameStart := tagStart + 1

		if isClose {
			nameStart++
		}

		if nameStart >= len(text) || !isTagNameStartByte(text[nameStart]) {
			i = tagStart + 1

			continue
		}

		nameEnd := nameStart + 1
		for nameEnd < len(text) && isTagNamePartByte(text[nameEnd]) {
			nameEnd++
		}

		tagEnd := findExprTagClose(text, nameEnd)
		if tagEnd < 0 {
			break
		}

		lower := strings.ToLower(text[nameStart:nameEnd])
		absStart := base + nameStart
		absEnd := base + nameEnd

		if !isClose {
			selfClosed := tagEnd > nameEnd && text[tagEnd-1] == '/'
			if !selfClosed && !isLinkedVoidTag(lower) {
				*stack = append(*stack, linkedTagOpen{lowerName: lower, nameStart: absStart, nameEnd: absEnd})
				openCount[lower]++
			}

			i = tagEnd + 1

			continue
		}

		if opener, matched := popLinkedOpen(stack, openCount, lower); matched {
			onOpen := cursor >= opener.nameStart && cursor <= opener.nameEnd
			onClose := cursor >= absStart && cursor <= absEnd

			if onOpen || onClose {
				return [2]int{opener.nameStart, opener.nameEnd}, [2]int{absStart, absEnd}, true
			}
		}

		i = tagEnd + 1
	}

	return [2]int{}, [2]int{}, false
}

func skipMarkupComment(text string, tagStart int) (int, bool) {
	if strings.HasPrefix(text[tagStart:], "<!---") {
		if end := strings.Index(text[tagStart+5:], "--->"); end >= 0 {
			return tagStart + 5 + end + 4, true
		}

		return -1, true
	}

	if strings.HasPrefix(text[tagStart:], "<!--") {
		if end := strings.Index(text[tagStart+4:], "-->"); end >= 0 {
			return tagStart + 4 + end + 3, true
		}

		return -1, true
	}

	return 0, false
}

func popLinkedOpen(stack *[]linkedTagOpen, openCount map[string]int, lower string) (linkedTagOpen, bool) {
	if openCount[lower] <= 0 {
		return linkedTagOpen{}, false
	}

	s := *stack

	for j, opener := range slices.Backward(s) {
		if opener.lowerName == lower {
			for k := j; k < len(s); k++ {
				openCount[s[k].lowerName]--
			}

			*stack = s[:j]

			return opener, true
		}
	}

	return linkedTagOpen{}, false
}

func isTagNameStartByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isTagNamePartByte(c byte) bool {
	return isTagNameStartByte(c) || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == ':'
}

func isLinkedVoidTag(lower string) bool {
	switch lower {
	case "cfset", "cfparam", "cfargument", "cfreturn", "cfinclude", "cfabort",
		"cfbreak", "cfcontinue", "cfexit", "cfthrow", "cfrethrow", "cfelse",
		"cfelseif", "cfproperty", "cfimport", "cfdump", "cflocation", "cfheader",
		"cfcookie", "cfcontent", "cflog", "cfqueryparam", "cfprocparam", "cfhttpparam",
		"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta",
		"param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}
