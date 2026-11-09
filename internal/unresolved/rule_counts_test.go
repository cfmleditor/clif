package unresolved

import (
	"sync"
	"testing"
)

func TestRuleTallyCountsRulesNotProofsOrBuiltins(t *testing.T) {
	var nilTally *ruleTally

	nilTally.add("sharedMember") // a scan with no tally must not panic

	tl := &ruleTally{}

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			tl.add("sharedMember")
			tl.add("")
			tl.add("builtin")
			tl.add("untypedHop")
		})
	}

	wg.Wait()

	got := tl.snapshot()
	if len(got) != 2 || got["sharedMember"] != 8 || got["untypedHop"] != 8 {
		t.Errorf("tally = %v, want sharedMember and untypedHop at 8 and nothing else", got)
	}
}
