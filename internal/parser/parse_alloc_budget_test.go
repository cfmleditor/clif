package parser

import "testing"

// parseAllocBudgets holds each benchmark fixture's parse to a number of
// allocations. An allocation count, unlike a time, is the same on every
// machine and every run, so it can fail the build: the collection-returns
// pass (bcac43a) took a script component from 442 allocations to 3,713 and a
// sixfold slower parse, and the benchmarks that showed it gate nothing.
//
// Each budget is the count when it was set with about a quarter's headroom.
// A change that needs more should raise it in the same commit and say why;
// one that lowers the count a lot should lower it, so the headroom stays a
// quarter rather than becoming room for the next regression.
var parseAllocBudgets = []struct {
	name    string
	content *string
	opts    ParseOptions
	budget  float64
}{
	{"script", &benchScriptCFC, ParseOptions{}, 1650},
	{"script, calls", &benchScriptCFC, ParseOptions{ExtractCalls: true}, 1720},
	{"tag", &benchTagCFC, ParseOptions{}, 425},
	{"tag, calls", &benchTagCFC, ParseOptions{ExtractCalls: true}, 1070},
}

func TestParseStaysWithinItsAllocationBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes allocation counts")
	}

	for i := range parseAllocBudgets {
		c := &parseAllocBudgets[i]
		got := testing.AllocsPerRun(20, func() {
			opts := c.opts
			ParseWithOptions("file:///bench.cfc", *c.content, &opts)
		})
		t.Logf("%s: %.0f allocations (budget %.0f)", c.name, got, c.budget)

		if got > c.budget {
			t.Errorf("%s: a parse made %.0f allocations, over its budget of %.0f", c.name, got, c.budget)
		}
	}
}
