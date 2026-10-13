package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestIsBuiltin(t *testing.T) {
	// A tag called as a function is built in too: cfheader( name = "x" ).
	// "cf" alone and a cf-prefixed name no tag has are not.
	for name, want := range map[string]bool{
		"trim": true, "TRIM": true, "append": true, "someUserDefinedFunctionXyz": false,
		"cfheader": true, "CFHTTP": true, "cf": false, "cfNotATag": false,
	} {
		if got := IsBuiltin(name); got != want {
			t.Errorf("IsBuiltin(%q) = %v, want %v", name, got, want)
		}
	}

	for name, want := range map[string]bool{"append": true, "APPEND": true, "notARealMemberFunctionXyz": false} {
		if got := IsMemberFunction(name); got != want {
			t.Errorf("IsMemberFunction(%q) = %v, want %v", name, got, want)
		}
	}
}

// TestClassifyCallIsTheOneDecision: a bare call by a builtin's name is a
// builtin, but a call chained onto another call is a call on what that returns,
// and a call the workspace defines is that definition. The code map had its own
// copy of this test that ignored the chain and used a shorter list of names, so
// it hid calls `unresolved` reported.
func TestClassifyCallIsTheOneDecision(t *testing.T) {
	dir := t.TempDir()
	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	r.Index.IndexFile(cfpath.ToURI(filepath.Join(dir, "Defs.cfc")), `component { function definedSomewhereXyz(){} }`)

	pr := &parser.ParseResult{}

	for name, tc := range map[string]struct {
		call parser.CallSite
		opt  ClassifyOptions
		want Verdict
	}{
		"bare builtin":          {call: parser.CallSite{FuncName: "append"}, want: Builtin},
		"member function":       {call: parser.CallSite{FuncName: "arrayAppend"}, want: Builtin},
		"chained builtin name":  {call: parser.CallSite{FuncName: "trim", Chain: []string{"f"}}, want: Unresolved},
		"receiver, same name":   {call: parser.CallSite{FuncName: "trim", Variable: "x"}, want: Unresolved},
		"not a builtin":         {call: parser.CallSite{FuncName: "someUserFunctionXyz"}, want: Unresolved},
		"global def, not asked": {call: parser.CallSite{FuncName: "definedSomewhereXyz"}, want: Unresolved},
		"global def, asked":     {call: parser.CallSite{FuncName: "definedSomewhereXyz"}, opt: ClassifyOptions{GlobalDefs: true}, want: GlobalDef},
	} {
		if got := r.ClassifyCall(&tc.call, pr, dir, tc.opt); got.Verdict != tc.want {
			t.Errorf("%s: verdict %v, want %v (%s)", name, got.Verdict, tc.want, got.Reason)
		}
	}
}
