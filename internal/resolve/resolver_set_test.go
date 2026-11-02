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

	if !r.parseResolverSet().Serves(r.Resolvers) {
		t.Fatal("the set does not serve the resolvers it was built from")
	}

	r.Resolvers = []parser.Resolver{{Match: "getB()", Resolve: "b.B", Prefix: "getB"}, {Match: "getA()", Resolve: "a.A", Prefix: "getA"}}

	if !r.parseResolverSet().Serves(r.Resolvers) {
		t.Fatal("after Resolvers was replaced, the set still serves the old slice")
	}
}
