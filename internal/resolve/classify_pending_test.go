package resolve

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

func declareIn(r *Resolver, dir string, files int, function string) {
	for i := range files {
		r.Index.IndexFile(cfpath.ToURI(filepath.Join(dir, fmt.Sprintf("C%d.cfc", i))), `component { function `+function+`(){} }`)
	}
}

// A call whose fate rests on how many components declare a name is left open
// when it is classified and decided against the index as it stands when it is
// settled: the same call classified immediately gives the answer for
// whatever the index held at that moment.
func TestACountRuleIsDecidedAgainstTheIndexAtSettleTime(t *testing.T) {
	dir := t.TempDir()
	pr := &parser.ParseResult{}
	deferred := ClassifyOptions{DeferCounts: true}

	t.Run("a name five components share, declared after the call was classified", func(t *testing.T) {
		r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
		call := parser.CallSite{FuncName: "sharedByManyXyz", Variable: "x"}

		c := r.ClassifyCall(&call, pr, dir, deferred)
		if c.Pending == nil {
			t.Fatalf("verdict %v with no pending; a shared-member call must wait for the counts", c.Verdict)
		}

		declareIn(r, dir, sharedMemberFiles, "sharedByManyXyz")

		if v, rule := c.Pending.Settle(r); v != Accepted || rule != "sharedMember" {
			t.Errorf("settled %v by %q, want Accepted by sharedMember", v, rule)
		}

		if now := r.ClassifyCall(&call, pr, dir, ClassifyOptions{}); now.Verdict != Accepted {
			t.Errorf("an immediate classification now gives %v, want Accepted", now.Verdict)
		}
	})

	t.Run("a member function's name no component declares, until one does", func(t *testing.T) {
		r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
		call := parser.CallSite{FuncName: "trim", Variable: "x"}

		c := r.ClassifyCall(&call, pr, dir, deferred)
		if c.Pending == nil {
			t.Fatalf("verdict %v with no pending", c.Verdict)
		}

		if v, rule := c.Pending.Settle(r); v != Builtin || rule != "undeclaredMember" {
			t.Errorf("settled %v by %q with no declaration, want Builtin by undeclaredMember", v, rule)
		}

		declareIn(r, dir, 1, "trim")

		// Still a built-in's call, but now by the rule that weighs a native
		// member against a component's: one declaration is not the receiver.
		if v, rule := c.Pending.Settle(r); v != Builtin || rule != "nativeMemberOfUnknown" {
			t.Errorf("settled %v by %q once a component declares it, want Builtin by nativeMemberOfUnknown", v, rule)
		}
	})

	t.Run("a rule that reads no count waits behind one that does", func(t *testing.T) {
		r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
		call := parser.CallSite{FuncName: "getFooBarXyz", Variable: "x"}

		c := r.ClassifyCall(&call, pr, dir, deferred)
		if c.Pending == nil {
			t.Fatalf("verdict %v with no pending; sharedMember comes first and may yet apply", c.Verdict)
		}

		if _, rule := c.Pending.Settle(r); rule != "beanAccessorOfUnknown" {
			t.Errorf("settled by %q, want beanAccessorOfUnknown while few components declare it", rule)
		}

		declareIn(r, dir, sharedMemberFiles, "getFooBarXyz")

		if _, rule := c.Pending.Settle(r); rule != "sharedMember" {
			t.Errorf("settled by %q, want sharedMember once five declare it", rule)
		}
	})
}

// A call no count rule can reach is decided at once, as it always was.
func TestACallNoCountRuleCanReachIsNotLeftOpen(t *testing.T) {
	dir := t.TempDir()
	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	pr := &parser.ParseResult{}
	opt := ClassifyOptions{DeferCounts: true}

	for name, call := range map[string]parser.CallSite{
		"a chained call":           {FuncName: "someFuncXyz", Variable: "x", Chain: []string{"f"}},
		"a bare call":              {FuncName: "someFuncXyz"},
		"a plain built-in":         {FuncName: "arrayAppend"},
		"a name no rule can match": {FuncName: "ab", Variable: "x", Chain: []string{"f"}},
	} {
		if c := r.ClassifyCall(&call, pr, dir, opt); c.Pending != nil {
			t.Errorf("%s was left open", name)
		}
	}

	// And with a global-definition check, which counts too, nothing is left open.
	call := parser.CallSite{FuncName: "sharedByManyXyz", Variable: "x"}
	if c := r.ClassifyCall(&call, pr, dir, ClassifyOptions{DeferCounts: true, GlobalDefs: true}); c.Pending != nil {
		t.Error("a call was left open with GlobalDefs set")
	}
}
