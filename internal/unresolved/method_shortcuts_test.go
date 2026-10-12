package unresolved

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

func TestMethodShortcutsRetainDiagnosticsAndAmbiguousTargets(t *testing.T) {
	dir := t.TempDir()
	sources := map[string]string{
		"A.cfc": `component { function other(){} }`,
		"B.cfc": `component { function ready(){} function shared(){} }`,
		"C.cfc": `component { function shared(){} }`,
		"Use.cfc": `component {
 function run(bean){
  bean.ready();
  bean.shared();
  var known=new A();
  known.ready();
  known.missing().ready();
  known.nonexistent();
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
	strict := Scan(vfs.OS{}, files, nil, opt)
	opt.SpeculativeMethods = true

	guessed := Scan(vfs.OS{}, files, nil, opt)
	if len(strict.Calls) != 6 || len(guessed.Calls) != 2 || len(guessed.SpeculativeMethodConnections) != 4 || guessed.Resolved != strict.Resolved {
		t.Fatalf("strict=%+v guessed=%+v", strict, guessed)
	}

	if guessed.Calls[0].Function != "missing" || guessed.Calls[1].Function != "nonexistent" {
		t.Fatal(guessed.Calls)
	}

	original := map[string]bool{}
	for _, c := range strict.Calls {
		original[c.Reason] = true
	}

	sawAmbiguous, sawChain, sawKnown := false, false, false

	for _, c := range guessed.SpeculativeMethodConnections {
		if c.Ownership != "unproven" || c.Target.Confidence != ConfidenceLow || !original[c.Reason] {
			t.Fatal(c)
		}

		if c.Function == "shared" {
			sawAmbiguous = true

			if c.Policy != "ranked-global-method" || c.TargetCount != 2 || len(c.Alternatives) != 1 || filepath.Base(c.Target.File) != "B.cfc" {
				t.Fatal(c)
			}
		}

		if c.Function == "ready" && c.Variable == "known" {
			sawKnown = true

			if c.MethodEvidence.Function == "missing" {
				sawChain = true
			}
		}
	}

	if !sawAmbiguous || !sawChain || !sawKnown {
		t.Fatalf("missing coverage: %+v", guessed)
	}

	opt.SpeculativeMethods = false

	again := Scan(vfs.OS{}, files, nil, opt)
	if len(again.Calls) != len(strict.Calls) || len(again.SpeculativeMethodConnections) != 0 {
		t.Fatal("off did not restore strict behavior")
	}

	opt.SpeculativeMethods = true
	opt.AssumeOpenWorld = true

	combined := Scan(vfs.OS{}, files, nil, opt)
	if len(combined.Calls) != 2 || len(combined.SpeculativeMethodConnections) != 4 || len(combined.AssumedCalls) != 0 {
		t.Fatal(combined)
	}
	// Evidence including original diagnostics must survive serialization.
	b, err := json.Marshal(guessed.SpeculativeMethodConnections)
	if err != nil {
		t.Fatal(err)
	}

	var back []SpeculativeMethodConnection
	if err = json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}

	if len(back) != 4 || back[0].Reason == "" || back[0].Target.File == "" {
		t.Fatal(string(b))
	}
}
