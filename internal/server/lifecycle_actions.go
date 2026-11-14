package server

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
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
func (s *Server) lifecycleActions(docURI uri.URI, content string) []protocol.CodeAction {
	if !strings.EqualFold(filepath.Base(cfpath.FromURI(string(docURI))), "Application.cfc") {
		return nil
	}

	end, tags, ok := parser.ComponentBodyEnd(content)
	if !ok {
		return nil
	}

	declared := declaredFunctionNames(content, s.parseResultOf(docURI))

	var missing []lifecycleStub

	for _, st := range lifecycleStubs {
		if !declared[strings.ToLower(st.name)] {
			missing = append(missing, st)
		}
	}

	if len(missing) == 0 {
		return nil
	}

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

	return actions
}

// parseResultOf is docURI's parse, or nil.
func (s *Server) parseResultOf(docURI uri.URI) *parser.ParseResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.parseResults[docURI]
}

// declaredFunctionNames are the lowercased names of the functions content
// declares, from pr when it is the parse of content, else from a fresh parse.
func declaredFunctionNames(content string, pr *parser.ParseResult) map[string]bool {
	if pr == nil || pr.Content != content {
		pr = parser.ParseWithOptions("", content, &parser.ParseOptions{Shallow: true})
	}

	out := make(map[string]bool, len(pr.Funcs))
	for i := range pr.Funcs {
		out[strings.ToLower(pr.Funcs[i].Name)] = true
	}

	return out
}

// insertStubs is the edit that adds stubs before the end of the component's
// body at end: on a line of their own, indented as the body's functions are
// and set apart by blank lines.
func insertStubs(content string, end int, tags bool, stubs []lifecycleStub) protocol.TextEdit {
	var body strings.Builder

	for i, st := range stubs {
		if i > 0 {
			body.WriteString("\n\n")
		}

		text := st.script
		if tags {
			text = st.tags
		}

		body.WriteString(indentLines(text, "\t"))
	}

	// The closing line starts at the closer when nothing but blanks stands
	// before it; otherwise the stubs go on lines of their own before the closer.
	lineStart := strings.LastIndexByte(content[:end], '\n') + 1
	atLineStart := strings.TrimLeft(content[lineStart:end], " \t") == ""

	at, text := end, "\n"+body.String()+"\n"
	if atLineStart {
		at, text = lineStart, body.String()+"\n"
		// A blank line before the stubs, unless there already is one.
		if prev := strings.TrimRight(content[:lineStart], " \t\n"); prev != "" && !strings.HasSuffix(content[:lineStart], "\n\n") {
			text = "\n" + text
		}
	}

	lines := parser.NewLineTable(content)
	line := lines.LineOf(at)
	start, _ := lines.Start(line)
	pos := protocol.Position{Line: conv.Uint32(line), Character: newColMapper(content).col(conv.Uint32(line), conv.Uint32(at-start))}

	return protocol.TextEdit{Range: protocol.Range{Start: pos, End: pos}, NewText: text}
}

// indentLines prefixes every non-blank line of text with indent.
func indentLines(text, indent string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = indent + l
		}
	}

	return strings.Join(lines, "\n")
}
