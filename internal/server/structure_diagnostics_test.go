package server

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func diagnosticCodes(ds []protocol.Diagnostic) []string {
	var out []string

	for i := range ds {
		d := &ds[i]
		if src, _ := d.Source.Get(); src == "clif" {
			code, _ := d.Code.(protocol.String)
			out = append(out, string(code))
		}
	}

	slices.Sort(out)

	return out
}

// waitForStructure waits until docURI's structure diagnostics hold want.
func waitForStructure(t *testing.T, s *Server, docURI uri.URI, want []string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for {
		got := diagnosticCodes(s.diagnosticsFor(docURI))
		if slices.Equal(got, want) {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("structure diagnostics are %q, want %q", got, want)
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// The ranges are the problem's own text, in the UTF-16 columns of the editor.
func TestStructureDiagnosticRanges(t *testing.T) {
	content := "<cfset s = \"é\">\n<p>é</p><cfif a>\n<cffunction>x</cffunction>\n"
	ds := structureDiagnostics(content, nil)

	if len(ds) != 2 {
		t.Fatalf("%d diagnostics, want 2: %+v", len(ds), ds)
	}

	byCode := map[string]protocol.Diagnostic{}

	for _, d := range ds {
		code, _ := d.Code.(protocol.String)
		byCode[string(code)] = d
	}

	// "<p>é</p>" is eight UTF-16 units before the "<cfif" on line 1.
	open := byCode["unclosed-tag"].Range
	if open.Start.Line != 1 || open.Start.Character != 8 || open.End.Character != 13 {
		t.Errorf("unclosed <cfif> is at %+v, want line 1, columns 8 to 13", open)
	}

	if r := byCode["missing-attribute"].Range; r.Start.Line != 2 || r.Start.Character != 0 || r.End.Character != 11 {
		t.Errorf("missing name is at %+v, want line 2, columns 0 to 11", r)
	}

	for _, d := range ds {
		if d.Severity != protocol.DiagnosticSeverityError {
			t.Errorf("%v has severity %v, want error", d.Code, d.Severity)
		}
	}
}

func TestStructureDiagnosticsNameAFunctionsRepeatedArgument(t *testing.T) {
	content := "component {\n\tfunction f( a, b, A ) { return a; }\n\tfunction g( a, b ) { return a; }\n}\n"
	pr := parser.ParseWithOptions(testURI(), content, &parser.ParseOptions{})
	ds := structureDiagnostics(content, pr)

	if len(ds) != 1 {
		t.Fatalf("%d diagnostics, want 1: %+v", len(ds), ds)
	}

	d := ds[0]
	if code, _ := d.Code.(protocol.String); code != "duplicate-argument" || d.Range.Start.Line != 1 || d.Range.Start.Character != 19 || d.Range.End.Character != 20 {
		t.Errorf("got %v at %+v, want duplicate-argument at line 1, columns 19 to 20", d.Code, d.Range)
	}

	// A parse of other text says nothing of this one.
	if got := structureDiagnostics(content, parser.ParseWithOptions(testURI(), content+" ", &parser.ParseOptions{})); len(got) != 0 {
		t.Errorf("a stale parse gave %d diagnostics", len(got))
	}
}

func testURI() uri.URI { return uri.File("/tmp/structure_test.cfc") }

func TestStructureDiagnosticsFollowTheDocument(t *testing.T) {
	s := newTestServer()
	s.Features.StructureDiagnostics = true

	ctx := context.Background()
	docURI := uri.File("/tmp/structure_session.cfc")

	openDoc(t, s, docURI, "<cfcomponent>\n<cffunction name=\"f\">\n<cfif a>\n</cffunction>\n</cfcomponent>\n")
	waitForStructure(t, s, docURI, []string{"unclosed-tag"})

	// Closing the tag clears it, once the document has been left alone.
	change := makeCall(t, protocol.MethodTextDocumentDidChange, protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI}, Version: 2},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			&protocol.TextDocumentContentChangeWholeDocument{Text: "<cfcomponent>\n<cffunction name=\"f\">\n<cfif a></cfif>\n</cffunction>\n</cfcomponent>\n"},
		},
	})
	if _, err := s.handleDidChange(ctx, change); err != nil {
		t.Fatal(err)
	}

	waitForStructure(t, s, docURI, nil)

	// And a close takes what is left.
	change = makeCall(t, protocol.MethodTextDocumentDidChange, protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI}, Version: 3},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			&protocol.TextDocumentContentChangeWholeDocument{Text: "<cfcomponent>\n<cfcatch>\n</cfcomponent>\n"},
		},
	})
	if _, err := s.handleDidChange(ctx, change); err != nil {
		t.Fatal(err)
	}

	waitForStructure(t, s, docURI, []string{"misplaced-tag", "unclosed-tag"})

	closeCall := makeCall(t, protocol.MethodTextDocumentDidClose, protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
	})
	if _, err := s.handleDidClose(ctx, closeCall); err != nil {
		t.Fatal(err)
	}

	waitForStructure(t, s, docURI, nil)
}

// The feature is off unless asked for: an editor that draws its own errors
// would show each twice.
func TestStructureDiagnosticsAreOffByDefault(t *testing.T) {
	s := newTestServer()
	docURI := uri.File("/tmp/structure_off.cfc")

	openDoc(t, s, docURI, "<cfcomponent><cfif a></cfcomponent>")
	time.Sleep(structureDelay + 200*time.Millisecond)

	if got := diagnosticCodes(s.diagnosticsFor(docURI)); len(got) != 0 {
		t.Errorf("structure diagnostics %q with the feature off", got)
	}
}
