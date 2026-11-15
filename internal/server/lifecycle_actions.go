package server

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// lifecycleStub is one of the functions an Application.cfc may declare, as
// script and as tags.
type lifecycleStub struct {
	name   string
	script string // the declaration, its body indented one level
	tags   string
}

var lifecycleStubs = []lifecycleStub{
	{
		"onApplicationStart", "public boolean function onApplicationStart() {\n\treturn true;\n}",
		`<cffunction name="onApplicationStart" access="public" returntype="boolean">` + "\n\t<cfreturn true>\n</cffunction>",
	},
	{
		"onApplicationEnd", "public void function onApplicationEnd( struct applicationScope = {} ) {\n\treturn;\n}",
		`<cffunction name="onApplicationEnd" access="public" returntype="void">` + "\n\t" + `<cfargument name="applicationScope" type="struct" required="false" default="#{}#">` + "\n</cffunction>",
	},
	{
		"onSessionStart", "public void function onSessionStart() {\n\treturn;\n}",
		`<cffunction name="onSessionStart" access="public" returntype="void">` + "\n</cffunction>",
	},
	{
		"onSessionEnd", "public void function onSessionEnd( required struct sessionScope, struct applicationScope = {} ) {\n\treturn;\n}",
		`<cffunction name="onSessionEnd" access="public" returntype="void">` + "\n\t" + `<cfargument name="sessionScope" type="struct" required="true">` + "\n\t" + `<cfargument name="applicationScope" type="struct" required="false" default="#{}#">` + "\n</cffunction>",
	},
	{
		"onRequestStart", "public boolean function onRequestStart( required string targetPage ) {\n\treturn true;\n}",
		`<cffunction name="onRequestStart" access="public" returntype="boolean">` + "\n\t" + `<cfargument name="targetPage" type="string" required="true">` + "\n\t<cfreturn true>\n</cffunction>",
	},
	{
		"onRequest", "public void function onRequest( required string targetPage ) {\n\tinclude arguments.targetPage;\n}",
		`<cffunction name="onRequest" access="public" returntype="void">` + "\n\t" + `<cfargument name="targetPage" type="string" required="true">` + "\n\t" + `<cfinclude template="#arguments.targetPage#">` + "\n</cffunction>",
	},
	{
		"onRequestEnd", "public void function onRequestEnd() {\n\treturn;\n}",
		`<cffunction name="onRequestEnd" access="public" returntype="void">` + "\n</cffunction>",
	},
	{
		"onCFCRequest", "public void function onCFCRequest( string cfcName, string method, struct args ) {\n\treturn;\n}",
		`<cffunction name="onCFCRequest" access="public" returntype="void">` + "\n\t" + `<cfargument name="cfcName" type="string">` + "\n\t" + `<cfargument name="method" type="string">` + "\n\t" + `<cfargument name="args" type="struct">` + "\n</cffunction>",
	},
	{
		"onError", "public void function onError( required any exception, required string eventName ) {\n\treturn;\n}",
		`<cffunction name="onError" access="public" returntype="void">` + "\n\t" + `<cfargument name="exception" type="any" required="true">` + "\n\t" + `<cfargument name="eventName" type="string" required="true">` + "\n</cffunction>",
	},
	{
		"onAbort", "public void function onAbort( required string targetPage ) {\n\treturn;\n}",
		`<cffunction name="onAbort" access="public" returntype="void">` + "\n\t" + `<cfargument name="targetPage" type="string" required="true">` + "\n</cffunction>",
	},
	{
		"onMissingTemplate", "public boolean function onMissingTemplate( required string targetPage ) {\n\treturn true;\n}",
		`<cffunction name="onMissingTemplate" access="public" returntype="boolean">` + "\n\t" + `<cfargument name="targetPage" type="string" required="true">` + "\n\t<cfreturn true>\n</cffunction>",
	},
}

// lifecycleActions are the code actions that add the application events an
// Application.cfc does not yet declare: one for each, and one for all of
// them. They are offered in an Application.cfc only, written as the component
// is written, tags or script.
func (s *Server) lifecycleActions(docURI uri.URI, content string, at protocol.Range) []protocol.CodeAction {
	if !strings.EqualFold(filepath.Base(cfpath.FromURI(string(docURI))), "Application.cfc") {
		return nil
	}

	// Asked on every cursor move, and the answer is the text's: made once for
	// each version of it.
	s.mu.RLock()
	cached, ok := s.lifecycleCache[docURI]
	s.mu.RUnlock()

	if !ok || cached.content != content {
		// Read from the text alone: nothing here waits on the parse an edit
		// defers.
		cached = lifecycleCached{content: content}
		cached.actions, cached.lines = s.buildLifecycleActions(docURI, content)
	}

	// Kept while the document is open, decided under the lock a close takes
	// the document away under, so one racing a close does not outlive it.
	s.mu.Lock()
	if _, open := s.documents[docURI]; open {
		s.lifecycleCache[docURI] = cached
	}
	s.mu.Unlock()

	// Offered where the component is declared or closed, not on every line.
	for _, line := range cached.lines {
		if int(at.Start.Line) <= line && line <= int(at.End.Line) {
			return cached.actions
		}
	}

	return nil
}

// lifecycleCached is the lifecycle actions of one version of a document.
type lifecycleCached struct {
	content string
	actions []protocol.CodeAction
	lines   []int // the lines they are offered on: the component's declaration and its end
}

// buildLifecycleActions is lifecycleActions for an Application.cfc, uncached.
func (s *Server) buildLifecycleActions(docURI uri.URI, content string) ([]protocol.CodeAction, []int) {
	declared := declaredFunctionNames(content)

	var missing []lifecycleStub

	for _, st := range lifecycleStubs {
		if !declared[strings.ToLower(st.name)] {
			missing = append(missing, st)
		}
	}

	if len(missing) == 0 {
		return nil, nil
	}

	// The body's end is a scan of the file: only an application that lacks an
	// event is worth it.
	end, tags, ok := parser.ComponentBodyEnd(content)
	if !ok {
		return nil, nil
	}

	lines := parser.NewLineTable(content)

	openAt, textEnd, _ := parser.ComponentDeclarationBounds(content)
	where := []int{lines.LineOf(openAt), lines.LineOf(textEnd), lines.LineOf(end)}

	var actions []protocol.CodeAction

	add := func(title string, stubs []lifecycleStub) {
		kind := protocol.CodeActionKindRefactor

		actions = append(actions, protocol.CodeAction{
			Title: title,
			Kind:  &kind,
			Edit: &protocol.WorkspaceEdit{Changes: map[uri.URI][]protocol.TextEdit{
				docURI: {insertStubs(content, end, tags, stubs)},
			}},
		})
	}

	if len(missing) > 1 {
		add("Add all missing application events to Application.cfc", missing)
	}

	for _, st := range missing {
		add("Add "+st.name+"() to Application.cfc", []lifecycleStub{st})
	}

	return actions, where
}

// parseResultOf is docURI's parse, or nil.
func (s *Server) parseResultOf(docURI uri.URI) *parser.ParseResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.parseResults[docURI]
}

// declaredFunctionNames are the lowercased names content declares as events:
// a function (`function onError(`, or a <cffunction name>) or a property set to
// one (`this.onError = function( e ) { ... }`). They are read from the tokens in
// one pass, so a comment or a string that spells one declares nothing, and
// nothing waits on the parse an edit defers.
func declaredFunctionNames(content string) map[string]bool {
	out := map[string]bool{}

	if parser.IndexFold(content, "<cffunction") >= 0 {
		for _, tag := range parser.CFTags(content, "cffunction") {
			if name, ok := tag.Attr("name"); ok && !tag.Close {
				out[strings.ToLower(name)] = true
			}
		}
	}

	// A window of the last five tokens read, the oldest first: the tokens are
	// not all held.
	var win [5]parser.Token

	n := 0

	sc := parser.NewSourceScanner(content)
	for t := sc.NextSkipComments(); ; t = sc.NextSkipComments() {
		copy(win[:], win[1:])
		win[len(win)-1] = t

		n++

		markDeclared(out, win[:], n)

		if t.Kind == parser.TokEOF {
			break
		}
	}

	return out
}

// insertStubs is the edit that adds stubs before the end of the component's
// body at end: on a line of their own, indented as the body's functions are
// and set apart by blank lines.
func insertStubs(content string, end int, tags bool, stubs []lifecycleStub) protocol.TextEdit {
	var body strings.Builder

	unit := indentUnit(content[:end])

	for i, st := range stubs {
		if i > 0 {
			body.WriteString("\n\n")
		}

		text := st.script
		if tags {
			text = st.tags
		}

		body.WriteString(indentLines(text, unit))
	}

	// The closing line starts at the closer when nothing but blanks stands
	// before it; otherwise the stubs go on lines of their own before the closer.
	lineStart := strings.LastIndexByte(content[:end], '\n') + 1
	atLineStart := strings.TrimLeft(content[lineStart:end], " \t") == ""

	// A closer that follows code on its line is moved to the line after the
	// stubs, which are set apart from that code by a blank line.
	at, text := end, "\n\n"+body.String()+"\n"
	if atLineStart {
		at, text = lineStart, body.String()+"\n"
		// A blank line before the stubs, unless there already is one: the line
		// before is blank when only blanks stand on it.
		before := strings.ReplaceAll(content[:lineStart], "\r\n", "\n")
		lastLine := strings.TrimSuffix(before, "\n")
		lastLine = lastLine[strings.LastIndexByte(lastLine, '\n')+1:]

		if strings.TrimSpace(before) != "" && strings.TrimSpace(lastLine) != "" {
			text = "\n" + text
		}
	}

	// A file written with CRLF line endings is given them.
	if strings.Contains(content, "\r\n") {
		text = strings.ReplaceAll(text, "\n", "\r\n")
	}

	pos := rangePositioner(content, parser.NewLineTable(content))(at)

	return protocol.TextEdit{Range: protocol.Range{Start: pos, End: pos}, NewText: text}
}

// indentUnit is the indentation of the file's functions: the leading whitespace
// of the first line that declares one (`function`, a modifier before it, or
// <cffunction>), as the stubs are functions that stand beside them. A file with
// none is indented as its first indented line is, or with a tab.
func indentUnit(content string) string {
	var fallback string

	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSuffix(line, "\r")

		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || isCommentLine(trimmed) {
			continue
		}

		lead := line[:len(line)-len(trimmed)]
		lower := strings.ToLower(trimmed)

		if strings.HasPrefix(lower, "<cffunction") || strings.HasPrefix(lower, "function ") || strings.Contains(lower, " function ") {
			return lead
		}

		if fallback == "" && lead != "" {
			fallback = lead
		}
	}

	if fallback != "" {
		return fallback
	}

	return "\t"
}

// nestedUnit is what a level of nesting inside a function is indented with, given
// the indentation of the function itself: the same when the file indents at all,
// a tab when its functions stand flush left.
func nestedUnit(unit string) string {
	if unit == "" {
		return "\t"
	}

	return unit
}

// indentLines prefixes every non-blank line of text with indent, and writes
// the tabs the text is indented with in the file's unit for a nested level.
func indentLines(text, indent string) string {
	nested := nestedUnit(indent)

	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if l == "" {
			continue
		}

		rest := strings.TrimLeft(l, "\t")
		lines[i] = indent + strings.Repeat(nested, len(l)-len(rest)) + rest
	}

	return strings.Join(lines, "\n")
}

// isCommentLine reports whether a line, with its indentation removed, is a
// comment or a doc comment's line: its blanks are not the code's indentation.
func isCommentLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "<!--")
}

// markDeclared adds to out what the window win, the last five tokens read (n
// read in all) ends the declaration of: `function name(`, or `this.name =` (and
// not `==`), each looked for where the token before it was not a `.`.
func markDeclared(out map[string]bool, win []parser.Token, n int) {
	// win[4] is the newest. `this . name =` is win[0..3] with win[4] after it.
	if n >= 5 {
		a, b, c, d, e := win[0], win[1], win[2], win[3], win[4]

		if a.Kind == parser.TokIdent && strings.EqualFold(a.Value, "this") &&
			b.Kind == parser.TokDot && c.Kind == parser.TokIdent && d.Kind == parser.TokEquals && e.Kind != parser.TokEquals {
			out[strings.ToLower(c.Value)] = true
		}
	}

	if n >= 3 {
		a, b, c := win[2], win[3], win[4]

		if a.Kind == parser.TokIdent && strings.EqualFold(a.Value, "function") && b.Kind == parser.TokIdent && c.Kind == parser.TokLParen {
			out[strings.ToLower(b.Value)] = true
		}
	}
}
