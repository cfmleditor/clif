//go:build !wasip1

package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cfmleditor/clif/internal/confine"
	"github.com/cfmleditor/clif/internal/daemon"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// mcpDaemonClient connects clif mcp to a running clif daemon over its Unix
// socket, reusing the daemon's already-populated shared workspace index
// instead of indexing a second copy in-process.
type mcpDaemonClient struct {
	root     string
	sockPath string

	mu     sync.Mutex
	raw    net.Conn
	conn   jsonrpc2.Conn
	closed bool
}

func connectMCPDaemon(root, explicitSocket string) (*mcpDaemonClient, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving root %s: %w", root, err)
	}

	sockPath := strings.TrimSpace(explicitSocket)
	if sockPath == "" {
		cfg, _ := daemon.FindConfig(absRoot)
		if cfg == nil {
			return nil, errors.New("no .clif.json found to locate daemon socket; pass --daemon-socket <path>")
		}

		sockPath = cfg.SocketPath()
	}

	dialer := net.Dialer{Timeout: 2 * time.Second}

	raw, err := dialer.DialContext(context.Background(), "unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("connecting to daemon at %s: %w", sockPath, err)
	}

	conn := jsonrpc2.NewConn(jsonrpc2.NewStream(raw))
	// Accept server notifications (window/logMessage, window/showMessage, etc.)
	// without blocking the daemon's write loop.
	conn.Go(context.Background(), func(_ context.Context, _ *jsonrpc2.Request) (any, error) {
		return nil, nil
	})

	c := &mcpDaemonClient{
		root:     absRoot,
		sockPath: sockPath,
		raw:      raw,
		conn:     conn,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	initParams := protocol.InitializeParams{
		WorkspaceFolders: protocol.NewNullable([]protocol.WorkspaceFolder{{
			URI:  cfpath.ToURI(absRoot),
			Name: filepath.Base(absRoot),
		}}),
	}

	var initRes protocol.InitializeResult
	if _, err := conn.Call(ctx, protocol.MethodInitialize, initParams, &initRes); err != nil {
		_ = c.Close()

		return nil, fmt.Errorf("daemon initialize failed: %w", err)
	}

	if err := conn.Notify(ctx, protocol.MethodInitialized, struct{}{}); err != nil {
		_ = c.Close()

		return nil, fmt.Errorf("daemon initialized notify failed: %w", err)
	}

	return c, nil
}

func (c *mcpDaemonClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	if c.conn != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_, _ = c.conn.Call(ctx, protocol.MethodShutdown, nil, nil)
		_ = c.conn.Notify(ctx, protocol.MethodExit, nil)

		cancel()
	}

	if c.raw != nil {
		return c.raw.Close()
	}

	return nil
}

func (c *mcpDaemonClient) executeCommand(ctx context.Context, command string, args ...any) (jsontext.Value, error) {
	lspArgs := make([]protocol.LSPAny, 0, len(args))
	for _, a := range args {
		raw, err := json.Marshal(a)
		if err != nil {
			return nil, err
		}

		lspArgs = append(lspArgs, raw)
	}

	params := protocol.ExecuteCommandParams{
		Command:   command,
		Arguments: lspArgs,
	}

	var result jsontext.Value
	if _, err := c.conn.Call(ctx, protocol.MethodWorkspaceExecuteCommand, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *mcpDaemonClient) ExplainCall(file string, line int, match string) (string, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", file, err)
	}

	if err := withinRoot(file, abs, confine.Roots(c.root)); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	raw, err := c.executeCommand(ctx, "clif.explainCall", string(cfpath.ToURI(abs)), line-1, match)
	if err != nil {
		return "", err
	}

	var msg string
	if err := json.Unmarshal(raw, &msg); err != nil {
		return string(raw), nil
	}

	return msg, nil
}

func (c *mcpDaemonClient) ResolveRoute(routeStr string) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	raw, err := c.executeCommand(ctx, "clif.resolveRoute", strings.TrimSpace(routeStr))
	if err != nil {
		return nil, err
	}

	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *mcpDaemonClient) WorkspaceSymbols(query string) ([]protocol.SymbolInformation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var syms []protocol.SymbolInformation
	if _, err := c.conn.Call(ctx, protocol.MethodWorkspaceSymbol, protocol.WorkspaceSymbolParams{Query: query}, &syms); err != nil {
		return nil, err
	}

	return syms, nil
}

func (c *mcpDaemonClient) isTransportFailure(err error) bool {
	if err == nil {
		return false
	}

	select {
	case <-c.conn.Done():
		return true
	default:
	}

	if _, ok := errors.AsType[net.Error](err); ok {
		return true
	}

	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed)
}

func (c *mcpDaemonClient) LiveExplore(query string, includeSource bool) (any, error) {
	// Handle file-path queries directly (matching mcpLiveExplore).
	candidate := query
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(c.root, filepath.FromSlash(candidate))
	}

	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		outline, oerr := mcpOutline(c.root, candidate)
		if oerr != nil {
			return nil, oerr
		}

		out := map[string]any{
			"query":   query,
			"mode":    "daemon-file",
			"outline": outline,
		}

		if includeSource {
			if src, serr := mcpReadSnippet(c.root, candidate, 0, 100); serr == nil {
				out["source"] = src
			}
		}

		return out, nil
	}

	syms, err := c.WorkspaceSymbols(query)
	if err != nil {
		return nil, err
	}

	defs := make([]map[string]any, 0, min(len(syms), 5))

	for i := range syms {
		if len(defs) >= 5 {
			break
		}

		sym := &syms[i]
		defPath := cfpath.FromURI(string(sym.Location.URI))
		line := int(sym.Location.Range.Start.Line) + 1

		entry := map[string]any{
			"name": sym.Name,
			"file": defPath,
			"line": line,
		}

		if includeSource {
			if src, serr := mcpReadSnippet(c.root, defPath, line, 60); serr == nil {
				entry["source"] = src
			}
		}

		defs = append(defs, entry)
	}

	return map[string]any{
		"query":       query,
		"mode":        "daemon-symbol",
		"definitions": defs,
		"total":       len(syms),
	}, nil
}
