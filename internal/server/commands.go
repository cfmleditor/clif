// workspace/executeCommand: the dispatch and each command it runs.

package server

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/cache"
	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/deps"
	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/refs"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// writeRefsReport writes the reference report for funcName beside the file the
// request came from, as markdown and as DOT, and tells the client where it
// went. Only clif.findRefs' explicit export argument reaches here.
func (s *Server) writeRefsReport(ctx context.Context, funcName, sourceFile string, result *refs.TraceResult) {
	outDir := filepath.Dir(sourceFile)
	if sourceFile == "" {
		// With no source file the report goes to the workspace root. It used
		// to go to the system temp directory, which reportPath now refuses.
		if roots := s.searchRoots(); len(roots) > 0 {
			outDir = roots[0]
		}
	}

	outFile, err := s.reportPath(outDir, "refs-"+funcName+".md")
	if err != nil {
		s.notifyError(ctx, err.Error())

		return
	}

	output := result.Summary + "\n\n```mermaid\n" + result.Graph.Mermaid() + "\n```"

	if err := writeReport(outFile, []byte(output)); err != nil {
		s.log.Error("failed to write file", cflog.String("path", outFile), cflog.Err(err))

		return
	}

	dotFile := strings.TrimSuffix(outFile, ".md") + ".dot"
	if err := writeReport(dotFile, []byte(result.Graph.DOT())); err != nil {
		s.log.Error("failed to write file", cflog.String("path", dotFile), cflog.Err(err))
	}

	s.notify(ctx, protocol.MethodWindowShowMessage, &protocol.ShowMessageParams{
		Type:    protocol.MessageTypeInfo,
		Message: "Wrote " + outFile,
	})
}

// legacyCommandPrefix is what every command was called while this server was
// cfmleditor-lsp. A client still sends it: the cfmleditor extension runs
// cfmleditor.generateCodeMap itself, and keybindings name the commands an older
// release advertised. Only the clif. names are advertised, so a command picker
// lists each once, and both are run.
const legacyCommandPrefix = "cfmleditor."

// canonicalCommand is the clif. name for a command sent under either name.
func canonicalCommand(name string) string {
	if rest, ok := strings.CutPrefix(name, legacyCommandPrefix); ok {
		return "clif." + rest
	}

	return name
}

func (s *Server) handleExecuteCommand(ctx context.Context, rawParams []byte) (any, error) {
	var params protocol.ExecuteCommandParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	switch canonicalCommand(params.Command) {
	case "clif.reindex":
		return s.cmdReindex(ctx, params.Arguments)
	case "clif.format":
		return s.cmdFormat(ctx, params.Arguments)
	case "clif.showComponentPath":
		return s.cmdShowComponentPath(ctx, params.Arguments)
	case "clif.restartDaemon":
		return s.cmdRestartDaemon(ctx, params.Arguments)
	case "clif.showResolvers":
		return s.cmdShowResolvers(ctx, params.Arguments)
	case "clif.showFileIndex":
		return s.cmdShowFileIndex(ctx, params.Arguments)
	case "clif.showConnections":
		return s.cmdShowConnections(ctx, params.Arguments)
	case "clif.openActiveApplicationFile":
		return s.cmdOpenActiveApplicationFile(ctx, params.Arguments)
	case "clif.goToMatchingTag":
		return s.cmdGoToMatchingTag(ctx, params.Arguments)
	case "clif.copyPackage":
		return s.cmdCopyPackage(ctx, params.Arguments)
	case "clif.findRefs":
		return s.cmdFindRefs(ctx, params.Arguments)
	case "clif.exportDeps":
		return s.cmdExportDeps(ctx, params.Arguments)
	case "clif.resolveRoute":
		return s.handleResolveRoute(params.Arguments)
	case "clif.generateCodeMap":
		return s.handleGenerateCodeMap(params.Arguments)
	case "clif.exportUnresolved":
		return s.handleExport(config.GenerateUnresolved)
	case "clif.exportCFLint":
		return s.handleExport(config.GenerateCFLint)
	case "clif.showCodeMapStats":
		return s.handleCodeMapStats(ctx, params.Arguments)
	case "clif.explainCall":
		return s.handleExplainCall(ctx, params.Arguments)
	case "clif.scanWorkspace":
		return s.cmdScanWorkspace(ctx, params.Arguments)
	default:
		return nil, fmt.Errorf("unknown command: %s", params.Command)
	}
}

// cmdReindex runs clif.reindex.
func (s *Server) cmdReindex(_ context.Context, _ []protocol.LSPAny) (any, error) {
	s.invalidateResolveCache()
	cfpath.InvalidateAppMappingsCache()
	s.safeGo("reindex", s.indexWorkspace)
	s.log.Info("reindex triggered via command")

	return nil, nil
}

// cmdFormat runs clif.format.
func (s *Server) cmdFormat(ctx context.Context, args []protocol.LSPAny) (any, error) {
	if len(args) == 0 {
		return nil, errors.New("clif.format requires a document URI argument")
	}

	docURI, _ := argString(args, 0)
	if docURI == "" {
		return nil, errors.New("clif.format: invalid URI argument")
	}

	// The same gate textDocument/formatting has. Without it this command
	// formatted with whatever s.Formatting happened to hold, and an
	// unconfigured one is the zero value — every flag false, including
	// WhitespaceOnly, which is the guard that stops the formatter writing
	// back a file whose non-whitespace content it changed.
	if !s.Formatting.Enabled {
		return nil, nil
	}

	fileURI := s.documentURI(docURI)

	content, ok := s.getDocument(fileURI)
	if !ok {
		// Said rather than ignored: a client that gets nothing back cannot
		// tell a document it named wrongly from one with nothing to format.
		return nil, fmt.Errorf("clif.format: %s is not an open document", docURI)
	}

	// A snapshot, as the formatting handler takes: the formatter reads it
	// throughout, and the server's settings can be replaced meanwhile.
	cfg := s.Formatting

	formatted, err := formatDocument(content, protocol.FormattingOptions{InsertSpaces: true, TabSize: conv.Uint32(cfg.IndentWidth)}, &cfg)
	if err != nil {
		return nil, err
	}

	if formatted == content {
		return nil, nil
	}

	lines := parser.CountNewlines(content)
	formatLabel := "Format document"
	s.call(ctx, protocol.MethodWorkspaceApplyEdit, &protocol.ApplyWorkspaceEditParams{
		Label: &formatLabel,
		Edit: protocol.WorkspaceEdit{
			Changes: map[uri.URI][]protocol.TextEdit{
				fileURI: {{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: conv.Uint32(lines + 1), Character: 0},
					},
					NewText: formatted,
				}},
			},
		},
	}, nil)

	return nil, nil
}

// cmdShowComponentPath runs clif.showComponentPath.
func (s *Server) cmdShowComponentPath(ctx context.Context, args []protocol.LSPAny) (any, error) {
	if len(args) == 0 {
		return nil, errors.New("clif.showComponentPath requires a dot-path argument")
	}

	dotPath, _ := argString(args, 0)
	if dotPath == "" {
		return nil, errors.New("clif.showComponentPath: invalid argument")
	}

	var baseDir string

	if len(args) > 1 {
		if docURI, ok := argString(args, 1); ok {
			baseDir = filepath.Dir(cfpath.FromURI(docURI))
		}
	}

	if roots := s.searchRoots(); baseDir == "" && len(roots) > 0 {
		baseDir = roots[0]
	}

	resolved := s.getResolver().ComponentPath(dotPath, baseDir)
	if resolved == "" {
		s.notify(ctx, protocol.MethodWindowShowMessage, &protocol.ShowMessageParams{
			Type:    protocol.MessageTypeInfo,
			Message: "Cannot resolve: " + dotPath,
		})
	} else {
		s.notify(ctx, protocol.MethodWindowShowMessage, &protocol.ShowMessageParams{
			Type:    protocol.MessageTypeInfo,
			Message: fmt.Sprintf("%s → %s", dotPath, resolved),
		})
	}

	return resolved, nil
}

// cmdRestartDaemon runs clif.restartDaemon.
func (s *Server) cmdRestartDaemon(ctx context.Context, _ []protocol.LSPAny) (any, error) {
	s.notify(ctx, protocol.MethodWindowShowMessage, &protocol.ShowMessageParams{
		Type:    protocol.MessageTypeInfo,
		Message: "Restarting daemon: clearing all caches and re-indexing",
	})
	s.invalidateResolveCache()
	cfpath.InvalidateAppMappingsCache()
	s.mu.Lock()
	s.parseResults = make(map[uri.URI]*parser.ParseResult)
	s.funcRanges = make(map[uri.URI][]cache.FuncRange)
	s.mu.Unlock()
	s.compCache.InvalidateAll()
	s.safeGo("reindex", s.indexWorkspace)
	s.log.Info("daemon restart triggered via command")

	return nil, nil
}

// cmdShowResolvers runs clif.showResolvers.
func (s *Server) cmdShowResolvers(ctx context.Context, _ []protocol.LSPAny) (any, error) {
	var lines []string

	lines = append(lines, fmt.Sprintf("Workspace folders: %v", s.WorkspaceFolders))
	if len(s.Mappings) > 0 {
		lines = append(lines, "Mappings:")
		for k, v := range s.Mappings {
			lines = append(lines, fmt.Sprintf("  %s → %s", k, v))
		}
	}

	if len(s.ComponentResolvers) > 0 {
		lines = append(lines, "Component resolvers:")

		for i := range s.ComponentResolvers {
			r := &s.ComponentResolvers[i]
			lines = append(lines, fmt.Sprintf("  match=%q resolve=%q prefix=%q", r.MatchText(), r.Resolve, r.Prefix))
		}
	}

	if len(s.PropertyResolvers) > 0 {
		lines = append(lines, "Property resolvers:")
		for _, r := range s.PropertyResolvers {
			lines = append(lines, fmt.Sprintf("  match=%q resolve=%q attr=%q", r.Match, r.Resolve, r.Attribute))
		}
	}

	msg := strings.Join(lines, "\n")
	s.notify(ctx, protocol.MethodWindowShowMessage, &protocol.ShowMessageParams{
		Type:    protocol.MessageTypeInfo,
		Message: msg,
	})

	return msg, nil
}

// cmdShowFileIndex runs clif.showFileIndex.
func (s *Server) cmdShowFileIndex(ctx context.Context, args []protocol.LSPAny) (any, error) {
	if len(args) == 0 {
		return nil, errors.New("clif.showFileIndex requires a document URI argument")
	}

	docURI, _ := argString(args, 0)
	if docURI == "" {
		return nil, errors.New("clif.showFileIndex: invalid argument")
	}

	fileURI := s.documentURI(docURI)
	funcs := s.index.FunctionsForFile(fileURI)
	compRefs := s.index.RefsForFile(fileURI)

	var lines []string

	lines = append(lines, "File: "+docURI, fmt.Sprintf("Functions (%d):", len(funcs)))

	for _, f := range funcs {
		lines = append(lines, fmt.Sprintf("  %s (line %d)", f.Name, f.Line))
	}

	lines = append(lines, fmt.Sprintf("Component refs (%d):", len(compRefs)))
	for _, r := range compRefs {
		lines = append(lines, fmt.Sprintf("  %s → %s (line %d)", r.Variable, r.Component, r.Line))
	}

	msg := strings.Join(lines, "\n")
	s.notify(ctx, protocol.MethodWindowShowMessage, &protocol.ShowMessageParams{
		Type:    protocol.MessageTypeInfo,
		Message: msg,
	})

	return msg, nil
}

// cmdShowConnections runs clif.showConnections.
func (s *Server) cmdShowConnections(ctx context.Context, _ []protocol.LSPAny) (any, error) {
	s.mu.RLock()
	openDocs := len(s.documents)
	s.mu.RUnlock()
	msg := fmt.Sprintf("Open documents: %d\nWorkspace folders: %d\nIndex globs: %d", openDocs, len(s.WorkspaceFolders), len(s.IndexGlobs))
	s.notify(ctx, protocol.MethodWindowShowMessage, &protocol.ShowMessageParams{
		Type:    protocol.MessageTypeInfo,
		Message: msg,
	})

	return msg, nil
}

// cmdOpenActiveApplicationFile runs clif.openActiveApplicationFile.
func (s *Server) cmdOpenActiveApplicationFile(ctx context.Context, args []protocol.LSPAny) (any, error) {
	if len(args) == 0 {
		return nil, errors.New("clif.openActiveApplicationFile requires a document URI argument")
	}

	docURI, _ := argString(args, 0)
	if docURI == "" {
		return nil, nil
	}

	baseDir := filepath.Dir(cfpath.FromURI(docURI))

	appDir := s.getResolver().FindApplicationRoot(baseDir)
	if appDir == "" {
		s.notify(ctx, protocol.MethodWindowShowMessage, &protocol.ShowMessageParams{
			Type:    protocol.MessageTypeInfo,
			Message: "No Application.cfc found",
		})

		return nil, nil
	}
	// Find the actual file
	for _, name := range []string{"Application.cfc", "Application.cfm"} {
		if _, err := s.FS.Stat(filepath.Join(appDir, name)); err == nil {
			target := string(cfpath.ToURI(filepath.Join(appDir, name)))
			s.call(ctx, "window/showDocument", map[string]any{
				"uri":       target,
				"takeFocus": true,
			}, nil)

			return target, nil
		}
	}

	return nil, nil
}

// cmdGoToMatchingTag runs clif.goToMatchingTag.
func (s *Server) cmdGoToMatchingTag(_ context.Context, args []protocol.LSPAny) (any, error) {
	if len(args) < 2 {
		return nil, errors.New("clif.goToMatchingTag requires [documentURI, line, char]")
	}

	docURI, _ := argString(args, 0)
	if docURI == "" {
		return nil, nil
	}

	content, ok := s.getDocument(s.documentURI(docURI))
	if !ok {
		return nil, fmt.Errorf("clif.goToMatchingTag: %s is not an open document", docURI)
	}

	var line, char int

	if len(args) >= 3 {
		if v, ok := argFloat(args, 1); ok {
			line = int(v)
		}

		if v, ok := argFloat(args, 2); ok {
			char = int(v)
		}
	}

	// The arguments are the editor's cursor, so char counts UTF-16 units,
	// and the answer is a position the editor moves to; FindMatchingTag
	// reads and answers in bytes. Converted on both sides, as a handler's
	// params are (see position.go).
	pos := parser.FindMatchingTag(content, line, byteCol(content, line, uint32(max(char, 0))))
	if pos == nil {
		return nil, nil
	}

	if l, ok := pos["line"].(int); ok {
		if c, ok := pos["character"].(int); ok {
			pos["character"] = lineCol(parser.LineTextAt(content, l), c)
		}
	}

	return pos, nil
}

// cmdCopyPackage runs clif.copyPackage.
func (s *Server) cmdCopyPackage(_ context.Context, args []protocol.LSPAny) (any, error) {
	if len(args) == 0 {
		return nil, errors.New("clif.copyPackage requires a document URI argument")
	}

	docURI, _ := argString(args, 0)
	if docURI == "" {
		return nil, nil
	}

	filePath := cfpath.FromURI(docURI)
	dotPath := s.fileToPackage(filePath)

	return dotPath, nil
}

// cmdFindRefs runs clif.findRefs.
func (s *Server) cmdFindRefs(ctx context.Context, args []protocol.LSPAny) (any, error) {
	if len(args) == 0 {
		return nil, errors.New("clif.findRefs requires a function name argument")
	}

	funcName, _ := argString(args, 0)
	if funcName == "" {
		return nil, nil
	}

	sourceURI := ""
	if len(args) > 1 {
		sourceURI, _ = argString(args, 1)
	}

	// A walk of the workspace on disk: nothing here reads the open
	// documents, so it can run off the read loop.
	releaseReadLoop(ctx)

	s.log.Debug("findRefs: searching", cflog.String("funcName", funcName), cflog.Strings("roots", s.searchRoots()))

	sourceFile := uri.URI(sourceURI).Path()
	findOpts := refs.Options{
		FuncName:   funcName,
		SourceFile: sourceFile,
		Resolver:   s.getResolver(),
		Explain:    true,
	}
	entries := refs.Trace(s.FS, s.searchRoots(), &findOpts)
	result := refs.FormatResult(entries, funcName, sourceURI, s.searchRoots())

	s.log.Debug("findRefs: complete", cflog.String("funcName", funcName), cflog.Int("results", len(entries)))

	// Writing the report is opt-in, via a third argument. It used to be
	// unconditional, and the caller that fires most often is a code action
	// on an ordinary editor gesture — so asking "find all references" left
	// refs-<name>.md and refs-<name>.dot beside the file being read, inside
	// the user's source tree, ready to be committed by accident. The
	// summary is returned to the client either way, so the files duplicate
	// something the caller already has; only a caller that wants them on
	// disk asks for them.
	if argBool(args, 2) {
		s.writeRefsReport(ctx, funcName, sourceFile, &result)
	}

	return result.Summary, nil
}

// cmdExportDeps runs clif.exportDeps.
func (s *Server) cmdExportDeps(ctx context.Context, args []protocol.LSPAny) (any, error) {
	if len(args) == 0 {
		return nil, errors.New("clif.exportDeps requires a document URI")
	}

	docURI, _ := argString(args, 0)
	if docURI == "" {
		return nil, nil
	}

	funcName := ""
	if len(args) > 1 {
		funcName, _ = argString(args, 1)
	}

	fileURI := s.documentURI(docURI)

	// A fresh parse, not s.parseResults: the cached result loses every call
	// inside a function on an edit outside one (see parseForCalls), and the
	// graph silently fell back to bare component refs. The result is
	// private, so no document lock is needed.
	var pr *parser.ParseResult

	content, ok := s.getDocument(fileURI)

	// The graph is built from that text and the index, which has its own
	// lock, so the parse and the walk can run off the read loop.
	releaseReadLoop(ctx)

	if ok {
		pr = s.parseForDeps(fileURI, content)
	} else if data, err := s.FS.ReadFile(cfpath.FromURI(docURI)); err == nil {
		pr = s.parseForDeps(fileURI, string(data))
	}

	// The graph is the command line's (deps.BuildDocument): this is only how the
	// editor parses a file for it.
	result := deps.BuildDocument(&deps.Document{
		URI:      docURI,
		FuncName: funcName,
		PR:       pr,
		Parse: func(target uri.URI) *parser.ParseResult {
			data, err := s.FS.ReadFile(cfpath.FromURI(string(target)))
			if err != nil {
				return nil
			}

			return s.parseForDeps(target, string(data))
		},
		IndexRefs: func() []parser.ComponentRef {
			ptrs := s.index.RefsForFile(fileURI)
			refs := make([]parser.ComponentRef, 0, len(ptrs))

			for _, p := range ptrs {
				refs = append(refs, *p)
			}

			return refs
		},
		Index:    s.index,
		Resolver: s.getResolver(),
	})

	filePath := cfpath.FromURI(docURI)

	suffix := funcName
	if suffix == "" {
		suffix = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}

	mermaid := result.Graph.Mermaid()

	outFile, err := s.reportPath(filepath.Dir(filePath), "deps-"+suffix+".md")
	if err != nil {
		s.notifyError(ctx, err.Error())

		return mermaid, nil
	}

	if err := writeReport(outFile, []byte("```mermaid\n"+mermaid+"\n```\n")); err != nil {
		s.log.Error("failed to write file", cflog.String("path", outFile), cflog.Err(err))
	}

	dotFile := strings.TrimSuffix(outFile, ".md") + ".dot"
	if err := writeReport(dotFile, []byte(result.Graph.DOT())); err != nil {
		s.log.Error("failed to write file", cflog.String("path", dotFile), cflog.Err(err))
	}

	s.notify(ctx, protocol.MethodWindowShowMessage, &protocol.ShowMessageParams{
		Type:    protocol.MessageTypeInfo,
		Message: "Wrote " + outFile,
	})

	return mermaid, nil
}

// cmdScanWorkspace runs clif.scanWorkspace.
func (s *Server) cmdScanWorkspace(_ context.Context, _ []protocol.LSPAny) (any, error) {
	// Same reasoning as runDiagnostics: this goroutine outlives the
	// handler, so the request ctx (pooled/reset on return) is unsafe here.
	s.safeGo("scanWorkspace", func() { s.scanWorkspace(context.Background()) })

	return nil, nil
}
