// Package supervise keeps an editor's language server session alive across a
// crash of the server. Go recovers a panic only on the goroutine that raised
// it, and a stack overflow, a concurrent map write or running out of memory is
// not recoverable at all, so the one thing that survives every failure is a
// separate process. The editor talks to the supervisor; the supervisor runs
// the server as a child, relays every message both ways and remembers what a
// new server needs to carry on: the initialize request, the configuration and
// workspace folders the editor sent, and the current text of every open
// document. When the child exits without having been asked to, the
// supervisor starts another, replays that state to it, fails the requests the
// dead one left unanswered (ContentModified, which an editor retries or drops
// quietly), and tells the user.
package supervise

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cfmleditor/clif/internal/lspdoc"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// Child is one running language server.
type Child interface {
	// Stream is the server's stdio as an LSP stream.
	Stream() jsonrpc2.Stream
	// Wait blocks until the server exits and returns its exit code.
	Wait() int
	// Kill ends a server that has stopped answering.
	Kill()
}

// Options configures Run.
type Options struct {
	// Start starts a language server.
	Start func() (Child, error)
	// Editor is the editor's end of the session.
	Editor jsonrpc2.Stream
	// Log records one line about the supervisor's own work (a restart);
	// nil for none.
	Log func(msg string)
	// MaxRestarts within Window, past which the supervisor gives up and
	// exits as the server did, so a server that crashes at once every time
	// is not restarted forever. Zero values take 5 and three minutes.
	MaxRestarts int
	Window      time.Duration
	// ReplayTimeout bounds how long a new server may take to answer the
	// replayed initialize. Zero takes a minute.
	ReplayTimeout time.Duration
}

// contentModified is LSP's ContentModified: an editor retries the request or
// drops its answer, rather than showing an error.
const contentModified jsonrpc2.Code = -32801

type document struct {
	languageID string
	version    int32
	text       string
}

type exitEvent struct {
	gen  int
	code int
}

// notificationKind is what a notification from the editor is to the replay.
type notificationKind int

const (
	relayed  notificationKind = iota // passed on as it is
	replayed                         // state the replay sends as it stands: initialized, configuration, folders, document sync
)

type supervisor struct {
	opts Options

	// mu guards every field below. No write to either stream is made while
	// it is held: relaying the server's output takes it, so a write to the
	// server blocked on a full pipe while holding it stopped the server's
	// output being read, and a server that then wrote a pipe's worth before
	// reading again waited on the supervisor as the supervisor waited on it.
	mu         sync.Mutex
	child      Child
	stream     jsonrpc2.Stream
	gen        int
	restarting bool
	snapshot   bool // the replay has taken its snapshot of the session
	queue      []jsonrpc2.Message

	initID       jsonrpc2.ID
	initParams   jsonrpc2.RawMessage
	initAnswered bool
	initialized  bool
	config       jsonrpc2.RawMessage
	folders      []jsonrpc2.RawMessage
	docs         map[string]*document
	order        []string // docs' URIs in the order they were opened
	notice       string   // a restart notice held until the editor's initialize is answered

	registered    map[string]bool          // registration ids the editor has accepted
	registering   map[jsonrpc2.ID][]string // registration ids sent to the editor, by request, until it answers
	editorPending map[jsonrpc2.ID]string   // editor requests the server has not answered
	serverPending map[jsonrpc2.ID]int      // server requests the editor has not answered, by generation
	replayID      jsonrpc2.ID
	replayDone    chan struct{}

	shutdown     bool
	editorClosed bool
	restarts     []time.Time
	exits        chan exitEvent
}

// Run relays the session until it ends, restarting the server when it dies
// unasked, and returns the exit code the process should end with.
func Run(ctx context.Context, opts Options) int {
	if opts.MaxRestarts == 0 {
		opts.MaxRestarts = 5
	}

	if opts.Window == 0 {
		opts.Window = 3 * time.Minute
	}

	if opts.ReplayTimeout == 0 {
		opts.ReplayTimeout = time.Minute
	}

	s := &supervisor{
		opts:          opts,
		docs:          map[string]*document{},
		editorPending: map[jsonrpc2.ID]string{},
		serverPending: map[jsonrpc2.ID]int{},
		registered:    map[string]bool{},
		registering:   map[jsonrpc2.ID][]string{},
		exits:         make(chan exitEvent, 4),
	}

	child, err := opts.Start()
	if err != nil {
		s.logf("cannot start the language server: %v", err)

		return 1
	}

	s.adopt(child)

	editorDone := make(chan struct{})

	go func() {
		s.pumpEditor(ctx)
		close(editorDone)
	}()

	return s.watch(ctx, editorDone)
}

// watch waits for the server to exit, and restarts it while the session goes
// on.
func (s *supervisor) watch(ctx context.Context, editorDone <-chan struct{}) int {
	for {
		var ev exitEvent

		select {
		case ev = <-s.exits:
		case <-editorDone:
			// The editor has gone: let the server see its stdin close and
			// end as it would have with no supervisor.
			s.closeChild()

			ev = <-s.exits
		}

		s.mu.Lock()
		current, ended := ev.gen == s.gen, s.shutdown || s.editorClosed
		s.mu.Unlock()

		if !current {
			continue
		}

		if ended {
			s.answerShutdown(ctx)

			return ev.code
		}

		if !s.restart(ctx, ev.code) {
			return max(ev.code, 1)
		}
	}
}

// answerShutdown answers a shutdown the server died before answering: the
// session is over either way, and the editor otherwise waits for its own
// timeout.
func (s *supervisor) answerShutdown(ctx context.Context) {
	s.mu.Lock()

	var ids []jsonrpc2.ID

	for id, method := range s.editorPending {
		if method == protocol.MethodShutdown {
			ids = append(ids, id)
			delete(s.editorPending, id)
		}
	}

	s.mu.Unlock()

	for _, id := range ids {
		_, _ = s.opts.Editor.Write(ctx, jsonrpc2.NewResponse(id, jsonrpc2.RawMessage("null"), nil))
	}
}

// adopt makes child the server the session talks to.
func (s *supervisor) adopt(child Child) {
	s.mu.Lock()
	s.gen++
	gen := s.gen
	s.child, s.stream = child, child.Stream()
	s.mu.Unlock()

	go s.pumpChild(child, gen)
}

func (s *supervisor) closeChild() {
	s.mu.Lock()
	stream := s.stream
	s.mu.Unlock()

	_ = stream.Close()
}

// restart starts a new server after one exited with code, and replays the
// session to it; it reports false when the supervisor gives up.
func (s *supervisor) restart(ctx context.Context, code int) bool {
	now := time.Now()

	s.mu.Lock()
	s.restarts = append(s.restarts, now)

	for len(s.restarts) > 0 && now.Sub(s.restarts[0]) > s.opts.Window {
		s.restarts = s.restarts[1:]
	}

	tooMany := len(s.restarts) > s.opts.MaxRestarts
	s.restarting, s.snapshot = true, false
	s.mu.Unlock()

	s.failPending(ctx)

	if tooMany {
		s.logf("the language server exited %d times in %v; not restarting it", len(s.restarts), s.opts.Window)
		s.notify(ctx, protocol.MessageTypeError, fmt.Sprintf("clif's language server stopped %d times in %v and was not restarted. The crash log has the details.", len(s.restarts), s.opts.Window))

		return false
	}

	child, err := s.opts.Start()
	if err != nil {
		s.logf("cannot restart the language server: %v", err)

		return false
	}

	s.adopt(child)
	s.logf("the language server exited (code %d); restarted it", code)

	if !s.replay(ctx, child) {
		// The new server died, or was killed for not answering, during
		// the replay: its exit restarts it again, within the limit.
		return true
	}

	if code != 0 {
		s.noticeOnce(ctx, "clif's language server stopped after an internal error and was restarted. The crash log has the details.")
	}

	return true
}

// noticeOnce tells the user, or holds the notice until the editor's
// initialize is answered: before that a server may send it nothing.
func (s *supervisor) noticeOnce(ctx context.Context, message string) {
	s.mu.Lock()
	answered := s.initAnswered

	if !answered {
		s.notice = message
	}
	s.mu.Unlock()

	if answered {
		s.notify(ctx, protocol.MessageTypeWarning, message)
	}
}

// replay brings the new server to where the old one was, then sends what the
// editor said meanwhile. The writes are made without s.mu (see its comment);
// pumpEditor queues while s.restarting holds, and skips the state the
// snapshot already carries.
func (s *supervisor) replay(ctx context.Context, child Child) bool {
	stream := child.Stream()

	s.mu.Lock()
	gen := s.gen
	answered, params, initID := s.initAnswered, s.initParams, s.initID
	s.mu.Unlock()

	if params != nil && !answered {
		// The editor's initialize was never answered: send it again as it
		// was, and let the answer through.
		_, _ = stream.Write(ctx, jsonrpc2.NewCall(initID, protocol.MethodInitialize, params))
	}

	if params != nil && answered && !s.replayInitialize(ctx, child, gen) {
		return false
	}

	for _, m := range s.takeSnapshot() {
		_, _ = stream.Write(ctx, m)
	}

	for {
		s.mu.Lock()

		if gen != s.gen {
			s.mu.Unlock()

			return false
		}

		queued := s.queue
		s.queue = nil

		if len(queued) == 0 {
			s.restarting, s.snapshot = false, false
			s.mu.Unlock()

			return true
		}

		s.mu.Unlock()

		for _, m := range queued {
			_, _ = stream.Write(ctx, m)
		}
	}
}

// replayInitialize sends the editor's initialize to the new server under an
// id of the supervisor's own and waits for, and swallows, its answer. A server
// that does not answer in time is killed, and its exit restarts it.
func (s *supervisor) replayInitialize(ctx context.Context, child Child, gen int) bool {
	done := make(chan struct{})

	s.mu.Lock()
	s.replayID, s.replayDone = jsonrpc2.NewStringID(fmt.Sprintf("clif-supervisor-initialize-%d", gen)), done
	id, params := s.replayID, s.initParams
	s.mu.Unlock()

	_, _ = child.Stream().Write(ctx, jsonrpc2.NewCall(id, protocol.MethodInitialize, params))

	select {
	case <-done:
		return true
	case <-time.After(s.opts.ReplayTimeout):
		s.logf("the restarted language server did not answer initialize in %v; ending it", s.opts.ReplayTimeout)
		child.Kill()

		return false
	case <-ctx.Done():
		return false
	}
}

// takeSnapshot is the session as the new server needs it, and marks the
// moment: replayed state the editor sends after it is queued, not skipped.
func (s *supervisor) takeSnapshot() []jsonrpc2.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []jsonrpc2.Message

	if s.initialized {
		out = append(out, jsonrpc2.NewNotification(protocol.MethodInitialized, jsonrpc2.RawMessage("{}")))
	}

	if s.config != nil {
		out = append(out, jsonrpc2.NewNotification(protocol.MethodWorkspaceDidChangeConfiguration, s.config))
	}

	for _, f := range s.folders {
		out = append(out, jsonrpc2.NewNotification(protocol.MethodWorkspaceDidChangeWorkspaceFolders, f))
	}

	for _, u := range s.order {
		d := s.docs[u]
		params, _ := json.Marshal(map[string]any{"textDocument": map[string]any{
			"uri": u, "languageId": d.languageID, "version": d.version, "text": d.text,
		}})

		out = append(out, jsonrpc2.NewNotification(protocol.MethodTextDocumentDidOpen, params))
	}

	s.snapshot = true

	return out
}

// failPending answers every editor request the dead server left unanswered,
// except an initialize not yet answered, which the replay sends again, a
// shutdown, which answerShutdown answers, and one still queued.
func (s *supervisor) failPending(ctx context.Context) {
	s.mu.Lock()
	ids := make([]jsonrpc2.ID, 0, len(s.editorPending))

	// A request still in the queue reached no server: it goes to the next
	// one, and failing it as well gave the editor two answers to it.
	queued := map[jsonrpc2.ID]bool{}

	for _, m := range s.queue {
		if c, ok := m.(*jsonrpc2.Call); ok {
			queued[c.ID()] = true
		}
	}

	for id, method := range s.editorPending {
		if method == protocol.MethodInitialize && !s.initAnswered || method == protocol.MethodShutdown || queued[id] {
			continue
		}

		ids = append(ids, id)
		delete(s.editorPending, id)
	}

	s.serverPending = map[jsonrpc2.ID]int{}
	s.registering = map[jsonrpc2.ID][]string{}
	s.mu.Unlock()

	for _, id := range ids {
		_, _ = s.opts.Editor.Write(ctx, jsonrpc2.NewResponse(id, nil,
			jsonrpc2.NewError(contentModified, "clif's language server restarted; try again")))
	}
}

func (s *supervisor) notify(ctx context.Context, kind protocol.MessageType, message string) {
	params, _ := json.Marshal(map[string]any{"type": kind, "message": message})
	_, _ = s.opts.Editor.Write(ctx, jsonrpc2.NewNotification(protocol.MethodWindowShowMessage, params))
}

func (s *supervisor) logf(format string, args ...any) {
	if s.opts.Log != nil {
		s.opts.Log(fmt.Sprintf(format, args...))
	}
}

// pumpEditor relays the editor's messages to the server, noting what a new
// server would need, until the editor closes the session. It is the one
// writer of editor messages to the server, which keeps them in order though
// each write is made without s.mu.
func (s *supervisor) pumpEditor(ctx context.Context) {
	for {
		msg, _, err := s.opts.Editor.Read(ctx)
		if err != nil {
			s.mu.Lock()
			s.editorClosed = true
			s.mu.Unlock()

			return
		}

		out, stream, closeAfter := s.fromEditor(msg)
		if out == nil {
			continue
		}

		_, _ = stream.Write(ctx, out)

		// The server ends when its stdin does, not on exit itself; the
		// editor may keep the pipe open after exit, and then nothing would.
		if closeAfter {
			_ = stream.Close()
		}
	}
}

// fromEditor notes what msg changes and returns what to send the server and
// on which stream, or nil when it is queued or dropped.
func (s *supervisor) fromEditor(msg jsonrpc2.Message) (out jsonrpc2.Message, stream jsonrpc2.Stream, closeAfter bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch m := msg.(type) {
	case *jsonrpc2.Call:
		method := strings.Clone(m.Method())
		s.editorPending[m.ID()] = method

		switch method {
		case protocol.MethodInitialize:
			s.initID, s.initParams = m.ID(), clone(m.Params())
		case protocol.MethodShutdown:
			s.shutdown = true
		}

		msg = jsonrpc2.NewCall(m.ID(), method, clone(m.Params()))
	case *jsonrpc2.Notification:
		method := strings.Clone(m.Method())
		params := clone(m.Params())

		if s.track(method, params) == replayed && s.restarting && !s.snapshot {
			// The replay's snapshot, still to be taken, carries it.
			return nil, nil, false
		}

		msg = jsonrpc2.NewNotification(method, params)
		closeAfter = method == protocol.MethodExit
	case *jsonrpc2.Response:
		if gen, ok := s.serverPending[m.ID()]; !ok || gen != s.gen {
			return nil, nil, false // an answer for a server that has gone
		}

		delete(s.serverPending, m.ID())

		if ids, ok := s.registering[m.ID()]; ok {
			delete(s.registering, m.ID())

			if m.Err() == nil {
				for _, id := range ids {
					s.registered[id] = true
				}
			}
		}
	}

	if s.restarting {
		s.queue = append(s.queue, msg)

		return nil, nil, false
	}

	return msg, s.stream, closeAfter
}

// track notes what a notification changes in the session, and says whether
// the replay carries it.
func (s *supervisor) track(method string, params jsonrpc2.RawMessage) notificationKind {
	switch method {
	case protocol.MethodInitialized:
		s.initialized = true
	case protocol.MethodExit:
		s.shutdown = true

		return relayed
	case protocol.MethodWorkspaceDidChangeConfiguration:
		s.config = params
	case protocol.MethodWorkspaceDidChangeWorkspaceFolders:
		s.folders = append(s.folders, params)
	case protocol.MethodTextDocumentDidOpen:
		s.trackOpen(params)
	case protocol.MethodTextDocumentDidChange:
		var p protocol.DidChangeTextDocumentParams
		if json.Unmarshal(params, &p) == nil {
			if d := s.docs[string(p.TextDocument.URI)]; d != nil {
				d.text, d.version = lspdoc.ApplyChanges(d.text, p.ContentChanges), p.TextDocument.Version
			}
		}
	case protocol.MethodTextDocumentDidClose:
		s.trackClose(params)
	default:
		return relayed
	}

	return replayed
}

func (s *supervisor) trackOpen(params jsonrpc2.RawMessage) {
	var p protocol.DidOpenTextDocumentParams
	if json.Unmarshal(params, &p) != nil {
		return
	}

	u := string(p.TextDocument.URI)
	if _, open := s.docs[u]; !open {
		s.order = append(s.order, u)
	}

	s.docs[u] = &document{languageID: string(p.TextDocument.LanguageID), version: p.TextDocument.Version, text: p.TextDocument.Text}
}

func (s *supervisor) trackClose(params jsonrpc2.RawMessage) {
	var p protocol.DidCloseTextDocumentParams
	if json.Unmarshal(params, &p) != nil {
		return
	}

	u := string(p.TextDocument.URI)
	delete(s.docs, u)

	for i, o := range s.order {
		if o == u {
			s.order = append(s.order[:i], s.order[i+1:]...)

			break
		}
	}
}

// pumpChild relays the server's messages to the editor until the server's
// stdout closes, then reports its exit.
func (s *supervisor) pumpChild(child Child, gen int) {
	ctx := context.Background()
	stream := child.Stream()

	for {
		msg, _, err := stream.Read(ctx)
		if err != nil {
			break
		}

		s.fromChild(ctx, msg, gen, stream)
	}

	s.exits <- exitEvent{gen: gen, code: child.Wait()}
}

func (s *supervisor) fromChild(ctx context.Context, msg jsonrpc2.Message, gen int, stream jsonrpc2.Stream) {
	out, toServer, notice := s.routeFromChild(msg, gen)

	if toServer != nil {
		_, _ = stream.Write(ctx, toServer)
	}

	if out != nil {
		if _, err := s.opts.Editor.Write(ctx, out); err != nil && !errors.Is(err, context.Canceled) {
			s.logf("cannot write to the editor: %v", err)
		}
	}

	if notice != "" {
		s.notify(ctx, protocol.MessageTypeWarning, notice)
	}
}

// routeFromChild decides what a message from the server becomes: out for the
// editor, toServer answered on its behalf, and a held restart notice to send
// once the editor's initialize has been answered.
func (s *supervisor) routeFromChild(msg jsonrpc2.Message, gen int) (out, toServer jsonrpc2.Message, notice string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if gen != s.gen {
		return nil, nil, ""
	}

	switch m := msg.(type) {
	case *jsonrpc2.Response:
		if s.replayDone != nil && m.ID() == s.replayID {
			close(s.replayDone)
			s.replayDone = nil

			return nil, nil, ""
		}

		if method, ok := s.editorPending[m.ID()]; ok {
			delete(s.editorPending, m.ID())

			if method == protocol.MethodInitialize {
				s.initAnswered = true
				notice, s.notice = s.notice, ""
			}
		}

		return msg, nil, notice
	case *jsonrpc2.Call:
		method := strings.Clone(m.Method())

		if method == protocol.MethodClientRegisterCapability {
			ids := registrationIDs(m.Params())
			if len(ids) > 0 && s.allRegistered(ids) {
				// A restarted server registering what the editor holds
				// from the first: registering an id twice is an error to
				// an editor.
				return nil, jsonrpc2.NewResponse(m.ID(), jsonrpc2.RawMessage("null"), nil), ""
			}

			s.registering[m.ID()] = ids
		}

		s.serverPending[m.ID()] = gen

		return jsonrpc2.NewCall(m.ID(), method, clone(m.Params())), nil, ""
	case *jsonrpc2.Notification:
		return jsonrpc2.NewNotification(strings.Clone(m.Method()), clone(m.Params())), nil, ""
	}

	return msg, nil, ""
}

// registrationIDs is the ids a client/registerCapability registers.
func registrationIDs(params jsonrpc2.RawMessage) []string {
	var p struct {
		Registrations []struct {
			ID string `json:"id"`
		} `json:"registrations"`
	}

	if json.Unmarshal(params, &p) != nil {
		return nil
	}

	ids := make([]string, 0, len(p.Registrations))
	for _, r := range p.Registrations {
		ids = append(ids, r.ID)
	}

	return ids
}

// allRegistered reports whether the editor accepted every one of ids from an
// earlier server; s.mu is held.
func (s *supervisor) allRegistered(ids []string) bool {
	for _, id := range ids {
		if !s.registered[id] {
			return false
		}
	}

	return true
}

// clone copies params out of the stream's read buffer, which the next Read
// reuses.
func clone(p jsonrpc2.RawMessage) jsonrpc2.RawMessage {
	if p == nil {
		return nil
	}

	return append(jsonrpc2.RawMessage(nil), p...)
}
