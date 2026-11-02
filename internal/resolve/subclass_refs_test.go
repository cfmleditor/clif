package resolve

import (
	"path/filepath"
	"strings"
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
// mappings, so the leaves it finds are filed under them, apart from the
// owner's, which resolves by directory.
func TestLeafSubclassesAreKeptApartByMappings(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"h/Base.cfc": `component { function run(){ variables.svc.go(); } }`,
		"h/A.cfc":    `component extends="Base" {}`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "h/Base.cfc")

	view := r.forCaller(dir)
	if view == r {
		t.Skip("the resolver served itself, so there is no view to keep apart")
	}

	base := filepath.Join(dir, "h", "Base.cfc")
	if leaves := view.leafSubclasses(base); len(leaves) != 1 {
		t.Fatalf("leaves = %v, want A", leaves)
	}

	if leaves := r.leafSubclasses(base); len(leaves) != 1 {
		t.Fatalf("leaves = %v, want A", leaves)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	views, owners := 0, 0

	for key := range r.leafSubs {
		switch {
		case strings.HasPrefix(key, "view:"+view.mappingsKey+"\x00"):
			views++
		case strings.HasPrefix(key, "owner\x00"):
			owners++
		}
	}

	if views != 1 || owners != 1 {
		t.Fatalf("%d entries under the view's mappings and %d under the owner's, want one each: %q", views, owners, r.leafSubs)
	}
}
