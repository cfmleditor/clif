package server

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// quickFixActions produces concrete CodeActionKindQuickFix edits for diagnostics
// reported on the current range (structureDiagnostics and CFLint MISSING_VAR).
func (s *Server) quickFixActions(docURI uri.URI, content string, diags []protocol.Diagnostic) []protocol.CodeAction {
	lines := strings.Split(content, "\n")

	var out []protocol.CodeAction

	for i := range diags {
		d := diags[i]
		code := diagCodeString(d.Code)

		switch code {
		case "duplicate-argument":
			if act, ok := s.fixDuplicateArgument(docURI, lines, &d); ok {
				out = append(out, act)
			}

		case "duplicate-attribute":
			if act, ok := s.fixDuplicateAttribute(docURI, lines, &d); ok {
				out = append(out, act)
			}

		case "MISSING_VAR":
			out = append(out, s.fixMissingVar(docURI, lines, &d)...)
		}
	}

	return out
}

var quickFixKind = protocol.CodeActionKindQuickFix

func diagCodeString(code any) string {
	if s, ok := code.(protocol.String); ok {
		return string(s)
	}

	if s, ok := code.(string); ok {
		return s
	}

	return ""
}

func (s *Server) fixDuplicateArgument(docURI uri.URI, lines []string, d *protocol.Diagnostic) (protocol.CodeAction, bool) {
	lineIdx := int(d.Range.Start.Line)
	if lineIdx < 0 || lineIdx >= len(lines) {
		return protocol.CodeAction{}, false
	}

	trimmed := strings.TrimSpace(lines[lineIdx])
	if !strings.HasPrefix(strings.ToLower(trimmed), "<cfargument") || !strings.HasSuffix(trimmed, ">") {
		return protocol.CodeAction{}, false
	}

	delRange := protocol.Range{
		Start: protocol.Position{Line: d.Range.Start.Line, Character: 0},
		End:   protocol.Position{Line: d.Range.Start.Line + 1, Character: 0},
	}
	if lineIdx == len(lines)-1 {
		delRange.End = protocol.Position{
			Line:      d.Range.Start.Line,
			Character: conv.Uint32(lex.UTF16Len(lines[lineIdx])),
		}
	}

	preferred := true

	return protocol.CodeAction{
		Title:       "Remove duplicate <cfargument>",
		Kind:        &quickFixKind,
		Diagnostics: []protocol.Diagnostic{*d},
		IsPreferred: &preferred,
		Edit:        s.documentEdit(docURI, []protocol.TextEdit{{Range: delRange, NewText: ""}}),
	}, true
}

func (s *Server) fixDuplicateAttribute(docURI uri.URI, lines []string, d *protocol.Diagnostic) (protocol.CodeAction, bool) {
	lineIdx := int(d.Range.Start.Line)
	if lineIdx < 0 || lineIdx >= len(lines) || d.Range.Start.Line != d.Range.End.Line {
		return protocol.CodeAction{}, false
	}

	delRange := d.Range
	lineText := lines[lineIdx]
	startByte := byteCol(lineText, 0, delRange.Start.Character)

	if startByte > 0 && startByte <= len(lineText) && lineText[startByte-1] == ' ' {
		delRange.Start.Character--
	}

	preferred := true

	return protocol.CodeAction{
		Title:       "Remove duplicate attribute",
		Kind:        &quickFixKind,
		Diagnostics: []protocol.Diagnostic{*d},
		IsPreferred: &preferred,
		Edit:        s.documentEdit(docURI, []protocol.TextEdit{{Range: delRange, NewText: ""}}),
	}, true
}

func (s *Server) fixMissingVar(docURI uri.URI, lines []string, d *protocol.Diagnostic) []protocol.CodeAction {
	lineIdx := int(d.Range.Start.Line)
	if lineIdx < 0 || lineIdx >= len(lines) {
		return nil
	}

	lineText := lines[lineIdx]
	insertByte, ok := missingVarInsertOffset(lineText)

	if !ok {
		return nil
	}

	col := conv.Uint32(lex.UTF16Len(lineText[:insertByte]))
	pos := protocol.Position{Line: d.Range.Start.Line, Character: col}
	insRange := protocol.Range{Start: pos, End: pos}
	preferred := true

	return []protocol.CodeAction{
		{
			Title:       "Add `var` declaration",
			Kind:        &quickFixKind,
			Diagnostics: []protocol.Diagnostic{*d},
			IsPreferred: &preferred,
			Edit:        s.documentEdit(docURI, []protocol.TextEdit{{Range: insRange, NewText: "var "}}),
		},
		{
			Title:       "Prefix with `local.`",
			Kind:        &quickFixKind,
			Diagnostics: []protocol.Diagnostic{*d},
			Edit:        s.documentEdit(docURI, []protocol.TextEdit{{Range: insRange, NewText: "local."}}),
		},
	}
}

// missingVarInsertOffset finds the byte offset of the assigned variable name
// in either a CFScript assignment (`foo = ...`) or a `<cfset foo = ...>` tag,
// returning false if the variable is already `var`-declared or scoped.
func missingVarInsertOffset(lineText string) (int, bool) {
	i := 0
	for i < len(lineText) && (lineText[i] == ' ' || lineText[i] == '\t') {
		i++
	}

	if i >= len(lineText) {
		return 0, false
	}

	// Tag form: `<cfset foo = ...>`.
	if strings.HasPrefix(strings.ToLower(lineText[i:]), "<cfset") && i+6 < len(lineText) && isTagWhitespace(lineText[i+6]) {
		i += 6
		for i < len(lineText) && (lineText[i] == ' ' || lineText[i] == '\t') {
			i++
		}
	}

	if i >= len(lineText) || !lex.IsWordStart(lineText[i]) {
		return 0, false
	}

	identStart := i
	for i < len(lineText) && isSemIdentByte(lineText[i]) {
		i++
	}

	ident := lineText[identStart:i]
	if strings.EqualFold(ident, "var") {
		return 0, false
	}

	// Skip whitespace after identifier; must be followed by `=` (not `.`, `==`, `(`).
	j := i
	for j < len(lineText) && (lineText[j] == ' ' || lineText[j] == '\t') {
		j++
	}

	if j >= len(lineText) || lineText[j] != '=' || (j+1 < len(lineText) && lineText[j+1] == '=') {
		return 0, false
	}

	return identStart, true
}
