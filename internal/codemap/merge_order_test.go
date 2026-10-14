package codemap

import "testing"

// TestAnExternalNodeKeepsOneSpellingInAnyMergeOrder: an external node's id is
// its lowercased label, so `expandPath` and `Expandpath` are one node, and the
// report named it after whichever file was merged first. Files finish in no
// fixed order, so the same workspace printed either name from run to run.
func TestAnExternalNodeKeepsOneSpellingInAnyMergeOrder(t *testing.T) {
	a := &FileGraph{Provisional: []Node{{ID: ExternalID("expandPath"), Kind: KindExternal, Name: "expandPath"}}}
	b := &FileGraph{Provisional: []Node{{ID: ExternalID("Expandpath"), Kind: KindExternal, Name: "Expandpath"}}}

	names := map[string]bool{}

	for _, order := range [][]*FileGraph{{a, b}, {b, a}} {
		c := newCollector(&Map{})
		for _, g := range order {
			c.merge(g)
		}

		n, ok := c.nodes[ExternalID("expandPath")]
		if !ok {
			t.Fatal("the node is missing")
		}

		names[n.Name] = true
	}

	if len(names) != 1 {
		t.Errorf("the node's name depends on merge order: %v", names)
	}
}
