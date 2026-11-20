package server

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func TestWorkflowCompletionResolve(t *testing.T) {
	root := t.TempDir()
	client := startWorkflowWithCapabilities(t, root, map[string]any{
		"textDocument": map[string]any{"completion": map[string]any{"completionItem": map[string]any{
			"resolveSupport": map[string]any{"properties": []string{"documentation", "detail"}},
		}}},
	})

	provider := client.capabilities.CompletionProvider
	if provider == nil || provider.ResolveProvider == nil || !*provider.ResolveProvider {
		t.Fatal("initialize did not advertise completion item resolution")
	}

	doc := client.open(filepath.Join(root, "list.cfm"), "<cfscript>\narrayLen(items);\n</cfscript>")

	var result protocol.CompletionList
	client.call(protocol.MethodTextDocumentCompletion, protocol.CompletionParams{
		TextDocument: doc,
		Position:     protocol.Position{Line: 1},
		Context:      protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
	}, &result)

	index := slices.IndexFunc(result.Items, func(item protocol.CompletionItem) bool {
		return strings.EqualFold(item.Label, "arrayLen")
	})
	if index < 0 {
		t.Fatal("completion did not include arrayLen")
	}

	item := result.Items[index]
	if _, hasDetail := item.Detail.Get(); hasDetail || item.Documentation != nil {
		t.Fatalf("expected deferred documentation and detail for %s", item.Label)
	}

	var resolved protocol.CompletionItem
	client.call(protocol.MethodCompletionItemResolve, item, &resolved)

	detail, ok := resolved.Detail.Get()
	if resolved.Label != item.Label || !ok || !strings.Contains(strings.ToLower(detail), "arraylen") || resolved.Documentation == nil {
		t.Fatalf("resolve did not fill in arrayLen documentation and detail: %+v", resolved)
	}
}

func TestWorkflowEditorQueries(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, filepath.Join(root, ".clif.json"), `{"features":{"inlayHints":true,"typeDefinition":true},"references":{"enabled":true}}`)
	model := filepath.Join(root, "Person.cfc")
	writeCFC(t, model, "component {}")

	const source = `component {
  function greet(required string name) {
    return "Hello " & arguments.name;
  }
  function run() {
    var person = new Person();
    return greet("Ada");
  }
}`

	path := filepath.Join(root, "Greeter.cfc")
	writeCFC(t, path, source)
	client := startWorkflow(t, root, false)
	doc := client.open(path, source)
	typeAt := protocol.Position{Line: 5, Character: 10}
	callAt := protocol.Position{Line: 6, Character: 12}
	td := protocol.TextDocumentPositionParams{TextDocument: doc, Position: callAt}

	for _, tc := range []struct {
		method string
		params any
		want   string
	}{
		{protocol.MethodTextDocumentTypeDefinition, protocol.TypeDefinitionParams{TextDocument: doc, Position: typeAt}, string(cfpath.ToURI(model))},
		{protocol.MethodTextDocumentHover, protocol.HoverParams{TextDocument: doc, Position: callAt}, "greet"},
		{protocol.MethodTextDocumentSignatureHelp, protocol.SignatureHelpParams{
			TextDocument: doc,
			Position:     protocol.Position{Line: 6, Character: 17},
		}, "name"},
		{protocol.MethodTextDocumentInlayHint, protocol.InlayHintParams{
			TextDocument: doc,
			Range:        protocol.Range{End: protocol.Position{Line: 8}},
		}, "Path: (Greeter)"},
		{protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
			TextDocument: doc,
			Range:        protocol.Range{Start: callAt, End: callAt},
		}, "clif.explainCall"},
	} {
		var result jsonrpc2.RawMessage
		client.call(tc.method, tc.params, &result)

		if !bytes.Contains(result, []byte(tc.want)) {
			t.Errorf("%s = %s, want %q", tc.method, result, tc.want)
		}
	}

	var highlights []protocol.DocumentHighlight
	client.call(protocol.MethodTextDocumentDocumentHighlight, protocol.DocumentHighlightParams{
		TextDocument: td.TextDocument,
		Position:     td.Position,
	}, &highlights)

	if len(highlights) != 2 || !slices.ContainsFunc(highlights, func(hit protocol.DocumentHighlight) bool {
		return hit.Range.Start.Line == 1 && hit.Range.Start.Character == 11
	}) {
		t.Errorf("highlights = %+v, want the declaration and call", highlights)
	}

	var refs []protocol.Location
	client.call(protocol.MethodTextDocumentReferences, protocol.ReferenceParams{
		TextDocument: doc,
		Position:     protocol.Position{Line: 1, Character: 12},
		Context:      protocol.ReferenceContext{IncludeDeclaration: false},
	}, &refs)

	want := protocol.Location{URI: doc.URI, Range: protocol.Range{
		Start: protocol.Position{Line: 6, Character: 11},
		End:   protocol.Position{Line: 6, Character: 16},
	}}
	if !slices.Equal(refs, []protocol.Location{want}) {
		t.Errorf("references = %+v, want %+v", refs, want)
	}
}

func TestWorkflowFormattingSelectionAndOnType(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, filepath.Join(root, ".clif.json"), `{"formatting":{"enabled":true,"whitespaceOnly":true}}`)
	client := startWorkflow(t, root, false)

	const source = "component {\nfunction first() {\nreturn 1;\n}\nfunction second() {\nreturn 2;\n}\n}\n"

	doc := client.open(filepath.Join(root, "Layout.cfc"), source)
	opts := protocol.FormattingOptions{TabSize: 4, InsertSpaces: true}

	var edits []protocol.TextEdit
	client.call(protocol.MethodTextDocumentRangeFormatting, protocol.DocumentRangeFormattingParams{
		TextDocument: doc,
		Range:        protocol.Range{Start: protocol.Position{Line: 1}, End: protocol.Position{Line: 4}},
		Options:      opts,
	}, &edits)

	if len(edits) == 0 {
		t.Fatal("format selection returned no edits")
	}

	formatted := applyTextEdits(t, source, edits)
	if !strings.Contains(formatted, "\n        return 1;\n") || !strings.HasSuffix(formatted, "function second() {\nreturn 2;\n}\n}\n") {
		t.Errorf("format selection did not reindent only the selected function:\n%s", formatted)
	}

	const page = "<cfoutput>>Hello</cfoutput>"

	tag := client.open(filepath.Join(root, "hello.cfm"), page)
	client.call(protocol.MethodTextDocumentOnTypeFormatting, protocol.DocumentOnTypeFormattingParams{
		TextDocument: tag,
		Position:     protocol.Position{Character: 10},
		Ch:           ">",
		Options:      opts,
	}, &edits)

	if got := applyTextEdits(t, page, edits); got != "<cfoutput>Hello</cfoutput>" {
		t.Errorf("on-type result = %q, want a single closing angle bracket", got)
	}
}

func TestWorkflowDiagnosticsPublishAndClear(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, filepath.Join(root, ".clif.json"), `{"features":{"structureDiagnostics":true},"linting":{"enabled":false}}`)
	client := startWorkflowWithCapabilities(t, root, map[string]any{}, protocol.MethodTextDocumentPublishDiagnostics)

	const broken = "<cfif ready>\n<cfinclude template=\"chapter.cfm\">\n"

	doc := client.open(filepath.Join(root, "status.cfm"), broken)

	waitDiagnostics := func(codes ...string) {
		t.Helper()

		client.waitForEvent(protocol.MethodTextDocumentPublishDiagnostics, func(raw []byte) bool {
			var params protocol.PublishDiagnosticsParams
			if err := json.Unmarshal(raw, &params); err != nil {
				t.Fatal(err)
			}

			return params.URI == doc.URI && slices.Equal(diagnosticCodes(params.Diagnostics), codes)
		})
	}
	waitDiagnostics("unclosed-tag")

	fixed := broken + "</cfif>\n"
	client.notify(protocol.MethodTextDocumentDidChange, protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: doc, Version: 2},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			&protocol.TextDocumentContentChangeWholeDocument{Text: fixed},
		},
	})
	waitDiagnostics()

	writeCFC(t, filepath.Join(root, "status.cfm"), fixed)
	client.notify(protocol.MethodTextDocumentDidSave, protocol.DidSaveTextDocumentParams{TextDocument: doc})

	var links []protocol.DocumentLink
	client.call(protocol.MethodTextDocumentDocumentLink, protocol.DocumentLinkParams{TextDocument: doc}, &links)

	if len(links) != 1 {
		t.Fatalf("open document links = %v, want one include", links)
	}

	client.notify(protocol.MethodTextDocumentDidClose, protocol.DidCloseTextDocumentParams{TextDocument: doc})

	links = nil
	client.call(protocol.MethodTextDocumentDocumentLink, protocol.DocumentLinkParams{TextDocument: doc}, &links)

	if len(links) != 0 {
		t.Errorf("closed document still returned links: %v", links)
	}

	client.call(protocol.MethodShutdown, nil, nil)
	client.notify(protocol.MethodExit, nil)
}

func TestWorkflowWatchedFilesPublishIndexChanges(t *testing.T) {
	root := t.TempDir()
	client := startWorkflowWithCapabilities(t, root, map[string]any{"workspace": map[string]any{
		"didChangeWatchedFiles": map[string]bool{"dynamicRegistration": true},
	}}, protocol.MethodWindowLogMessage)
	client.waitForEvent(protocol.MethodWindowLogMessage, func(raw []byte) bool {
		return bytes.Contains(raw, []byte("indexing complete"))
	})

	path := filepath.Join(root, "External.cfc")

	for _, step := range []struct {
		kind protocol.FileChangeType
		name string
		want bool
	}{
		{protocol.FileChangeTypeCreated, "externalOld", true},
		{protocol.FileChangeTypeChanged, "externalNew", true},
		{protocol.FileChangeTypeDeleted, "externalNew", false},
	} {
		if step.want {
			writeCFC(t, path, "component { function "+step.name+"() {} }")
		} else if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}

		client.notify(protocol.MethodWorkspaceDidChangeWatchedFiles, protocol.DidChangeWatchedFilesParams{
			Changes: []protocol.FileEvent{{URI: cfpath.ToURI(path), Type: step.kind}},
		})
		client.waitForEvent(protocol.MethodWindowLogMessage, func(raw []byte) bool {
			return bytes.Contains(raw, []byte("watched files applied"))
		})

		if got := hasWorkflowSymbol(client, step.name); got != step.want {
			t.Errorf("workspace symbol %s after %v = %v, want %v", step.name, step.kind, got, step.want)
		}

		if step.kind == protocol.FileChangeTypeChanged && hasWorkflowSymbol(client, "externalOld") {
			t.Error("workspace symbols still include the function replaced on disk")
		}
	}
}

func TestWorkflowBackgroundReportsComplete(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, filepath.Join(root, ".clif.json"), `{"linting":{"enabled":false}}`)
	writeCFC(t, filepath.Join(root, "Tasks.cfc"), "component { function run() { workflowMissingTask(); } }")
	client := startWorkflowWithCapabilities(t, root, map[string]any{},
		protocol.MethodWindowShowMessage, protocol.MethodWindowLogMessage)
	client.waitForEvent(protocol.MethodWindowLogMessage, func(raw []byte) bool {
		return bytes.Contains(raw, []byte("indexing complete"))
	})

	for _, tc := range []struct {
		command string
		args    []any
		message string
		file    string
		content string
	}{
		{"clif.scanWorkspace", nil, "Scan complete: 0 parse errors in 1 files", "", ""},
		{
			"clif.generateCodeMap",
			[]any{map[string]any{"out": "workflow-map.json", "format": "json"}},
			"Code map:", "workflow-map.json", "Tasks.cfc",
		},
		{"clif.exportUnresolved", nil, "The unresolved report:", ".clif-unresolved.txt", "workflowMissingTask"},
	} {
		var result jsonrpc2.RawMessage
		client.call(protocol.MethodWorkspaceExecuteCommand, map[string]any{
			"command": tc.command, "arguments": tc.args,
		}, &result)

		if tc.file != "" && !bytes.Contains(result, []byte(`"started":true`)) {
			t.Fatalf("%s did not acknowledge the job: %s", tc.command, result)
		}

		client.waitForEvent(protocol.MethodWindowShowMessage, func(raw []byte) bool {
			var params protocol.ShowMessageParams
			if err := json.Unmarshal(raw, &params); err != nil {
				t.Fatal(err)
			}

			if params.Type == protocol.MessageTypeError {
				t.Fatalf("%s reported an error: %s", tc.command, params.Message)
			}

			return strings.Contains(params.Message, tc.message)
		})

		if tc.file != "" {
			data, err := os.ReadFile(filepath.Join(root, tc.file))
			if err != nil || !bytes.Contains(data, []byte(tc.content)) {
				t.Errorf("%s report = %q, error = %v, want %q", tc.command, data, err, tc.content)
			}
		}
	}
}

func TestWorkflowBackgroundReportsFailWithoutWedgingSession(t *testing.T) {
	root := t.TempDir()
	missingCFLint := filepath.Join(root, "missing-cflint")
	t.Setenv("CLIF_CFLINT", missingCFLint)
	writeCFC(t, filepath.Join(root, ".clif.json"), `{"linting":{"enabled":false}}`)

	client := startWorkflowWithCapabilities(t, root, map[string]any{},
		protocol.MethodWindowLogMessage, protocol.MethodWindowShowMessage)
	client.waitForEvent(protocol.MethodWindowLogMessage, func(raw []byte) bool {
		return bytes.Contains(raw, []byte("indexing complete"))
	})

	for _, tc := range []struct {
		command string
		args    []any
		failure string
		detail  string
		output  string
	}{
		{
			"clif.generateCodeMap",
			[]any{map[string]any{"out": "empty-map.json", "format": "json"}},
			"Code map failed:", "no CFML files", "empty-map.json",
		},
		{"clif.exportCFLint", nil, "The cflint report failed:", "missing-cflint", ".clif-cflint.txt"},
	} {
		var result struct {
			Started bool `json:"started"`
		}
		client.call(protocol.MethodWorkspaceExecuteCommand, map[string]any{
			"command": tc.command, "arguments": tc.args,
		}, &result)

		if !result.Started {
			t.Fatalf("%s did not accept the background job", tc.command)
		}

		client.waitForEvent(protocol.MethodWindowShowMessage, func(raw []byte) bool {
			var params protocol.ShowMessageParams
			if err := json.Unmarshal(raw, &params); err != nil {
				t.Fatal(err)
			}

			if params.Type != protocol.MessageTypeError {
				return false
			}

			if !strings.Contains(params.Message, tc.failure) || !strings.Contains(params.Message, tc.detail) {
				t.Fatalf("%s reported the wrong error: %s", tc.command, params.Message)
			}

			return true
		})

		if _, err := os.Stat(filepath.Join(root, tc.output)); !os.IsNotExist(err) {
			t.Errorf("%s left a report after failure, stat error = %v", tc.command, err)
		}
	}

	var entries []struct {
		Name string `json:"name"`
	}
	client.call(protocol.MethodWorkspaceExecuteCommand, map[string]any{
		"command": "clif.searchDocs", "arguments": []any{"arrayLen", "function", 1},
	}, &entries)

	if len(entries) != 1 || !strings.EqualFold(entries[0].Name, "arrayLen") {
		t.Errorf("docs search after the failed jobs = %+v", entries)
	}
}
