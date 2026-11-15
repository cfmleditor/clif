package server

import (
	"context"
	"time"

	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
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
	timer     *time.Timer
	latest    uint64 // the generation of the text: an edit gives it a new one
	published bool   // a check has published for this opening
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
	if _, open := s.documents[docURI]; !open || s.structureStopped {
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
	// A check already running finds its document closed after its send, and
	// takes its own publication back.
	s.mu.Lock()

	published := false

	if st := s.structure[docURI]; st != nil {
		published = st.published

		if st.timer != nil {
			st.timer.Stop()
		}
	}

	delete(s.structure, docURI)

	s.mu.Unlock()

	// Only what was published is taken back.
	if published {
		s.setDiagnostics(ctx, docURI, sourceStructure, nil)
	}
}

// structureRun is a check of one opening of a document: the state of that
// opening, and the generation of the text the check began with. An edit gives
// the state a new generation, and a close ends the state, so a run is the latest
// only until either, and a run of an earlier opening is never the latest of a
// reopened one.
type structureRun struct {
	st  *structureState
	gen uint64
}

// startStructureRun is the run of a check of docURI as it stands. ok is false
// when the document is not open.
func (s *Server) startStructureRun(docURI uri.URI) (run structureRun, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st := s.structure[docURI]
	if st == nil {
		return structureRun{}, false
	}

	return structureRun{st: st, gen: st.latest}, true
}

// structureRunIsLatest reports whether run is still the latest check of docURI.
func (s *Server) structureRunIsLatest(docURI uri.URI, run structureRun) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return run.st != nil && s.structure[docURI] == run.st && run.st.latest == run.gen
}

// publishStructureDiagnostics checks the structure of docURI as it stands and
// publishes what it finds. lockDoc runs the reparse an edit deferred, so the
// parse the spans are read from is the document's.
func (s *Server) publishStructureDiagnostics(docURI uri.URI) {
	// The lock is held to catch up the deferred reparse and take the text and
	// its spans; the analysis is of that text, which does not change, and runs
	// without it so hover and typing are not left waiting on it. The run is
	// numbered with the text: a check that starts after this one is of newer
	// text, and this one, if it finishes later, has nothing to publish. (The
	// document's lock is the one it had while open: docLocks keeps its
	// entries.)
	unlock := s.lockDoc(docURI)

	content, open := s.getDocument(docURI)

	var run structureRun
	if open {
		run, open = s.startStructureRun(docURI)
	}

	// An edit changes a parse in place, so what is read from it is read here.
	spans := s.functionSpansOfDoc(docURI, content)

	unlock()

	// A document closed since the timer fired has nothing to check.
	if !open {
		return
	}

	found := structureDiagnosticsFrom(content, spans)

	// Whether the run is still the latest is asked under the lock that orders the
	// sends, so no newer run can be sent between the question and the send. A
	// close that comes after has already taken the document's diagnostics back,
	// so the send is followed by a look at whether the opening it was for
	// is still the one open: if the document was closed, the publication is
	// taken back; if it was reopened, the new opening's own run publishes, and
	// the close has already cleared this one.
	if !s.setDiagnosticsIf(context.Background(), docURI, sourceStructure, found, func() bool { return s.structureRunIsLatest(docURI, run) }) {
		return // a newer check, or a close, has the say
	}

	s.mu.Lock()

	current := s.structure[docURI]
	if current == run.st {
		run.st.published = true
	}

	s.mu.Unlock()

	if current == nil {
		s.setDiagnostics(context.Background(), docURI, sourceStructure, nil)
	}
}

// stopStructureDiagnostics stops every pending check: the server is going.
func (s *Server) stopStructureDiagnostics() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.structureStopped = true

	for docURI, st := range s.structure {
		if st.timer != nil {
			st.timer.Stop()
		}

		delete(s.structure, docURI)
	}
}

// functionSpansOfDoc are the function spans of content, the text of docURI:
// those of the parse the server holds when it is of this text, and those of a
// fresh parse when it is not or there is none. The caller holds the document's
// lock, which an edit changes a parse in place under.
func (s *Server) functionSpansOfDoc(docURI uri.URI, content string) []parser.FunctionSpan {
	if pr := s.parseResultOf(docURI); pr != nil && pr.Content == content {
		return pr.FunctionSpans()
	}

	return parser.FunctionSpansOf(cfpath.FromURI(string(docURI)), content)
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
