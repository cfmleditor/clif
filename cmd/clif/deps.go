package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cfpath "github.com/cfmleditor/clif/internal/path"

	"github.com/cfmleditor/clif/internal/deps"
	"github.com/cfmleditor/clif/internal/graph"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
	"go.lsp.dev/uri"
)

// DepEdge is one dependency edge, attributed to the file it was found in.
//
// From and To are the graph's node labels, so they carry the resolved component
// and the line the dependency was written on. Dashed marks a dependency whose
// component could not be resolved to a file — the previous output had no way to
// express that, and reported resolvable and unresolvable dependencies alike.
type DepEdge struct {
	FromFile string `json:"fromFile"`
	From     string `json:"from"`
	To       string `json:"to"`
	Dashed   bool   `json:"dashed,omitempty"`
}

// DepResult holds the full dependency graph.
type DepResult struct {
	Files int       `json:"files"`
	Edges []DepEdge `json:"edges"`
}

func cmdDeps(args []string) {
	format := "json"

	var filteredArgs []string

	for _, a := range args {
		if a == "--mermaid" {
			format = "mermaid"

			continue
		}

		filteredArgs = append(filteredArgs, positional(a, depsUsage))
	}

	args = filteredArgs

	if len(args) == 0 {
		fmt.Fprint(os.Stderr, depsUsage)
		os.Exit(1)
	}

	fsys := vfs.OS{}

	files := cfpath.CollectCFML(fsys, args)
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no CFML files found\n")
		os.Exit(1)
	}

	resolver := workspaceResolver(args, files)
	idx := resolver.Index

	var (
		edges []DepEdge
		seen  = make(map[string]bool)
	)

	for _, f := range files {
		for _, e := range depsForFile(f, resolver, idx) {
			key := f + "\x00" + e.From + "\x00" + e.To
			if seen[key] {
				continue
			}

			seen[key] = true

			edges = append(edges, DepEdge{FromFile: f, From: e.From, To: e.To, Dashed: e.Dashed})
		}
	}

	result := DepResult{Files: len(files), Edges: edges}

	if format == "mermaid" {
		g := graph.Graph{Direction: "LR"}
		for _, e := range edges {
			g.Edges = append(g.Edges, graph.Edge{From: e.From, To: e.To, Dashed: e.Dashed})
		}

		fmt.Println(g.Mermaid())

		return
	}

	writeJSONOrExit(result)
}

// depsForFile builds the dependency graph rooted at one file, the same way
// clif.exportDeps does for a whole document (deps.BuildDocument) — so the CLI and
// the editor command answer with the same graph rather than each having their
// own idea of what a dependency is.
func depsForFile(path string, resolver *resolve.Resolver, idx *index.Index) []graph.Edge {
	content, err := vfs.ReadRegular(path)
	if err != nil {
		return nil
	}

	abs, _ := filepath.Abs(path)

	result := deps.BuildDocument(&deps.Document{
		URI: string(uri.File(abs)),
		PR:  resolver.ParseFile(abs, string(content), resolve.DepsSpec()),
		Parse: func(fileURI uri.URI) *parser.ParseResult {
			file := strings.TrimPrefix(string(fileURI), "file://")

			data, err := vfs.ReadRegular(file)
			if err != nil {
				return nil
			}

			return resolver.ParseFile(file, string(data), resolve.DepsSpec())
		},
		Index:    idx,
		Resolver: resolver,
	})

	return result.Graph.Edges
}
