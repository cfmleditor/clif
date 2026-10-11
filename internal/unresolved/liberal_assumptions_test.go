package unresolved

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestSpeculativeReceiverRoleRejectsAmbiguousSameLineChains(t *testing.T) {
	call := &Call{Line: 2, Variable: "factory", Function: "ready", Caller: "use"}

	sites := []parser.CallSite{
		{Line: 2, Variable: "factory", FuncName: "ready", Caller: "use", Chain: []string{parser.CallHop("makeA()")}},
		{Line: 2, Variable: "factory", FuncName: "ready", Caller: "use", Chain: []string{parser.CallHop("makeB()")}},
	}
	if role, ok := speculativeReceiverRole(sites, call); ok {
		t.Fatal("ambiguous role accepted", role)
	}

	sites[1].Chain = sites[0].Chain
	if role, ok := speculativeReceiverRole(sites, call); !ok || role != "makeA" {
		t.Fatalf("role=%q ok=%v", role, ok)
	}
}

func TestRankedSpeculationUsesBaseRoleAndPreservesTies(t *testing.T) {
	dir := t.TempDir()
	sources := map[string]string{
		"Product.cfc": `component {function ready(){}}`,
		"Factory.cfc": `component {function make(){return new Product();}}`,
		"Other.cfc":   `component {function make(){return new Product();}}`,
		"Use.cfc": `component {
function ranked(factory){factory.make().ready();}
function tied(value){value.make().ready();}
function known(){var x=new Other();x.absent();}
}`,
	}

	var files []string

	for i := range 10 {
		sources[fmt.Sprintf("Spare%d.cfc", i)] = sources["Other.cfc"]
	}

	for name, src := range sources {
		file := filepath.Join(dir, name)
		if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, file)
	}

	opt := &Options{WorkspaceFolders: []string{dir}}
	strict := Scan(vfs.OS{}, files, nil, opt)
	opt.SpeculativeRanked = true

	guessed := Scan(vfs.OS{}, files, nil, opt)
	if len(guessed.SpeculativeConnections) != 1 {
		t.Fatalf("connections=%+v calls=%+v", guessed.SpeculativeConnections, guessed.Calls)
	}

	c := guessed.SpeculativeConnections[0]
	if c.Caller != "ranked" || c.Component != filepath.Join(dir, "Factory.cfc") || c.Policy != "ranked-cfc-role" {
		t.Fatal(c)
	}

	if len(guessed.Calls) != len(strict.Calls)-2 {
		t.Fatalf("strict=%+v guessed=%+v", strict.Calls, guessed.Calls)
	}

	for _, call := range guessed.Calls {
		if call.Caller == "ranked" {
			t.Fatal(call)
		}
	}

	opt.SpeculativeRanked = false

	again := Scan(vfs.OS{}, files, nil, opt)
	if len(again.Calls) != len(strict.Calls) || len(again.SpeculativeConnections) != 0 {
		t.Fatal("off failed")
	}
}

func TestOpenWorldRetainsEvidenceAndDoesNotInflateResolvedCount(t *testing.T) {
	dir := t.TempDir()
	sources := map[string]string{
		"Known.cfc":   `component {function ready(){}}`,
		"Factory.cfc": `component {function make(){}}`,
		"Use.cfc": `component {
function unknown(value){value.ready();}
function known(){var x=new Known();x.absent();}
function explicitThis(){this.absent();}
function external(){var x=new ExternalService();x.ready();}
function mixin(){externalHook();}
function returns(){var f=new Factory();f.make().ready();}
}`,
		"Broken.cfc": `component extends="MissingBase" {function use(){inheritedHook();}}`,
	}

	var files []string

	for name, src := range sources {
		file := filepath.Join(dir, name)
		if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, file)
	}

	opt := &Options{WorkspaceFolders: []string{dir}, Candidates: true}
	strict := Scan(vfs.OS{}, files, nil, opt)
	opt.AssumeOpenWorld = true

	open := Scan(vfs.OS{}, files, nil, opt)
	if len(open.AssumedCalls) < 4 || len(open.Calls) != 3 {
		t.Fatalf("%+v", open)
	}

	if open.Resolved != strict.Resolved || len(open.Calls)+len(open.AssumedCalls) != len(strict.Calls) || open.Indexed != strict.Indexed || open.Scanned != strict.Scanned {
		t.Fatal("lost coverage or inflated resolutions")
	}

	policies := map[string]bool{}

	for _, assumed := range open.AssumedCalls {
		if assumed.Reason == "" || assumed.Policy == "" {
			t.Fatal(assumed)
		}

		policies[assumed.Policy] = true
	}

	for _, policy := range []string{"runtime-receiver", "runtime-return", "external-component", "runtime-function-mixin"} {
		if !policies[policy] {
			t.Fatalf("missing %s: %+v", policy, open)
		}
	}

	for _, call := range open.Calls {
		if call.Unchecked == 0 && call.Caller != "known" && call.Caller != "explicitThis" {
			t.Fatal(call)
		}
	}

	opt.AssumeOpenWorld = false

	again := Scan(vfs.OS{}, files, nil, opt)
	if len(again.Calls) != len(strict.Calls) || len(again.AssumedCalls) != 0 {
		t.Fatal("off failed")
	}
}
