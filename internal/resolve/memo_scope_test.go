package resolve

import (
	"testing"

	"github.com/cfmleditor/clif/internal/index"
)

// A parse memo whose answer resolves paths is the resolver's, for one index
// generation: the parse outlives a component appearing on disk, and two
// resolvers (one per config) can read the same parse. A snapshot scan's
// index only grows with what is already on disk, so it keeps one scope.
func TestAPathResolvingMemoIsScopedToItsResolverAndIndex(t *testing.T) {
	idx := index.New()
	a, b := &Resolver{Index: idx}, &Resolver{Index: idx}

	if a.memoScope() == b.memoScope() {
		t.Error("two resolvers share a memo scope")
	}

	before := a.memoScope()

	idx.IndexFile("file:///Repo.cfc", "component { function one(){} }")

	if a.memoScope() == before {
		t.Error("an index write left the memo scope unchanged")
	}

	snap := &Resolver{Index: idx, SnapshotScan: true}
	before = snap.memoScope()

	idx.IndexFile("file:///Other.cfc", "component { function two(){} }")

	if snap.memoScope() != before {
		t.Error("a snapshot scan's memo scope moved with the index")
	}
}
