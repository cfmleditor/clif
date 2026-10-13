package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cfmleditor/clif/internal/daemon"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/unresolved"
	"github.com/cfmleditor/clif/internal/vfs"
)

// cmdExplain prints, for one or more call sites on a given line, every resolution
// step CanResolveCall walked through to reach its verdict — which ComponentRef or
// componentResolver set the receiver's type, which FuncLookup hop fired for a chained
// call, and why the final method-exists check passed or failed. This turns a manual
// trace through script_parser.go/tag_parser.go/result.go/resolve.go into one command.
func cmdExplain(args []string) {
	var (
		configRoot string
		fl         unresolvedFlags
	)

	var filteredArgs []string

	for i := 0; i < len(args); i++ {
		if args[i] == "--no-infer-args" {
			fl.noInferArgs = true

			continue
		}

		if args[i] == "--root" && i+1 < len(args) {
			configRoot = args[i+1]
			i++

			continue
		}

		filteredArgs = append(filteredArgs, positional(args[i], explainUsage))
	}

	args = filteredArgs

	if len(args) < 2 {
		fmt.Fprint(os.Stderr, explainUsage)
		os.Exit(1)
	}

	file, err := filepath.Abs(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	line, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid line number %q\n", args[1])
		os.Exit(1)
	}

	var filter string
	if len(args) > 2 {
		filter = args[2]
	}

	fsys := vfs.OS{}

	searchDir := filepath.Dir(file)

	if configRoot != "" {
		abs, err := filepath.Abs(configRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		searchDir = abs
	}

	env := newExplainEnv(fsys, searchDir, false, &fl)

	data, err := fsys.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", file, err)
		os.Exit(1)
	}

	if !env.explain(os.Stdout, file, string(data), line, filter) {
		fmt.Fprintf(os.Stderr, "no call sites found on %s:%d\n", file, line)
		os.Exit(1)
	}
}

// explainEnv is what an explanation is made with: the unresolved scan's own
// options and resolver, so the trace explains the verdict the report gives. The
// CLI and the MCP tool both build one; the MCP tool had its own, without the
// bean resolvers, the assumed framework, or the scan's indexing and parse.
type explainEnv struct {
	opt      *unresolved.Options
	resolver *resolve.Resolver
}

// newExplainEnv takes the flags the unresolved report was run with, so a trace
// can be made under the same ones; the MCP tool passes none.
func newExplainEnv(fsys vfs.FS, searchDir string, quiet bool, fl *unresolvedFlags) *explainEnv {
	cfg, _ := daemon.FindConfig(searchDir)
	if cfg != nil && !quiet {
		fmt.Fprintf(os.Stderr, "Using config: %s\n", cfg.Path)
	}

	opt := unresolvedOptions(cfg, []string{searchDir}, fl)
	files := collectCFMLFiles(fsys, opt.WorkspaceFolders)

	if !quiet {
		fmt.Fprintf(os.Stderr, "Indexing %d files...\n", len(files))
	}

	return &explainEnv{opt: opt, resolver: unresolved.NewResolver(fsys, files, opt)}
}

// explain writes the trace of every call on a 1-based line of file, narrowed by
// filter, and reports whether there were any. The selection and the report are
// shared with clif.explainCall, so the server's answer reads exactly as this
// one does.
func (e *explainEnv) explain(w io.Writer, file, content string, line int, filter string) bool {
	pr := unresolved.Parse(e.resolver, file, content, e.opt)

	matches := resolve.CallsOnLine(pr, line-1, filter)
	if len(matches) == 0 {
		return false
	}

	e.resolver.WriteExplanation(w, file, line-1, matches, pr, filepath.Dir(file))

	return true
}
