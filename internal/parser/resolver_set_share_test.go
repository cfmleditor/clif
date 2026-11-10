package parser

import "testing"

// A set lent to several parses shares its memo of answers, and keeps the soft
// components (from a dynamicIfMissing resolver) to the parse that met them.
func TestASharedResolverSetKeepsSoftComponentsPerParse(t *testing.T) {
	resolvers := []Resolver{{Match: "getWidget()", Resolve: "models.Widget", Prefix: "getWidget", DynamicIfMissing: true}}
	shared := BuildResolverSet(resolvers)
	opts := func() *ParseOptions {
		return &ParseOptions{Resolvers: resolvers, ResolverSet: shared, ExtractCalls: true, ScanAllScopes: true}
	}

	a := ParseWithOptions("file:///a.cfc", "component { function f() { var w = getWidget(); w.go(); } }", opts())
	b := ParseWithOptions("file:///b.cfc", "component { function f() { var x = 1; } }", opts())

	if a.resolverSet == shared || b.resolverSet == shared {
		t.Fatal("a parse was handed the shared set itself, not a set of its own")
	}

	if !a.IsSoftComponent("models.Widget") {
		t.Fatal("the parse that met the dynamicIfMissing resolver does not report its component soft")
	}

	if b.IsSoftComponent("models.Widget") {
		t.Fatal("a parse that never met the resolver reports another parse's soft component")
	}

	if shared.memo.size() == 0 {
		t.Fatal("the parses did not fill the shared memo")
	}

	// A set built from another slice is not used.
	other := []Resolver{{Match: resolvers[0].Match, Resolve: resolvers[0].Resolve, Prefix: resolvers[0].Prefix}}
	if c := ParseWithOptions("file:///c.cfc", "component {}", &ParseOptions{Resolvers: other, ResolverSet: shared}); c.resolverSet.memo == shared.memo {
		t.Fatal("a parse used a set built from another slice of resolvers")
	}
}
