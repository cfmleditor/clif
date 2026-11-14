package server

import (
	"context"
	"strings"
	"time"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
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
	defer s.lockDoc(docURI)()

	content, ok := s.getDocument(docURI)
	if !ok {
		return
	}

	s.mu.RLock()
	pr := s.parseResults[docURI]
	s.mu.RUnlock()

	s.setDiagnostics(context.Background(), docURI, sourceStructure, structureDiagnostics(content, pr))
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
// underlined at the second.
func duplicateArguments(content string, pr *parser.ParseResult, at func(int) protocol.Position) []protocol.Diagnostic {
	if pr == nil || pr.Content != content {
		return nil
	}

	lines := parser.NewLineTable(content)

	var out []protocol.Diagnostic

	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		if f.Generated || len(f.Arguments) < 2 {
			continue
		}

		from, ok := lines.Start(int(f.Line))
		if !ok {
			continue
		}

		seen := map[string]bool{}

		for _, a := range f.Arguments {
			name := strings.ToLower(a.Name)
			if !seen[name] {
				seen[name] = true

				continue
			}

			// The second place the name is written as a word, from the
			// function's line on: its declaration, past the first.
			start, end := nthWord(content, from, a.Name, 2)
			if start < 0 {
				continue
			}

			out = append(out, protocol.Diagnostic{
				Range:    protocol.Range{Start: at(start), End: at(end)},
				Severity: protocol.DiagnosticSeverityError,
				Source:   protocol.NewOptional("clif"),
				Code:     protocol.String("duplicate-argument"),
				Message:  protocol.String("Duplicate argument '" + a.Name + "' in function '" + f.Name + "'"),
			})
		}
	}

	return out
}

// nthWord is where the nth whole-word, case-insensitive occurrence of word
// stands in content at or after from, or -1.
func nthWord(content string, from int, word string, n int) (start, end int) {
	for i := from; i+len(word) <= len(content); i++ {
		if !strings.EqualFold(content[i:i+len(word)], word) {
			continue
		}

		if i > 0 && lex.IsWord(content[i-1]) || i+len(word) < len(content) && lex.IsWord(content[i+len(word)]) {
			continue
		}

		n--

		if n == 0 {
			return i, i + len(word)
		}
	}

	return -1, -1
}
