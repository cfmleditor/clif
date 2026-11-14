package server

import (
	"context"
	"time"

	"github.com/cfmleditor/clif/internal/conv"
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

// scheduleStructureDiagnostics checks the structure of docURI once it has been
// quiet for structureDelay, and opened documents soon after.
func (s *Server) scheduleStructureDiagnostics(docURI uri.URI) {
	if !s.Features.StructureDiagnostics {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if timer, ok := s.structureTimers[docURI]; ok {
		timer.Stop()
	}

	s.structureTimers[docURI] = time.AfterFunc(structureDelay, func() {
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
	// document still has a timer: one that finishes after the close has
	// nothing left to publish.
	s.structureMu.Lock()
	defer s.structureMu.Unlock()

	s.mu.Lock()

	if timer, ok := s.structureTimers[docURI]; ok {
		timer.Stop()
		delete(s.structureTimers, docURI)
	}

	s.mu.Unlock()

	if s.Features.StructureDiagnostics {
		s.setDiagnostics(ctx, docURI, sourceStructure, nil)
	}
}

// publishStructureDiagnostics checks the structure of docURI as it stands and
// publishes what it finds. lockDoc runs the reparse an edit deferred, so the
// parse the arguments are read from is the document's.
func (s *Server) publishStructureDiagnostics(docURI uri.URI) {
	// A document closed since the timer fired has nothing to check, and
	// lockDoc would make a lock for it that nothing removes.
	if !s.structureWanted(docURI) {
		return
	}

	defer s.lockDoc(docURI)()

	content, ok := s.getDocument(docURI)
	if !ok {
		return
	}

	s.mu.RLock()
	pr := s.parseResults[docURI]
	s.mu.RUnlock()

	found := structureDiagnostics(content, pr)

	s.structureMu.Lock()
	defer s.structureMu.Unlock()

	if s.structureWanted(docURI) {
		s.setDiagnostics(context.Background(), docURI, sourceStructure, found)
	}
}

// structureWanted reports whether docURI is still open to be checked: it has a
// timer, which a close takes away.
func (s *Server) structureWanted(docURI uri.URI) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.structureTimers[docURI]

	return ok
}

// stopStructureDiagnostics stops every pending check: the server is going.
func (s *Server) stopStructureDiagnostics() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for docURI, timer := range s.structureTimers {
		timer.Stop()
		delete(s.structureTimers, docURI)
	}
}

// structureDiagnostics is the structure problems of content: those of its tags
// (parser.TagProblems) and the arguments a function declares twice, read from
// pr, the parse of content, when there is one.
func structureDiagnostics(content string, pr *parser.ParseResult) []protocol.Diagnostic {
	problems := parser.TagProblems(content)

	var out []protocol.Diagnostic

	lines := parser.NewLineTable(content)
	cols := newColMapper(content)

	at := func(offset int) protocol.Position {
		line := lines.LineOf(offset)
		start, _ := lines.Start(line)

		return protocol.Position{
			Line:      conv.Uint32(line),
			Character: cols.col(conv.Uint32(line), conv.Uint32(offset-start)),
		}
	}

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

	return append(out, duplicateArguments(content, pr, at)...)
}

// duplicateArguments are the arguments a function declares more than once,
// underlined at each declaration after the first.
func duplicateArguments(content string, pr *parser.ParseResult, at func(int) protocol.Position) []protocol.Diagnostic {
	if pr == nil || pr.Content != content {
		return nil
	}

	var out []protocol.Diagnostic

	for _, d := range parser.DuplicateArguments(content, pr.FunctionSpans()) {
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
