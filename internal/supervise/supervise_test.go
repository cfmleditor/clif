package supervise

import (
	"context"
	"encoding/json/v2"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/lspdoc"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// fakeServer is an in-process language server: it answers initialize and
// shutdown, keeps every open document's text, registers a capability when
// initialized, never answers test/slow, reports its state for test/state, and
// dies with exit code 2 on test/crash.
type fakeServer struct {
	stream jsonrpc2.Stream
	code   chan int

	inits       int
	initialized bool
	docs        map[string]string
	versions    map[string]int32
}

func startFake() (Child, *fakeServer) {
	ours, theirs := jsonrpc2.NewChannelStreamPair(64)
	f := &fakeServer{stream: ours, code: make(chan int, 1), docs: map[string]string{}, versions: map[string]int32{}}

	go f.serve(theirs)

	return f, f
}

func (f *fakeServer) Stream() jsonrpc2.Stream { return f.stream }
func (f *fakeServer) Wait() int               { return <-f.code }

func (f *fakeServer) serve(s jsonrpc2.Stream) {
	ctx := context.Background()

	for {
		msg, _, err := s.Read(ctx)
		if err != nil {
			f.code <- 1

			return
		}

		if code, done := f.handle(ctx, s, msg); done {
			_ = s.Close()

			f.code <- code

			return
		}
	}
}

func (f *fakeServer) handle(ctx context.Context, s jsonrpc2.Stream, msg jsonrpc2.Message) (code int, done bool) {
	reply := func(id jsonrpc2.ID, v any) {
		b, _ := json.Marshal(v)
		_, _ = s.Write(ctx, jsonrpc2.NewResponse(id, b, nil))
	}

	switch m := msg.(type) {
	case *jsonrpc2.Call:
		switch m.Method() {
		case protocol.MethodInitialize:
			f.inits++

			reply(m.ID(), map[string]any{"capabilities": map[string]any{}})
		case protocol.MethodShutdown:
			reply(m.ID(), nil)
		case "test/state":
			reply(m.ID(), map[string]any{"inits": f.inits, "initialized": f.initialized, "docs": f.docs, "versions": f.versions})
		case "test/crash":
			return 2, true
		}
	case *jsonrpc2.Notification:
		params := append([]byte(nil), m.Params()...)

		switch m.Method() {
		case protocol.MethodInitialized:
			f.initialized = true
			_, _ = s.Write(ctx, jsonrpc2.NewCall(jsonrpc2.NewNumberID(1), protocol.MethodClientRegisterCapability, jsonrpc2.RawMessage(`{"registrations":[{"id":"clif-watched-files","method":"workspace/didChangeWatchedFiles"}]}`)))
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
			return 0, true
		}
	}

	return 0, false
}

// editor is the test's end of the session: it sends, and collects what the
// supervisor sends back, answering the server's registerCapability.
type editor struct {
	t      *testing.T
	stream jsonrpc2.Stream

	mu        sync.Mutex
	responses map[jsonrpc2.ID]*jsonrpc2.Response
	shown     []string
	registers int
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
		case *jsonrpc2.Notification:
			if m.Method() == protocol.MethodWindowShowMessage {
				e.shown = append(e.shown, string(m.Params()))
			}
		case *jsonrpc2.Call:
			if m.Method() == protocol.MethodClientRegisterCapability {
				e.registers++
				_, _ = e.stream.Write(ctx, jsonrpc2.NewResponse(m.ID(), jsonrpc2.RawMessage("null"), nil))
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

	deadline := time.Now().Add(5 * time.Second)

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

// A server that dies unasked is restarted with the session replayed: the
// editor's initialize, initialized and every open document as it stands, its
// unanswered request failed with ContentModified, its user told, and the
// replayed registration answered without troubling the editor again.
func TestACrashedServerIsRestartedWithTheSessionReplayed(t *testing.T) {
	ours, theirs := jsonrpc2.NewChannelStreamPair(64)
	e := &editor{t: t, stream: ours, responses: map[jsonrpc2.ID]*jsonrpc2.Response{}}

	go e.read()

	var (
		mu    sync.Mutex
		fakes []*fakeServer
	)

	start := func() (Child, error) {
		mu.Lock()
		defer mu.Unlock()

		if len(fakes) == 2 {
			return nil, errors.New("a third server was started for one crash")
		}

		c, f := startFake()
		fakes = append(fakes, f)

		return c, nil
	}

	done := make(chan int, 1)

	go func() {
		done <- Run(context.Background(), Options{Start: start, Editor: theirs})
	}()

	e.call(1, protocol.MethodInitialize, `{"processId":1,"capabilities":{}}`)
	e.await(1)
	e.notify(protocol.MethodInitialized, `{}`)
	e.notify(protocol.MethodTextDocumentDidOpen, `{"textDocument":{"uri":"file:///a.cfm","languageId":"cfml","version":1,"text":"<cfset a = 1>"}}`)
	e.notify(protocol.MethodTextDocumentDidChange, `{"textDocument":{"uri":"file:///a.cfm","version":2},"contentChanges":[{"range":{"start":{"line":0,"character":11},"end":{"line":0,"character":12}},"text":"2"}]}`)
	e.notify(protocol.MethodTextDocumentDidOpen, `{"textDocument":{"uri":"file:///b.cfm","languageId":"cfml","version":1,"text":"b"}}`)
	e.notify(protocol.MethodTextDocumentDidClose, `{"textDocument":{"uri":"file:///b.cfm"}}`)
	e.call(2, "test/slow", `{}`)
	e.call(3, "test/crash", `{}`)

	for _, id := range []int64{2, 3} {
		var rpcErr *jsonrpc2.Error
		if r := e.await(id); !errors.As(r.Err(), &rpcErr) || rpcErr.Code != contentModified {
			t.Fatalf("request %d: got %v, want ContentModified", id, r.Err())
		}
	}

	e.call(4, "test/state", `{}`)

	var state struct {
		Inits       int               `json:"inits"`
		Initialized bool              `json:"initialized"`
		Docs        map[string]string `json:"docs"`
		Versions    map[string]int32  `json:"versions"`
	}

	if err := json.Unmarshal(e.await(4).Result(), &state); err != nil {
		t.Fatal(err)
	}

	if state.Inits != 1 || !state.Initialized {
		t.Fatalf("the new server saw %d initialize and initialized=%v, want 1 and true", state.Inits, state.Initialized)
	}

	if len(state.Docs) != 1 || state.Docs["file:///a.cfm"] != "<cfset a = 2>" || state.Versions["file:///a.cfm"] != 2 {
		t.Fatalf("the new server holds %v at versions %v, want a.cfm at version 2 as edited and b.cfm closed", state.Docs, state.Versions)
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

	mu.Lock()
	servers := len(fakes)
	mu.Unlock()

	if servers != 2 {
		t.Fatalf("%d servers started, want 2", servers)
	}

	e.call(5, protocol.MethodShutdown, `null`)
	e.await(5)
	e.notify(protocol.MethodExit, `null`)

	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("exit code %d after a clean shutdown, want 0", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the supervisor did not end after shutdown and exit")
	}
}

// A server that dies at once every time is not restarted forever: past the
// limit the supervisor gives up with the server's exit code.
func TestAServerThatKeepsCrashingIsGivenUpOn(t *testing.T) {
	ours, theirs := jsonrpc2.NewChannelStreamPair(64)
	e := &editor{t: t, stream: ours, responses: map[jsonrpc2.ID]*jsonrpc2.Response{}}

	go e.read()

	starts := 0
	start := func() (Child, error) {
		starts++
		if starts > 4 {
			return nil, errors.New("started again after giving up")
		}

		c, _ := startFake()
		_ = c.Stream().Close() // dies at once

		return c, nil
	}

	done := make(chan int, 1)

	go func() {
		done <- Run(context.Background(), Options{Start: start, Editor: theirs, MaxRestarts: 3})
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
