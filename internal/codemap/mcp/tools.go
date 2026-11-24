package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/codemap/store"
)

type tool struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

// readOnly marks every tool here. MCP clients use it to decide what needs
// confirmation, and nothing in this server writes.
var readOnly = map[string]any{"readOnlyHint": true, "destructiveHint": false, "idempotentHint": true}

// runsCFLint marks lint, which writes nothing either but starts a CFLint
// process, and in a build without CFLint embedded downloads it on first use:
// open-world, in MCP's terms.
var runsCFLint = map[string]any{"readOnlyHint": true, "destructiveHint": false, "idempotentHint": true, "openWorldHint": true}

func strList(desc string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
}

const pathsHelp = "Files or directories, absolute or relative to the server's working directory. " +
	"The .clif.json above the first one decides how calls resolve."

func obj(props map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

func num(desc string) map[string]any { return map[string]any{"type": "integer", "description": desc} }

func flag(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

const idHelp = "Node id. A function is \"<file>::<lowercased name>\", e.g. " +
	"\"packages/tass/core/user.cfc::checkpermslist\". A file is its path relative to the map root. " +
	"Use search_symbols to find one rather than guessing."

func (s *Server) tools() []tool {
	var list []tool

	if s.Store != nil || s.LiveExplore != nil {
		list = append(list, exploreTool)
	}

	list = append(list, s.taskTools()...)

	if s.Store != nil {
		list = append(list, mapTools...)
	}

	if s.Explain != nil {
		list = append(list, explainTool)
	}

	if len(s.AllowedTools) == 0 {
		return list
	}

	allow := make(map[string]struct{}, len(s.AllowedTools))
	for _, name := range s.AllowedTools {
		allow[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}

	filtered := make([]tool, 0, len(allow))
	for _, t := range list {
		if _, ok := allow[strings.ToLower(t.Name)]; ok {
			filtered = append(filtered, t)
		}
	}

	return filtered
}

var exploreTool = tool{
	Name:  "explore",
	Title: "Explore symbol or file context",
	Description: "One-shot context lookup for a symbol name, node id, or file path. Returns " +
		"matching symbol metadata, verbatim line-numbered source, direct callers and callees, " +
		"and transitive blast radius (when a code map is loaded) in a single call — avoiding " +
		"multi-step search/read round-trips.",
	InputSchema: obj(map[string]any{
		"query":          str("Symbol name, node id (e.g. \"svc.cfc::getuser\"), or file path."),
		"depth":          num("Transitive blast-radius depth when a code map is loaded (default 2, max 10)."),
		"max_symbols":    num("Maximum matching symbols to expand (default 3, max 10)."),
		"include_source": flag("Include verbatim line-numbered source snippets (default true)."),
	}, "query"),
	Annotations: readOnly,
}

// mapTools query the code map, and are offered only when one is loaded.
var mapTools = []tool{
	{
		Name:  "search_symbols",
		Title: "Search symbols",
		Description: "Find functions and files by name, path or component. Start here: every other " +
			"tool takes a node id, and this is how you get one. A blank query with filters set " +
			"lists by those filters alone (for example every unreachable function under a path).",
		InputSchema: obj(map[string]any{
			"query":  str("Text to match against name, file and component. Substring matching, so \"perms\" finds checkPermsList."),
			"kind":   map[string]any{"type": "string", "enum": []string{"function", "file", "package", "external"}, "description": "Restrict to one kind of node."},
			"file":   str("Restrict to nodes whose path starts with this prefix."),
			"island": num("Restrict to one connected component of the graph."),
			"only_unreachable": flag("Only nodes no entry point reaches. Deletion candidates, " +
				"not proven dead code — see get_stats for how many calls went unresolved."),
			"sort_by_fan_in": flag("Order by how many things depend on the node, most first, rather than by name."),
			"limit":          num("Maximum results (default 50)."),
		}),
		Annotations: readOnly,
	},
	{
		Name:        "get_symbol",
		Title:       "Get one symbol",
		Description: "Full detail for one node: file, line, access, island, and how many things depend on it.",
		InputSchema: obj(map[string]any{"id": str(idHelp)}, "id"),
		Annotations: readOnly,
	},
	{
		Name:  "get_callers",
		Title: "Get callers",
		Description: "What depends on this node, and how. An empty result on a function that " +
			"clearly runs usually means the call was made through a receiver the resolver could " +
			"not type, not that nothing calls it.",
		InputSchema: obj(map[string]any{
			"id":    str(idHelp),
			"limit": num("Maximum results (default 200)."),
		}, "id"),
		Annotations: readOnly,
	},
	{
		Name:        "get_callees",
		Title:       "Get callees",
		Description: "What this node depends on: what it calls, instantiates, extends or includes.",
		InputSchema: obj(map[string]any{
			"id":    str(idHelp),
			"limit": num("Maximum results (default 200)."),
		}, "id"),
		Annotations: readOnly,
	},
	{
		Name:  "find_path",
		Title: "Find a dependency path",
		Description: "The shortest chain of dependencies from one node to another, or nothing if " +
			"there is none within the depth limit. Use it to answer \"how does this page reach that " +
			"component\" without reading the files in between.",
		InputSchema: obj(map[string]any{
			"from":      str("Starting node id."),
			"to":        str("Target node id."),
			"max_depth": num("How many hops to search (default 12)."),
		}, "from", "to"),
		Annotations: readOnly,
	},
	{
		Name:  "impact_analysis",
		Title: "Analyze transitive impact (blast radius)",
		Description: "Walk the reverse dependency graph from a node up to max_depth hops to see " +
			"everything transitively affected by changing it: affected files, entry points reached, " +
			"and impacted callers ordered by hop distance.",
		InputSchema: obj(map[string]any{
			"id":        str(idHelp),
			"max_depth": num("Maximum reverse hops to walk (default 4, max 10)."),
			"limit":     num("Maximum impacted nodes returned (default 100)."),
		}, "id"),
		Annotations: readOnly,
	},
	{
		Name:  "list_islands",
		Title: "List islands",
		Description: "The disconnected pieces of the graph, largest first. The biggest is the " +
			"application; the rest are detached subsystems — either genuinely dead, or live code " +
			"reached through a call the resolver could not follow.",
		InputSchema: obj(map[string]any{
			"detached_only": flag("Only islands no entry point reaches."),
			"limit":         num("Maximum results (default 50)."),
		}),
		Annotations: readOnly,
	},
	{
		Name:  "list_orphans",
		Title: "List unreferenced symbols",
		Description: "Nodes nothing depends on and no entry point reaches. Deletion candidates. " +
			"Always read get_stats alongside: an unresolved call is an edge the map does not have, " +
			"so a high unresolved count makes this list longer than the truth.",
		InputSchema: obj(map[string]any{
			"kind":  map[string]any{"type": "string", "enum": []string{"function", "file"}, "description": "Restrict to one kind."},
			"limit": num("Maximum results (default 50)."),
		}),
		Annotations: readOnly,
	},
	{
		Name:  "get_stats",
		Title: "Map statistics",
		Description: "Counts for the whole map, including how many call sites resolved to a " +
			"definition. That ratio is the map's confidence in itself: at 50%, half the call graph " +
			"is missing and an empty caller list proves much less than it looks like it does.",
		InputSchema: obj(map[string]any{}),
		Annotations: readOnly,
	},
}

var explainTool = tool{
	Name:  "explain_call",
	Title: "Explain a call site",
	Description: "Trace, step by step, how a call site's receiver was typed and why the method " +
		"check passed or failed. This is the tool for \"why is this reported unresolved\" and for " +
		"\"where did that component path come from\" — it names the exact componentResolver that " +
		"fired. It re-parses the file, so it reflects what is on disk now, not what any map holds.",
	InputSchema: obj(map[string]any{
		"file":  str("Path to the CFML file, absolute or relative to the working directory."),
		"line":  num("1-based line number of the call site."),
		"match": str("Optional substring to pick one call on that line."),
	}, "file", "line"),
	Annotations: readOnly,
}

// taskTools run what the CLI runs, from source on disk; none needs a map.
func (s *Server) taskTools() []tool {
	var list []tool

	if s.Unresolved != nil {
		list = append(list, tool{
			Name:  "find_unresolved_calls",
			Title: "Find unresolved calls",
			Description: "Every component or method call under the given paths that does not resolve: " +
				"a receiver with no known component, a method the component does not declare, a " +
				"component that does not exist. What `clif unresolved` reports. Pass " +
				"explain_call a result's file and line to see why. It indexes the workspace on " +
				"every call, which takes seconds on a large one.",
			InputSchema: obj(map[string]any{
				"paths":       strList(pathsHelp),
				"global_defs": flag("Accept a bare call to a function any indexed file declares."),
				"limit":       num("Maximum calls listed (default 200). The result gives the total."),
			}, "paths"),
			Annotations: readOnly,
		})
	}

	if s.FindRefs != nil {
		list = append(list, tool{
			Name:  "find_references",
			Title: "Find references",
			Description: "Every reference under the given paths to a component (a dot-path such as " +
				"\"packages.finance.service\") or a function (a bare name such as \"getReport\"), with " +
				"whether each resolved to it. What `clif refs` reports. It reads source, so " +
				"it needs no map and reflects what is on disk now.",
			InputSchema: obj(map[string]any{
				"target": str("A component dot-path, or a function name."),
				"paths":  strList(pathsHelp),
				"limit":  num("Maximum references listed (default 200). The result gives the total."),
			}, "target", "paths"),
			Annotations: readOnly,
		})
	}

	if s.Lint != nil {
		list = append(list, tool{
			Name:  "lint",
			Title: "Run CFLint",
			Description: "Run CFLint over files or directories and return its findings: path, line, " +
				"column, severity, rule and message. linting.minSeverity in .clif.json sets the " +
				"least severe level reported. It starts a CFLint process (released clif builds carry " +
				"CFLint; others download it on first use) and writes no report. CFLint reports some rules (MISSING_VAR, IMPLICIT_SCOPE) " +
				"once per name per run, so lint one file to see all of its findings.",
			InputSchema: obj(map[string]any{
				"paths": strList("Files or directories, absolute or relative to the server's working directory."),
				"limit": num("Maximum findings listed (default 200). The result gives the total."),
			}, "paths"),
			Annotations: runsCFLint,
		})
	}

	if s.Outline != nil {
		list = append(list, tool{
			Name:  "get_outline",
			Title: "Get file outline",
			Description: "List a CFML file's component declaration (extends, accessors, persistent), " +
				"properties, and function signatures with 1-based line numbers. Parses only the " +
				"single file on disk, needs no code map, and is much faster than reading large CFCs.",
			InputSchema: obj(map[string]any{
				"file": str("Path to the CFML file, absolute or relative to the working directory."),
			}, "file"),
			Annotations: readOnly,
		})
	}

	if s.TypeHierarchy != nil {
		list = append(list, tool{
			Name:  "get_type_hierarchy",
			Title: "Get component type hierarchy",
			Description: "Return the inheritance hierarchy for a CFC file: supertypes (the extends " +
				"ancestor chain) and subtypes (components in the workspace that extend it). Uses " +
				"the live workspace index and needs no code map.",
			InputSchema: obj(map[string]any{
				"file": str("Path to the CFC file, absolute or relative to the working directory."),
			}, "file"),
			Annotations: readOnly,
		})
	}

	if s.LookupDocs != nil {
		list = append(list, tool{
			Name:  "lookup_docs",
			Title: "Look up CFML documentation",
			Description: "Look up built-in CFML tag, function, or member-function documentation " +
				"embedded in clif. Pass \"name\" for an exact entry (e.g. \"arrayMap\", \"cfquery\", " +
				"\"array.map\") or \"query\" for substring search.",
			InputSchema: obj(map[string]any{
				"name":  str("Exact tag, function, or type.member name to look up."),
				"query": str("Substring to search across built-in tag, function, and member names/descriptions."),
				"kind":  map[string]any{"type": "string", "enum": []string{"function", "tag", "member"}, "description": "Restrict search results to one kind."},
				"limit": num("Maximum search results (default 25)."),
			}),
			Annotations: readOnly,
		})
	}

	if s.Structure != nil {
		list = append(list, tool{
			Name:  "check_structure",
			Title: "Check CFML tag structure",
			Description: "Check CFML files for structural errors without starting CFLint: unclosed " +
				"or mismatched tags, misplaced <cfcatch>/<cfelse>, duplicate tag attributes, and " +
				"duplicate function arguments.",
			InputSchema: obj(map[string]any{
				"paths": strList("Files or directories, absolute or relative to the server's working directory."),
				"limit": num("Maximum problems listed (default 200). The result gives the total."),
			}, "paths"),
			Annotations: readOnly,
		})
	}

	if s.ResolveRoute != nil {
		list = append(list, tool{
			Name:  "resolve_route",
			Title: "Resolve a framework route",
			Description: "Resolve a framework route string (e.g. \"admin.users.edit\") against the " +
				"workspace's .clif.json routes configuration, returning matching controller methods " +
				"and view files.",
			InputSchema: obj(map[string]any{
				"route": str("The route string to resolve."),
			}, "route"),
			Annotations: readOnly,
		})
	}

	if s.CheckFormat != nil {
		list = append(list, tool{
			Name:  "check_format",
			Title: "Check CFML formatting",
			Description: "Check whether CFML files would change under `clif format` without writing " +
				"anything to disk. Optionally returns a line diff preview for changed files.",
			InputSchema: obj(map[string]any{
				"paths":        strList("Files or directories, absolute or relative to the server's working directory."),
				"include_diff": flag("Include a unified-style line diff preview for changed files (default false, automatically enabled for a single file)."),
				"limit":        num("Maximum changed/errored files listed (default 50)."),
			}, "paths"),
			Annotations: readOnly,
		})
	}

	return list
}

type callParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// callTool runs a tool and wraps the result.
//
// A tool that fails returns isError on a successful JSON-RPC response rather than
// a protocol-level error. That is what MCP specifies, and the reason is practical:
// a protocol error is the transport's problem and many clients drop the
// conversation over it, where a tool error is information the model should see and
// can act on.
func (s *Server) callTool(raw json.RawMessage) (any, *rpcError) {
	var p callParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpcError{codeInvalidParams, "bad tools/call params: " + err.Error()}
	}

	payload, err := s.run(p.Name, p.Arguments)
	if err != nil {
		return map[string]any{
			"content": []any{map[string]any{"type": "text", "text": err.Error()}},
			"isError": true,
		}, nil
	}

	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, &rpcError{codeInternal, "encoding result: " + err.Error()}
	}

	return map[string]any{
		"content":           []any{map[string]any{"type": "text", "text": string(encoded)}},
		"structuredContent": payload,
	}, nil
}

type toolArgs struct {
	Query           string   `json:"query"`
	Kind            string   `json:"kind"`
	File            string   `json:"file"`
	Island          *int     `json:"island"`
	OnlyUnreachable bool     `json:"only_unreachable"`
	SortByFanIn     bool     `json:"sort_by_fan_in"`
	Limit           int      `json:"limit"`
	ID              string   `json:"id"`
	From            string   `json:"from"`
	To              string   `json:"to"`
	MaxDepth        int      `json:"max_depth"`
	DetachedOnly    bool     `json:"detached_only"`
	Line            int      `json:"line"`
	Match           string   `json:"match"`
	Target          string   `json:"target"`
	Paths           []string `json:"paths"`
	GlobalDefs      bool     `json:"global_defs"`
	Name            string   `json:"name"`
	Route           string   `json:"route"`
	IncludeDiff     bool     `json:"include_diff"`
	Depth           int      `json:"depth"`
	MaxSymbols      int      `json:"max_symbols"`
	IncludeSource   *bool    `json:"include_source"`
}

// toolHandler runs one tool for already-parsed arguments.
type toolHandler func(s *Server, a *toolArgs) (any, error)

// toolHandlers is every tool run answers, by name.
var toolHandlers = map[string]toolHandler{
	"explore":               (*Server).explore,
	"search_symbols":        (*Server).searchSymbols,
	"get_symbol":            (*Server).getSymbol,
	"get_callers":           (*Server).getCallers,
	"get_callees":           (*Server).getCallees,
	"find_path":             (*Server).findPath,
	"impact_analysis":       (*Server).impactAnalysis,
	"list_islands":          (*Server).listIslands,
	"list_orphans":          (*Server).listOrphans,
	"get_stats":             (*Server).getStats,
	"explain_call":          (*Server).explainCall,
	"find_unresolved_calls": (*Server).findUnresolved,
	"find_references":       (*Server).findReferences,
	"lint":                  (*Server).lintPaths,
	"get_outline":           (*Server).getOutline,
	"get_type_hierarchy":    (*Server).getTypeHierarchy,
	"lookup_docs":           (*Server).lookupDocs,
	"check_structure":       (*Server).checkStructure,
	"resolve_route":         (*Server).resolveRoute,
	"check_format":          (*Server).checkFormat,
}

func (s *Server) run(name string, raw json.RawMessage) (any, error) {
	var a toolArgs

	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, fmt.Errorf("bad arguments for %s: %w", name, err)
		}
	}

	if s.Store == nil && slices.ContainsFunc(mapTools, func(t tool) bool { return t.Name == name }) {
		return nil, fmt.Errorf("%s needs a code map, and this server was started without one: build it with "+
			"`clif graph --db <file> <dir>` and start the server with --db <file>", name)
	}

	handler, ok := toolHandlers[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %q", name)
	}

	return handler(s, &a)
}

func (s *Server) searchSymbols(a *toolArgs) (any, error) {
	found, err := s.Store.SearchSymbols(&store.SearchOptions{
		Query: a.Query, Kind: a.Kind, File: a.File, Island: a.Island,
		OnlyDead: a.OnlyUnreachable, SortByFan: a.SortByFanIn, Limit: a.Limit,
	})

	return wrap("symbols", found), err
}

func (s *Server) getSymbol(a *toolArgs) (any, error) {
	if a.ID == "" {
		return nil, errMissing("id")
	}

	return s.Store.Node(a.ID)
}

func (s *Server) getCallers(a *toolArgs) (any, error) {
	if a.ID == "" {
		return nil, errMissing("id")
	}

	found, err := s.Store.Callers(a.ID, a.Limit)

	return wrap("callers", found), err
}

func (s *Server) getCallees(a *toolArgs) (any, error) {
	if a.ID == "" {
		return nil, errMissing("id")
	}

	found, err := s.Store.Callees(a.ID, a.Limit)

	return wrap("callees", found), err
}

func (s *Server) findPath(a *toolArgs) (any, error) {
	if a.From == "" || a.To == "" {
		return nil, errMissing("from and to")
	}

	path, err := s.Store.Path(a.From, a.To, a.MaxDepth)
	if err != nil {
		return nil, err
	}

	if path == nil {
		return map[string]any{
			"found": false,
			"note": "No dependency path within the depth limit. That can mean there is none, " +
				"or that a call on the way resolved to nothing — check get_stats.",
		}, nil
	}

	return map[string]any{"found": true, "hops": len(path) - 1, "path": path}, nil
}

func (s *Server) impactAnalysis(a *toolArgs) (any, error) {
	if a.ID == "" {
		return nil, errMissing("id")
	}

	return s.Store.Impact(a.ID, a.MaxDepth, a.Limit)
}

func (s *Server) explore(a *toolArgs) (any, error) {
	query := strings.TrimSpace(a.Query)
	if query == "" {
		return nil, errMissing("query")
	}

	includeSource := true
	if a.IncludeSource != nil {
		includeSource = *a.IncludeSource
	}

	if s.Store == nil {
		if s.LiveExplore == nil {
			return nil, errors.New("explore is not available on this server")
		}

		return s.LiveExplore(query, includeSource)
	}

	maxSymbols := a.MaxSymbols
	if maxSymbols <= 0 {
		maxSymbols = 3
	}

	if maxSymbols > 10 {
		maxSymbols = 10
	}

	depth := a.Depth
	if depth <= 0 {
		depth = 2
	}

	var matched []store.Symbol
	if sym, err := s.Store.Node(query); err == nil {
		matched = []store.Symbol{sym}
	} else {
		found, serr := s.Store.SearchSymbols(&store.SearchOptions{
			Query:     query,
			SortByFan: true,
			Limit:     maxSymbols,
		})
		if serr != nil {
			return nil, serr
		}

		matched = found
	}

	items := make([]map[string]any, 0, len(matched))

	for i := range matched {
		sym := matched[i]
		callers, _ := s.Store.Callers(sym.ID, 25)
		callees, _ := s.Store.Callees(sym.ID, 25)

		entry := map[string]any{
			"symbol":  sym,
			"callers": callers,
			"callees": callees,
		}

		if imp, ierr := s.Store.Impact(sym.ID, depth, 20); ierr == nil && imp != nil {
			entry["impact"] = map[string]any{
				"maxDepth":      imp.MaxDepth,
				"totalImpacted": imp.Total,
				"affectedFiles": imp.AffectedFiles,
				"entryPoints":   imp.EntryPoints,
				"nodes":         imp.Nodes,
			}
		}

		if includeSource && s.ReadSnippet != nil && sym.File != "" {
			if snippet, rerr := s.ReadSnippet(sym.File, int(sym.Line), 80); rerr == nil && snippet != "" {
				entry["source"] = snippet
			}
		}

		items = append(items, entry)
	}

	return map[string]any{
		"query":   query,
		"mode":    "codemap",
		"count":   len(items),
		"results": items,
	}, nil
}

func (s *Server) listIslands(a *toolArgs) (any, error) {
	found, err := s.Store.Islands(a.DetachedOnly, a.Limit)

	return wrap("islands", found), err
}

func (s *Server) listOrphans(a *toolArgs) (any, error) {
	found, err := s.Store.Orphans(a.Kind, a.Limit)

	return wrap("orphans", found), err
}

func (s *Server) getStats(_ *toolArgs) (any, error) {
	stats, err := s.Store.Stats()
	if err != nil {
		return nil, err
	}

	if s.SyncStatus == nil {
		return stats, nil
	}

	return map[string]any{
		"stats": stats,
		"watch": s.SyncStatus(),
	}, nil
}

func (s *Server) explainCall(a *toolArgs) (any, error) {
	if s.Explain == nil {
		return nil, errors.New("explain_call is not available: this server was started without a workspace resolver")
	}

	if a.File == "" || a.Line <= 0 {
		return nil, errMissing("file and a 1-based line")
	}

	text, err := s.Explain(a.File, a.Line, a.Match)
	if err != nil {
		return nil, err
	}

	return map[string]any{"explanation": text}, nil
}

func (s *Server) findUnresolved(a *toolArgs) (any, error) {
	if s.Unresolved == nil {
		return nil, errors.New("find_unresolved_calls is not available on this server")
	}

	if len(a.Paths) == 0 {
		return nil, errMissing("paths")
	}

	return s.Unresolved(a.Paths, a.GlobalDefs, limitOr(a.Limit))
}

func (s *Server) findReferences(a *toolArgs) (any, error) {
	if s.FindRefs == nil {
		return nil, errors.New("find_references is not available on this server")
	}

	if strings.TrimSpace(a.Target) == "" || len(a.Paths) == 0 {
		return nil, errMissing("target and paths")
	}

	return s.FindRefs(a.Target, a.Paths, limitOr(a.Limit))
}

func (s *Server) lintPaths(a *toolArgs) (any, error) {
	if s.Lint == nil {
		return nil, errors.New("lint is not available: start the server with --allow-lint to offer it")
	}

	if len(a.Paths) == 0 {
		return nil, errMissing("paths")
	}

	return s.Lint(a.Paths, limitOr(a.Limit))
}

func (s *Server) getOutline(a *toolArgs) (any, error) {
	if s.Outline == nil {
		return nil, errors.New("get_outline is not available on this server")
	}

	if strings.TrimSpace(a.File) == "" {
		return nil, errMissing("file")
	}

	return s.Outline(a.File)
}

func (s *Server) getTypeHierarchy(a *toolArgs) (any, error) {
	if s.TypeHierarchy == nil {
		return nil, errors.New("get_type_hierarchy is not available on this server")
	}

	if strings.TrimSpace(a.File) == "" {
		return nil, errMissing("file")
	}

	return s.TypeHierarchy(a.File)
}

func (s *Server) lookupDocs(a *toolArgs) (any, error) {
	if s.LookupDocs == nil {
		return nil, errors.New("lookup_docs is not available on this server")
	}

	if strings.TrimSpace(a.Name) == "" && strings.TrimSpace(a.Query) == "" {
		return nil, errMissing("name or query")
	}

	limit := a.Limit
	if limit <= 0 {
		limit = 25
	}

	return s.LookupDocs(a.Name, a.Query, a.Kind, limit)
}

func (s *Server) checkStructure(a *toolArgs) (any, error) {
	if s.Structure == nil {
		return nil, errors.New("check_structure is not available on this server")
	}

	if len(a.Paths) == 0 {
		return nil, errMissing("paths")
	}

	return s.Structure(a.Paths, limitOr(a.Limit))
}

func (s *Server) resolveRoute(a *toolArgs) (any, error) {
	if s.ResolveRoute == nil {
		return nil, errors.New("resolve_route is not available on this server")
	}

	if strings.TrimSpace(a.Route) == "" {
		return nil, errMissing("route")
	}

	return s.ResolveRoute(a.Route)
}

func (s *Server) checkFormat(a *toolArgs) (any, error) {
	if s.CheckFormat == nil {
		return nil, errors.New("check_format is not available on this server")
	}

	if len(a.Paths) == 0 {
		return nil, errMissing("paths")
	}

	limit := a.Limit
	if limit <= 0 {
		limit = 50
	}

	return s.CheckFormat(a.Paths, a.IncludeDiff, limit)
}

// wrap gives a list result a named field and a count. A bare JSON array tells a
// reader nothing about whether it was truncated; "count" next to the list does.
func wrap[T any](field string, items []T) map[string]any {
	if items == nil {
		items = []T{}
	}

	return map[string]any{field: items, "count": len(items)}
}

// defaultTaskLimit caps a task tool's list. A whole-workspace unresolved scan
// can find thousands, and every one of them lands in the model's context.
const defaultTaskLimit = 200

func limitOr(n int) int {
	if n <= 0 {
		return defaultTaskLimit
	}

	return n
}

// Limited is a task tool's list result: at most limit items, and how many
// there were in all, so a reader can tell a short answer from a cut one.
func Limited[T any](field string, items []T, limit int) map[string]any {
	total := len(items)
	if limit > 0 && total > limit {
		items = items[:limit]
	}

	out := wrap(field, items)
	out["total"] = total
	out["truncated"] = len(items) < total

	return out
}

func errMissing(what string) error {
	return fmt.Errorf("missing required argument: %s", strings.TrimSpace(what))
}
