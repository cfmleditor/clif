package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// TestABaseComponentsVariableIsWhatItsSubclassesHold: an abstract component
// calls variables.svc and never assigns it; the receiver is typed by what every
// concrete subclass holds, as alternatives. A subclass that does not type it,
// or no subclass at all, leaves it untyped.
func TestABaseComponentsVariableIsWhatItsSubclassesHold(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"models/PageService.cfc":  `component { function pageOnly(){} function shared(){} }`,
		"models/EntryService.cfc": `component { function entryOnly(){} function shared(){} }`,
		"h/Base.cfc": `component {
function run(){
	variables.svc.pageOnly();
	variables.svc.entryOnly();
	svc.shared();
	variables.svc.nope();
}
function shadowed(){
	var svc = getThing();
	svc.pageOnly();
}
}`,
		"h/A.cfc": `component extends="Base" { function init(){ variables.svc = new models.PageService(); return this; } }`,
		"h/B.cfc": `component extends="Base" { function init(){ variables.svc = new models.EntryService(); return this; } }`,

		"h/Partial.cfc": `component { function run(){ variables.svc.pageOnly(); } }`,
		"h/PA.cfc":      `component extends="Partial" { function init(){ variables.svc = new models.PageService(); return this; } }`,
		"h/PB.cfc":      `component extends="Partial" { function other(){} }`,

		"h/Alone.cfc": `component { function run(){ variables.svc.pageOnly(); } }`,

		"h/Top.cfc":   `component { function run(){ variables.svc.pageOnly(); } }`,
		"h/Mid.cfc":   `component extends="Top" { function init(){ variables.svc = new models.PageService(); return this; } }`,
		"h/Leaf1.cfc": `component extends="Mid" { function one(){} }`,
		"h/Leaf2.cfc": `component extends="Mid" { function two(){} }`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "h/Base.cfc"), map[string]string{
		"variables.svc.pageOnly":  "",
		"variables.svc.entryOnly": "",
		"svc.shared":              "",
		"variables.svc.nope":      "method 'nope' not found in models.EntryService|models.PageService",
		"svc.pageOnly":            "variable 'svc' has no component ref",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "h/Partial.cfc"), map[string]string{
		"variables.svc.pageOnly": "variable 'variables.svc' has no component ref",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "h/Alone.cfc"), map[string]string{
		"variables.svc.pageOnly": "variable 'variables.svc' has no component ref",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "h/Top.cfc"), map[string]string{
		"variables.svc.pageOnly": "",
	})
}

// TestLeafSubclassesIsCachedButNotShared: the leaves of a base are memoised,
// since every unanswered bare call in the base asks for them (tassweb's scan
// took twice as long without it). A caller narrowing the list in place must
// not narrow what the next caller gets, and a subclass indexed later is seen.
func TestLeafSubclassesIsCachedButNotShared(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"h/Base.cfc": `component { function run(){ variables.svc.go(); } }`,
		"h/A.cfc":    `component extends="Base" {}`,
		"h/B.cfc":    `component extends="Base" {}`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "h/Base.cfc")

	base := filepath.Join(dir, "h", "Base.cfc")

	first := r.leafSubclasses(base)
	if len(first) != 2 {
		t.Fatalf("leaves = %v, want A and B", first)
	}

	first[0] = "changed"

	if again := r.leafSubclasses(base); len(again) != 2 || again[0] == "changed" {
		t.Fatalf("a caller's edit reached the cache: %v", again)
	}

	c := filepath.Join(dir, "h", "C.cfc")
	src := `component extends="Base" {}`
	r.Index.IndexParsed(cfpath.ToURI(c), src, parser.Parse(cfpath.ToURI(c), src))

	if later := r.leafSubclasses(base); len(later) != 3 {
		t.Fatalf("a subclass indexed later is missing: %v", later)
	}
}

// A caller-context view resolves extends with its own application's
// mappings, so the leaves it finds are filed under them: views whose
// mappings agree share an answer, views whose mappings differ do not, and
// the owner, which resolves by directory, keeps its own.
func TestLeafSubclassesAreSharedByViewsThatMapAlike(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"h/Base.cfc":        `component { function run(){ variables.svc.go(); } }`,
		"h/A.cfc":           `component extends="Base" {}`,
		"a/Application.cfc": `component { this.name = "a"; }`,
		"b/Application.cfc": `component { this.name = "b"; }`,
		"c/Application.cfc": `component { this.name = "c"; this.mappings["/lib"] = getDirectoryFromPath(getCurrentTemplatePath()) & "lib/"; }`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "h/Base.cfc")

	va, vb, vc := r.forCaller(filepath.Join(dir, "a")), r.forCaller(filepath.Join(dir, "b")), r.forCaller(filepath.Join(dir, "c"))
	if va == r || vb == r || vc == r {
		t.Skip("the resolver served itself, so there are no views to keep apart")
	}

	if va == vb {
		t.Fatal("two applications were given one view; the test needs two")
	}

	if va.mappingsKey != vb.mappingsKey || va.mappingsKey == vc.mappingsKey {
		t.Fatalf("mappings keys a=%q b=%q c=%q: a and b should agree and c differ", va.mappingsKey, vb.mappingsKey, vc.mappingsKey)
	}

	base := filepath.Join(dir, "h", "Base.cfc")

	for _, res := range []*Resolver{va, vb, vc, r} {
		if leaves := res.leafSubclasses(base); len(leaves) != 1 {
			t.Fatalf("leaves = %v, want A", leaves)
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for scope, want := range map[string]int{"view:" + va.mappingsKey: 1, "view:" + vc.mappingsKey: 1, "owner": 1} {
		if got := len(r.leafSubs[scope]); got != want {
			t.Errorf("scope %q holds %d answers, want %d", scope, got, want)
		}
	}

	if len(r.leafSubs) != 3 {
		t.Errorf("%d scopes, want 3 (a and b sharing one): %q", len(r.leafSubs), r.leafSubs)
	}
}
