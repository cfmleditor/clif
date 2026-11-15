package server

import (
	"context"
	"time"

	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// sourceStructure is the diagnostics source for the structure of a
// document's tags: what features.structureDiagnostics publishes.
const sourceStructure = "structure"

// structureDelay is how long a document is left alone before its structure is
// checked: typing a tag out leaves it open for a moment, and flagging that on
// every keystroke is noise.
const structureDelay = 300 * time.Millisecond

// structureState is what an open document has of its structure check: the timer
// that will run it, and the number of the latest run that started. A close
// takes the whole of it away, so a run that finishes after that, or after a
// reopen's new state, finds its number is not the latest.
type structureState struct {
	timer  *time.Timer
	latest uint64
}

// scheduleStructureDiagnostics checks the structure of docURI once it has been
// quiet for structureDelay, and opened documents soon after.
func (s *Server) scheduleStructureDiagnostics(docURI uri.URI) {
	if !s.Features.StructureDiagnostics {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Of an open document only, decided under the lock a close takes the
	// document away under: a change that follows a close has nothing to
	// check, and no close left to take the check back.
	if _, open := s.documents[docURI]; !open {
		return
	}

	st := s.structure[docURI]
	if st == nil {
		st = &structureState{}
		s.structure[docURI] = st
	} else if st.timer != nil {
		st.timer.Stop()
	}

	// An edit makes a run in flight one of older text.
	s.structureSeq++
	st.latest = s.structureSeq

	st.timer = time.AfterFunc(structureDelay, func() {
		defer func() {
			if r := recover(); r != nil {
				cflog.Recovered(s.log, "goroutine panic", r, cflog.String("label", "structureDiagnostics"))
			}
		}()

		s.publishStructureDiagnostics(docURI)
	})
}

// cancelStructureDiagnostics stops a pending check of docURI, and takes its
// diagnostics back: the document is closed.
func (s *Server) cancelStructureDiagnostics(ctx context.Context, docURI uri.URI) {
	// A check already running publishes under this lock, and only while its
	// document still has its state: one that finishes after the close has
	// nothing left to publish.
	s.structureMu.Lock()
	defer s.structureMu.Unlock()

	s.mu.Lock()

	if st := s.structure[docURI]; st != nil && st.timer != nil {
		st.timer.Stop()
	}

	delete(s.structure, docURI)

	s.mu.Unlock()

	if s.Features.StructureDiagnostics {
		s.setDiagnostics(ctx, docURI, sourceStructure, nil)
	}
}

// startStructureRun numbers a check of docURI, which is then the latest, or
// reports that the document is not open.
func (s *Server) startStructureRun(docURI uri.URI) (run uint64, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.structure[docURI]
	if st == nil {
		return 0, false
	}

	s.structureSeq++
	st.latest = s.structureSeq

	return st.latest, true
}

// structureRunIsLatest reports whether run is still the latest check of docURI.
func (s *Server) structureRunIsLatest(docURI uri.URI, run uint64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st := s.structure[docURI]

	return st != nil && st.latest == run
}

// publishStructureDiagnostics checks the structure of docURI as it stands and
// publishes what it finds. lockDoc runs the reparse an edit deferred, so the
// parse the spans are read from is the document's.
func (s *Server) publishStructureDiagnostics(docURI uri.URI) {
	// A document closed since the timer fired has nothing to check, and
	// lockDoc would make a lock for it that nothing removes. A check that
	// starts after this one is of newer text: this one, if it finishes later,
	// has nothing to publish.
	run, ok := s.startStructureRun(docURI)
	if !ok {
		return
	}

	// The lock is held to catch up the deferred reparse and take the text and
	// its spans; the analysis is of that text, which does not change, and runs
	// without it so hover and typing are not left waiting on it.
	unlock := s.lockDoc(docURI)

	content, ok := s.getDocument(docURI)

	// An edit changes a parse in place, so what is read from it is read here.
	spans := s.functionSpansOfDoc(docURI, content)

	unlock()

	if !ok {
		return
	}

	found := structureDiagnosticsFrom(content, spans)

	s.structureMu.Lock()
	defer s.structureMu.Unlock()

	if s.structureRunIsLatest(docURI, run) {
		s.setDiagnostics(context.Background(), docURI, sourceStructure, found)
	}
}

// stopStructureDiagnostics stops every pending check: the server is going.
func (s *Server) stopStructureDiagnostics() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for docURI, st := range s.structure {
		if st.timer != nil {
			st.timer.Stop()
		}

		delete(s.structure, docURI)
	}
}

// functionSpansOf are the function spans of pr when it is the parse of
// content, and nil when it is not or there is none.
func functionSpansOf(content string, pr *parser.ParseResult) []parser.FunctionSpan {
	if pr == nil || pr.Content != content {
		return nil
	}

	return pr.FunctionSpans()
}

// functionSpansOfDoc are the function spans of the parse the server holds for
// docURI when it is of content, and nil when it is not.
func (s *Server) functionSpansOfDoc(docURI uri.URI, content string) []parser.FunctionSpan {
	return functionSpansOf(content, s.parseResultOf(docURI))
}

// structureDiagnosticsFrom is the structure problems of content: those of its
// tags (parser.TagProblems) and the arguments a function declares twice, read
// from the function spans of content, which are taken already.
func structureDiagnosticsFrom(content string, spans []parser.FunctionSpan) []protocol.Diagnostic {
	problems := parser.TagProblems(content)

	var out []protocol.Diagnostic

	at := rangePositioner(content, parser.NewLineTable(content))

	for _, p := range problems {
		severity := protocol.DiagnosticSeverityError
		if p.Warning {
			severity = protocol.DiagnosticSeverityWarning
		}

		out = append(out, protocol.Diagnostic{
			Range:    protocol.Range{Start: at(p.Start), End: at(p.End)},
			Severity: severity,
			Source:   protocol.NewOptional("clif"),
			Code:     protocol.String(string(p.Kind)),
			Message:  protocol.String(p.Message),
		})
	}

	return append(out, duplicateArguments(content, spans, at)...)
}

// duplicateArguments are the arguments a function declares more than once,
// underlined at each declaration after the first.
func duplicateArguments(content string, spans []parser.FunctionSpan, at func(int) protocol.Position) []protocol.Diagnostic {
	dups := parser.DuplicateArguments(content, spans)
	out := make([]protocol.Diagnostic, 0, len(dups))

	for _, d := range dups {
		out = append(out, protocol.Diagnostic{
			Range:    protocol.Range{Start: at(d.Start), End: at(d.End)},
			Severity: protocol.DiagnosticSeverityError,
			Source:   protocol.NewOptional("clif"),
			Code:     protocol.String("duplicate-argument"),
			Message:  protocol.String("Duplicate argument '" + d.Name + "' in function '" + d.Function + "'"),
		})
	}

	return out
}
