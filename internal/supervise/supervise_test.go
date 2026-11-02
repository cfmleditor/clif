package supervise

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/lspdoc"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// behaviour is how one fake server misbehaves.
type behaviour struct {
	hangInit      bool          // never answers initialize
	initDelay     time.Duration // answers initialize this late
	dieOnInit     bool          // exits with code 2 on initialize
	dieOnShutdown bool          // exits with code 2 on shutdown, unanswered
	capacity      int           // the stream's buffer, in messages; 64 when zero
	ignoreExit    bool          // ends, with code 0, only when its input does
}

// fakeServer is an in-process language server: it answers initialize and
// shutdown, keeps every open document's text, counts workspace folder
// changes, registers a capability when initialized, never answers test/slow,
// reports its state for test/state, floods its output on test/flood without
// reading, and dies with exit code 2 on test/crash.
type fakeServer struct {
	stream jsonrpc2.Stream
	code   chan int
	b      behaviour

	mu          sync.Mutex
	inits       int
	initialized bool
	folders     int
	docs        map[string]string
	versions    map[string]int32
}

func startFake(b behaviour) *fakeServer {
	capacity := b.capacity
	if capacity == 0 {
		capacity = 64
	}

	ours, theirs := jsonrpc2.NewChannelStreamPair(capacity)
	f := &fakeServer{stream: ours, code: make(chan int, 1), b: b, docs: map[string]string{}, versions: map[string]int32{}}

	go f.serve(theirs)

	return f
}

func (f *fakeServer) Stream() jsonrpc2.Stream { return f.stream }
func (f *fakeServer) Wait() int               { return <-f.code }
func (f *fakeServer) Kill()                   { _ = f.stream.Close() }

func (f *fakeServer) serve(s jsonrpc2.Stream) {
	ctx := context.Background()

	for {
		msg, _, err := s.Read(ctx)
		if err != nil {
			// One that ignores exit ends cleanly when its input does. (A
			// channel stream's close can overtake a message it buffered,
			// so whether it read the exit first is not a thing to test.)
			if f.b.ignoreExit {
				f.code <- 0
			} else {
				f.code <- 1
			}

			return
		}

		if code, done := f.handle(ctx, s, msg); done {
			_ = s.Close()

			f.code <- code

			return
		}
	}
}

func (f *fakeServer) reply(ctx context.Context, s jsonrpc2.Stream, id jsonrpc2.ID, v any) {
	b, _ := json.Marshal(v)
	_, _ = s.Write(ctx, jsonrpc2.NewResponse(id, b, nil))
}

func (f *fakeServer) handle(ctx context.Context, s jsonrpc2.Stream, msg jsonrpc2.Message) (code int, done bool) {
	switch m := msg.(type) {
	case *jsonrpc2.Call:
		return f.call(ctx, s, m)
	case *jsonrpc2.Notification:
		return f.notification(ctx, s, m)
	}

	return 0, false
}

func (f *fakeServer) call(ctx context.Context, s jsonrpc2.Stream, m *jsonrpc2.Call) (code int, done bool) {
	switch m.Method() {
	case protocol.MethodInitialize:
		if f.b.dieOnInit {
			return 2, true
		}

		f.mu.Lock()
		f.inits++
		f.mu.Unlock()

		if f.b.hangInit {
			return 0, false
		}

		time.Sleep(f.b.initDelay)
		f.reply(ctx, s, m.ID(), map[string]any{"capabilities": map[string]any{}})
	case protocol.MethodShutdown:
		if f.b.dieOnShutdown {
			return 2, true
		}

		f.reply(ctx, s, m.ID(), nil)
	case "test/state":
		f.mu.Lock()
		st := map[string]any{"inits": f.inits, "initialized": f.initialized, "folders": f.folders, "docs": f.docs, "versions": f.versions}
		f.mu.Unlock()

		f.reply(ctx, s, m.ID(), st)
	case "test/flood":
		// Write a lot before reading again, as a handler logging as it goes
		// does: the supervisor must keep draining while it writes to us.
		for range 200 {
			_, _ = s.Write(ctx, jsonrpc2.NewNotification(protocol.MethodWindowLogMessage, jsonrpc2.RawMessage(`{"type":4,"message":"working"}`)))
		}

		f.reply(ctx, s, m.ID(), "flooded")
	case "test/crash":
		return 2, true
	}

	return 0, false
}

func (f *fakeServer) notification(ctx context.Context, s jsonrpc2.Stream, m *jsonrpc2.Notification) (code int, done bool) {
	params := append([]byte(nil), m.Params()...)

	f.mu.Lock()
	defer f.mu.Unlock()

	switch m.Method() {
	case protocol.MethodInitialized:
		f.initialized = true
		_, _ = s.Write(ctx, jsonrpc2.NewCall(jsonrpc2.NewNumberID(1), protocol.MethodClientRegisterCapability, jsonrpc2.RawMessage(`{"registrations":[{"id":"clif-watched-files","method":"workspace/didChangeWatchedFiles"}]}`)))
	case protocol.MethodWorkspaceDidChangeWorkspaceFolders:
		f.folders++
	case protocol.MethodTextDocumentDidOpen:
		var p protocol.DidOpenTextDocumentParams
		if json.Unmarshal(params, &p) == nil {
			f.docs[string(p.TextDocument.URI)] = p.TextDocument.Text
			f.versions[string(p.TextDocument.URI)] = p.TextDocument.Version
		}
	case protocol.MethodTextDocumentDidChange:
		var p protocol.DidChangeTextDocumentParams
		if json.Unmarshal(params, &p) == nil {
			u := string(p.TextDocument.URI)
			f.docs[u] = lspdoc.ApplyChanges(f.docs[u], p.ContentChanges)
			f.versions[u] = p.TextDocument.Version
		}
	case protocol.MethodExit:
		return 0, !f.b.ignoreExit
	}

	return 0, false
}

// editor is the test's end of the session: it sends, and collects what the
// supervisor sends back in order, answering the server's registerCapability
// (with an error when reject is set).
type editor struct {
	t      *testing.T
	stream jsonrpc2.Stream
	reject bool
	delay  time.Duration // answers a registration this late

	mu        sync.Mutex
	responses map[jsonrpc2.ID]*jsonrpc2.Response
	order     []string // "response:<id>" and "notice", in arrival order
	shown     []string
	registers int
	answered  int // registrations answered, the write done
}

func newEditor(t *testing.T, capacity int) (*editor, jsonrpc2.Stream) {
	t.Helper()

	ours, theirs := jsonrpc2.NewChannelStreamPair(capacity)
	e := &editor{t: t, stream: ours, responses: map[jsonrpc2.ID]*jsonrpc2.Response{}}

	go e.read()

	return e, theirs
}

func (e *editor) read() {
	ctx := context.Background()

	for {
		msg, _, err := e.stream.Read(ctx)
		if err != nil {
			return
		}

		e.mu.Lock()

		switch m := msg.(type) {
		case *jsonrpc2.Response:
			e.responses[m.ID()] = m
			e.order = append(e.order, fmt.Sprintf("response:%v", m.ID()))
		case *jsonrpc2.Notification:
			if m.Method() == protocol.MethodWindowShowMessage {
				e.shown = append(e.shown, string(m.Params()))
				e.order = append(e.order, "notice")
			}
		case *jsonrpc2.Call:
			if m.Method() == protocol.MethodClientRegisterCapability {
				e.registers++

				var err error
				if e.reject {
					err = jsonrpc2.NewError(jsonrpc2.InvalidParams, "not supported")
				}

				// From a goroutine of its own, as an editor's writer is: a
				// reader that blocks on its own write is a deadlock of the
				// test's making.
				go func(id jsonrpc2.ID) {
					time.Sleep(e.delay)

					_, _ = e.stream.Write(ctx, jsonrpc2.NewResponse(id, jsonrpc2.RawMessage("null"), err))

					e.mu.Lock()
					e.answered++
					e.mu.Unlock()
				}(m.ID())
			}
		}

		e.mu.Unlock()
	}
}

func (e *editor) send(m jsonrpc2.Message) {
	if _, err := e.stream.Write(context.Background(), m); err != nil {
		e.t.Fatal(err)
	}
}

func (e *editor) call(id int64, method, params string) {
	e.send(jsonrpc2.NewCall(jsonrpc2.NewNumberID(id), method, jsonrpc2.RawMessage(params)))
}

func (e *editor) notify(method, params string) {
	e.send(jsonrpc2.NewNotification(method, jsonrpc2.RawMessage(params)))
}

// await waits for the response to id.
func (e *editor) await(id int64) *jsonrpc2.Response {
	e.t.Helper()

	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		e.mu.Lock()
		r := e.responses[jsonrpc2.NewNumberID(id)]
		e.mu.Unlock()

		if r != nil {
			return r
		}

		time.Sleep(5 * time.Millisecond)
	}

	e.t.Fatalf("no response to request %d", id)

	return nil
}

// settled waits until the editor has answered n registrations, then makes a
// round trip through the server so the supervisor has relayed the answers:
// it reads the editor's messages in order.
func (e *editor) settled(n int, id int64) {
	e.t.Helper()

	deadline := time.Now().Add(10 * time.Second)

	for {
		e.mu.Lock()
		answered := e.answered
		e.mu.Unlock()

		if answered >= n {
			break
		}

		if time.Now().After(deadline) {
			e.t.Fatalf("the editor answered %d registrations, want %d", answered, n)
		}

		time.Sleep(5 * time.Millisecond)
	}

	e.state(id)
}

func (e *editor) start() {
	e.call(1, protocol.MethodInitialize, `{"processId":1,"capabilities":{}}`)
	e.await(1)
	e.notify(protocol.MethodInitialized, `{}`)
}

type serverState struct {
	Inits       int               `json:"inits"`
	Initialized bool              `json:"initialized"`
	Folders     int               `json:"folders"`
	Docs        map[string]string `json:"docs"`
	Versions    map[string]int32  `json:"versions"`
}

func (e *editor) state(id int64) serverState {
	e.t.Helper()
	e.call(id, "test/state", `{}`)

	var st serverState
	if err := json.Unmarshal(e.await(id).Result(), &st); err != nil {
		e.t.Fatal(err)
	}

	return st
}

// servers starts a fake per call with the next behaviour in plan (the last
// one repeating), and records them.
type servers struct {
	mu    sync.Mutex
	plan  []behaviour
	fakes []*fakeServer
}

func (sv *servers) start() (Child, error) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	if len(sv.fakes) > 10 {
		return nil, errors.New("too many servers started")
	}

	f := startFake(sv.plan[min(len(sv.fakes), len(sv.plan)-1)])
	sv.fakes = append(sv.fakes, f)

	return f, nil
}

func (sv *servers) count() int {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	return len(sv.fakes)
}

func run(t *testing.T, editorEnd jsonrpc2.Stream, sv *servers, opts Options) <-chan int {
	t.Helper()

	opts.Start, opts.Editor = sv.start, editorEnd
	done := make(chan int, 1)

	go func() { done <- Run(context.Background(), opts) }()

	return done
}

func endSession(t *testing.T, e *editor, done <-chan int, id int64) int {
	t.Helper()

	e.call(id, protocol.MethodShutdown, `null`)
	e.await(id)
	e.notify(protocol.MethodExit, `null`)

	select {
	case code := <-done:
		return code
	case <-time.After(10 * time.Second):
		t.Fatal("the supervisor did not end after shutdown and exit")
	}

	return -1
}

func isContentModified(r *jsonrpc2.Response) bool {
	var rpcErr *jsonrpc2.Error

	return errors.As(r.Err(), &rpcErr) && rpcErr.Code == contentModified
}

// A server that dies unasked is restarted with the session replayed: the
// editor's initialize, initialized and every open document as it stands, its
// unanswered request failed with ContentModified, its user told, and the
// replayed registration answered without troubling the editor again.
func TestACrashedServerIsRestartedWithTheSessionReplayed(t *testing.T) {
	e, editorEnd := newEditor(t, 64)
	sv := &servers{plan: []behaviour{{}}}
	done := run(t, editorEnd, sv, Options{})

	e.start()
	e.notify(protocol.MethodTextDocumentDidOpen, `{"textDocument":{"uri":"file:///a.cfm","languageId":"cfml","version":1,"text":"<cfset a = 1>"}}`)
	e.notify(protocol.MethodTextDocumentDidChange, `{"textDocument":{"uri":"file:///a.cfm","version":2},"contentChanges":[{"range":{"start":{"line":0,"character":11},"end":{"line":0,"character":12}},"text":"2"}]}`)
	e.notify(protocol.MethodTextDocumentDidOpen, `{"textDocument":{"uri":"file:///b.cfm","languageId":"cfml","version":1,"text":"b"}}`)
	e.notify(protocol.MethodTextDocumentDidClose, `{"textDocument":{"uri":"file:///b.cfm"}}`)
	e.settled(1, 20)
	e.call(2, "test/slow", `{}`)
	e.call(3, "test/crash", `{}`)

	for _, id := range []int64{2, 3} {
		if r := e.await(id); !isContentModified(r) {
			t.Fatalf("request %d: got %v, want ContentModified", id, r.Err())
		}
	}

	st := e.state(4)

	if st.Inits != 1 || !st.Initialized {
		t.Fatalf("the new server saw %d initialize and initialized=%v, want 1 and true", st.Inits, st.Initialized)
	}

	if len(st.Docs) != 1 || st.Docs["file:///a.cfm"] != "<cfset a = 2>" || st.Versions["file:///a.cfm"] != 2 {
		t.Fatalf("the new server holds %v at versions %v, want a.cfm at version 2 as edited and b.cfm closed", st.Docs, st.Versions)
	}

	e.mu.Lock()
	shown, registers := len(e.shown), e.registers
	e.mu.Unlock()

	if shown != 1 {
		t.Fatalf("the editor was told %d times, want once", shown)
	}

	if registers != 1 {
		t.Fatalf("the editor was asked to register %d times, want once: the replay's registration should be answered by the supervisor", registers)
	}

	if n := sv.count(); n != 2 {
		t.Fatalf("%d servers started, want 2", n)
	}

	if code := endSession(t, e, done, 5); code != 0 {
		t.Fatalf("exit code %d after a clean shutdown, want 0", code)
	}
}

// A server that dies at once every time is not restarted forever: past the
// limit the supervisor gives up with the server's exit code.
func TestAServerThatKeepsCrashingIsGivenUpOn(t *testing.T) {
	_, editorEnd := newEditor(t, 64)

	starts := 0
	start := func() (Child, error) {
		starts++
		if starts > 4 {
			return nil, errors.New("started again after giving up")
		}

		f := startFake(behaviour{})
		_ = f.Stream().Close() // dies at once

		return f, nil
	}

	done := make(chan int, 1)

	go func() {
		done <- Run(context.Background(), Options{Start: start, Editor: editorEnd, MaxRestarts: 3})
	}()

	select {
	case code := <-done:
		if code == 0 {
			t.Fatal("gave up with exit code 0")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the supervisor kept restarting a server that dies at once")
	}

	if starts != 4 {
		t.Fatalf("%d starts, want 4 (the first and three restarts)", starts)
	}
}

// The supervisor never writes to the server while holding the lock its relay
// of the server's output needs: a server that writes a lot before it reads
// again, while the editor keeps sending, would otherwise wait on the
// supervisor as the supervisor waited on it.
func TestASlowServerWritingALotDoesNotDeadlockTheRelay(t *testing.T) {
	e, editorEnd := newEditor(t, 2)
	sv := &servers{plan: []behaviour{{capacity: 2}}}
	done := run(t, editorEnd, sv, Options{})

	e.start()
	e.notify(protocol.MethodTextDocumentDidOpen, `{"textDocument":{"uri":"file:///a.cfm","languageId":"cfml","version":1,"text":"x"}}`)
	e.call(2, "test/flood", `{}`)

	for i := range 50 {
		e.notify(protocol.MethodTextDocumentDidChange, `{"textDocument":{"uri":"file:///a.cfm","version":`+strconv.Itoa(i+2)+`},"contentChanges":[{"text":"x"}]}`)
	}

	if r := e.await(2); r.Err() != nil {
		t.Fatalf("flood: %v", r.Err())
	}

	if code := endSession(t, e, done, 3); code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
}

// A restarted server that does not answer the replayed initialize is ended
// and replaced, rather than the session queueing forever behind it.
func TestARestartedServerThatHangsIsReplaced(t *testing.T) {
	e, editorEnd := newEditor(t, 64)
	sv := &servers{plan: []behaviour{{}, {hangInit: true}, {}}}
	done := run(t, editorEnd, sv, Options{ReplayTimeout: 200 * time.Millisecond})

	e.start()
	e.call(2, "test/crash", `{}`)
	e.await(2)

	if st := e.state(3); st.Inits != 1 || !st.Initialized {
		t.Fatalf("the replacement saw %d initialize and initialized=%v", st.Inits, st.Initialized)
	}

	if n := sv.count(); n != 3 {
		t.Fatalf("%d servers started, want 3: the first, the one that hung, and its replacement", n)
	}

	if code := endSession(t, e, done, 4); code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
}

// A workspace folder change sent while the server restarts reaches the new
// server once: the replay carries it or the queue does, not both.
func TestAFolderChangeDuringARestartIsSentOnce(t *testing.T) {
	e, editorEnd := newEditor(t, 64)
	sv := &servers{plan: []behaviour{{}, {initDelay: 300 * time.Millisecond}}}
	done := run(t, editorEnd, sv, Options{})

	e.start()
	e.call(2, "test/crash", `{}`)
	e.await(2)

	// The new server is answering initialize late: the change arrives
	// during the replay.
	e.notify(protocol.MethodWorkspaceDidChangeWorkspaceFolders, `{"event":{"added":[{"uri":"file:///w2","name":"w2"}],"removed":[]}}`)

	if st := e.state(3); st.Folders != 1 {
		t.Fatalf("the new server saw the folder change %d times, want once", st.Folders)
	}

	if code := endSession(t, e, done, 4); code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
}

// A shutdown the server dies before answering is answered by the supervisor:
// the session is over either way.
func TestAShutdownTheServerDiesOnIsAnswered(t *testing.T) {
	e, editorEnd := newEditor(t, 64)
	sv := &servers{plan: []behaviour{{dieOnShutdown: true}}}
	done := run(t, editorEnd, sv, Options{})

	e.start()
	e.call(2, protocol.MethodShutdown, `null`)

	if r := e.await(2); r.Err() != nil {
		t.Fatalf("shutdown: %v", r.Err())
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the supervisor did not end")
	}
}

// A server that dies answering initialize is restarted, and the editor is
// told only after its initialize has been answered: before that a server may
// send it nothing.
func TestTheRestartNoticeWaitsForInitialize(t *testing.T) {
	e, editorEnd := newEditor(t, 64)
	// The replacement answers late, so a notice sent before the answer
	// would arrive first.
	sv := &servers{plan: []behaviour{{dieOnInit: true}, {initDelay: 200 * time.Millisecond}}}
	done := run(t, editorEnd, sv, Options{})

	e.call(1, protocol.MethodInitialize, `{"processId":1,"capabilities":{}}`)

	if r := e.await(1); r.Err() != nil {
		t.Fatalf("initialize: %v", r.Err())
	}

	deadline := time.Now().Add(5 * time.Second)

	for {
		e.mu.Lock()
		order := append([]string(nil), e.order...)
		e.mu.Unlock()

		if len(order) >= 2 {
			if order[0] != "response:1" || order[1] != "notice" {
				t.Fatalf("the editor received %v, want the initialize response and then the notice", order)
			}

			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("the editor received %v, want the initialize response and then the notice", order)
		}

		time.Sleep(5 * time.Millisecond)
	}

	e.notify(protocol.MethodInitialized, `{}`)
	endSession(t, e, done, 2)
}

// A registration the editor rejected is not taken as held: a restarted
// server's same registration goes to the editor again.
func TestARejectedRegistrationIsAskedAgainAfterARestart(t *testing.T) {
	e, editorEnd := newEditor(t, 64)
	e.reject = true
	sv := &servers{plan: []behaviour{{}}}
	done := run(t, editorEnd, sv, Options{})

	e.start()
	e.settled(1, 2)
	e.call(3, "test/crash", `{}`)
	e.await(3)
	e.state(4)

	e.mu.Lock()
	registers := e.registers
	e.mu.Unlock()

	if registers != 2 {
		t.Fatalf("the editor was asked to register %d times, want 2: it rejected the first", registers)
	}

	endSession(t, e, done, 5)
}

// A panic tracking the editor's messages does not end the supervisor: the
// document it could not follow is dropped, and a restart does not reopen it.
func TestAPanicTrackingADocumentDropsTheDocument(t *testing.T) {
	applyChanges = func(string, []protocol.TextDocumentContentChangeEvent) string { panic("cannot apply") }

	t.Cleanup(func() { applyChanges = lspdoc.ApplyChanges })

	e, editorEnd := newEditor(t, 64)
	sv := &servers{plan: []behaviour{{}}}
	done := run(t, editorEnd, sv, Options{})

	e.start()
	e.notify(protocol.MethodTextDocumentDidOpen, `{"textDocument":{"uri":"file:///a.cfm","languageId":"cfml","version":1,"text":"a"}}`)
	e.notify(protocol.MethodTextDocumentDidOpen, `{"textDocument":{"uri":"file:///b.cfm","languageId":"cfml","version":1,"text":"b"}}`)
	e.notify(protocol.MethodTextDocumentDidChange, `{"textDocument":{"uri":"file:///a.cfm","version":2},"contentChanges":[{"text":"aa"}]}`)

	if st := e.state(2); st.Docs["file:///a.cfm"] != "aa" {
		t.Fatalf("the server has %q, want the change relayed whatever the supervisor's copy", st.Docs["file:///a.cfm"])
	}

	e.call(3, "test/crash", `{}`)
	e.await(3)

	st := e.state(4)
	if _, ok := st.Docs["file:///a.cfm"]; ok {
		t.Fatalf("the restarted server was given a.cfm, which the supervisor could not follow")
	}

	if st.Docs["file:///b.cfm"] != "b" {
		t.Fatalf("the restarted server has b.cfm as %q, want it reopened", st.Docs["file:///b.cfm"])
	}

	endSession(t, e, done, 5)
}

// A replacement that dies on the replayed initialize is restarted when it
// dies, not when the replay times out.
func TestAReplacementDyingOnInitializeIsRestartedAtOnce(t *testing.T) {
	e, editorEnd := newEditor(t, 64)
	sv := &servers{plan: []behaviour{{}, {dieOnInit: true}, {}}}
	done := run(t, editorEnd, sv, Options{ReplayTimeout: time.Minute})

	e.start()

	began := time.Now()

	e.call(2, "test/crash", `{}`)
	e.await(2)
	e.state(3)

	if took := time.Since(began); took > 5*time.Second {
		t.Fatalf("the session came back after %v: it waited out the replay timeout", took)
	}

	if n := sv.count(); n != 3 {
		t.Fatalf("%d servers started, want 3", n)
	}

	endSession(t, e, done, 4)
}

// A shutdown and exit the editor sends while the server restarts end the
// session: the queued exit closes the new server's input, which is what ends
// a server that does not end on exit alone.
func TestAnExitSentDuringARestartEndsTheSession(t *testing.T) {
	e, editorEnd := newEditor(t, 64)
	sv := &servers{plan: []behaviour{{}, {initDelay: 300 * time.Millisecond, ignoreExit: true}}}
	done := run(t, editorEnd, sv, Options{})

	e.start()
	e.call(2, "test/crash", `{}`)
	e.await(2)

	// Both before the new server answers initialize, so both are queued.
	e.call(3, protocol.MethodShutdown, `null`)
	e.notify(protocol.MethodExit, `null`)

	if r := e.await(3); r.Err() != nil {
		t.Fatalf("shutdown: %v", r.Err())
	}

	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("exit code %d, want 0", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the supervisor did not end after a queued shutdown and exit")
	}
}

// A registration the editor accepts after its server died is held: the
// restarted server's same registration does not go to the editor again.
func TestARegistrationAcceptedAfterTheCrashIsNotAskedAgain(t *testing.T) {
	e, editorEnd := newEditor(t, 64)
	e.delay = 100 * time.Millisecond
	sv := &servers{plan: []behaviour{{}, {initDelay: 300 * time.Millisecond}}}
	done := run(t, editorEnd, sv, Options{})

	e.start()
	e.call(2, "test/crash", `{}`)
	e.await(2)
	e.settled(1, 3)
	e.state(4)

	e.mu.Lock()
	registers := e.registers
	e.mu.Unlock()

	if registers != 1 {
		t.Fatalf("the editor was asked to register %d times, want once", registers)
	}

	endSession(t, e, done, 5)
}

// A restart drops the queued notifications its replay carries as state, so a
// second restart before the queue is flushed does not send them twice.
func TestARestartDropsQueuedStateTheReplayCarries(t *testing.T) {
	note := func(method string) jsonrpc2.Message {
		return jsonrpc2.NewNotification(method, jsonrpc2.RawMessage(`{}`))
	}
	call := jsonrpc2.NewCall(jsonrpc2.NewNumberID(7), protocol.MethodTextDocumentHover, jsonrpc2.RawMessage(`{}`))
	queue := []jsonrpc2.Message{
		note(protocol.MethodWorkspaceDidChangeWorkspaceFolders),
		call,
		note(protocol.MethodTextDocumentDidOpen),
		note(protocol.MethodTextDocumentDidChange),
		note(protocol.MethodTextDocumentDidSave),
		note(protocol.MethodTextDocumentDidClose),
		note(protocol.MethodWorkspaceDidChangeConfiguration),
		note(protocol.MethodExit),
	}

	var got []string

	for _, m := range withoutReplayedState(queue) {
		switch m := m.(type) {
		case *jsonrpc2.Call:
			got = append(got, "call:"+m.Method())
		case *jsonrpc2.Notification:
			got = append(got, m.Method())
		}
	}

	want := []string{"call:" + protocol.MethodTextDocumentHover, protocol.MethodTextDocumentDidSave, protocol.MethodExit}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("kept %v, want %v", got, want)
	}
}
