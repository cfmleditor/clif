package resolve

import (
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// NewBatchResolver replaces Resolvers after BeanResolvers has already parsed
// files, so a set built before that must not be the one every later parse is
// handed.
func TestTheParseResolverSetFollowsTheResolvers(t *testing.T) {
	r := &Resolver{Resolvers: []parser.Resolver{{Match: "getA()", Resolve: "a.A", Prefix: "getA"}}}

	first := r.parseResolverSet(r.Resolvers)
	if !first.Serves(r.Resolvers) {
		t.Fatal("the set does not serve the resolvers it was built from")
	}

	if again := r.parseResolverSet(r.Resolvers); again != first {
		t.Fatal("a second parse with the same resolvers was not handed the same set")
	}

	oldResolvers := r.Resolvers
	r.Resolvers = []parser.Resolver{{Match: "getB()", Resolve: "b.B", Prefix: "getB"}, {Match: "getA()", Resolve: "a.A", Prefix: "getA"}}

	rebuilt := r.parseResolverSet(r.Resolvers)
	if !rebuilt.Serves(r.Resolvers) {
		t.Fatal("after Resolvers was replaced, the set still serves the old slice")
	}

	if again := r.parseResolverSet(r.Resolvers); again != rebuilt {
		t.Fatal("the rebuilt set was not kept: every later parse builds its own again")
	}

	// A resolver still holding the old slice keeps its own set, and asking
	// with it does not replace the new one.
	if old := r.parseResolverSet(oldResolvers); old != first {
		t.Fatal("the old slice was handed a new set rather than the one built for it")
	}

	if again := r.parseResolverSet(r.Resolvers); again != rebuilt {
		t.Fatal("asking with the old slice replaced the new slice's set")
	}
}
