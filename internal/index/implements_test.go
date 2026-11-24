package index

import (
	"testing"

	"go.lsp.dev/uri"
)

func TestImplementersByNameFollowsTheFile(t *testing.T) {
	idx := New()
	u := uri.URI("file:///w/Square.cfc")

	idx.IndexFile(u, "component implements=\"shapes.IShape, IDraw\" {\n}\n")

	got := idx.ImplementersByName()
	if len(got["ishape"]) != 1 || len(got["idraw"]) != 1 || got["ishape"][0].URI != string(u) {
		t.Fatalf("ImplementersByName = %+v", got)
	}

	// A re-index that no longer implements forgets the record.
	idx.IndexFile(u, "component {\n}\n")

	if got := idx.ImplementersByName(); len(got) != 0 {
		t.Errorf("after re-index = %+v, want none", got)
	}

	// SetImplements restores it after IndexFileFromResult, and RemoveFile drops it.
	idx.SetImplements(u, []string{"IShape"})

	if len(idx.ImplementersByName()["ishape"]) != 1 {
		t.Error("SetImplements did not record")
	}

	idx.RemoveFile(u)

	if got := idx.ImplementersByName(); len(got) != 0 {
		t.Errorf("after RemoveFile = %+v, want none", got)
	}
}
