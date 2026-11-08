package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

// reasonsFor resolves every call of a one-file workspace and returns the
// resolver's reason for each, keyed by "variable.name" (or "name" for a bare
// call). An empty reason is a call that resolved.
func reasonsFor(t *testing.T, files map[string]string, main string, settings func(*Resolver, string)) map[string]string {
	t.Helper()

	dir := t.TempDir()
	writeFiles(t, dir, files)

	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	if settings != nil {
		settings(r, dir)
	}

	for name := range files {
		if !strings.HasSuffix(name, ".cfc") || name == main {
			continue
		}

		path := filepath.Join(dir, name)

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		r.Index.IndexParsed(cfpath.ToURI(path), string(data), r.ParseFile(path, string(data), BatchSpec()))
	}

	path := filepath.Join(dir, main)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	pr := r.ParseFile(path, string(data), BatchSpec())
	r.Index.IndexParsed(cfpath.ToURI(path), string(data), pr)

	out := map[string]string{}

	calls := pr.AllCalls()
	for i := range calls {
		c := &calls[i]

		key := c.FuncName
		if c.Variable != "" {
			key = c.Variable + "." + c.FuncName
		}

		out[key] = r.CanResolveCall(c, pr, dir)
	}

	return out
}

// A scope word before a call is not a component. VARIABLES.cb() calls a
// function stored in the scope, ARGUMENTS.cb() one passed in: each is accepted
// when the file shows the value, and is otherwise an ordinary unresolved call.
func TestAScopeQualifiedCallIsAFunctionReference(t *testing.T) {
	reasons := reasonsFor(t, map[string]string{"A.cfc": `component {
	function init( cb ) { variables.stored = arguments.cb; return this; }
	function run( passed ) {
		variables.stored( 1 );
		arguments.passed( 2 );
		variables.neverSet( 3 );
	}
}`}, "A.cfc", nil)

	for _, c := range []struct {
		key      string
		resolved bool
	}{
		{"stored", true}, // variables.x() is recorded unqualified
		{"arguments.passed", true},
		{"neverSet", false},
	} {
		got, ok := reasons[c.key]
		if !ok {
			t.Fatalf("no call %s found among %v", c.key, reasons)
		}

		if (got == "") != c.resolved {
			t.Errorf("%s: reason %q, resolved want %v", c.key, got, c.resolved)
		}
	}
}

// A call whose qualifier is the VARIABLES scope itself is a function stored in
// it when the file assigns that name there.
func TestACallQualifiedByTheVariablesScopeNeedsAnAssignment(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A.cfc": `component {
	function init( cb ) { variables.stored = arguments.cb; return this; }
	function run() { }
}`})

	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	path := filepath.Join(dir, "A.cfc")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	pr := r.ParseFile(path, string(data), BatchSpec())
	r.Index.IndexParsed(cfpath.ToURI(path), string(data), pr)

	set := &parser.CallSite{Variable: "VARIABLES", FuncName: "stored", Caller: "run", Line: 3}
	if got := r.CanResolveCall(set, pr, dir); got != "" {
		t.Errorf("VARIABLES.stored: %q, want resolved", got)
	}

	unset := &parser.CallSite{Variable: "VARIABLES", FuncName: "neverSet", Caller: "run", Line: 3}
	if got := r.CanResolveCall(unset, pr, dir); got == "" {
		t.Error("VARIABLES.neverSet resolved, but nothing assigns it")
	}
}

// A receiver nothing in the file types is looked up through the configured
// resolvers by its name, then by the whole line; a resolver that says
// noFollow accepts the call without looking for the method.
func TestAReceiverIsMatchedByNameThenByLine(t *testing.T) {
	files := map[string]string{
		"Svc.cfc": `component { function known() {} }`,
		"A.cfc": `component {
	function run() {
		named.known();
		named.absent();
		other.known();
		free.whatever();
	}
}`,
	}

	reasons := reasonsFor(t, files, "A.cfc", func(r *Resolver, _ string) {
		r.Resolvers = []parser.Resolver{
			{Match: "named", Resolve: "Svc", Prefix: "named", Anchored: true},
			{Match: "free", Resolve: "nocheck", Prefix: "free", Anchored: true, NoFollow: true},
		}
	})

	if reasons["named.known"] != "" {
		t.Errorf("named.known: %q, want resolved through the name", reasons["named.known"])
	}

	if reasons["named.absent"] == "" {
		t.Error("named.absent resolved, but Svc has no such method")
	}

	if reasons["free.whatever"] != "" {
		t.Errorf("free.whatever: %q, want accepted by a noFollow resolver", reasons["free.whatever"])
	}

	if reasons["other.known"] == "" {
		t.Error("other.known resolved with no ref and no resolver for it")
	}
}
