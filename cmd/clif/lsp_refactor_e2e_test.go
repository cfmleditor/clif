package main

import (
	"bufio"
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/cfmleditor/clif/internal/daemon"
	"go.lsp.dev/uri"
)

// The refactorings, rename and the code actions that edit, are tested here
// through the binary as an editor runs it: a child process speaking JSON-RPC
// on stdin and stdout, configured by initializationOptions, sent the edits an
// editor sends. The unit tests call the handlers directly, so they cannot see
// what only the whole path shows: a capability not advertised, a feature the
// options do not reach, positions in UTF-16 lost in framing, a rename planned
// from a parse of text the editor has since changed, or a version on an edit
// that is not the version the client holds.
//
// The child is this test binary, run again with lspE2EEnv set, which makes it
// start the server the way main does with no arguments.

const lspE2EEnv = "CLIF_LSP_E2E_SERVER"

func TestMain(m *testing.M) {
	if os.Getenv(lspE2EEnv) == "1" {
		os.Args = []string{"clif"}

		main()
		os.Exit(0)
	}

	os.Exit(m.Run())
}

// lspClient is a minimal LSP client over a child's stdio: it frames requests,
// answers the server's own requests with null, and drops its notifications.
type lspClient struct {
	t      *testing.T
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	nextID int

	mu      sync.Mutex
	pending map[int]chan rpcResponse
	done    chan struct{}
	stderr  bytes.Buffer
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	Result jsontext.Value `json:"result"`
	Error  *rpcError      `json:"error"`
}

type rpcMessage struct {
	ID     jsontext.Value `json:"id"`
	Method string         `json:"method"`
	rpcResponse
}

// startLSP runs the server in dir, which must have no .clif.json above it, so
// the session is standalone and configured by the options alone.
func startLSP(t *testing.T, dir string) *lspClient {
	t.Helper()

	if cfg, _ := daemon.FindConfig(dir); cfg != nil {
		t.Skipf("a .clif.json above %s (%s) would put the server in daemon mode", dir, cfg.Path)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.CommandContext(t.Context(), exe, "-test.run=^$")
	cmd.Dir = dir
	// The crash log and the file log go to the test's own directory, never the
	// user's cache.
	cmd.Env = append(os.Environ(), lspE2EEnv+"=1", "CLIF_LOG="+filepath.Join(t.TempDir(), "clif.log"))

	c := &lspClient{t: t, cmd: cmd, pending: map[int]chan rpcResponse{}, done: make(chan struct{})}
	cmd.Stderr = &c.stderr

	c.stdin, err = cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	go c.read(bufio.NewReader(stdout))

	t.Cleanup(func() {
		_ = c.stdin.Close()

		select {
		case <-c.done:
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
		}

		_ = cmd.Wait()
	})

	return c
}

func (c *lspClient) read(r *bufio.Reader) {
	defer close(c.done)

	for {
		length := -1

		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}

			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}

			if v, ok := strings.CutPrefix(line, "Content-Length: "); ok {
				length, _ = strconv.Atoi(v)
			}
		}

		body := make([]byte, length)
		if _, err := io.ReadFull(r, body); err != nil {
			return
		}

		var msg rpcMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			continue
		}

		switch {
		case msg.Method != "" && len(msg.ID) > 0:
			// A request of the server's (client/registerCapability,
			// workspace/configuration): answered, so it does not wait on us.
			c.write(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": nil})
		case msg.Method == "":
			id, err := strconv.Atoi(string(msg.ID))
			if err != nil {
				continue
			}

			c.mu.Lock()
			ch := c.pending[id]
			delete(c.pending, id)
			c.mu.Unlock()

			if ch != nil {
				ch <- msg.rpcResponse
			}
		}
	}
}

func (c *lspClient) write(msg any) {
	body, err := json.Marshal(msg)
	if err != nil {
		c.t.Fatal(err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := fmt.Fprintf(c.stdin, "Content-Length: %d\r\n\r\n%s", len(body), body); err != nil {
		c.t.Fatalf("writing to the server: %v\nstderr:\n%s", err, c.stderr.String())
	}
}

func (c *lspClient) notify(method string, params any) {
	c.write(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

// call sends a request and waits for its answer.
func (c *lspClient) call(method string, params any) rpcResponse {
	c.t.Helper()

	c.nextID++
	id := c.nextID
	ch := make(chan rpcResponse, 1)

	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	c.write(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})

	select {
	case res := <-ch:
		return res
	case <-c.done:
		c.t.Fatalf("the server exited during %s\nstderr:\n%s", method, c.stderr.String())
	case <-time.After(30 * time.Second):
		c.t.Fatalf("no answer to %s\nstderr:\n%s", method, c.stderr.String())
	}

	return rpcResponse{}
}

// result is call's result decoded into out, failing on an error answer.
func (c *lspClient) result(method string, params, out any) {
	c.t.Helper()

	res := c.call(method, params)
	if res.Error != nil {
		c.t.Fatalf("%s: %s", method, res.Error.Message)
	}

	if err := json.Unmarshal(res.Result, out); err != nil {
		c.t.Fatalf("%s answered %s: %v", method, res.Result, err)
	}
}

// initialize opens the session, rooted at dir, with options as the editor's
// initializationOptions, and says whether the client takes versioned edits.
func (c *lspClient) initialize(dir string, versioned bool, options any) jsontext.Value {
	c.t.Helper()

	var res struct {
		Capabilities jsontext.Value `json:"capabilities"`
	}

	c.result("initialize", map[string]any{
		"processId": os.Getpid(),
		"rootUri":   uri.File(dir),
		"capabilities": map[string]any{
			"workspace": map[string]any{"workspaceEdit": map[string]any{"documentChanges": versioned}},
		},
		"initializationOptions": options,
	}, &res)

	c.notify("initialized", map[string]any{})

	return res.Capabilities
}

func (c *lspClient) shutdown() {
	c.t.Helper()

	if res := c.call("shutdown", nil); res.Error != nil {
		c.t.Fatalf("shutdown: %s", res.Error.Message)
	}

	c.notify("exit", nil)
}

type lspPos struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start lspPos `json:"start"`
	End   lspPos `json:"end"`
}

type textEdit struct {
	Range   lspRange `json:"range"`
	NewText string   `json:"newText"`
}

type workspaceEdit struct {
	Changes         map[string][]textEdit `json:"changes"`
	DocumentChanges []struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version *int   `json:"version"`
		} `json:"textDocument"`
		Edits []textEdit `json:"edits"`
	} `json:"documentChanges"`
}

// editsFor is the edits to docURI in e and the version they were made from (-1
// for the plain map, which carries none).
func (e workspaceEdit) editsFor(t *testing.T, docURI uri.URI) ([]textEdit, int) {
	t.Helper()

	if len(e.Changes) > 0 && len(e.DocumentChanges) > 0 {
		t.Fatal("the edit has both changes and documentChanges")
	}

	if len(e.Changes) > 0 {
		if len(e.Changes) != 1 {
			t.Fatalf("edits to %d documents, want 1", len(e.Changes))
		}

		edits, ok := e.Changes[string(docURI)]
		if !ok {
			t.Fatalf("no edits to %s in %v", docURI, e.Changes)
		}

		return edits, -1
	}

	if len(e.DocumentChanges) != 1 || e.DocumentChanges[0].TextDocument.URI != string(docURI) {
		t.Fatalf("document changes %+v, want one to %s", e.DocumentChanges, docURI)
	}

	dc := e.DocumentChanges[0]
	if dc.TextDocument.Version == nil {
		t.Fatal("a versioned edit with no version")
	}

	return dc.Edits, *dc.TextDocument.Version
}

// offsetOf is the byte offset of p, whose character is in UTF-16 code units,
// in text, as an editor counts it.
func offsetOf(t *testing.T, text string, p lspPos) int {
	t.Helper()

	off := 0

	for range p.Line {
		nl := strings.IndexByte(text[off:], '\n')
		if nl < 0 {
			t.Fatalf("line %d is past the end of the text", p.Line)
		}

		off += nl + 1
	}

	for units := 0; units < p.Character; {
		if off >= len(text) || text[off] == '\n' || text[off] == '\r' {
			t.Fatalf("character %d is past the end of line %d", p.Character, p.Line)
		}

		r, size := utf8.DecodeRuneInString(text[off:])
		units += len(utf16.Encode([]rune{r}))
		off += size

		if units > p.Character {
			t.Fatalf("character %d of line %d is inside a surrogate pair", p.Character, p.Line)
		}
	}

	return off
}

// posOf is the LSP position of the first needle in text, at its ¦ if it has one.
func posOf(t *testing.T, text, needle string) lspPos {
	t.Helper()

	shift := 0
	if i := strings.Index(needle, "¦"); i >= 0 {
		shift = i
		needle = needle[:i] + needle[i+len("¦"):]
	}

	at := strings.Index(text, needle)
	if at < 0 {
		t.Fatalf("%q is not in the text", needle)
	}

	at += shift
	line := strings.Count(text[:at], "\n")
	start := strings.LastIndexByte(text[:at], '\n') + 1

	return lspPos{Line: line, Character: len(utf16.Encode([]rune(text[start:at])))}
}

// apply applies edits to text as an editor does: every range read against the
// text before any edit, and overlapping ranges refused.
func apply(t *testing.T, text string, edits []textEdit) string {
	t.Helper()

	type span struct {
		start, end int
		text       string
	}

	spans := make([]span, len(edits))
	for i, e := range edits {
		spans[i] = span{offsetOf(t, text, e.Range.Start), offsetOf(t, text, e.Range.End), e.NewText}
	}

	for i := range spans {
		for j := range spans {
			if i != j && spans[i].start < spans[j].end && spans[j].start < spans[i].end {
				t.Fatalf("edits %d and %d overlap", i, j)
			}
		}
	}

	// From the end, so an edit does not move the ranges before it.
	slices.SortFunc(spans, func(a, b span) int { return b.start - a.start })

	out := text
	for _, s := range spans {
		out = out[:s.start] + s.text + out[s.end:]
	}

	return out
}

func textDoc(u uri.URI) map[string]any { return map[string]any{"uri": u} }

func versionedDoc(u uri.URI, version int) map[string]any {
	return map[string]any{"uri": u, "version": version}
}

// TestRenameOverStdio drives a rename the way an editor does: prepare, rename,
// apply the edit, send it back as a change, and rename again from the edited
// text.
func TestRenameOverStdio(t *testing.T) {
	dir := t.TempDir()

	// The emoji before #count# is two UTF-16 units and four bytes: a column
	// counted in either of the other units lands inside the name.
	src := "component {\n" +
		"\tfunction total( items ) {\n" +
		"\t\tvar count = 0;\n" +
		"\t\titems.each( function( i ) {\n" +
		"\t\t\tcount += i;\n" +
		"\t\t} );\n" +
		"\t\treturn \"😀 #count# 😀\" & count;\n" +
		"\t}\n" +
		"}\n"

	path := filepath.Join(dir, "Counter.cfc")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	doc := uri.File(path)

	c := startLSP(t, dir)

	caps := c.initialize(dir, true, map[string]any{"features": map[string]any{"rename": true}})

	var advertised struct {
		RenameProvider struct {
			PrepareProvider bool `json:"prepareProvider"`
		} `json:"renameProvider"`
		CodeActionProvider jsontext.Value `json:"codeActionProvider"`
	}
	if err := json.Unmarshal(caps, &advertised); err != nil {
		t.Fatal(err)
	}

	if !advertised.RenameProvider.PrepareProvider {
		t.Fatalf("rename with prepare is not advertised with features.rename on: %s", caps)
	}

	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": doc, "languageId": "cfml", "version": 1, "text": src,
	}})

	// An edit before the rename, so a rename planned from the parse of the
	// opened text would land a line early.
	header := "// Counts the items.\n"
	c.notify("textDocument/didChange", map[string]any{
		"textDocument": versionedDoc(doc, 2),
		"contentChanges": []map[string]any{{
			"range": lspRange{}, "text": header,
		}},
	})

	text := header + src

	var prep struct {
		Range       lspRange `json:"range"`
		Placeholder string   `json:"placeholder"`
	}

	c.result("textDocument/prepareRename", map[string]any{
		"textDocument": textDoc(doc), "position": posOf(t, text, "var co¦unt"),
	}, &prep)

	if want := posOf(t, text, "var ¦count"); prep.Placeholder != "count" || prep.Range.Start != want {
		t.Fatalf("prepareRename answered %+v, want count at %+v", prep, want)
	}

	// From the use inside the closure.
	var edit workspaceEdit

	c.result("textDocument/rename", map[string]any{
		"textDocument": textDoc(doc), "position": posOf(t, text, "\t\t\t¦count += i"), "newName": "sum",
	}, &edit)

	edits, version := edit.editsFor(t, doc)
	if version != 2 {
		t.Errorf("the edit is of version %d, want 2, the version the client holds", version)
	}

	renamed := apply(t, text, edits)

	want := header + strings.ReplaceAll(src, "count", "sum")
	if renamed != want {
		t.Fatalf("after the rename:\n%s\nwant:\n%s", renamed, want)
	}

	// The editor applies it and tells the server, in one full-text change.
	c.notify("textDocument/didChange", map[string]any{
		"textDocument":   versionedDoc(doc, 3),
		"contentChanges": []map[string]any{{"text": renamed}},
	})

	// The new name is renamed from the edited text, from inside the string.
	edit = workspaceEdit{}

	c.result("textDocument/rename", map[string]any{
		"textDocument": textDoc(doc), "position": posOf(t, renamed, "😀 #s¦um#"), "newName": "count",
	}, &edit)

	edits, version = edit.editsFor(t, doc)
	if version != 3 {
		t.Errorf("the second edit is of version %d, want 3", version)
	}

	if got := apply(t, renamed, edits); got != text {
		t.Errorf("renaming back:\n%s\nwant:\n%s", got, text)
	}

	// What cannot be renamed is refused, as an error the editor shows...
	for _, c2 := range []struct{ at, name, why string }{
		{"\t\t\t¦sum += i", "items", "argument"},
		{"\t\t\t¦sum += i", "return", "reserved"},
		{"\t\t\t¦sum += i", "two words", "not a valid"},
		{"( ¦items )", "list", "argument"},
	} {
		res := c.call("textDocument/rename", map[string]any{
			"textDocument": textDoc(doc), "position": posOf(t, renamed, c2.at), "newName": c2.name,
		})
		if res.Error == nil || !strings.Contains(res.Error.Message, c2.why) {
			t.Errorf("renaming %q to %q: %+v, want an error mentioning %q", c2.at, c2.name, res, c2.why)
		}
	}

	// ...and a place with nothing to rename is null, which an editor takes as
	// "not here" rather than as an error.
	res := c.call("textDocument/prepareRename", map[string]any{
		"textDocument": textDoc(doc), "position": posOf(t, renamed, "comp¦onent"),
	})
	if res.Error != nil || string(res.Result) != "null" {
		t.Errorf("prepareRename on the component keyword: %+v, want null", res)
	}

	c.shutdown()
}

// TestRenameOverStdioForAPlainClient: a client that does not take versioned
// edits gets the plain changes map, and an editor with the feature off is
// neither offered rename nor answered with an edit.
func TestRenameOverStdioForAPlainClient(t *testing.T) {
	src := "component {\n\tfunction f() {\n\t\tvar x = 1;\n\t\treturn x;\n\t}\n}\n"

	for _, on := range []bool{true, false} {
		t.Run(fmt.Sprintf("rename %v", on), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "F.cfc")

			if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}

			doc := uri.File(path)
			c := startLSP(t, dir)

			caps := c.initialize(dir, false, map[string]any{"features": map[string]any{"rename": on}})

			var advertised struct {
				RenameProvider jsontext.Value `json:"renameProvider"`
			}
			if err := json.Unmarshal(caps, &advertised); err != nil {
				t.Fatal(err)
			}

			if offered := len(advertised.RenameProvider) > 0 && string(advertised.RenameProvider) != "false"; offered != on {
				t.Errorf("renameProvider %s with the feature %v", advertised.RenameProvider, on)
			}

			c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{
				"uri": doc, "languageId": "cfml", "version": 1, "text": src,
			}})

			res := c.call("textDocument/rename", map[string]any{
				"textDocument": textDoc(doc), "position": posOf(t, src, "return ¦x"), "newName": "y",
			})

			if !on {
				if res.Error == nil || !strings.Contains(res.Error.Message, "features.rename") {
					t.Errorf("rename with the feature off: %+v, want an error naming features.rename", res)
				}

				c.shutdown()

				return
			}

			if res.Error != nil {
				t.Fatal(res.Error.Message)
			}

			var edit workspaceEdit
			if err := json.Unmarshal(res.Result, &edit); err != nil {
				t.Fatal(err)
			}

			edits, version := edit.editsFor(t, doc)
			if version != -1 {
				t.Errorf("a client that takes no versioned edits got one of version %d", version)
			}

			if got, want := apply(t, src, edits), strings.ReplaceAll(src, "x", "y"); got != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}

			c.shutdown()
		})
	}
}

type codeAction struct {
	Title string         `json:"title"`
	Kind  string         `json:"kind"`
	Edit  *workspaceEdit `json:"edit"`
}

func codeActions(c *lspClient, doc uri.URI, at lspPos, only ...string) []codeAction {
	c.t.Helper()

	ctx := map[string]any{"diagnostics": []any{}}
	if len(only) > 0 {
		ctx["only"] = only
	}

	var actions []codeAction

	c.result("textDocument/codeAction", map[string]any{
		"textDocument": textDoc(doc), "range": lspRange{Start: at, End: at}, "context": ctx,
	}, &actions)

	return actions
}

func titled(actions []codeAction, title string) (codeAction, bool) {
	for _, a := range actions {
		if a.Title == title {
			return a, true
		}
	}

	return codeAction{}, false
}

// TestRefactorCodeActionsOverStdio asks for the code actions of an
// Application.cfc as an editor's light bulb does, applies one, sends the
// change, and asks again.
func TestRefactorCodeActionsOverStdio(t *testing.T) {
	dir := t.TempDir()

	// CRLF, as many a Windows checkout has: the stub must follow it, and its
	// position must count the \r as part of the line.
	src := strings.ReplaceAll("component {\n\tthis.name = \"app\";\n\n\tpublic boolean function onApplicationStart() {\n\t\treturn true;\n\t}\n}\n", "\n", "\r\n")

	path := filepath.Join(dir, "Application.cfc")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	doc := uri.File(path)
	c := startLSP(t, dir)

	c.initialize(dir, true, nil)

	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": doc, "languageId": "cfml", "version": 1, "text": src,
	}})

	decl := posOf(t, src, "comp¦onent")

	// A client asking for quick fixes alone is not handed the refactorings.
	for _, a := range codeActions(c, doc, decl, "quickfix") {
		if a.Kind == "refactor" || a.Edit != nil {
			t.Errorf("asked for quick fixes, got %q (%s)", a.Title, a.Kind)
		}
	}

	// Nor is one asking for a narrower refactoring than these are.
	for _, a := range codeActions(c, doc, decl, "refactor.extract") {
		if a.Kind == "refactor" {
			t.Errorf("asked for refactor.extract, got %q", a.Title)
		}
	}

	actions := codeActions(c, doc, decl, "refactor")

	if _, ok := titled(actions, "Add onApplicationStart() to Application.cfc"); ok {
		t.Error("onApplicationStart is offered though the component declares it")
	}

	add, ok := titled(actions, "Add onSessionStart() to Application.cfc")
	if !ok || add.Kind != "refactor" || add.Edit == nil {
		t.Fatalf("onSessionStart is not offered as a refactor with an edit: %+v", actions)
	}

	edits, version := add.Edit.editsFor(t, doc)
	if version != 1 {
		t.Errorf("the edit is of version %d, want 1", version)
	}

	added := apply(t, src, edits)

	if !strings.Contains(added, "\r\n\tpublic void function onSessionStart() {\r\n\t\treturn;\r\n\t}\r\n}") {
		t.Fatalf("the stub is not added with the file's line breaks before the closing brace:\n%q", added)
	}

	if strings.Count(added, "\n") != strings.Count(added, "\r\n") {
		t.Errorf("the stub brought a bare line feed into a CRLF file:\n%q", added)
	}

	c.notify("textDocument/didChange", map[string]any{
		"textDocument":   versionedDoc(doc, 2),
		"contentChanges": []map[string]any{{"text": added}},
	})

	// The edited text is what it answers for: the event is declared now, and
	// the rest are still offered, at the version the client holds.
	actions = codeActions(c, doc, posOf(t, added, "comp¦onent"), "refactor")

	if _, ok := titled(actions, "Add onSessionStart() to Application.cfc"); ok {
		t.Error("onSessionStart is offered again after it was added")
	}

	end, ok := titled(actions, "Add onSessionEnd() to Application.cfc")
	if !ok {
		t.Fatalf("onSessionEnd is not offered after the edit: %+v", actions)
	}

	if _, version := end.Edit.editsFor(t, doc); version != 2 {
		t.Errorf("the edit after the change is of version %d, want 2", version)
	}

	// The non-editing actions still come with no filter.
	if _, ok := titled(codeActions(c, doc, decl), "Scan workspace for parse errors"); !ok {
		t.Error("an unfiltered request lost the workspace actions")
	}

	c.shutdown()
}

// The server's process ends once the client closes the connection after
// exit, rather than lingering as an orphan.
func TestServerOverStdioEndsAfterExit(t *testing.T) {
	dir := t.TempDir()
	c := startLSP(t, dir)

	c.initialize(dir, false, nil)
	c.shutdown()

	_ = c.stdin.Close()

	select {
	case <-c.done:
	case <-time.After(10 * time.Second):
		t.Fatal("the server did not end after exit and a closed stdin")
	}

	if err := c.cmd.Wait(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 0 {
			t.Errorf("the server ended with %v\nstderr:\n%s", err, c.stderr.String())
		}
	}
}
