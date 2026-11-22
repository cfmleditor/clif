package server

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// Function-scope refs are indexed lazily, keyed by the function they came from.
// The key used to be the function's line range, which moves the moment a line
// is inserted above it — so the next indexing wrote a new entry instead of
// replacing the old one. The refs accumulated exactly as they had before
// SetFuncRefs existed, and the stale entry won LookupComponentRefInFile's
// tie-break, so hover answered with the component the variable held *before*
// the edit.
func TestFuncScopeRefsSurviveLineShifts(t *testing.T) {
	srv := newTestServer()
	docURI := uri.URI("file:///scope.cfc")

	before := "component {\n" +
		"\tpublic void function Go() {\n" +
		"\t\tvar svc = new pkg.Foo();\n" +
		"\t\tsvc.Run();\n" +
		"\t}\n" +
		"}\n"

	open := makeCall(t, protocol.MethodTextDocumentDidOpen, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, Text: before},
	})

	if _, err := srv.handleDidOpen(context.Background(), open); err != nil {
		t.Fatal(err)
	}

	srv.ensureFuncRefsIndexed(docURI, 3)

	if got := len(srv.index.LookupComponentRef("svc")); got != 1 {
		t.Fatalf("first indexing produced %d refs, want 1", got)
	}

	// An incremental edit *inside* the function body: that takes didChange's
	// EditInFunc path, which shifts line numbers without reindexing the file —
	// so nothing else clears the previous entry and the key has to do the work.
	// A whole-document replace would reindex and mask this.
	change := makeCall(t, protocol.MethodTextDocumentDidChange, protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI},
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			&protocol.TextDocumentContentChangePartial{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 0},
					End:   protocol.Position{Line: 3, Character: 0},
				},
				Text: "\t\t// a note\n",
			},
		},
	})

	if _, err := srv.handleDidChange(context.Background(), change); err != nil {
		t.Fatal(err)
	}

	// Hover again inside the same function, whose range has now moved.
	srv.ensureFuncRefsIndexed(docURI, 3)

	refs := srv.index.LookupComponentRef("svc")
	if len(refs) != 1 {
		t.Errorf("re-indexing one function left %d refs, want 1: %+v", len(refs), refs)
	}
}

// A function whose last component ref is deleted must lose its indexed ref too.
// The old code skipped SetFuncRefs entirely when a function had none left, so
// the previous ref stayed until a full reindex.
func TestFuncScopeRefsClearWhenEmptied(t *testing.T) {
	srv := newTestServer()
	docURI := uri.URI("file:///cleared.cfc")

	withRef := "component {\n\tpublic void function Go() {\n\t\tvar svc = new pkg.Foo();\n\t\tsvc.Run();\n\t}\n}\n"

	open := makeCall(t, protocol.MethodTextDocumentDidOpen, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, Text: withRef},
	})

	if _, err := srv.handleDidOpen(context.Background(), open); err != nil {
		t.Fatal(err)
	}

	srv.ensureFuncRefsIndexed(docURI, 2)

	if got := len(srv.index.LookupComponentRef("svc")); got != 1 {
		t.Fatalf("expected the ref to be indexed, got %d", got)
	}

	withoutRef := "component {\n\tpublic void function Go() {\n\t\tvar svc = 1;\n\t\tsvc.Run();\n\t}\n}\n"

	change := makeCall(t, protocol.MethodTextDocumentDidChange, protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI},
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			&protocol.TextDocumentContentChangeWholeDocument{Text: withoutRef},
		},
	})

	if _, err := srv.handleDidChange(context.Background(), change); err != nil {
		t.Fatal(err)
	}

	// ensureFuncRefsIndexed's callers hold the document's lock, and taking it
	// runs the reparse the whole-document change deferred.
	release := srv.lockDoc(docURI)
	srv.ensureFuncRefsIndexed(docURI, 2)
	release()

	if got := srv.index.LookupComponentRef("svc"); len(got) != 0 {
		t.Errorf("stale ref survived after its assignment was removed: %+v", got)
	}
}

func TestDotCompletionAndReassignmentInsideFunction(t *testing.T) {
	dir := t.TempDir()
	fooPath := filepath.Join(dir, "Foo.cfc")
	barPath := filepath.Join(dir, "Bar.cfc")

	if err := os.WriteFile(fooPath, []byte("component {\n\tpublic void function fromFoo(required string fooArg) {}\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(barPath, []byte("component {\n\tpublic void function fromBar(required string barArg) {}\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := newTestServer()
	srv.WorkspaceFolders = []string{dir}
	docURI := uri.File(filepath.Join(dir, "Consumer.cfc"))

	// Declares variables.mapper as a string first, then assigns a component in
	// init() (CFML Editor #75), plus a function-local var and typed argument.
	src := "component {\n" +
		"\tvariables.mapper = \"\";\n" +
		"\tpublic void function init(Foo argFoo) {\n" +
		"\t\tvariables.mapper = new Bar();\n" +
		"\t\tvar localSvc = new Foo();\n" +
		"\t\tvariables.mapper.\n" +
		"\t\tlocalSvc.\n" +
		"\t\targFoo.\n" +
		"\t\tvariables.mapper.fromBar()\n" +
		"\t}\n" +
		"}\n"

	open := makeCall(t, protocol.MethodTextDocumentDidOpen, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, Text: src},
	})
	if _, err := srv.handleDidOpen(context.Background(), open); err != nil {
		t.Fatal(err)
	}

	hasLabel := func(line, col uint32, want string) bool {
		req := makeCall(t, protocol.MethodTextDocumentCompletion, protocol.CompletionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: line, Character: col},
			Context:      protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindTriggerCharacter, TriggerCharacter: new(".")},
		})

		res, err := srv.handleCompletion(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}

		list, _ := res.(*protocol.CompletionList)

		return list != nil && slices.ContainsFunc(list.Items, func(it protocol.CompletionItem) bool {
			return it.Label == want
		})
	}

	// variables.mapper. at line 5 -> Bar.fromBar
	if !hasLabel(5, 19, "fromBar") {
		t.Error("expected dot completion on reassigned variables.mapper. to include fromBar")
	}

	// localSvc. at line 6 -> Foo.fromFoo (without any prior hover/definition call)
	if !hasLabel(6, 11, "fromFoo") {
		t.Error("expected dot completion on function-local localSvc. to include fromFoo")
	}

	// argFoo. at line 7 -> Foo.fromFoo
	if !hasLabel(7, 9, "fromFoo") {
		t.Error("expected dot completion on typed parameter argFoo. to include fromFoo")
	}

	// Named-argument completion inside variables.mapper.fromBar(|) at line 8
	argReq := makeCall(t, protocol.MethodTextDocumentCompletion, protocol.CompletionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Position:     protocol.Position{Line: 8, Character: 27},
	})

	argRes, err := srv.handleCompletion(context.Background(), argReq)
	if err != nil {
		t.Fatal(err)
	}

	argList, _ := argRes.(*protocol.CompletionList)
	if argList == nil || !slices.ContainsFunc(argList.Items, func(it protocol.CompletionItem) bool { return it.Label == "barArg=" }) {
		t.Errorf("expected named argument completion barArg= inside variables.mapper.fromBar(), got %+v", argList)
	}

	// Incremental EditInFunc: change `var localSvc = new Foo();` on line 4 to `var localSvc = new Bar();`
	change := makeCall(t, protocol.MethodTextDocumentDidChange, protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI},
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			&protocol.TextDocumentContentChangePartial{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 21},
					End:   protocol.Position{Line: 4, Character: 24},
				},
				Text: "Bar",
			},
		},
	})
	if _, err := srv.handleDidChange(context.Background(), change); err != nil {
		t.Fatal(err)
	}

	// After the incremental edit inside init(), localSvc. at line 6 must offer fromBar, not stale fromFoo.
	if !hasLabel(6, 11, "fromBar") {
		t.Error("expected dot completion on localSvc. after EditInFunc to update to fromBar")
	}
}
