package index

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.lsp.dev/uri"
)

// TestExtendsForFileDistinguishesUnknownFromNone pins the tri-state, which is
// the whole reason ExtendsForFile returns a second value.
//
// Only one of the two doors into the index can fill this in for free: IndexFile
// parses, so it knows; IndexFileFromResult is handed funcs and refs by a caller
// that already parsed. Collapsing "nobody has looked" into "extends nothing"
// would stop the resolver walking the chain for every file indexed through the
// second door — which loses resolutions silently rather than slowly.
func TestExtendsForFileDistinguishesUnknownFromNone(t *testing.T) {
	t.Parallel()

	idx := New()
	u := uri.File("/ws/Never.cfc")

	if _, ok := idx.ExtendsForFile(u); ok {
		t.Error("a file nobody has indexed reports a known extends")
	}

	// A component that extends nothing: known, and empty.
	base := uri.File("/ws/Base.cfc")
	idx.IndexFile(base, "component {\n\tfunction go() {}\n}\n")

	ext, ok := idx.ExtendsForFile(base)
	if !ok {
		t.Error("IndexFile did not record extends")
	}

	if ext != "" {
		t.Errorf("extends = %q, want empty", ext)
	}

	// A component that extends something.
	child := uri.File("/ws/Child.cfc")
	idx.IndexFile(child, "component extends=\"Base\" {\n}\n")

	if ext, ok := idx.ExtendsForFile(child); !ok || ext != "Base" {
		t.Errorf("ExtendsForFile = (%q, %v), want (\"Base\", true)", ext, ok)
	}
}

// TestExtendsIsForgottenWithTheRestOfTheFile is what removes the need for any
// invalidation logic of its own. The per-file views all have to be cleared
// together; an extends record left behind by removeFileEntries would outlive
// the parse it came from and answer for a file that no longer says that.
func TestExtendsIsForgottenWithTheRestOfTheFile(t *testing.T) {
	t.Parallel()

	idx := New()
	u := uri.File("/ws/Child.cfc")

	idx.IndexFile(u, "component extends=\"Base\" {\n}\n")

	if _, ok := idx.ExtendsForFile(u); !ok {
		t.Fatal("extends not recorded")
	}

	// The other door: re-indexed from a parse the caller already had, which
	// cannot supply extends. The record must be gone, not stale.
	idx.IndexFileFromResult(u, nil, nil)

	if ext, ok := idx.ExtendsForFile(u); ok {
		t.Errorf("extends survived a re-index as (%q, true); it would answer for a parse that no longer exists", ext)
	}
}

// ExtendersByName is FilesExtendingName for every name at once.
func TestExtendersByNameAnswersAsFilesExtendingNameDoes(t *testing.T) {
	idx := New()

	files := map[string]string{
		"file:///w/A.cfc":     "component extends=\"x.Base\" {}",
		"file:///w/B.cfc":     "component extends=\"base\" {}",
		"file:///w/C.cfc":     "component extends=\"y.z.BASE\" {}",
		"file:///w/D.cfc":     "component extends=\"Other\" {}",
		"file:///w/E.cfc":     "component {}",
		"file:///w/sub/F.cfc": "component extends=\"a.Other\" {}",
	}

	for u, content := range files {
		idx.IndexFile(uri.URI(u), content)
	}

	got := idx.ExtendersByName()
	if len(got) != 2 {
		t.Fatalf("groups = %v, want base and other", got)
	}

	for _, name := range []string{"base", "Base", "OTHER", "other", "missing"} {
		var uris []string
		for _, e := range got[strings.ToLower(name)] {
			uris = append(uris, e.URI)
		}

		if want := idx.FilesExtendingName(name); !slices.Equal(uris, want) {
			t.Errorf("ExtendersByName[%q] = %v, FilesExtendingName says %v", name, uris, want)
		}
	}

	if len(got["base"]) != 3 {
		t.Errorf("base is extended by %v, want three files", got["base"])
	}
}

// The grouping is kept until the index changes, and then read again.
func TestExtendersByNameFollowsTheIndex(t *testing.T) {
	idx := New()
	idx.IndexFile(uri.URI("file:///w/A.cfc"), "component extends=\"Base\" {}")

	first := idx.ExtendersByName()
	if len(first["base"]) != 1 {
		t.Fatalf("base is extended by %v, want one file", first["base"])
	}

	if again := idx.ExtendersByName(); len(again) != len(first) || len(again["base"]) != 1 {
		t.Errorf("a second ask changed the answer: %v", again)
	}

	idx.IndexFile(uri.URI("file:///w/B.cfc"), "component extends=\"Base\" {}")

	if got := idx.ExtendersByName(); len(got["base"]) != 2 {
		t.Errorf("after indexing another extender, base is extended by %v, want two files", got["base"])
	}
}

// uriKey's memory answers as the computation does, for a URI asked for again and
// for one asked for first, and stays bounded.
func TestURIKeyMemoryAnswersAsTheComputationDoes(t *testing.T) {
	inputs := []string{
		"file:///My%20Documents/Sub/A.cfc", "file:///My Documents/Sub/A.cfc", "file:///a/B.CFC", "file:///a/b.cfc",
		"file:///%zz/bad", "", "untitled:Untitled-1", "file:///C%3A/Users/X/y.cfc",
	}

	for range 2 {
		for _, in := range inputs {
			if got, want := uriKey(uri.URI(in)), uriKeyUncached(in); got != want {
				t.Errorf("uriKey(%q) = %q, want %q", in, got, want)
			}
		}
	}

	for i := range uriKeyShards * maxURIKeyShard * 2 {
		_ = uriKey(uri.URI("file:///bulk/File" + strconv.Itoa(i) + ".cfc"))
	}

	for i := range uriKeys.shards {
		sh := &uriKeys.shards[i]

		sh.mu.RLock()
		n := len(sh.m)
		sh.mu.RUnlock()

		if n > maxURIKeyShard {
			t.Fatalf("shard %d holds %d keys, over the %d it is held to", i, n, maxURIKeyShard)
		}
	}
}
