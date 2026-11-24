package main

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/codemap/mcp"
	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
	"go.lsp.dev/uri"
)

type outlineArg struct {
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type outlineFunc struct {
	Name       string       `json:"name"`
	Line       uint32       `json:"line"`
	ReturnType string       `json:"returnType,omitempty"`
	Arguments  []outlineArg `json:"arguments,omitempty"`
	Generated  bool         `json:"generated,omitempty"`
}

type outlineProp struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
	Line uint32 `json:"line"`
}

func mcpOutline(root, file string) (any, error) {
	absList, err := absPaths(root, []string{file})
	if err != nil {
		return nil, err
	}

	abs := absList[0]

	data, err := vfs.ReadRegular(abs)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", file, err)
	}

	pr := parser.Parse(uri.File(abs), string(data))

	props := make([]outlineProp, 0, len(pr.PropertySymbols()))
	for _, p := range pr.PropertySymbols() {
		props = append(props, outlineProp{
			Name: p.Name,
			Type: p.Type,
			Line: p.Line + 1,
		})
	}

	funcs := make([]outlineFunc, 0, len(pr.Funcs))

	for i := range pr.Funcs {
		fn := &pr.Funcs[i]

		args := make([]outlineArg, 0, len(fn.Arguments))
		for _, a := range fn.Arguments {
			args = append(args, outlineArg{
				Name:     a.Name,
				Type:     a.Type,
				Required: a.Required,
			})
		}

		funcs = append(funcs, outlineFunc{
			Name:       fn.Name,
			Line:       fn.Line + 1,
			ReturnType: fn.ReturnType,
			Arguments:  args,
			Generated:  fn.Generated,
		})
	}

	out := map[string]any{
		"file":       abs,
		"properties": props,
		"functions":  funcs,
	}

	if pr.Extends != "" {
		out["extends"] = pr.Extends
	}

	if pr.Accessors {
		out["accessors"] = true
	}

	if pr.Persistent {
		out["persistent"] = true
	}

	return out, nil
}

type hierarchyNode struct {
	Name    string `json:"name"`
	File    string `json:"file,omitempty"`
	Extends string `json:"extends,omitempty"`
}

func mcpTypeHierarchy(ws *mcpWorkspace, root, file string) (any, error) {
	absList, err := absPaths(root, []string{file})
	if err != nil {
		return nil, err
	}

	abs := absList[0]

	env, _, err := ws.get()
	if err != nil {
		return nil, err
	}

	data, err := vfs.ReadRegular(abs)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", file, err)
	}

	pr := parser.Parse(uri.File(abs), string(data))
	compName := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))

	// Walk supertype chain with cycle guard.
	supertypes := make([]hierarchyNode, 0, 4)
	visited := map[string]bool{strings.ToLower(abs): true}
	curDir := filepath.Dir(abs)
	curExtends := pr.Extends

	for curExtends != "" {
		parentPath := env.resolver.ComponentPath(curExtends, curDir)
		if parentPath == "" {
			supertypes = append(supertypes, hierarchyNode{Name: curExtends})

			break
		}

		key := strings.ToLower(parentPath)
		if visited[key] {
			break
		}

		visited[key] = true

		parentExtends, _ := env.resolver.Index.ExtendsForFile(uri.File(parentPath))
		if parentExtends == "" {
			if pdata, rerr := vfs.ReadRegular(parentPath); rerr == nil {
				parentExtends = parser.Parse(uri.File(parentPath), string(pdata)).Extends
			}
		}

		supertypes = append(supertypes, hierarchyNode{
			Name:    curExtends,
			File:    parentPath,
			Extends: parentExtends,
		})

		curDir = filepath.Dir(parentPath)
		curExtends = parentExtends
	}

	// Find direct subtypes using the index's ExtendersByName map.
	extenders := env.resolver.Index.ExtendersByName()
	candidates := extenders[strings.ToLower(compName)]
	subtypes := make([]hierarchyNode, 0, len(candidates))

	for _, c := range candidates {
		childPath := cfpath.FromURI(c.URI)
		if strings.EqualFold(childPath, abs) {
			continue
		}

		// Verify the child's extends actually resolves to this file when possible.
		resolvedParent := env.resolver.ComponentPath(c.Extends, filepath.Dir(childPath))
		if resolvedParent != "" && !strings.EqualFold(resolvedParent, abs) {
			continue
		}

		subtypes = append(subtypes, hierarchyNode{
			Name:    strings.TrimSuffix(filepath.Base(childPath), filepath.Ext(childPath)),
			File:    childPath,
			Extends: c.Extends,
		})
	}

	slices.SortFunc(subtypes, func(a, b hierarchyNode) int {
		return strings.Compare(a.File, b.File)
	})

	return map[string]any{
		"file":       abs,
		"name":       compName,
		"extends":    pr.Extends,
		"supertypes": supertypes,
		"subtypes":   subtypes,
	}, nil
}

type docSearchHit struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description,omitempty"`
	Syntax      string `json:"syntax,omitempty"`
}

func mcpLookupDocs(name, query, kind string, limit int) (any, error) {
	name = strings.TrimSpace(name)
	query = strings.TrimSpace(query)
	kind = strings.ToLower(strings.TrimSpace(kind))

	if name != "" {
		if entry, ok := lookupExactDoc(name); ok {
			return map[string]any{"found": true, "entry": entry}, nil
		}

		return map[string]any{"found": false, "name": name}, nil
	}

	q := strings.ToLower(query)

	var hits []docSearchHit

	if kind == "" || kind == "function" {
		for _, e := range docs.AllFunctions() {
			if docMatches(e.Name, e.Description, q) {
				hits = append(hits, docSearchHit{Name: e.Name, Kind: "function", Description: e.Description, Syntax: e.Syntax})
			}
		}
	}

	if kind == "" || kind == "tag" {
		for _, e := range docs.AllTags() {
			if docMatches(e.Name, e.Description, q) {
				hits = append(hits, docSearchHit{Name: e.Name, Kind: "tag", Description: e.Description, Syntax: e.Syntax})
			}
		}
	}

	if kind == "" || kind == "member" {
		for _, m := range docs.AllMemberFunctions() {
			desc := ""
			syntax := ""

			if m.Entry != nil {
				desc = m.Entry.Description
				syntax = m.Entry.Member
			}

			if docMatches(m.Name, desc, q) {
				hits = append(hits, docSearchHit{Name: m.Name, Kind: "member", Description: desc, Syntax: syntax})
			}
		}
	}

	slices.SortFunc(hits, func(a, b docSearchHit) int {
		aExact := strings.EqualFold(a.Name, query)
		bExact := strings.EqualFold(b.Name, query)

		if aExact != bExact {
			if aExact {
				return -1
			}

			return 1
		}

		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return mcp.Limited("results", hits, limit), nil
}

func lookupExactDoc(name string) (*docs.Entry, bool) {
	if e, ok := docs.LookupFunction(name); ok {
		return e, true
	}

	if e, ok := docs.LookupTag(name); ok {
		return e, true
	}

	// Support "type.member" (e.g. "array.map" or "struct.each").
	if typ, mem, ok := strings.Cut(name, "."); ok && typ != "" && mem != "" {
		for _, mf := range docs.MemberFunctionsForType(strings.ToLower(typ)) {
			if strings.EqualFold(mf.Name, mem) && mf.Entry != nil {
				return mf.Entry, true
			}
		}
	}

	return nil, false
}

func docMatches(name, desc, q string) bool {
	return strings.Contains(strings.ToLower(name), q) || strings.Contains(strings.ToLower(desc), q)
}

type structureProblem struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

func mcpCheckStructure(root string, paths []string, limit int) (any, error) {
	abs, err := absPaths(root, paths)
	if err != nil {
		return nil, err
	}

	files := cfpath.CollectCFML(vfs.OS{}, abs)
	if len(files) == 0 {
		return nil, errors.New("no CFML files under the paths given")
	}

	var problems []structureProblem

	for _, f := range files {
		data, rerr := vfs.ReadRegular(f)
		if rerr != nil {
			continue
		}

		content := string(data)
		lines := parser.NewLineTable(content)

		for _, p := range parser.TagProblems(content) {
			line := lines.LineOf(p.Start)
			col := p.Start - int(lines[line]) + 1
			sev := "error"

			if p.Warning {
				sev = "warning"
			}

			problems = append(problems, structureProblem{
				File:     f,
				Line:     line + 1,
				Column:   col,
				Severity: sev,
				Code:     string(p.Kind),
				Message:  p.Message,
			})
		}

		spans := parser.FunctionSpansOf(f, content)
		for _, d := range parser.DuplicateArguments(content, spans) {
			line := lines.LineOf(d.Start)
			col := d.Start - int(lines[line]) + 1
			problems = append(problems, structureProblem{
				File:     f,
				Line:     line + 1,
				Column:   col,
				Severity: "error",
				Code:     "duplicate-argument",
				Message:  "Duplicate argument '" + d.Name + "' in function '" + d.Function + "'",
			})
		}
	}

	out := mcp.Limited("problems", problems, limit)
	out["filesChecked"] = len(files)

	return out, nil
}

type routeTargetResult struct {
	Kind      string `json:"kind"`
	Component string `json:"component,omitempty"`
	Method    string `json:"method,omitempty"`
	File      string `json:"file,omitempty"`
	Alias     string `json:"alias,omitempty"`
}

func mcpResolveRoute(ws *mcpWorkspace, root, routeStr string) (any, error) {
	env, cfg, err := ws.get()
	if err != nil {
		return nil, err
	}

	if cfg == nil {
		return map[string]any{
			"route":      routeStr,
			"configured": false,
			"targets":    []routeTargetResult{},
			"note":       "No routes block is configured in .clif.json for " + root + ".",
		}, nil
	}

	rr := routeResolver(vfs.OS{}, env.resolver, cfg)
	if rr == nil || !rr.Config.Enabled() {
		return map[string]any{
			"route":      routeStr,
			"configured": false,
			"targets":    []routeTargetResult{},
			"note":       "No routes block is configured in .clif.json for " + root + ".",
		}, nil
	}

	rawTargets := rr.Resolve(strings.TrimSpace(routeStr))
	targets := make([]routeTargetResult, 0, len(rawTargets))

	for i := range rawTargets {
		t := &rawTargets[i]
		targets = append(targets, routeTargetResult{
			Kind:      string(t.Kind),
			Component: t.Component,
			Method:    t.Method,
			File:      t.Path,
			Alias:     t.Alias,
		})
	}

	return map[string]any{
		"route":      routeStr,
		"configured": true,
		"targets":    targets,
		"count":      len(targets),
	}, nil
}

type formatCheckEntry struct {
	File      string `json:"file"`
	Formatted bool   `json:"formatted"`
	Error     string `json:"error,omitempty"`
	Diff      string `json:"diff,omitempty"`
}

func mcpCheckFormat(root string, paths []string, includeDiff bool, limit int) (any, error) {
	abs, err := absPaths(root, paths)
	if err != nil {
		return nil, err
	}

	files := cfpath.CollectCFML(vfs.OS{}, abs)
	if len(files) == 0 {
		return nil, errors.New("no CFML files under the paths given")
	}

	if len(files) == 1 {
		includeDiff = true
	}

	optsFor := formatOptionsFor(root, false)

	var (
		changed []formatCheckEntry
		errored int
	)

	for _, f := range files {
		opts := optsFor(f)

		orig, formatted, ferr := formatFile(f, &opts)
		if ferr != nil {
			errored++

			changed = append(changed, formatCheckEntry{
				File:      f,
				Formatted: false,
				Error:     ferr.Error(),
			})

			continue
		}

		if bytes.Equal(orig, formatted) {
			continue
		}

		entry := formatCheckEntry{
			File:      f,
			Formatted: false,
		}

		if includeDiff {
			entry.Diff = simpleLineDiff(string(orig), string(formatted), 60)
		}

		changed = append(changed, entry)
	}

	out := mcp.Limited("files", changed, limit)
	out["filesChecked"] = len(files)
	out["needsFormatting"] = len(changed) - errored
	out["errors"] = errored

	return out, nil
}

// simpleLineDiff produces a compact line-by-line preview of the first maxLines
// differing lines between before and after without allocating a full Myers matrix.
func simpleLineDiff(before, after string, maxLines int) string {
	a := strings.Split(before, "\n")
	b := strings.Split(after, "\n")
	n := max(len(a), len(b))

	var sb strings.Builder

	emitted := 0

	for i := range n {
		var la, lb string
		if i < len(a) {
			la = a[i]
		}

		if i < len(b) {
			lb = b[i]
		}

		if la == lb {
			continue
		}

		if emitted >= maxLines {
			sb.WriteString("... (diff truncated)\n")

			break
		}

		fmt.Fprintf(&sb, "L%d - %s\nL%d + %s\n", i+1, la, i+1, lb)

		emitted++
	}

	return sb.String()
}
