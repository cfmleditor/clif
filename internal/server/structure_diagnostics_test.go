package server

import (
	"context"
	"fmt"
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
	ds := structureDiagnosticsFrom(content, functionSpansOf(content, nil))

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
		code, _ := d.Code.(protocol.String)
		if want := map[bool]protocol.DiagnosticSeverity{true: protocol.DiagnosticSeverityError, false: protocol.DiagnosticSeverityWarning}[code == "missing-attribute"]; d.Severity != want {
			t.Errorf("%v has severity %v, want %v", d.Code, d.Severity, want)
		}
	}
}

func TestStructureDiagnosticsNameAFunctionsRepeatedArgument(t *testing.T) {
	content := "component {\n\tfunction f( a, b, A ) { return a; }\n\tfunction g( a, b ) { return a; }\n}\n"
	pr := parser.ParseWithOptions(testURI(), content, &parser.ParseOptions{})
	ds := structureDiagnosticsFrom(content, functionSpansOf(content, pr))

	if len(ds) != 1 {
		t.Fatalf("%d diagnostics, want 1: %+v", len(ds), ds)
	}

	d := ds[0]
	if code, _ := d.Code.(protocol.String); code != "duplicate-argument" || d.Range.Start.Line != 1 || d.Range.Start.Character != 19 || d.Range.End.Character != 20 {
		t.Errorf("got %v at %+v, want duplicate-argument at line 1, columns 19 to 20", d.Code, d.Range)
	}

	// A parse of other text says nothing of this one.
	if got := structureDiagnosticsFrom(content, functionSpansOf(content, parser.ParseWithOptions(testURI(), content+" ", &parser.ParseOptions{}))); len(got) != 0 {
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

func TestDuplicateArgumentsAreUnderlinedAtTheRepeat(t *testing.T) {
	for _, c := range []struct {
		name, src string
		want      []string
	}{
		{"script", "component {\n\tpublic void function id( string id, string id, string ID ) {}\n}\n", []string{"1:44-46", "1:55-57"}},
		{"tags", "<cffunction name=\"t\"><cfargument name=\"t\"><cfargument name=\"t\"></cffunction>", []string{"0:60-61"}},
	} {
		pr := parser.ParseWithOptions(testURI(), c.src, &parser.ParseOptions{})

		var cols []string

		for _, d := range structureDiagnosticsFrom(c.src, functionSpansOf(c.src, pr)) {
			if code, _ := d.Code.(protocol.String); code == "duplicate-argument" {
				cols = append(cols, fmt.Sprintf("%d:%d-%d", d.Range.Start.Line, d.Range.Start.Character, d.Range.End.Character))
			}
		}

		if !slices.Equal(cols, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, cols, c.want)
		}
	}
}

// A check that finishes after its document closed publishes nothing.
func TestAStructureCheckAfterACloseLeavesNothingBehind(t *testing.T) {
	s := newTestServer()
	s.Features.StructureDiagnostics = true

	docURI := uri.File("/tmp/structure_closed.cfc")
	openDoc(t, s, docURI, "<cfcomponent>\n<cfif a>\n</cfcomponent>\n")
	waitForStructure(t, s, docURI, []string{"unclosed-tag"})

	s.cancelStructureDiagnostics(context.Background(), docURI)
	waitForStructure(t, s, docURI, nil)

	s.publishStructureDiagnostics(docURI) // the check that was already running

	waitForStructure(t, s, docURI, nil)
}

// A check that began before a close and a reopen is not the latest of the
// reopened document, whatever its number was.
func TestAStructureRunOfAnEarlierOpeningIsNotTheLatest(t *testing.T) {
	s := newTestServer()
	s.Features.StructureDiagnostics = true

	docURI := uri.File("/tmp/structure_reopened.cfc")
	s.setDocument(docURI, "<cfif a>")
	s.scheduleStructureDiagnostics(docURI)

	old, ok := s.startStructureRun(docURI)
	if !ok {
		t.Fatal("an open document has no run")
	}

	s.cancelStructureDiagnostics(context.Background(), docURI)

	if _, ok := s.startStructureRun(docURI); ok {
		t.Error("a run started on a closed document")
	}

	s.scheduleStructureDiagnostics(docURI)

	fresh, _ := s.startStructureRun(docURI)
	if s.structureRunIsLatest(docURI, old) || !s.structureRunIsLatest(docURI, fresh) {
		t.Errorf("old run %d latest %v, fresh run %d latest %v", old, s.structureRunIsLatest(docURI, old), fresh, s.structureRunIsLatest(docURI, fresh))
	}

	s.stopStructureDiagnostics()
}

// An edit makes a run already in flight one of older text.
func TestAnEditSupersedesAStructureRunInFlight(t *testing.T) {
	s := newTestServer()
	s.Features.StructureDiagnostics = true

	docURI := uri.File("/tmp/structure_inflight.cfc")
	s.setDocument(docURI, "<cfif a>")
	s.scheduleStructureDiagnostics(docURI)

	run, _ := s.startStructureRun(docURI)

	s.scheduleStructureDiagnostics(docURI) // the next keystroke

	if s.structureRunIsLatest(docURI, run) {
		t.Error("a run begun before an edit is still the latest")
	}

	s.stopStructureDiagnostics()
}

// A change for a document that is not open leaves nothing behind.
func TestAChangeToAnUnopenedDocumentSchedulesNothing(t *testing.T) {
	s := newTestServer()
	s.Features.StructureDiagnostics = true

	docURI := uri.File("/tmp/structure_unopened.cfc")
	change := makeCall(t, protocol.MethodTextDocumentDidChange, protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI}, Version: 2},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			&protocol.TextDocumentContentChangeWholeDocument{Text: "<cfif a>"},
		},
	})

	if _, err := s.handleDidChange(context.Background(), change); err != nil {
		t.Fatal(err)
	}

	s.mu.RLock()
	_, scheduled := s.structure[docURI]
	s.mu.RUnlock()

	if scheduled {
		t.Error("a structure check is scheduled for a document that is not open")
	}
}

// A parse that is missing or of other text does not take the spans away: a
// fresh parse stands in.
func TestFunctionSpansOfADocWithoutAFreshParse(t *testing.T) {
	s := newTestServer()
	docURI := uri.File("/tmp/spans_fallback.cfc")
	content := "component {\n function f( a, a ) {}\n}\n"

	if got := s.functionSpansOfDoc(docURI, content); len(got) != 1 {
		t.Errorf("%d spans with no parse held, want 1", len(got))
	}
}

// After shutdown nothing is scheduled.
func TestNoStructureCheckIsScheduledAfterShutdown(t *testing.T) {
	s := newTestServer()
	s.Features.StructureDiagnostics = true

	docURI := uri.File("/tmp/structure_stopped.cfc")
	s.setDocument(docURI, "<cfif a>")
	s.stopStructureDiagnostics()
	s.scheduleStructureDiagnostics(docURI)

	s.mu.RLock()
	_, scheduled := s.structure[docURI]
	s.mu.RUnlock()

	if scheduled {
		t.Error("a check was scheduled after the server stopped")
	}
}
