package resolve

import "testing"

func TestProducerOnlyTracksIntermediateStatesForCatch(t *testing.T) {
	budget := 100
	e := &producerEvaluation{budget: &budget}
	env := producerEnvironment{"original": producerUnknown()}
	nodes := []producerNode{{kind: "param", target: "later"}}

	plain := e.execute(nodes, env)
	if plain.observed != nil {
		t.Fatal("ordinary evaluation retained unused intermediate states")
	}

	if _, ok := plain.environment["later"]; !ok {
		t.Fatal("lost final environment")
	}

	if _, ok := env["later"]; ok {
		t.Fatal("modified caller environment")
	}
	// Catch observes both the initial and modified literal, so it must not
	// claim either literal is known even though the primitive type is known.
	e.wanted = map[string]bool{"x": true}
	initial := producerEnvironment{"x": {primitive: true, literalKnown: true, literal: "old"}}
	node := &producerNode{kind: "try", body: []producerNode{{kind: "set", target: "x", expression: "'new'"}}, alternative: []producerNode{{kind: "return", expression: "x"}}}

	got := e.branches(node, initial)
	if len(got.returns) != 1 || !got.returns[0].primitive || got.returns[0].unknown || got.returns[0].literalKnown {
		t.Fatalf("catch lost intermediate states: %+v", got.returns)
	}
}
