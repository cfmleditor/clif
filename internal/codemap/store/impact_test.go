//go:build !wasip1

package store_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/codemap"
	"github.com/cfmleditor/clif/internal/codemap/store"
)

func TestStoreImpactMultiHop(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "impact.db")

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = st.Close() }()

	// Graph: entry.cfm -> mid.cfc::middle -> leaf.cfc::leaf
	// Plus cycle: leaf.cfc::leaf -> mid.cfc::middle (must not infinite-loop).
	m := &codemap.Map{
		Root: "/ws",
		Nodes: []codemap.Node{
			{ID: "entry.cfm", Kind: codemap.KindFile, Name: "entry.cfm", File: "entry.cfm", Entry: true, Reachable: true},
			{ID: "mid.cfc::middle", Kind: codemap.KindFunction, Name: "middle", File: "mid.cfc", Line: 5, Reachable: true},
			{ID: "leaf.cfc::leaf", Kind: codemap.KindFunction, Name: "leaf", File: "leaf.cfc", Line: 10, Reachable: true},
		},
		Edges: []codemap.Edge{
			{From: "entry.cfm", To: "mid.cfc::middle", Kind: codemap.EdgeCalls, Count: 1},
			{From: "mid.cfc::middle", To: "leaf.cfc::leaf", Kind: codemap.EdgeCalls, Count: 2},
			{From: "leaf.cfc::leaf", To: "mid.cfc::middle", Kind: codemap.EdgeCalls, Count: 1},
		},
	}

	if err := st.Save(m); err != nil {
		t.Fatal(err)
	}

	res, err := st.Impact("leaf.cfc::leaf", 4, 50)
	if err != nil {
		t.Fatalf("Impact error: %v", err)
	}

	if res.Total != 2 || len(res.Nodes) != 2 {
		t.Fatalf("got total=%d nodes=%+v, want 2", res.Total, res.Nodes)
	}

	if res.Nodes[0].ID != "mid.cfc::middle" || res.Nodes[0].Depth != 1 {
		t.Errorf("node[0] = %+v, want mid.cfc::middle at depth 1", res.Nodes[0])
	}

	if res.Nodes[1].ID != "entry.cfm" || res.Nodes[1].Depth != 2 {
		t.Errorf("node[1] = %+v, want entry.cfm at depth 2", res.Nodes[1])
	}

	if len(res.EntryPoints) != 1 || res.EntryPoints[0] != "entry.cfm" {
		t.Errorf("entryPoints = %v, want [entry.cfm]", res.EntryPoints)
	}

	if len(res.AffectedFiles) != 2 {
		t.Errorf("affectedFiles = %v, want 2 files", res.AffectedFiles)
	}

	// Depth=1 should only include direct caller.
	shallow, err := st.Impact("leaf.cfc::leaf", 1, 50)
	if err != nil {
		t.Fatal(err)
	}

	if shallow.Total != 1 {
		t.Errorf("depth=1 total = %d, want 1", shallow.Total)
	}

	// Unknown node returns ErrNotFound.
	if _, err := st.Impact("nope", 4, 50); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("unknown node err = %v, want ErrNotFound", err)
	}
}
