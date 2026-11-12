package resolve

import (
	"sync"
	"testing"
)

// Workers that miss a source together keep the first one stored: what it has
// derived is not derived again for the second.
func TestASourceReadTwiceAtOnceKeepsOneDerivation(t *testing.T) {
	r := &Resolver{}
	content := "component { function a() {} }"

	first := r.adoptSourceInfo("/w/A.cfc", sourceInfo{content: content, derived: &sourceDerived{path: "/w/A.cfc"}})
	second := r.adoptSourceInfo("/w/A.cfc", sourceInfo{content: string([]byte(content)), derived: &sourceDerived{path: "/w/A.cfc"}, size: 9})

	if second.derived != first.derived {
		t.Fatal("a second read of the same text replaced the first's derivations")
	}

	if second.size != 9 {
		t.Errorf("the kept source has size %d, want the newer stamp's 9", second.size)
	}

	changed := r.adoptSourceInfo("/w/A.cfc", sourceInfo{content: content + " ", derived: &sourceDerived{path: "/w/A.cfc"}})
	if changed.derived == first.derived {
		t.Error("changed text kept the old derivations")
	}
}

// Views asking for one startup source at once share one reading of it.
func TestStartupFactsAreReadOncePerText(t *testing.T) {
	owner := &Resolver{}
	views := []*Resolver{{indexer: owner}, {indexer: owner}}
	content := "<cfset application.svc = createObject(\"component\", \"a.Svc\")>"

	got := make([]*startupFileFacts, 16)

	var wg sync.WaitGroup
	for i := range got {
		wg.Go(func() { got[i] = views[i%2].startupFacts("/w/Application.cfm", content) })
	}

	wg.Wait()

	for i := range got {
		if got[i] != got[0] {
			t.Fatalf("call %d read its own facts", i)
		}
	}

	if len(got[0].shared) == 0 {
		t.Error("the shared facts hold no assignment")
	}

	if again := views[0].startupFacts("/w/Application.cfm", content+"\n"); again == got[0] {
		t.Error("changed text kept the old facts")
	}
}

// Every view's FW/1 scope reads one parse of each file.
func TestFW1ParsesAreSharedAcrossViews(t *testing.T) {
	owner := &Resolver{}
	a, b := &Resolver{indexer: owner}, &Resolver{indexer: owner}
	text := "component extends=\"framework.one\" { function setupApplication() { setBeanFactory(x); } }"

	first := a.fw1Parse("/w/Application.cfc", text)
	if b.fw1Parse("/w/Application.cfc", text) != first {
		t.Fatal("a second view parsed the file again")
	}

	if a.fw1Parse("/w/Application.cfc", text+" ") == first {
		t.Error("changed text kept the old parse")
	}
}
