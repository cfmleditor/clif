package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
)

func TestThisMethodDefinitionStaysInComponent(t *testing.T) {
	root := t.TempDir()
	srv := newTestServer()
	srv.WorkspaceFolders = []string{root}
	srv.GlobalFunctionResolution = true

	caller := filepath.Join(root, "Caller.cfc")
	callerSrc := `component extends="Base" {
	private function ownMethod() {}
	function run() {
		THIS.ownMethod();
		this.peerOnly();
		other.this.peerOnly();
		this.baseMethod();
	}
}`
	peer := filepath.Join(root, "Peer.cfc")
	peerSrc := `component {
	private function ownMethod() {}
	function peerOnly() {}
}`
	base := filepath.Join(root, "Base.cfc")
	baseSrc := `component { function baseMethod() {} }`

	for path, src := range map[string]string{caller: callerSrc, peer: peerSrc, base: baseSrc} {
		writeCFC(t, path, src)
		u := cfpath.ToURI(path)

		params := mustJSON(t, protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{URI: u, LanguageID: "cfml", Version: 1, Text: src},
		})
		if _, err := srv.handleDidOpen(context.Background(), params); err != nil {
			t.Fatal(err)
		}
	}

	locs := func(snippet string) []protocol.Location {
		t.Helper()

		line, col := cursorPosition(t, callerSrc, snippet)

		return conformanceLocations(t, definitionAt(t, srv, cfpath.ToURI(caller), line, col))
	}

	if got := locs("THIS.|ownMethod()"); len(got) != 1 || got[0].URI != cfpath.ToURI(caller) {
		t.Errorf("own private method definition = %+v", got)
	}

	if got := locs("\t\tthis.|peerOnly()"); len(got) != 0 {
		t.Errorf("unrelated method definition = %+v", got)
	}

	if got := locs("other.this.|peerOnly()"); len(got) != 1 || got[0].URI != cfpath.ToURI(peer) {
		t.Errorf("nested receiver fallback definition = %+v", got)
	}

	if got := locs("this.|baseMethod()"); len(got) != 1 || got[0].URI != cfpath.ToURI(base) {
		t.Errorf("inherited method definition = %+v", got)
	}
}

func TestClosedFileMethodDefinitionUsesParsedNameRange(t *testing.T) {
	root := t.TempDir()

	scriptTarget := "component {\n\t/* 😀 */ public\n\tstring\n\tfunction\n\trender() {\n\t\treturn \"ok\";\n\t}\n\tthis.helper = function() {\n\t\treturn 1;\n\t};\n}\n"
	tagTarget := "<cfcomponent>\n\t<cffunction\n\t\taccess=\"public\"\n\t\tname=\"tagRender\"\n\t\treturntype=\"string\">\n\t\t<cfreturn \"ok\">\n\t</cffunction>\n</cfcomponent>\n"
	callerSrc := "component {\n\tfunction run() {\n\t\tvar s = new ScriptTarget();\n\t\ts.render();\n\t\ts.helper();\n\t\tvar t = new TagTarget();\n\t\tt.tagRender();\n\t}\n}\n"

	scriptPath := filepath.Join(root, "ScriptTarget.cfc")
	tagPath := filepath.Join(root, "TagTarget.cfc")
	callerPath := filepath.Join(root, "Caller.cfc")

	for path, content := range map[string]string{
		scriptPath: scriptTarget,
		tagPath:    tagTarget,
		callerPath: callerSrc,
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	client := startWorkflow(t, root, false)
	// Open only Caller.cfc so ScriptTarget.cfc and TagTarget.cfc stay closed.
	doc := client.open(callerPath, callerSrc)

	cases := []struct {
		name        string
		snippet     string
		targetPath  string
		targetText  string
		declaration string
	}{
		{
			name:        "multiline script method in closed file",
			snippet:     "s.|render();",
			targetPath:  scriptPath,
			targetText:  scriptTarget,
			declaration: "\t|render|() {",
		},
		{
			name:        "function value in closed file",
			snippet:     "s.|helper();",
			targetPath:  scriptPath,
			targetText:  scriptTarget,
			declaration: "this.|helper| = function()",
		},
		{
			name:        "multiline tag method in closed file",
			snippet:     "t.|tagRender();",
			targetPath:  tagPath,
			targetText:  tagTarget,
			declaration: `name="|tagRender|"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			line, char := cursorPosition(t, callerSrc, tc.snippet)
			loc := client.definition(doc, line, char)

			want := protocol.Location{
				URI:   cfpath.ToURI(tc.targetPath),
				Range: conformanceDeclarationRange(t, tc.targetText, tc.declaration),
			}
			if loc == nil || *loc != want {
				t.Errorf("definition = %+v, want %+v", loc, want)
			}
		})
	}
}

func TestClosedFileMethodDefinitionDoesNotReadTargetFromDisk(t *testing.T) {
	root := t.TempDir()
	srv := newTestServer()
	srv.WorkspaceFolders = []string{root}
	srv.GlobalFunctionResolution = true

	// Index a component whose file is never written to disk and never opened.
	missingPath := filepath.Join(root, "ClosedOnly.cfc")
	missingURI := cfpath.ToURI(missingPath)
	missingSrc := "<cfcomponent>\n\t<cffunction\n\t\taccess=\"public\"\n\t\tname=\"remoteMethod\">\n\t</cffunction>\n</cfcomponent>\n"

	pr := parser.Parse(missingURI, missingSrc)
	srv.index.IndexFileFromResult(missingURI, pr.Funcs, pr.ComponentRefs)

	callerPath := filepath.Join(root, "Caller.cfc")
	callerURI := cfpath.ToURI(callerPath)
	callerSrc := "component {\n\tfunction run() {\n\t\tunknownObj.remoteMethod();\n\t}\n}\n"
	srv.setDocument(callerURI, callerSrc)

	callerPR := parser.Parse(callerURI, callerSrc)
	srv.index.IndexFileFromResult(callerURI, callerPR.Funcs, callerPR.ComponentRefs)
	srv.mu.Lock()
	srv.parseResults[callerURI] = callerPR
	srv.mu.Unlock()

	line, char := cursorPosition(t, callerSrc, "unknownObj.|remoteMethod();")
	locs := conformanceLocations(t, definitionAt(t, srv, callerURI, line, char))

	want := protocol.Location{
		URI:   missingURI,
		Range: conformanceDeclarationRange(t, missingSrc, `name="|remoteMethod|"`),
	}
	if len(locs) != 1 || locs[0] != want {
		t.Fatalf("definition = %+v, want %+v", locs, want)
	}
}

func TestDidChangeShiftsMultilineMethodDefinitionRange(t *testing.T) {
	root := t.TempDir()
	srv := newTestServer()
	srv.WorkspaceFolders = []string{root}

	docURI := cfpath.ToURI(filepath.Join(root, "ShiftDef.cfc"))
	src := "component {\n\tfunction first() {\n\t\tsecond();\n\t}\n\tpublic\n\tstring\n\tfunction\n\tsecond() {\n\t\treturn \"ok\";\n\t}\n}\n"

	openParams := mustJSON(t, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, LanguageID: "cfml", Version: 1, Text: src},
	})
	if _, err := srv.handleDidOpen(context.Background(), openParams); err != nil {
		t.Fatal(err)
	}

	// Insert two lines inside first() above second()'s multiline header.
	changeParams := mustJSON(t, protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI},
			Version:                2,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			&protocol.TextDocumentContentChangePartial{
				Range: protocol.Range{
					Start: protocol.Position{Line: 2, Character: 0},
					End:   protocol.Position{Line: 2, Character: 0},
				},
				Text: "\t\tvar a = 1;\n\t\tvar b = 2;\n",
			},
		},
	})
	if _, err := srv.handleDidChange(context.Background(), changeParams); err != nil {
		t.Fatal(err)
	}

	edited, ok := srv.getDocument(docURI)
	if !ok {
		t.Fatal("document missing after didChange")
	}

	line, char := cursorPosition(t, edited, "\t\t|second();")
	locs := conformanceLocations(t, definitionAt(t, srv, docURI, line, char))

	want := protocol.Location{
		URI:   docURI,
		Range: conformanceDeclarationRange(t, edited, "\t|second|() {"),
	}
	if len(locs) != 1 || locs[0] != want {
		t.Fatalf("definition after didChange = %+v, want %+v", locs, want)
	}
}
