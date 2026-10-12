package unresolved

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

func TestMethodEvidenceSeparatesNamesFromOwnershipAndFailedHops(t *testing.T) {
	dir := t.TempDir()
	sources := map[string]string{
		"A.cfc": `component { function other(){} }`,
		"B.cfc": `component { function ready(){} function shared(){} function missing(){} function finish(){} }`,
		"C.cfc": `component { function shared(){} function finish(){} }`,
		"Use.cfc": `component {
 function use(bean){
  bean.ready();
  bean.shared();
  bean.nowhere();
  var known=new A();
  known.missing().finish();
 }
}`,
	}

	var files []string

	for n, s := range sources {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, p)
	}

	opt := &Options{WorkspaceFolders: []string{dir}, Candidates: true}
	rep := Scan(vfs.OS{}, files, []string{filepath.Join(dir, "Use.cfc")}, opt)

	by := map[string]Call{}
	for _, c := range rep.Calls {
		by[c.Function] = c
	}

	for name, want := range map[string]string{"ready": "unique-source-declaration", "shared": "ambiguous-source-declarations", "nowhere": "no-source-declaration"} {
		c, ok := by[name]
		if !ok {
			t.Fatalf("missing diagnostic %s: %+v", name, rep.Calls)
		}

		e := c.MethodEvidence
		if e == nil || e.NameStatus != want || e.Ownership != "unproven" || e.Relationship != "receiver-type-unproven" {
			t.Errorf("%s evidence=%+v", name, e)
		}
	}

	c, ok := by["finish"]
	if !ok {
		t.Fatalf("missing chain diagnostic: %+v", rep.Calls)
	}

	if e := c.MethodEvidence; e == nil || e.Function != "missing" || e.SourceDeclarations != 1 || e.Relationship != "selected-contract-mismatch" {
		t.Errorf("failed hop evidence=%+v", e)
	}

	if c.CandidateCount != 1 || len(c.Candidates) != 1 || filepath.Base(c.Candidates[0].File) != "B.cfc" {
		t.Errorf("failed hop candidates=%+v", c)
	}
	// The unrelated unique declaration must never repair the known A receiver.
	opt.Speculative = true
	guessed := Scan(vfs.OS{}, files, []string{filepath.Join(dir, "Use.cfc")}, opt)
	found := false

	for _, c := range guessed.Calls {
		if c.Function == "finish" {
			found = true
		}
	}

	if !found {
		t.Fatal("global uniqueness overrode a known receiver mismatch")
	}
}

func TestSpeculativeConnectionExposesUnprovenMethods(t *testing.T) {
	dir := t.TempDir()

	var files []string

	for n, s := range map[string]string{"A.cfc": `component { function ready(){} }`, "Use.cfc": `component { function use(bean){bean.ready();} }`} {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, p)
	}

	rep := Scan(vfs.OS{}, files, nil, &Options{WorkspaceFolders: []string{dir}, Speculative: true})
	if len(rep.SpeculativeConnections) != 1 {
		t.Fatal(rep)
	}

	c := rep.SpeculativeConnections[0]
	if c.Ownership != "unproven" || len(c.Methods) != 1 || c.Methods[0] != "ready" || c.Policy != "sole-cfc-candidate" {
		t.Fatalf("missing fallback evidence: %+v", c)
	}
}
