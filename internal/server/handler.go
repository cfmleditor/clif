package server

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfmleditor/clif/internal/lex"

	"github.com/cfmleditor/clif/internal/cache"
	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/lspdoc"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/resolve"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// docOf digs the document URI out of a request's parameters for logging, and
// returns "" for a request that names no document. Best-effort by design: this
// runs only on a request already known to be slow, and a log line is not worth
// failing over.
func docOf(params []byte) string {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return ""
	}

	return p.TextDocument.URI
}

// Handler returns a jsonrpc2.Handler that dispatches LSP method calls.
func (s *Server) Handler() jsonrpc2.Handler {
	return func(ctx context.Context, req *jsonrpc2.Request) (result any, err error) {
		start := time.Now()

		noteActivity()

		defer func() {
			if r := recover(); r != nil {
				cflog.Recovered(s.log, "handler panic", r, cflog.String("method", req.Method()))
				result, err = nil, fmt.Errorf("internal error: %v", r)
			}

			if dur := time.Since(start); dur > 100*time.Millisecond {
				// The document, not just the method. "textDocument/definition
				// took 3s" repeated every few seconds is indistinguishable from
				// one stuck request and from many slow ones, and names nothing
				// to open and try.
				s.log.Warn("slow request",
					cflog.String("method", req.Method()),
					cflog.Duration("dur", dur),
					cflog.String("doc", docOf(req.Params())))
			}
		}()

		switch req.Method() {
		case protocol.MethodInitialize:
			return s.handleInitialize(ctx, req.Params())
		case protocol.MethodInitialized:
			return s.handleInitialized(ctx)
		case protocol.MethodShutdown:
			// Stop forwarding before the connection goes: a notification written
			// to a closing connection is at best ignored and at worst an error
			// logged about logging.
			s.detachLogSink()

			return nil, nil
		case protocol.MethodExit:
			s.detachLogSink()

			return nil, nil
		case protocol.MethodTextDocumentDidOpen:
			return s.handleDidOpen(ctx, req.Params())
		case protocol.MethodTextDocumentDidChange:
			return s.handleDidChange(ctx, req.Params())
		case protocol.MethodTextDocumentDidClose:
			return s.handleDidClose(ctx, req.Params())
		case protocol.MethodTextDocumentDidSave:
			return s.handleDidSave(ctx, req.Params())
		case protocol.MethodTextDocumentCompletion:
			return s.handleCompletion(ctx, req.Params())
		case protocol.MethodCompletionItemResolve:
			return s.handleCompletionResolve(ctx, req.Params())
		case protocol.MethodTextDocumentDefinition:
			return s.handleDefinition(ctx, req.Params())
		case protocol.MethodTextDocumentTypeDefinition:
			return s.handleTypeDefinition(ctx, req.Params())
		case protocol.MethodTextDocumentReferences:
			return s.handleReferences(ctx, req.Params())
		case protocol.MethodTextDocumentFormatting:
			return s.handleFormatting(ctx, req.Params())
		case protocol.MethodTextDocumentRangeFormatting:
			return s.handleRangeFormatting(ctx, req.Params())
		case protocol.MethodTextDocumentOnTypeFormatting:
			return s.handleOnTypeFormatting(ctx, req.Params())
		case protocol.MethodTextDocumentDocumentHighlight:
			return s.handleDocumentHighlight(ctx, req.Params())
		case protocol.MethodTextDocumentFoldingRange:
			return s.handleFoldingRange(ctx, req.Params())
		case protocol.MethodTextDocumentDocumentSymbol:
			return s.handleDocumentSymbol(ctx, req.Params())
		case protocol.MethodWorkspaceSymbol:
			return s.handleWorkspaceSymbol(ctx, req.Params())
		case protocol.MethodTextDocumentHover:
			return s.handleHover(ctx, req.Params())
		case protocol.MethodTextDocumentSignatureHelp:
			return s.handleSignatureHelp(ctx, req.Params())
		case protocol.MethodTextDocumentDocumentLink:
			return s.handleDocumentLink(ctx, req.Params())
		case protocol.MethodDocumentLinkResolve:
			return s.handleDocumentLinkResolve(ctx, req.Params())
		case protocol.MethodTextDocumentCodeAction:
			return s.handleCodeAction(ctx, req.Params())
		case protocol.MethodWorkspaceDidChangeWorkspaceFolders:
			return s.handleDidChangeWorkspaceFolders(ctx, req.Params())
		case protocol.MethodWorkspaceDidChangeWatchedFiles:
			return s.handleDidChangeWatchedFiles(ctx, req.Params())
		case protocol.MethodWorkspaceExecuteCommand:
			return s.handleExecuteCommand(ctx, req.Params())
		default:
			return jsonrpc2.MethodNotFoundHandler(ctx, req)
		}
	}
}

func (s *Server) handleInitialize(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.InitializeParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	s.initialized = true
	s.watchedFilesDynamic = clientWatchesFiles(params.Capabilities)
	s.completionDefer = clientDefers(params.Capabilities)

	folders, _ := params.WorkspaceFolders.Get()

	s.log.Debug("initialize params workspace folders", cflog.Int("count", len(folders)))

	for i, folder := range folders {
		s.log.Debug("workspace folder", cflog.Int("index", i), cflog.String("name", folder.Name), cflog.String("uri", string(folder.URI)))
	}

	// cfpath.FromURI rather than URI.Path(), for the same reason every other
	// inbound conversion goes through it (0.2.8, #45). Path() returns the
	// *empty string* for the non-canonical `file://C:\Users\q\proj` several
	// clients send — which would leave the roots as [""] and send every search
	// that falls back to them walking nothing — and for the canonical
	// `file:///c%3A/...` it returns `/c:/Users/q/proj`, a leading slash that is
	// not a Windows path. These roots are what searchRoots falls back to, so a
	// bad one here is a workspace-wide search covering the wrong place.
	for _, folder := range folders {
		if root, ok := s.usableWorkspaceRoot(string(folder.URI)); ok {
			s.addWorkspaceRoot(root)
		}
	}

	if len(s.editorRoots()) == 0 && params.RootURI != nil && *params.RootURI != "" { //nolint:staticcheck // rootUri is deprecated, but a client without workspaceFolders still sends only it
		if root, ok := s.usableWorkspaceRoot(string(*params.RootURI)); ok { //nolint:staticcheck // as above
			s.addWorkspaceRoot(root)
		}
	}

	// Apply this session's configuration. See configureSession for which file
	// governs and why the editor's initializationOptions are always merged.
	//
	// The condition here used to be "has this session no component resolvers
	// yet", standing in for "has it been configured". That made everything
	// else depend on an unrelated config key: adding one resolver to a working
	// .clif.json silently changed what a session did at startup, which is
	// how the completions defaults came to be dropped for some workspaces and
	// not others.
	//
	// This MUST happen before the goroutines below are spawned: applyConfig is
	// what sets s.Linting, and initLinter returns early when it reads that
	// field as false, which would leave s.linter nil and silently disable
	// diagnostics for the rest of the session. Spawning after the writes also
	// gives the goroutines a happens-before edge to them, so no locking is
	// needed for config that is only written here.
	s.configureSession(s.editorConfig(params.InitializationOptions))

	// From here the client can receive window/logMessage, so the server's own
	// logs stop being indistinguishable from failures in its Output panel.
	// Attached after configureSession and before anything is logged about the
	// session, so the first line a reader sees is the one below.
	s.attachLogSink()

	s.safeGo("indexWorkspace", s.indexWorkspace)
	s.safeGo("initLinter", s.initLinter)

	// The switches as resolved, not as written: a feature that looks off in the
	// config and on in the server is otherwise invisible until someone times a
	// request and disbelieves the answer.
	s.log.Info("features resolved",
		cflog.Bool("routes", s.Features.Routes),
		cflog.Bool("variableDefinitions", s.Features.VariableDefinitions),
		cflog.Bool("documentHighlight", s.Features.DocumentHighlight),
		cflog.Bool("folding", s.Features.Folding),
		cflog.Bool("typeDefinition", s.Features.TypeDefinition),
		cflog.Bool("watchedFiles", s.Features.WatchedFiles),
		cflog.Bool("rangeFormatting", s.Features.RangeFormatting),
		cflog.Bool("routesConfigured", s.Routes.Enabled()))

	s.log.Info("CFML LSP initialized", cflog.Strings("workspaceRoots", s.editorRoots()))

	return protocol.InitializeResult{
		Capabilities: s.capabilities(),
		ServerInfo: protocol.ServerInfo{
			Name:    "clif",
			Version: optStr(s.Version),
		},
	}, nil
}

// clientWatchesFiles reports whether the client will accept a
// workspace/didChangeWatchedFiles registration. There is no static way to ask
// for file watching — the protocol only offers dynamic registration — so a
// client that does not advertise it cannot be watched at all.
func clientWatchesFiles(caps protocol.ClientCapabilities) bool {
	w := caps.Workspace
	if w == nil || w.DidChangeWatchedFiles == nil {
		return false
	}

	return w.DidChangeWatchedFiles.DynamicRegistration != nil && *w.DidChangeWatchedFiles.DynamicRegistration
}

func (s *Server) handleInitialized(_ context.Context) (any, error) {
	// Known issues are published whether or not files are watched: watching
	// only decides whether an edit to the file is picked up without a restart.
	if len(s.KnownIssues) > 0 {
		s.safeGo("loadKnownIssues", func() { s.loadKnownIssues(context.Background()) })
	}

	if !s.Features.WatchedFiles {
		s.log.Info("file watching disabled by config; the index will not track on-disk changes")

		return nil, nil
	}

	if !s.watchedFilesDynamic {
		// Not an error, but worth saying once: this session's index is a
		// startup snapshot, and only clif.reindex will refresh it.
		s.log.Info("client does not support file watching; index will not track on-disk changes")

		return nil, nil
	}

	// registerCapability is a request, so it waits for the client to answer.
	// This handler serves a notification on the read loop, which must not
	// block; and the request context is recycled once the handler returns, so
	// the call carries its own.
	s.safeGo("registerFileWatchers", func() { s.registerFileWatchers(context.Background()) })

	return nil, nil
}

func (s *Server) handleDidOpen(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.DidOpenTextDocumentParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI

	if !cfpath.IsCFMLFile(string(docURI)) {
		return nil, nil
	}

	defer s.lockDoc(docURI)()

	// Lines end at a bare \r too, as the editor counts them; a document that
	// ends its lines that way alone gets them back in formatting edits.
	text := lex.NormalizeLineBreaks(params.TextDocument.Text)
	s.setCROnly(docURI, lex.CROnly(params.TextDocument.Text))

	s.setDocument(docURI, text)
	s.diagnosticsOpened(context.Background(), docURI)

	pr := s.parseContent(docURI, text)
	s.log.Debug("document opened: parse result",
		cflog.String("uri", string(docURI)),
		cflog.Int("funcs", len(pr.Funcs)),
		cflog.Int("refs", len(pr.ComponentRefs)),
		cflog.Int("resolvers", len(pr.Resolvers)))

	for i := range pr.ComponentRefs {
		ref := &pr.ComponentRefs[i]

		s.log.Debug("document opened: ref", cflog.String("var", ref.Variable), cflog.String("component", ref.Component))
	}

	s.mu.Lock()
	s.parseResults[docURI] = pr
	s.funcRanges[docURI] = scopesToFuncRanges(pr)
	s.mu.Unlock()

	s.reindexFromParseResult(docURI, pr)

	s.safeGo("rebuildFileCompletionCacheFromPR", func() {
		defer s.lockDoc(docURI)()

		s.rebuildFileCompletionCacheFromPR(docURI, pr)
	})
	s.log.Debug("document opened", cflog.String("uri", string(docURI)))

	return nil, nil
}

func (s *Server) handleDidChange(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.DidChangeTextDocumentParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI

	if len(params.ContentChanges) == 0 {
		return nil, nil
	}

	if !cfpath.IsCFMLFile(string(docURI)) {
		return nil, nil
	}

	// Not lockDoc: that would run the reparse deferred by the last keystroke
	// before this one, which is the cost deferring exists to avoid.
	defer s.lockDocOnly(docURI)()

	s.mu.RLock()
	pr := s.parseResults[docURI]
	s.mu.RUnlock()

	content, ok := s.getDocument(docURI)
	if !ok {
		return nil, nil
	}

	changeCount := s.noteChange(docURI, time.Now())

	// A burst of edits inside a function stays incremental: each costs a few
	// milliseconds that way, and the burst path's deferred reparse of the
	// whole file is what the next request then waited for — ~220ms on a
	// 65,000-line component, every sixth keystroke of a held-down key. The
	// burst path remains for everything else, and for a notification of
	// more than 50 changes, which ApplyEdits streams in one pass.
	rapidChanges := len(params.ContentChanges) > 50 ||
		changeCount > 5 && !s.editsInPlace(docURI, pr, params.ContentChanges)

	totalBytes := 0

	for _, c := range params.ContentChanges {
		_, text, _ := changeRangeAndText(c)
		totalBytes += len(text)
	}

	s.log.Debug("didChange",
		cflog.String("uri", string(docURI)),
		cflog.Int("changeCount", changeCount),
		cflog.Int("edits", len(params.ContentChanges)),
		cflog.Int("bytes", totalBytes),
	)

	if rapidChanges {
		// Too many changes in quick succession — just apply text and defer reindex.
		content = applyContentChanges(content, params.ContentChanges)

		s.setDocument(docURI, content)

		// Deferred reparse gets its own timer map. Sharing cacheTimers with
		// debounceCacheRebuild meant the two kinds of pending work occupied one
		// slot per document: the first non-rapid keystroke after a burst armed a
		// cache rebuild, which Stop()ped the reindex sitting in the slot and
		// dropped it. The paste that triggered the burst then never made it into
		// the index at all.
		s.scheduleReparse(docURI)

		return nil, nil
	}

	// A parse already lagging (a deferred reparse not yet run) must not be
	// edited incrementally: its scopes are out of step with the positions in
	// these changes. The text is applied and the reparse stays deferred.
	content, deferred, lastKind, editLine := applyChanges(content, pr, params.ContentChanges, pr != nil && s.reparseIsPending(docURI))

	s.setDocument(docURI, content)

	if deferred {
		s.scheduleReparse(docURI)

		return nil, nil
	}

	if pr == nil {
		// No cached parse result — fall back to full parse
		pr = s.parseContent(docURI, content)
		s.mu.Lock()
		s.parseResults[docURI] = pr
		s.funcRanges[docURI] = scopesToFuncRanges(pr)
		s.mu.Unlock()
		s.reindexFromParseResult(docURI, pr)

		return nil, nil
	}

	s.afterEdit(docURI, pr, content, lastKind, editLine, params.ContentChanges)

	return nil, nil
}

// applyChanges applies changes to content, editing the parse in place for as
// long as each one is inside a function body. An edit outside one, or a parse
// already lagging (deferred), applies the rest as text alone and reports
// deferred, for the caller to schedule the reparse. lastKind and editLine say
// what the last incremental edit did to the parse and where.
func applyChanges(content string, pr *parser.ParseResult, changes []protocol.TextDocumentContentChangeEvent, deferred bool) (out string, isDeferred bool, lastKind parser.EditKind, editLine int) {
	for i, change := range changes {
		r, text, isFull := changeRangeAndText(change)

		// An edit outside a function body would reparse the whole file's
		// signatures here, inline; defer it (see scheduleReparse). Changes
		// before this one were inside functions and are already applied.
		if !deferred && pr != nil && (isFull || !pr.EditInFunction(int(r.Start.Line), int(r.End.Line), int(r.Start.Character))) {
			deferred = true
		}

		if deferred {
			content = applyContentChanges(content, changes[i:])

			break
		}

		if isFull {
			// Full document replacement
			content = text
			if pr != nil {
				pr.ApplyFullReplace(content)

				lastKind = parser.EditFull
			}
		} else {
			content = applyEdit(content, r, text)
			editLine = int(r.Start.Line)

			if pr != nil {
				// The edited text is handed over rather than rebuilt: one copy
				// of the document per keystroke instead of two.
				lastKind = pr.ApplyEditResult(
					int(r.Start.Line), int(r.Start.Character),
					int(r.End.Line), int(r.End.Character),
					text, content,
				)
			}
		}
	}

	return content, deferred, lastKind, editLine
}

// afterEdit brings the server's views of docURI up to an incremental edit
// that has been applied to its parse.
func (s *Server) afterEdit(docURI uri.URI, pr *parser.ParseResult, content string, lastKind parser.EditKind, editLine int, changes []protocol.TextDocumentContentChangeEvent) {
	// Update funcRanges from the parse result
	switch lastKind {
	case parser.EditGlobal, parser.EditFull:
		// Signatures changed — update funcRanges and the index
		s.mu.Lock()
		s.funcRanges[docURI] = scopesToFuncRanges(pr)
		s.mu.Unlock()
		s.reindexFromParseResult(docURI, pr)
	case parser.EditInFunc:
		// Only function body changed — shift index lines and rebuild local vars
		if delta := lineDelta(changes); delta != 0 {
			s.index.ShiftLines(docURI, editLine, delta)
		}

		s.debounceCacheRebuild(docURI, content, editLine)
	}
}

// noteChange counts a change to docURI made at now in the current burst: the
// count restarts when more than the window (200ms) has passed since the burst
// began, and the result is the number of changes in it, this one included.
func (s *Server) noteChange(docURI uri.URI, now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ws, ok := s.changeWindowStart[docURI]; !ok || now.Sub(ws) > 200*time.Millisecond {
		s.changeWindowStart[docURI] = now
		s.changeCount[docURI] = 0
	}

	s.changeCount[docURI]++

	return s.changeCount[docURI]
}

// lineDelta is how many lines changes add to the document: the lines each
// writes less the lines it replaces. A whole-document replacement has no range
// to measure and counts for nothing.
func lineDelta(changes []protocol.TextDocumentContentChangeEvent) int {
	delta := 0

	for _, change := range changes {
		r, text, isFull := changeRangeAndText(change)
		if isFull {
			continue
		}

		delta += strings.Count(text, "\n") - (int(r.End.Line) - int(r.Start.Line))
	}

	return delta
}

// editsInPlace reports whether changes can be applied to pr incrementally: a
// parse that is not lagging its text, and a first change that is a partial
// edit inside a function. handleDidChange checks each later change as it
// applies it and defers from the first that is not.
func (s *Server) editsInPlace(docURI uri.URI, pr *parser.ParseResult, changes []protocol.TextDocumentContentChangeEvent) bool {
	if pr == nil || len(changes) == 0 || s.reparseIsPending(docURI) {
		return false
	}

	r, _, isFull := changeRangeAndText(changes[0])

	return !isFull && pr.EditInFunction(int(r.Start.Line), int(r.End.Line), int(r.Start.Character))
}

const cacheRebuildDelay = 150 * time.Millisecond

// debounceCacheRebuild resets the debounce timer for a file's completion cache rebuild.
func (s *Server) debounceCacheRebuild(docURI uri.URI, content string, editLine int) {
	s.mu.Lock()
	if t, ok := s.cacheTimers[docURI]; ok {
		t.Stop()
	}

	s.cacheTimers[docURI] = time.AfterFunc(cacheRebuildDelay, func() {
		defer func() {
			if r := recover(); r != nil {
				cflog.Recovered(s.log, "goroutine panic", r, cflog.String("label", "cacheRebuild"))
			}
		}()

		s.rebuildCompletionCache(docURI, content, editLine)
	})
	s.mu.Unlock()
}

// applyEdit replaces the text in the given range with newText.
func applyEdit(content string, r protocol.Range, newText string) string {
	return parser.ApplyEdit(content, int(r.Start.Line), int(r.Start.Character), int(r.End.Line), int(r.End.Character), newText)
}

// applyContentChanges is lspdoc.ApplyChanges, which the supervisor in front of
// the server shares, so the two agree on every open document's text.
func applyContentChanges(content string, changes []protocol.TextDocumentContentChangeEvent) string {
	return lspdoc.ApplyChanges(content, changes)
}

func (s *Server) handleDidClose(ctx context.Context, rawParams []byte) (any, error) {
	var params protocol.DidCloseTextDocumentParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI

	// lockDocOnly: a deferred reparse of a document being closed is wasted.
	release := s.lockDocOnly(docURI)

	s.removeDocument(docURI)

	// The completion cache goes with the document.
	//
	// It was the one per-file structure didClose left behind, and it is the
	// largest: an entry per function scope, each holding the completion items
	// for that scope. Nothing invalidated it but a watched-file change, so a
	// daemon that outlives many editor sessions accumulated one for every file
	// any of them had ever opened. Reopening the file rebuilds it; keeping it
	// for a buffer nobody has costs memory for as long as the daemon runs.
	s.compCache.Invalidate(docURI)

	s.mu.Lock()
	delete(s.parseResults, docURI)
	delete(s.funcRanges, docURI)
	delete(s.changeCount, docURI)
	delete(s.changeWindowStart, docURI)
	delete(s.reparsePending, docURI)

	// Both timers reach for this document's ParseResult, which is about to stop
	// existing. Stopping them is also the only thing that bounds these maps: a
	// long session that opens and closes many files otherwise keeps a timer and
	// a mutex per file it has ever seen.
	for _, timers := range []map[uri.URI]*time.Timer{s.cacheTimers, s.reindexTimers} {
		if t, ok := timers[docURI]; ok {
			t.Stop()
			delete(timers, docURI)
		}
	}

	s.mu.Unlock()

	release()

	// s.docLocks deliberately keeps its entry. Stop() reports false for a timer
	// whose goroutine has already started, and that goroutine is by then
	// holding or waiting on this document's mutex; dropping the entry would let
	// a later didOpen mint a fresh mutex and run alongside it, which is the
	// exact race the lock exists to prevent. One sync.Mutex per file the
	// session has opened is the cheaper side of that trade.
	s.log.Debug("document closed", cflog.String("uri", string(docURI)))

	// Clear the document's own diagnostics on close. A known issue is about the
	// file, not the buffer, and stays — unless its source is open-only.
	s.diagnosticsClosed(docURI)
	s.storeDiagnostics(docURI, sourceCFLint, nil)
	s.setDiagnostics(ctx, docURI, sourceParse, nil)

	return nil, nil
}

func (s *Server) handleDidSave(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.DidSaveTextDocumentParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI

	s.invalidateSourceFile(docURI.Path())

	filePath := docURI.Path()

	// A saved file's known issues are anchored again, since an edit above one
	// moves the line it is on.
	if s.hasKnownIssues(docURI) {
		s.safeGo("loadKnownIssues", func() { s.loadKnownIssues(context.Background()) })
	}

	if cfpath.IsCFMLFile(filePath) {
		// runDiagnostics outlives this handler, so it must not carry the
		// request-scoped ctx: jsonrpc2 pools/resets incoming request contexts
		// once the handler returns, and using the stale context afterward
		// panics ("cannot create context from nil parent").
		s.safeGo("runDiagnostics", func() { s.runDiagnostics(context.Background(), docURI) })
		s.safeGo("rebuildFileCompletionCache", func() { s.rebuildFileCompletionCache(docURI) })
	}

	return nil, nil
}

func (s *Server) runDiagnostics(ctx context.Context, docURI uri.URI) {
	// initLinter publishes s.linter from its own goroutine under s.mu, so this
	// read has to take the lock too.
	s.mu.RLock()
	linter := s.linter
	s.mu.RUnlock()

	if linter == nil || s.conn == nil {
		s.log.Debug("cflint diagnostics skipped", cflog.String("reason", "linter not available (linting disabled or cflint not found)"))

		return
	}

	// Cancel any in-flight scan for this file
	s.mu.Lock()
	if cancel, ok := s.lintCancels[docURI]; ok {
		cancel()
	}

	scanCtx, cancel := context.WithCancel(ctx)
	s.lintCancels[docURI] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.lintCancels, docURI)
		s.mu.Unlock()
		cancel()
	}()

	filePath := docURI.Path()
	s.log.Info("cflint scan starting", cflog.String("file", filePath))

	// Show progress
	s.notify(scanCtx, protocol.MethodProgress, map[string]any{
		"token": "cflint",
		"value": map[string]any{"kind": "begin", "title": "CFLint", "message": filepath.Base(filePath)},
	})

	diags, err := linter.Scan(scanCtx, filePath)

	s.notify(scanCtx, protocol.MethodProgress, map[string]any{
		"token": "cflint",
		"value": map[string]any{"kind": "end"},
	})

	if scanCtx.Err() != nil {
		s.log.Debug("cflint scan cancelled", cflog.String("file", filePath))

		return
	}

	if err != nil {
		s.log.Warn("cflint scan failed", cflog.String("file", filePath), cflog.Err(err))

		return
	}

	if diags == nil {
		diags = []protocol.Diagnostic{}
	}

	s.log.Info("cflint scan complete", cflog.String("file", filePath), cflog.Int("issues", len(diags)))

	s.diagnosticsRan(docURI, sourceCFLint)
	s.setDiagnostics(ctx, docURI, sourceCFLint, diags)
}

func (s *Server) reindexIfCFC(docURI uri.URI, content string) {
	if !cfpath.IsCFMLFile(string(docURI)) {
		return
	}

	if cfpath.IsCFCFile(string(docURI)) && len(s.WorkspaceFolders) > 0 && !s.isIncludedPath(string(docURI)) {
		return
	}

	// Indexed with the workspace's rules, as every other door indexes a
	// component; a bare parse typed nothing a resolver or a setter would.
	path := cfpath.FromURI(string(docURI))
	s.index.IndexParsed(docURI, content, s.getResolver().ParseFile(path, content, resolve.IndexSpec()))
}

// reindexFromParseResult updates the index using an existing ParseResult.
func (s *Server) reindexFromParseResult(docURI uri.URI, pr *parser.ParseResult) {
	if !cfpath.IsCFMLFile(string(docURI)) {
		return
	}

	s.index.IndexFileFromResult(docURI, pr.Funcs, pr.ComponentRefs)
	s.index.SetThisVars(docURI, pr.ThisVars())
	s.index.SetIncludes(docURI, parser.ExtractIncludes(pr.Content))
	// The re-index above forgets the file's extends and delegates, and the
	// resolver's answer to a forgotten one is to read the file from disk and
	// parse it: the saved text rather than the buffer, and on a 65,000-line
	// component twice per keystroke, once for each. The document's own full
	// parse already holds both.
	s.index.SetExtends(docURI, pr.Extends)
	s.index.SetDelegates(docURI, pr.Delegates)
	// The entity by the index's own rule. A component outside the workspace,
	// which reindexIfCFC does not index either, adds none (an open file from
	// another project would otherwise name entities for every client), but
	// one the resolver registered follows the buffer.
	if !cfpath.IsCFCFile(string(docURI)) || len(s.WorkspaceFolders) == 0 || s.isIncludedPath(string(docURI)) {
		s.index.IndexEntity(docURI, pr)
	} else {
		s.index.RefreshEntity(docURI, pr)
	}
}

// resolverRefs scans content for assignments whose RHS matches a component resolver.
// scopesToFuncRanges converts ParseResult scopes to cache.FuncRange slice.
//
// The name comes straight off the scope. It used to be recovered instead by
// scanning pr.Funcs for a definition declared on the scope's opening line,
// which is quadratic in the file's function count, and every caller runs this
// holding s.mu — the lock the whole server takes to reach any of its per-URI
// maps. On a 200-function component that was 62µs of write-locked work per
// didOpen and per signature-changing edit, against 3.6µs for the copy.
//
// The scan looked defensive because tagParser builds its own FuncScopes
// without a Name, but those never reach pr.Scopes: a tag region's scopes come
// from findTagFuncScopes and a script region's from scriptParser, and both name
// them (internal/parser/result.go). TestFuncScopesAreNamedByBothParsers pins
// that, and the two spellings were checked to agree on every function in the
// 5,624-file corpus before the scan was removed.
func scopesToFuncRanges(pr *parser.ParseResult) []cache.FuncRange {
	ranges := make([]cache.FuncRange, 0, len(pr.Scopes))

	for _, sc := range pr.Scopes {
		ranges = append(ranges, cache.FuncRange{
			Name:  sc.Name,
			Start: sc.Start,
			End:   sc.End,
		})
	}

	return ranges
}

func (s *Server) handleDidChangeWorkspaceFolders(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.DidChangeWorkspaceFoldersParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	for _, removed := range params.Event.Removed {
		root := cfpath.FromURI(string(removed.URI))
		if !s.isWorkspaceFolder(root) {
			s.index.RemoveFilesUnder(string(removed.URI))
		}

		s.removeWorkspaceRoot(root)
		s.log.Info("workspace folder removed", cflog.String("uri", string(removed.URI)))
	}

	for _, added := range params.Event.Added {
		root, ok := s.usableWorkspaceRoot(string(added.URI))
		if !ok {
			continue
		}

		s.addWorkspaceRoot(root)
		s.indexRoot(root)
		s.log.Info("workspace folder added", cflog.String("uri", string(added.URI)))
	}

	return nil, nil
}

// safeGo runs fn in a goroutine with panic recovery.
func (s *Server) safeGo(label string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				cflog.Recovered(s.log, "goroutine panic", r, cflog.String("label", label))
			}
		}()

		fn()
	}()
}

// fileToPackage converts a file path to a CFML dot-path relative to workspace.
func (s *Server) fileToPackage(filePath string) string {
	for _, root := range s.searchRoots() {
		if strings.HasPrefix(filePath, root+"/") {
			rel := filePath[len(root)+1:]
			rel = strings.TrimSuffix(rel, filepath.Ext(rel))

			return strings.ReplaceAll(rel, "/", ".")
		}
	}

	base := filepath.Base(filePath)

	return strings.TrimSuffix(base, filepath.Ext(base))
}
