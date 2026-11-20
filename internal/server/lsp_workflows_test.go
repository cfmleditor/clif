package server

import (
	"context"
	"encoding/json/v2"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	cflog "github.com/cfmleditor/clif/internal/log"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

type workflowClient struct {
	t          *testing.T
	conn       jsonrpc2.Conn
	registered chan protocol.RegistrationParams
}

func startWorkflow(t *testing.T, root string, watchesFiles bool) *workflowClient {
	t.Helper()

	ctx := t.Context()

	a, b := net.Pipe()

	t.Cleanup(func() {
		_ = a.Close()
		_ = b.Close()
	})

	serverConn := jsonrpc2.NewConn(jsonrpc2.NewStream(a))
	srv := NewServer(serverConn, cflog.NewLogger(false))
	client := &workflowClient{
		t: t, conn: jsonrpc2.NewConn(jsonrpc2.NewStream(b)),
		registered: make(chan protocol.RegistrationParams, 1),
	}

	serverConn.Go(ctx, srv.Handler())
	client.conn.Go(ctx, func(_ context.Context, req *jsonrpc2.Request) (any, error) {
		if req.Method() == protocol.MethodClientRegisterCapability {
			var params protocol.RegistrationParams
			if err := json.Unmarshal(req.Params(), &params); err != nil {
				return nil, err
			}

			select {
			case client.registered <- params:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		return nil, nil
	})

	var result protocol.InitializeResult
	client.call(protocol.MethodInitialize, map[string]any{
		"workspaceFolders": []protocol.WorkspaceFolder{{URI: cfpath.ToURI(root), Name: filepath.Base(root)}},
		"capabilities": map[string]any{"workspace": map[string]any{
			"didChangeWatchedFiles": map[string]bool{"dynamicRegistration": watchesFiles},
		}},
	}, &result)
	client.notify(protocol.MethodInitialized, protocol.InitializedParams{})

	return client
}

func (c *workflowClient) call(method string, params, result any) {
	c.t.Helper()

	ctx, cancel := context.WithTimeout(c.t.Context(), 15*time.Second)
	defer cancel()

	if _, err := c.conn.Call(ctx, method, params, result); err != nil {
		c.t.Fatalf("%s: %v", method, err)
	}
}

func (c *workflowClient) notify(method string, params any) {
	c.t.Helper()

	ctx, cancel := context.WithTimeout(c.t.Context(), 15*time.Second)
	defer cancel()

	if err := c.conn.Notify(ctx, method, params); err != nil {
		c.t.Fatalf("%s: %v", method, err)
	}
}

func (c *workflowClient) open(path, text string) protocol.TextDocumentIdentifier {
	c.t.Helper()

	docURI := cfpath.ToURI(path)
	c.notify(protocol.MethodTextDocumentDidOpen, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, LanguageID: "cfml", Version: 1, Text: text},
	})

	return protocol.TextDocumentIdentifier{URI: docURI}
}

func (c *workflowClient) definition(doc protocol.TextDocumentIdentifier, line, col uint32) *protocol.Location {
	c.t.Helper()

	var loc *protocol.Location
	c.call(protocol.MethodTextDocumentDefinition, protocol.DefinitionParams{
		TextDocument: doc,
		Position:     protocol.Position{Line: line, Character: col},
	}, &loc)

	return loc
}

// Adapted from testGotoFunctionFromCfinvoke and testGotoFunctionArgumentFromArgumentsScope
// in quetwo/intellilucee at 83e5e802d98d06845b220dfdd72db23da47c8be7.
// The fixtures exercise clif through LSP using the same editor actions.
func TestIntelliLuceeNavigation(t *testing.T) {
	root := t.TempDir()
	t.Run("invoke page function", func(t *testing.T) {
		client := startWorkflow(t, root, false)

		doc := client.open(filepath.Join(root, "jobs.cfm"), "<cffunction name=\"runJob\">\n</cffunction>\n\n<cfinvoke method=\"runJob\">")
		if loc := client.definition(doc, 3, 20); loc == nil || loc.URI != doc.URI || loc.Range.Start.Line != 0 {
			t.Fatalf("invoke definition = %+v, want function on line 0 of %s", loc, doc.URI)
		}
	})

	t.Run("arguments scope", func(t *testing.T) {
		client := startWorkflow(t, root, false)

		const prefix = `    return "Hello " & arguments.`

		doc := client.open(filepath.Join(root, "Greeter.cfc"), `component {
  function greet(
    required string name,
    numeric age = 30
  ) {
`+prefix+`name;
  }
  function farewell(required string name) {
    return arguments.name;
  }
}`)

		loc := client.definition(doc, 5, uint32(len(prefix))+1)
		if loc == nil || loc.URI != doc.URI {
			t.Fatalf("argument definition = %+v, want parameter in %s", loc, doc.URI)
		}

		offset := uint32(strings.Index("    required string name,", "name"))

		want := protocol.Range{
			Start: protocol.Position{Line: 2, Character: offset},
			End:   protocol.Position{Line: 2, Character: offset + 4},
		}
		if loc.Range != want {
			t.Errorf("argument definition range = %v, want %v", loc.Range, want)
		}
	})
}

// Adapted from testTagsInsideCommentsNotFolded and testSingleLineTagsDoNotFold
// in IntelliLucee's CFMLFoldingBuilderTest at the revision above. The comment
// itself is also foldable in clif.
func TestIntelliLuceeFolding(t *testing.T) {
	root := t.TempDir()
	client := startWorkflow(t, root, false)
	doc := client.open(filepath.Join(root, "report.cfm"), `<!---
<cfif ready>
    <cfquery name="ignored">
        SELECT 1
    </cfquery>
</cfif>
--->
<cfquery name="report">
    SELECT id
    FROM orders
</cfquery>
<cfquery name="count">SELECT COUNT(*) FROM orders</cfquery>`)

	var folds []protocol.FoldingRange

	client.call(protocol.MethodTextDocumentFoldingRange, protocol.FoldingRangeParams{TextDocument: doc}, &folds)

	if got, want := foldSet(folds), []string{"0-6 comment", "7-9"}; !slices.Equal(got, want) {
		t.Fatalf("folds = %v, want %v", got, want)
	}
}

func TestWorkflowMappedTemplate(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, filepath.Join(root, "Application.cfc"), `component { this.mappings["/shared"] = "./templates"; }`)

	header := filepath.Join(root, "templates", "banner.cfm")
	if err := os.Mkdir(filepath.Dir(header), 0o700); err != nil {
		t.Fatal(err)
	}

	writeCFC(t, header, "<cfoutput>Orders</cfoutput>")

	client := startWorkflow(t, root, false)
	doc := client.open(filepath.Join(root, "orders.cfm"), `<cfinclude template="/shared/banner.cfm">`)
	want := cfpath.ToURI(header)

	if definition := client.definition(doc, 0, 31); definition == nil || definition.URI != want {
		t.Fatalf("definition = %+v, want %s", definition, want)
	}

	var links []protocol.DocumentLink
	client.call(protocol.MethodTextDocumentDocumentLink, protocol.DocumentLinkParams{TextDocument: doc}, &links)

	if len(links) != 1 {
		t.Fatalf("document links = %+v, want one", links)
	}

	var resolved protocol.DocumentLink
	client.call(protocol.MethodDocumentLinkResolve, links[0], &resolved)

	if resolved.Target == nil {
		t.Fatalf("resolved document link has no target, want %s", want)
	}

	if *resolved.Target != want {
		t.Fatalf("resolved document link target = %q, want %s", *resolved.Target, want)
	}
}

func TestWorkflowWorkspaceFolderChangesComponentPath(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()

	service := filepath.Join(second, "models", "OrderService.cfc")
	if err := os.Mkdir(filepath.Dir(service), 0o700); err != nil {
		t.Fatal(err)
	}

	writeCFC(t, service, "component {}")
	client := startWorkflow(t, first, false)

	resolve := func() string {
		t.Helper()

		var path string
		client.call(protocol.MethodWorkspaceExecuteCommand, map[string]any{
			"command":   "clif.showComponentPath",
			"arguments": []string{"models.OrderService", string(cfpath.ToURI(filepath.Join(first, "page.cfm")))},
		}, &path)

		return path
	}
	changeFolders := func(event protocol.WorkspaceFoldersChangeEvent) {
		t.Helper()
		client.notify(protocol.MethodWorkspaceDidChangeWorkspaceFolders, protocol.DidChangeWorkspaceFoldersParams{Event: event})
	}

	if path := resolve(); path != "" {
		t.Fatalf("component resolved before folder was added: %q", path)
	}

	folders := []protocol.WorkspaceFolder{{URI: cfpath.ToURI(second), Name: "shared"}}
	changeFolders(protocol.WorkspaceFoldersChangeEvent{Added: folders})

	if path := resolve(); path != service {
		t.Errorf("component after adding workspace folder = %q, want %q", path, service)
	}

	changeFolders(protocol.WorkspaceFoldersChangeEvent{Removed: folders})

	if path := resolve(); path != "" {
		t.Errorf("component resolved after folder was removed: %q", path)
	}
}

func TestWorkflowRegistersAllCFMLWatchers(t *testing.T) {
	client := startWorkflow(t, t.TempDir(), true)

	select {
	case params := <-client.registered:
		for _, registration := range params.Registrations {
			if registration.Method != protocol.MethodWorkspaceDidChangeWatchedFiles {
				continue
			}

			var options protocol.DidChangeWatchedFilesRegistrationOptions
			if err := json.Unmarshal(registration.RegisterOptions, &options); err != nil {
				t.Fatal(err)
			}

			for _, ext := range []string{"cfc", "cfm", "cfml", "cfs"} {
				glob := "**/*." + ext
				if !slices.ContainsFunc(options.Watchers, func(w protocol.FileSystemWatcher) bool {
					return w.GlobPattern == protocol.Pattern(glob)
				}) {
					t.Errorf("watcher registration is missing %s", glob)
				}
			}

			return
		}

		t.Fatal("didChangeWatchedFiles was not registered")
	case <-time.After(15 * time.Second):
		t.Fatal("client did not receive watcher registration")
	}
}

// Adapted from IntelliLucee's testCompletionNamedParametersCreateObject.
// The receiver exposes a public field and a method in a separate component.
func TestIntelliLuceeCompletion(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, filepath.Join(root, "Application.cfc"), "component {}")

	models := filepath.Join(root, "models")
	if err := os.Mkdir(models, 0o700); err != nil {
		t.Fatal(err)
	}

	writeCFC(t, filepath.Join(models, "Product.cfc"), "component {\n  this.price = 0;\n  function calculateTax() {}\n}")
	client := startWorkflow(t, root, false)

	const (
		before = "<cfscript>\n  var product = "
		after  = ";\n  product.\n</cfscript>"
	)

	for i, args := range []string{
		`type="component", component="models.Product"`,
		`className: "models.Product", componentType: "component"`,
		`"component", "models.Product"`,
	} {
		doc := client.open(filepath.Join(root, "catalog"+itoa(i)+".cfm"), before+"createObject("+args+")"+after)
		assertProductCompletion(t, client, doc, args)
	}
}

func assertProductCompletion(t *testing.T, client *workflowClient, doc protocol.TextDocumentIdentifier, args string) {
	t.Helper()

	var completions protocol.CompletionList
	client.call(protocol.MethodTextDocumentCompletion, protocol.CompletionParams{
		TextDocument: doc,
		Position:     protocol.Position{Line: 2, Character: 10},
		Context:      protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindTriggerCharacter, TriggerCharacter: new(".")},
	}, &completions)

	for _, label := range []string{"calculateTax", "price"} {
		if !slices.ContainsFunc(completions.Items, func(item protocol.CompletionItem) bool {
			return item.Label == label
		}) {
			t.Errorf("createObject(%s): completion is missing %q (%d items)", args, label, len(completions.Items))
		}
	}
}

// Adapted from IntelliLucee's testRenameLocalVariableFromUsage. Another
// function has a same-named variable that the returned edit must leave alone.
func TestIntelliLuceeRename(t *testing.T) {
	root := t.TempDir()
	writeCFC(t, filepath.Join(root, ".clif.json"), `{"features":{"rename":true}}`)

	client := startWorkflow(t, root, false)
	doc := client.open(filepath.Join(root, "Counter.cfc"), `component {
  function addUser() {
    var userCount = 10;
    userCount = userCount + 1;
    return userCount;
  }
  function activeUsers() {
    var userCount = 3;
    return userCount;
  }
}`)
	pos := protocol.Position{Line: 3, Character: 7}

	var prepared protocol.PrepareRenamePlaceholder
	client.call(protocol.MethodTextDocumentPrepareRename, protocol.PrepareRenameParams{
		TextDocument: doc,
		Position:     pos,
	}, &prepared)

	if prepared.Placeholder != "userCount" {
		t.Fatalf("prepare rename placeholder = %q, want userCount", prepared.Placeholder)
	}

	var result protocol.WorkspaceEdit

	client.call(protocol.MethodTextDocumentRename, protocol.RenameParams{
		TextDocument: doc,
		Position:     pos,
		NewName:      "totalUsers",
	}, &result)

	edits := result.Changes[doc.URI]
	if len(edits) != 4 {
		t.Fatalf("rename returned %d edits, want four: %+v", len(edits), edits)
	}

	want := []protocol.Position{{Line: 2, Character: 8}, {Line: 3, Character: 4}, {Line: 3, Character: 16}, {Line: 4, Character: 11}}
	for _, start := range want {
		end := protocol.Position{Line: start.Line, Character: start.Character + uint32(len("userCount"))}
		if !slices.ContainsFunc(edits, func(edit protocol.TextEdit) bool {
			return edit.Range == (protocol.Range{Start: start, End: end}) && edit.NewText == "totalUsers"
		}) {
			t.Errorf("rename is missing an edit at %v: %+v", start, edits)
		}
	}
}

func TestWorkflowParameterDefinitionUsesItsDeclaration(t *testing.T) {
	root := t.TempDir()
	client := startWorkflow(t, root, false)

	doc := client.open(filepath.Join(root, "Report.cfc"), `component {
  function mode(required string mode) {
    function nested(string mode) {
      return arguments.mode;
    }
    return arguments.mode;
  }
}`)

	for _, tc := range []struct{ useLine, declLine, declCol uint32 }{
		{3, 2, 27}, {5, 1, 32},
	} {
		loc := client.definition(doc, tc.useLine, 23)

		start := protocol.Position{Line: tc.declLine, Character: tc.declCol}

		want := protocol.Range{Start: start, End: protocol.Position{Line: tc.declLine, Character: tc.declCol + 4}}
		if loc == nil || loc.URI != doc.URI || loc.Range != want {
			t.Errorf("definition at line %d = %+v, want range %v", tc.useLine, loc, want)
		}
	}
}

func TestWorkflowRemovingFolderKeepsOverlappingSymbols(t *testing.T) {
	for _, removeParent := range []bool{false, true} {
		name := "remove child"
		if removeParent {
			name = "remove parent"
		}

		t.Run(name, func(t *testing.T) {
			root := t.TempDir()

			parent := filepath.Join(root, "app")

			child := filepath.Join(parent, "lib")
			if err := os.MkdirAll(child, 0o700); err != nil {
				t.Fatal(err)
			}

			writeCFC(t, filepath.Join(parent, "Parent.cfc"), "component { function parentOnly() {} }")
			writeCFC(t, filepath.Join(child, "Service.cfc"), "component { function childSymbol() {} }")

			client := startWorkflow(t, t.TempDir(), false)
			parentFolder := protocol.WorkspaceFolder{URI: cfpath.ToURI(parent), Name: "app"}
			childFolder := protocol.WorkspaceFolder{URI: cfpath.ToURI(child), Name: "lib"}
			client.notify(protocol.MethodWorkspaceDidChangeWorkspaceFolders, protocol.DidChangeWorkspaceFoldersParams{
				Event: protocol.WorkspaceFoldersChangeEvent{Added: []protocol.WorkspaceFolder{parentFolder, childFolder}},
			})

			if !hasWorkflowSymbol(client, "childSymbol") || !hasWorkflowSymbol(client, "parentOnly") {
				t.Fatal("both files should be indexed before removing a folder")
			}

			removed := childFolder
			if removeParent {
				removed = parentFolder
			}

			client.notify(protocol.MethodWorkspaceDidChangeWorkspaceFolders, protocol.DidChangeWorkspaceFoldersParams{
				Event: protocol.WorkspaceFoldersChangeEvent{Removed: []protocol.WorkspaceFolder{removed}},
			})

			if !hasWorkflowSymbol(client, "childSymbol") {
				t.Error("child symbol disappeared from the remaining workspace")
			}

			if hasWorkflowSymbol(client, "parentOnly") == removeParent {
				t.Errorf("parent-only symbol visibility after removing %s is wrong", removed.Name)
			}
		})
	}
}

func hasWorkflowSymbol(client *workflowClient, name string) bool {
	var symbols []protocol.SymbolInformation
	client.call(protocol.MethodWorkspaceSymbol, protocol.WorkspaceSymbolParams{Query: name}, &symbols)

	return slices.ContainsFunc(symbols, func(symbol protocol.SymbolInformation) bool { return symbol.Name == name })
}

func TestWorkflowDefinitionOfInlineCreateObject(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Service.cfc")
	other := filepath.Join(root, "Unrelated.cfc")

	const method = "function runOrder() {}"
	writeCFC(t, target, "component { "+method+" }")
	writeCFC(t, other, "component { "+method+" }")
	client := startWorkflow(t, root, false)
	client.open(target, "component { "+method+" }")
	client.open(other, "component { "+method+" }")

	for i, args := range []string{
		`"component", "Service"`,
		`type="component", component="Service"`,
		`class: "Service", componentType: "component"`,
	} {
		line := "writeOutput(createObject(" + args + ").runOrder());"
		doc := client.open(filepath.Join(root, "calls"+itoa(i)+".cfm"), "<cfscript>\n"+line+"\n</cfscript>")

		var result jsonrpc2.RawMessage
		client.call(protocol.MethodTextDocumentDefinition, protocol.DefinitionParams{
			TextDocument: doc,
			Position:     protocol.Position{Line: 1, Character: uint32(strings.Index(line, "runOrder")) + 1},
		}, &result)

		var loc protocol.Location

		if err := json.Unmarshal(result, &loc); err != nil || loc.URI != cfpath.ToURI(target) {
			t.Errorf("createObject(%s) definition = %s, want a single location in %s", args, result, target)
		}
	}
}
