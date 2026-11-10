// Package lspdoc applies an LSP didChange to a document's text. The language
// server and the supervisor in front of it (internal/supervise) both keep each
// open document's text, and must arrive at the same bytes.
package lspdoc

import (
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
)

// ChangeRangeAndText extracts the range, text and whether the change is a
// whole-document replacement from a TextDocumentContentChangeEvent union value.
func ChangeRangeAndText(change protocol.TextDocumentContentChangeEvent) (r protocol.Range, text string, isFull bool) {
	switch c := change.(type) {
	// The text is the analysis's, whose lines end at a bare \r too
	// (lex.NormalizeLineBreaks); the range is the editor's, which already
	// counts them so.
	case *protocol.TextDocumentContentChangePartial:
		return c.Range, lex.NormalizeLineBreaks(c.Text), false
	case *protocol.TextDocumentContentChangeWholeDocument:
		return protocol.Range{}, lex.NormalizeLineBreaks(c.Text), true
	default:
		return protocol.Range{}, "", true
	}
}

// ApplyChanges applies a didChange's changes in one pass where it can (see
// parser.ApplyEdits). A batch is what an editor sends when it undoes a
// reformat, an edit per line, and applying those one at a time cost the whole
// document per edit. A whole-document change replaces everything before it.
func ApplyChanges(content string, changes []protocol.TextDocumentContentChangeEvent) string {
	var edits []parser.Edit

	for _, change := range changes {
		r, text, isFull := ChangeRangeAndText(change)
		if isFull {
			content, edits = text, edits[:0]

			continue
		}

		edits = append(edits, parser.Edit{
			StartLine: int(r.Start.Line), StartChar: int(r.Start.Character),
			EndLine: int(r.End.Line), EndChar: int(r.End.Character),
			Text: text,
		})
	}

	if len(edits) == 0 {
		return content
	}

	return parser.ApplyEdits(content, edits)
}
