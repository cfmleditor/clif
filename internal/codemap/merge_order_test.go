package codemap

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"
)

// permutations returns every ordering of graphs.
func permutations(graphs []*FileGraph) [][]*FileGraph {
	if len(graphs) <= 1 {
		return [][]*FileGraph{slices.Clone(graphs)}
	}

	var out [][]*FileGraph

	for i := range graphs {
		rest := slices.Concat(graphs[:i:i], graphs[i+1:])
		for _, p := range permutations(rest) {
			out = append(out, append([]*FileGraph{graphs[i]}, p...))
		}
	}

	return out
}

// mergedIn merges graphs in the order given and returns the finished map.
func mergedIn(order []*FileGraph) *Map {
	m := &Map{}
	c := newCollector(m)

	for _, g := range order {
		c.merge(g)
	}

	c.finish()

	return m
}

// TestMergeOrderDoesNotChangeTheMap: an external node's id is its lowercased
// label, so `expandPath` and `Expandpath` are one node, and the report named it
// after whichever file was merged first. Files finish in no fixed order, so the
// same workspace printed either name from run to run. Every ordering of three
// placeholders, two of which agree on the name and differ elsewhere, must give
// one map — nodes and the compact form the viewer and the store read.
func TestMergeOrderDoesNotChangeTheMap(t *testing.T) {
	id := ExternalID("expandPath")
	graphs := []*FileGraph{
		{Provisional: []Node{{ID: id, Kind: KindExternal, Name: "expandPath"}}},
		{Provisional: []Node{{ID: id, Kind: KindExternal, Name: "Expandpath"}}},
		{Provisional: []Node{{ID: id, Kind: KindExternal, Name: "Expandpath", File: "b.cfc", Line: 4}}},
	}

	var first []byte

	for _, order := range permutations(graphs) {
		m := mergedIn(order)

		if len(m.Nodes) != 1 {
			t.Fatalf("want one node, got %d", len(m.Nodes))
		}

		encoded, err := json.Marshal(struct {
			Nodes   []Node      `json:"nodes"`
			Compact *compactMap `json:"compact"`
		}{m.Nodes, m.compact("t")})
		if err != nil {
			t.Fatal(err)
		}

		if first == nil {
			first = encoded

			continue
		}

		if !bytes.Equal(encoded, first) {
			t.Errorf("merge order changed the map:\n%s\n%s", first, encoded)
		}
	}
}

// TestARealNodeOutranksAPlaceholderInAnyOrder: the file that owns a node knows
// its line and access; a caller's placeholder, however it sorts, never
// displaces it.
func TestARealNodeOutranksAPlaceholderInAnyOrder(t *testing.T) {
	id := FuncID("a.cfc", "run")
	graphs := []*FileGraph{
		{Nodes: []Node{{ID: id, Kind: KindFunction, Name: "run", File: "a.cfc", Line: 7, Access: "public"}}},
		{Provisional: []Node{{ID: id, Kind: KindFunction, Name: "Aardvark"}}},
		{Provisional: []Node{{ID: id, Kind: KindFunction, Name: "Zebra"}}},
	}

	for _, order := range permutations(graphs) {
		m := mergedIn(order)

		if len(m.Nodes) != 1 || m.Nodes[0].Line != 7 || m.Nodes[0].Name != "run" {
			t.Errorf("a placeholder displaced the real node: %+v", m.Nodes)
		}
	}
}
