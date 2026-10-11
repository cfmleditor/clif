package parser

import "testing"

func TestElseIfBranchesAreNotCalls(t *testing.T) {
	source := `component {function run(){if(first()) {one();} elseif(second()) {two();} else {three();} receiver.elseif();}}`
	pr := ParseWithOptions("file:///tmp/Branches.cfc", source, &ParseOptions{ExtractCalls: true})

	counts := map[string]int{}
	for _, call := range pr.AllCalls() {
		counts[call.Variable+"."+call.FuncName]++
	}

	if counts[".elseif"] != 0 {
		t.Fatalf("branch extracted as call: %v", counts)
	}

	for _, name := range []string{".first", ".second", ".one", ".two", ".three", "receiver.elseif"} {
		if counts[name] != 1 {
			t.Fatalf("lost %s: %v", name, counts)
		}
	}
}
