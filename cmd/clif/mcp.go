package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/codemap/mcp"
	"github.com/cfmleditor/clif/internal/codemap/store"
	"github.com/cfmleditor/clif/internal/confine"
	"github.com/cfmleditor/clif/internal/vfs"
)

const mcpUsage = `usage: clif mcp [--db <file>] [--root <dir>] [--tools <list>] [--allow-lint] [--map-only] [--no-explain]

Serve a CFML workspace over the Model Context Protocol on stdio, so an
assistant can ask about the code instead of grepping for it. It writes nothing.

Always offered, reading source on disk:
  explore                 one-shot symbol/file context (source, callers, callees, impact)
  find_unresolved_calls   what "unresolved" reports
  find_references         what "refs" reports
  explain_call            what "explain" reports
  get_outline             component declaration, properties, and function signatures
  get_type_hierarchy      supertypes and subtypes of a CFC
  lookup_docs             embedded CFML tag, function, and member documentation
  check_structure         tag-structure and duplicate-argument diagnostics
  resolve_route           resolve a framework route via .clif.json routes config
  check_format            read-only format check and optional diff preview
With --db, the code map "graph --db" built:
  search_symbols, get_symbol, get_callers, get_callees, find_path,
  impact_analysis, list_islands, list_orphans, get_stats
With --allow-lint:
  lint                    CFLint's findings, as "cflint" reports them; starts a
                          CFLint process (carried by released builds)

  --db <file>     the database written by "graph --db"
  --root <dir>    workspace root: explain_call indexes it, and every tool's
                  paths must lie in it (default: the map's own root, or the
                  working directory without a map)
  --tools <list>  comma-separated allowlist of MCP tools to advertise (or "slim"
                  for explore,find_unresolved_calls,explain_call); also read
                  from CLIF_MCP_TOOLS
  --allow-lint    offer the lint tool
  --map-only      offer only the map tools, and never read source (needs --db)
  --no-explain    do not offer explain_call

Paths given to a tool are relative to the server's working directory, and
must lie inside the workspace root (--root, the map's root, or the working
directory), after symlinks.

Register it with your MCP client, for example:
  {"command": "clif", "args": ["mcp", "--allow-lint"], "cwd": "/path/to/project"}
or with a map, built first by: clif graph --db .clif/codemap.db .
  {"command": "clif", "args": ["mcp", "--db", ".clif/codemap.db"]}
`

// fatalf prints to stderr and exits. It exists so a caller that has already
// released its resources can exit without gocritic flagging the os.Exit as
// skipping a defer it deliberately does not need.
func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

type mcpFlags struct {
	db, root, tools               string
	allowLint, mapOnly, noExplain bool
}

func parseMCPFlags(args []string) mcpFlags {
	var f mcpFlags

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--db", "--root", "--tools":
			if i+1 >= len(args) {
				fatalf("%s needs a value\n\n%s", args[i], mcpUsage)
			}

			switch args[i] {
			case "--db":
				f.db = args[i+1]
			case "--root":
				f.root = args[i+1]
			case "--tools":
				f.tools = args[i+1]
			}

			i++
		case "--allow-lint":
			f.allowLint = true
		case "--map-only":
			f.mapOnly = true
		case "--no-explain":
			f.noExplain = true
		default:
			fatalf("unknown option %q\n\n%s", args[i], mcpUsage)
		}
	}

	if f.mapOnly && f.db == "" {
		fatalf("--map-only needs --db: without a map there is nothing to offer\n\n%s", mcpUsage)
	}

	if f.mapOnly && f.allowLint {
		fatalf("--map-only and --allow-lint cannot be used together\n\n%s", mcpUsage)
	}

	return f
}

// openMap opens and checks the map at path, or exits saying how to build one.
func openMap(path string) *store.Store {
	if _, err := os.Stat(path); err != nil {
		fatalf("No map at %s. Build one first:\n  clif graph --db %s <dir>\n", path, path)
	}

	// Read-only: the command says it writes nothing, and Open would create
	// the schema in whatever file --db names.
	db, err := store.OpenReadOnly(path)
	if err != nil {
		fatalf("%v\n", err)
	}

	if _, err := db.Stats(); err != nil {
		// Closed explicitly and then exited via a helper, because os.Exit runs no
		// defers and a WAL-mode database left open strands its -wal and -shm files.
		_ = db.Close()

		fatalf("%s holds no map yet. Run: clif graph --db %s <dir>\n", path, path)
	}

	return db
}

// mcpAllowedTools parses a comma-separated tool allowlist (from --tools or
// CLIF_MCP_TOOLS), expanding the "slim" preset.
func mcpAllowedTools(spec string) []string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		spec = strings.TrimSpace(os.Getenv("CLIF_MCP_TOOLS"))
	}

	if spec == "" {
		return nil
	}

	if strings.EqualFold(spec, "slim") {
		return []string{"explore", "find_unresolved_calls", "explain_call"}
	}

	parts := strings.Split(spec, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if name := strings.TrimSpace(p); name != "" {
			out = append(out, name)
		}
	}

	return out
}

// mcpServer builds the server the flags ask for. db may be nil.
func mcpServer(f *mcpFlags, db *store.Store) *mcp.Server {
	srv := &mcp.Server{
		Store:        db,
		Name:         "clif",
		Version:      version,
		AllowedTools: mcpAllowedTools(f.tools),
	}

	if f.mapOnly {
		return srv
	}

	root := mcpRoot(f, db)
	ws := newMCPWorkspace(root)

	srv.ReadSnippet = func(file string, line, maxLines int) (string, error) {
		return mcpReadSnippet(root, file, line, maxLines)
	}
	srv.LiveExplore = func(query string, includeSource bool) (any, error) {
		return mcpLiveExplore(ws, root, query, includeSource)
	}
	srv.Unresolved = func(paths []string, globalDefs bool, limit int) (any, error) {
		return mcpUnresolved(root, paths, globalDefs, limit)
	}
	srv.FindRefs = func(target string, paths []string, limit int) (any, error) {
		return mcpFindRefsWithWorkspace(ws, root, target, paths, limit)
	}
	srv.Outline = func(file string) (any, error) {
		return mcpOutline(root, file)
	}
	srv.TypeHierarchy = func(file string) (any, error) {
		return mcpTypeHierarchy(ws, root, file)
	}
	srv.LookupDocs = mcpLookupDocs
	srv.Structure = func(paths []string, limit int) (any, error) {
		return mcpCheckStructure(root, paths, limit)
	}
	srv.ResolveRoute = func(routeStr string) (any, error) {
		return mcpResolveRoute(ws, root, routeStr)
	}
	srv.CheckFormat = func(paths []string, includeDiff bool, limit int) (any, error) {
		return mcpCheckFormat(root, paths, includeDiff, limit)
	}

	if f.allowLint {
		srv.Lint = newMCPLinter(root)
	}

	if !f.noExplain {
		srv.Explain = newWorkspaceExplainer(ws, root)
	}

	return srv
}

// mcpRoot is the workspace the task tools serve, and every path a tool is
// given must lie in it: --root, or the map's own root, or the working
// directory.
func mcpRoot(f *mcpFlags, db *store.Store) string {
	root := f.root
	if root == "" && db != nil {
		root, _ = db.Meta("root")
	}

	if root == "" {
		root = mustGetwd()
	}

	return root
}

func cmdMCP(args []string) {
	f := parseMCPFlags(args)

	limitLongLivedCaches()

	var db *store.Store
	if f.db != "" {
		db = openMap(f.db)
		defer func() { _ = db.Close() }()
	}

	srv := mcpServer(&f, db)

	// Diagnostics go to stderr. Stdout is the protocol channel, and one stray line
	// on it is a parse error at the other end rather than a log message.
	if f.db != "" {
		fmt.Fprintf(os.Stderr, "clif MCP server on stdio (db: %s)\n", f.db)
	} else {
		fmt.Fprintf(os.Stderr, "clif MCP server on stdio (no code map)\n")
	}

	if err := srv.Serve(os.Stdin, os.Stdout); err != nil {
		// Close before exiting rather than relying on the defer, which os.Exit
		// skips: this is a WAL-mode database and an unclosed one strands its -wal
		// and -shm files next to it for the next reader to recover.
		if db != nil {
			_ = db.Close()
		}

		fatalf("%v\n", err)
	}
}

// newWorkspaceExplainer traces call sites using the shared, incrementally
// refreshed workspace index.
func newWorkspaceExplainer(ws *mcpWorkspace, root string) mcp.Explainer {
	return func(file string, line int, match string) (string, error) {
		env, _, err := ws.get()
		if err != nil {
			return "", err
		}

		return explainAt(root, env, file, line, match)
	}
}

func explainAt(root string, env *explainEnv, file string, line int, match string) (string, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", file, err)
	}

	// The answer quotes the line it explains, so a file outside the
	// workspace would be read back to the caller; see absPaths.
	if err := withinRoot(file, abs, confine.Roots(root)); err != nil {
		return "", err
	}

	data, err := vfs.ReadRegular(abs)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", file, err)
	}

	var b strings.Builder

	if !env.explain(&b, abs, string(data), line, match) {
		return fmt.Sprintf("No call site found at %s:%d%s.\n", file, line,
			map[bool]string{true: "", false: " matching " + match}[match == ""]), nil
	}

	return b.String(), nil
}
