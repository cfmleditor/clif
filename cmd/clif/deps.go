package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

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

	files := collectCFMLFiles(fsys, args)
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

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "writing JSON: %v\n", err)
		os.Exit(1)
	}
}

// depsParseSpec is how deps reads a file: the resolver's own parse, typing a
// call by the configured resolvers and not by what the called function declares
// it returns. A scan types `getService("UserService")` as `$any`, because
// getService returns any; a dependency graph wants the component the resolver
// names, which is also what clif.exportDeps reads. The resolver and the index
// behind it are the batch ones; this is the one place deps differs.
var depsParseSpec = resolve.ParseSpec{Calls: true, ByResolversOnly: true}

// depsForFile builds the dependency graph rooted at one file, the same way
// clif.exportDeps does for a whole document — so the CLI and the editor
// command answer with the same graph rather than each having their own idea of
// what a dependency is.
func depsForFile(path string, resolver *resolve.Resolver, idx *index.Index) []graph.Edge {
	content, err := vfs.ReadRegular(path)
	if err != nil {
		return nil
	}

	abs, _ := filepath.Abs(path)
	fileURI := uri.File(abs)

	pr := resolver.ParseFile(abs, string(content), depsParseSpec)

	var calls []parser.CallSite
	for _, sc := range pr.Scopes {
		calls = append(calls, pr.FuncCalls(sc.Start, sc.End)...)
	}

	result := deps.Build(&deps.Options{
		DocURI:    string(fileURI),
		Calls:     calls,
		Refs:      pr.ComponentRefs,
		Index:     idx,
		Resolver:  resolver,
		LoadCalls: depsCallLoader(resolver),
		MaxDepth:  10,
	})

	return result.Graph.Edges
}

// depsCallLoader parses a target file on demand so the traversal can continue
// past the first hop, memoised for the life of one command.
func depsCallLoader(resolver *resolve.Resolver) func(uri.URI, string) ([]parser.CallSite, []parser.ComponentRef) {
	parsed := make(map[uri.URI]*parser.ParseResult)

	return func(fileURI uri.URI, funcName string) ([]parser.CallSite, []parser.ComponentRef) {
		pr, ok := parsed[fileURI]
		if !ok {
			data, err := vfs.ReadRegular(strings.TrimPrefix(string(fileURI), "file://"))
			if err != nil {
				parsed[fileURI] = nil

				return nil, nil
			}

			pr = resolver.ParseFile(strings.TrimPrefix(string(fileURI), "file://"), string(data), depsParseSpec)
			parsed[fileURI] = pr
		}

		if pr == nil {
			return nil, nil
		}

		for _, sc := range pr.Scopes {
			for i := range pr.Funcs {
				f := &pr.Funcs[i]

				if !strings.EqualFold(f.Name, funcName) || int(f.Line) != sc.Start {
					continue
				}

				// Function locals first, then file-level: a `var x = new Foo()`
				// shadows a `variables.x`, and the consumer takes the first match.
				refs := slices.Concat(pr.FuncComponentRefs(sc.Start, sc.End), pr.ComponentRefs)

				return pr.FuncCalls(sc.Start, sc.End), refs
			}
		}

		return nil, nil
	}
}
