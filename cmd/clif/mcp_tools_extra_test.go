package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func mustMap(t *testing.T, v any) map[string]any {
	t.Helper()

	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T (%#v)", v, v)
	}

	return m
}

func TestMCPOutline(t *testing.T) {
	root := t.TempDir()
	cfc := filepath.Join(root, "User.cfc")

	content := strings.Join([]string{
		`component extends="BaseEntity" accessors="true" {`,
		`	property name="email" type="string";`,
		`	public string function getName(required numeric id, boolean upper=false) {`,
		`		return "";`,
		`	}`,
		`}`,
	}, "\n")

	if err := os.WriteFile(cfc, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := mcpOutline(root, cfc)
	if err != nil {
		t.Fatalf("mcpOutline error: %v", err)
	}

	m := mustMap(t, res)
	if m["extends"] != "BaseEntity" {
		t.Errorf("extends = %v, want BaseEntity", m["extends"])
	}

	if acc, _ := m["accessors"].(bool); !acc {
		t.Errorf("accessors = %v, want true", m["accessors"])
	}

	props, ok := m["properties"].([]outlineProp)
	if !ok || len(props) != 1 || props[0].Name != "email" || props[0].Line != 2 {
		t.Errorf("properties = %+v", m["properties"])
	}

	funcs, ok := m["functions"].([]outlineFunc)
	if !ok {
		t.Fatalf("functions = %T", m["functions"])
	}

	foundGetName := false

	for _, fn := range funcs {
		if fn.Name == "getName" {
			foundGetName = true

			if fn.Line != 3 || fn.ReturnType != "string" || len(fn.Arguments) != 2 {
				t.Errorf("getName = %+v", fn)
			}
		}
	}

	if !foundGetName {
		t.Errorf("getName not found in %+v", funcs)
	}
}

func TestMCPTypeHierarchyAndWorkspaceRefresh(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "Base.cfc")
	child := filepath.Join(root, "Child.cfc")

	if err := os.WriteFile(base, []byte("component {\n\tfunction rootMethod() {}\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(child, []byte("component extends=\"Base\" {\n\tfunction childMethod() {}\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ws := newMCPWorkspace(root)

	res, err := mcpTypeHierarchy(ws, root, child)
	if err != nil {
		t.Fatalf("mcpTypeHierarchy(child) error: %v", err)
	}

	m := mustMap(t, res)

	supers, ok := m["supertypes"].([]hierarchyNode)
	if !ok || len(supers) != 1 || supers[0].File != base {
		t.Fatalf("supertypes = %+v, want [%s]", m["supertypes"], base)
	}

	// Add a new sibling subtype on disk AFTER initial indexing — workspace cache
	// must pick it up incrementally on the next call.
	sibling := filepath.Join(root, "Sibling.cfc")
	if err := os.WriteFile(sibling, []byte("component extends=\"Base\" {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(sibling, future, future)

	baseRes, err := mcpTypeHierarchy(ws, root, base)
	if err != nil {
		t.Fatalf("mcpTypeHierarchy(base) error: %v", err)
	}

	bm := mustMap(t, baseRes)

	subs, ok := bm["subtypes"].([]hierarchyNode)
	if !ok || len(subs) != 2 {
		t.Fatalf("subtypes of Base after adding Sibling = %+v, want 2", bm["subtypes"])
	}
}

func TestMCPLookupDocs(t *testing.T) {
	// Exact function lookup.
	fnRes, err := mcpLookupDocs("arrayMap", "", "", 10)
	if err != nil {
		t.Fatal(err)
	}

	if found, _ := mustMap(t, fnRes)["found"].(bool); !found {
		t.Fatalf("expected arrayMap to be found, got %+v", fnRes)
	}

	// Exact tag lookup.
	tagRes, err := mcpLookupDocs("cfquery", "", "", 10)
	if err != nil {
		t.Fatal(err)
	}

	if found, _ := mustMap(t, tagRes)["found"].(bool); !found {
		t.Fatalf("expected cfquery to be found, got %+v", tagRes)
	}

	// Exact type.member lookup.
	memRes, err := mcpLookupDocs("array.map", "", "", 10)
	if err != nil {
		t.Fatal(err)
	}

	if found, _ := mustMap(t, memRes)["found"].(bool); !found {
		t.Fatalf("expected array.map to be found, got %+v", memRes)
	}

	// Substring search.
	searchRes, err := mcpLookupDocs("", "array", "function", 5)
	if err != nil {
		t.Fatal(err)
	}

	sm := mustMap(t, searchRes)

	cnt, ok := sm["count"].(int)
	if !ok || cnt == 0 {
		t.Fatalf("expected non-empty search results for 'array', got %+v", sm)
	}
}

func TestMCPCheckStructure(t *testing.T) {
	root := t.TempDir()
	bad := filepath.Join(root, "bad.cfc")

	content := strings.Join([]string{
		`<cfcomponent>`,
		`	<cffunction name="dup">`,
		`		<cfargument name="id">`,
		`		<cfargument name="id">`,
		`		<cfif true>`,
		`	</cffunction>`,
		`</cfcomponent>`,
	}, "\n")

	if err := os.WriteFile(bad, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := mcpCheckStructure(root, []string{bad}, 50)
	if err != nil {
		t.Fatalf("mcpCheckStructure error: %v", err)
	}

	m := mustMap(t, res)

	probs, ok := m["problems"].([]structureProblem)
	if !ok || len(probs) < 2 {
		t.Fatalf("expected at least 2 structure problems (unclosed cfif + duplicate arg), got %+v", m["problems"])
	}
}

func TestMCPResolveRouteUnconfigured(t *testing.T) {
	root := t.TempDir()
	ws := newMCPWorkspace(root)

	res, err := mcpResolveRoute(ws, root, "admin.users.edit")
	if err != nil {
		t.Fatalf("mcpResolveRoute error: %v", err)
	}

	m := mustMap(t, res)
	if configured, ok := m["configured"].(bool); !ok || configured {
		t.Fatalf("expected configured=false without .clif.json routes, got %+v", m)
	}
}

func TestMCPCheckFormatReadOnly(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "ugly.cfm")
	orig := "<CFSET   x=1>\n"

	if err := os.WriteFile(file, []byte(orig), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := mcpCheckFormat(root, []string{file}, true, 10)
	if err != nil {
		t.Fatalf("mcpCheckFormat error: %v", err)
	}

	m := mustMap(t, res)
	if m["needsFormatting"] != 1 {
		t.Fatalf("needsFormatting = %v, want 1 (%+v)", m["needsFormatting"], m)
	}

	entries, ok := m["files"].([]formatCheckEntry)
	if !ok || len(entries) != 1 || entries[0].Diff == "" {
		t.Fatalf("expected non-empty diff preview, got %+v", m["files"])
	}

	// Verify read-only: file on disk must be untouched.
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	if string(after) != orig {
		t.Fatalf("check_format modified file on disk: got %q, want %q", string(after), orig)
	}
}

func TestMCPLiveExploreAndSnippet(t *testing.T) {
	root := t.TempDir()
	svc := filepath.Join(root, "Svc.cfc")
	content := strings.Join([]string{
		`component {`,
		`	public string function greet(required string who) {`,
		`		return "hi " & who;`,
		`	}`,
		`}`,
	}, "\n")

	if err := os.WriteFile(svc, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	ws := newMCPWorkspace(root)

	// Explore by file name.
	fileRes, err := mcpLiveExplore(ws, root, "Svc.cfc", true)
	if err != nil {
		t.Fatalf("mcpLiveExplore(file) error: %v", err)
	}

	fm := mustMap(t, fileRes)
	if fm["mode"] != "live-file" {
		t.Errorf("mode = %v, want live-file", fm["mode"])
	}

	if src, _ := fm["source"].(string); !strings.Contains(src, "   2: \tpublic string function greet") {
		t.Errorf("unexpected file snippet: %q", src)
	}

	// Explore by symbol name.
	symRes, err := mcpLiveExplore(ws, root, "greet", true)
	if err != nil {
		t.Fatalf("mcpLiveExplore(symbol) error: %v", err)
	}

	sm := mustMap(t, symRes)
	if sm["mode"] != "live-symbol" {
		t.Errorf("mode = %v, want live-symbol", sm["mode"])
	}

	defs, ok := sm["definitions"].([]map[string]any)
	if !ok || len(defs) != 1 || defs[0]["name"] != "greet" {
		t.Fatalf("definitions = %+v, want 1 greet definition", sm["definitions"])
	}
}

func TestMCPAllowedToolsFlagAndEnv(t *testing.T) {
	t.Setenv("CLIF_MCP_TOOLS", "explore,lookup_docs")

	gotEnv := mcpAllowedTools("")
	if len(gotEnv) != 2 || gotEnv[0] != "explore" || gotEnv[1] != "lookup_docs" {
		t.Fatalf("mcpAllowedTools(env) = %v", gotEnv)
	}

	gotSlim := mcpAllowedTools("slim")
	if len(gotSlim) != 3 || gotSlim[0] != "explore" {
		t.Fatalf("mcpAllowedTools(slim) = %v", gotSlim)
	}
}
