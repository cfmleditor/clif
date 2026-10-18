package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/codemap/mcp"
	"github.com/cfmleditor/clif/internal/codemap/store"
	"github.com/cfmleditor/clif/internal/confine"
	"github.com/cfmleditor/clif/internal/vfs"
)

const mcpUsage = `usage: clif mcp [--db <file>] [--root <dir>] [--allow-lint] [--map-only] [--no-explain]

Serve a CFML workspace over the Model Context Protocol on stdio, so an
assistant can ask about the code instead of grepping for it. It writes nothing.

Always offered, reading source on disk:
  find_unresolved_calls   what "unresolved" reports
  find_references         what "refs" reports
  explain_call            what "explain" reports
With --db, the code map "graph --db" built:
  search_symbols, get_symbol, get_callers, get_callees, find_path,
  list_islands, list_orphans, get_stats
With --allow-lint:
  lint                    CFLint's findings, as "cflint" reports them; starts a
                          CFLint process (carried by released builds)

  --db <file>     the database written by "graph --db"
  --root <dir>    workspace root: explain_call indexes it, and every tool's
                  paths must lie in it (default: the map's own root, or the
                  working directory without a map)
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
	db, root                      string
	allowLint, mapOnly, noExplain bool
}

func parseMCPFlags(args []string) mcpFlags {
	var f mcpFlags

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--db", "--root":
			if i+1 >= len(args) {
				fatalf("%s needs a value\n\n%s", args[i], mcpUsage)
			}

			if args[i] == "--db" {
				f.db = args[i+1]
			} else {
				f.root = args[i+1]
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

// mcpServer builds the server the flags ask for. db may be nil.
func mcpServer(f *mcpFlags, db *store.Store) *mcp.Server {
	srv := &mcp.Server{Store: db, Name: "clif", Version: version}

	if f.mapOnly {
		return srv
	}

	root := mcpRoot(f, db)

	srv.Unresolved = func(paths []string, globalDefs bool, limit int) (any, error) {
		return mcpUnresolved(root, paths, globalDefs, limit)
	}
	srv.FindRefs = func(target string, paths []string, limit int) (any, error) {
		return mcpFindRefs(root, target, paths, limit)
	}

	if f.allowLint {
		srv.Lint = newMCPLinter(root)
	}

	if !f.noExplain {
		srv.Explain = newExplainer(root)
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

// newExplainer builds the resolver lazily, on the first explain_call.
//
// Indexing the workspace costs seconds on a large one, and most sessions never
// call this tool. Paying it at startup would make every client's connection
// handshake time out on exactly the codebases where the map is most useful.
func newExplainer(root string) mcp.Explainer {
	var (
		once sync.Once
		env  *explainEnv
	)

	return func(file string, line int, match string) (string, error) {
		once.Do(func() {
			if abs, err := filepath.Abs(root); err == nil {
				env = newExplainEnv(vfs.OS{}, abs, true, &unresolvedFlags{})
			}
		})

		if env == nil {
			return "", fmt.Errorf("could not index the workspace at %s", root)
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
