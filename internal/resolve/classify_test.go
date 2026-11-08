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
	r.Index.IndexFile(cfpath.ToURI(filepath.Join(dir, "Defs.cfc")), `component { function definedSomewhereXyz(){} function toQuery(){} }`)

	pr := &parser.ParseResult{}

	for name, tc := range map[string]struct {
		call parser.CallSite
		opt  ClassifyOptions
		want Verdict
	}{
		"bare builtin":         {call: parser.CallSite{FuncName: "append"}, want: Builtin},
		"member function":      {call: parser.CallSite{FuncName: "arrayAppend"}, want: Builtin},
		"chained builtin name": {call: parser.CallSite{FuncName: "trim", Chain: []string{"f"}}, want: Unresolved},
		// A receiver nothing is known of, called by a member function's name no
		// component declares, can only be that member function; one a
		// component does declare may be that component's.
		"receiver, same name":   {call: parser.CallSite{FuncName: "trim", Variable: "x"}, want: Builtin},
		"receiver, declared":    {call: parser.CallSite{FuncName: "toQuery", Variable: "x"}, want: Unresolved},
		"not a builtin":         {call: parser.CallSite{FuncName: "someUserFunctionXyz"}, want: Unresolved},
		"global def, not asked": {call: parser.CallSite{FuncName: "definedSomewhereXyz"}, want: Unresolved},
		"global def, asked":     {call: parser.CallSite{FuncName: "definedSomewhereXyz"}, opt: ClassifyOptions{GlobalDefs: true}, want: GlobalDef},
	} {
		if got := r.ClassifyCall(&tc.call, pr, dir, tc.opt); got.Verdict != tc.want {
			t.Errorf("%s: verdict %v, want %v (%s)", name, got.Verdict, tc.want, got.Reason)
		}
	}
}

// TestExplanationEndsWithWhatTheReportDoes: a trace that stops at the resolver's
// reason says "unresolved" for a call the report sets aside as a built-in, and
// nothing about a call it folds into a missing base. The last line is the
// classification the report makes, from the reason printed above it.
func TestExplanationEndsWithWhatTheReportDoes(t *testing.T) {
	dir := t.TempDir()
	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	pr := &parser.ParseResult{}

	for name, tc := range map[string]struct {
		call parser.CallSite
		want string
	}{
		"builtin":    {parser.CallSite{FuncName: "append"}, "report: set aside as a built-in"},
		"unresolved": {parser.CallSite{FuncName: "someUserFunctionXyz"}, "report: reported as unresolved"},
	} {
		var b strings.Builder

		r.WriteExplanation(&b, "f.cfc", 0, []parser.CallSite{tc.call}, pr, dir)

		if !strings.Contains(b.String(), tc.want) {
			t.Errorf("%s: want %q in:\n%s", name, tc.want, b.String())
		}
	}

	if got := MissingBase.Describe("coldbox.system.Base"); !strings.Contains(got, "coldbox.system.Base") || !strings.Contains(got, "once per file") {
		t.Errorf("missing-base description: %q", got)
	}
}

// TestAnUntypedHopIsAcceptedAsDynamic: `svc.get().foo()` with get() declaring no
// return type is as dynamic as `x = svc.get(); x.foo()`, which the run accepts.
func TestAnUntypedHopIsAcceptedAsDynamic(t *testing.T) {
	r := &Resolver{FS: vfs.OS{}, Index: index.New()}
	call := &parser.CallSite{FuncName: "foo", Variable: "svc", Chain: []string{"get"}}

	reason := "method 'get' in Svc has no component return type (chain to 'foo')"
	if v, _ := r.ClassifyReason(call, reason, ClassifyOptions{}); v != Accepted {
		t.Errorf("an untyped hop: verdict %v, want accepted", v)
	}

	if v, _ := r.ClassifyReason(call, "method 'foo' not found in Svc", ClassifyOptions{}); v != Unresolved {
		t.Errorf("a missing method: verdict %v, want unresolved", v)
	}
}

// TestAVariableAssignedFromAnUntypedFunctionIsAccepted: `x = helper()` where
// helper() declares no return type and gives none to infer holds whatever it
// returns, as `x = svc.get()` does for an untyped get(): the run accepts calls
// on it. A function with a return type, or one nothing declares, is not that.
func TestAVariableAssignedFromAnUntypedFunctionIsAccepted(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Use.cfc": `component {
	function helper( a ) { var r = something( a ); return r; }
	string function typed() { return "a"; }
	function run() {
		var x = helper( 1 );
		x.zorkify();
		var y = typed();
		y.zorkify();
		var z = notDeclared();
		z.zorkify();
	}
}`,
	})

	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	file := filepath.Join(dir, "Use.cfc")
	got := map[string]Verdict{}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	pr := r.ParseFile(file, string(content), BatchSpec())
	r.Index.IndexParsed(cfpath.ToURI(file), string(content), pr)

	for _, c := range pr.AllCalls() {
		if c.Variable != "" {
			got[c.Variable] = r.ClassifyCall(&c, pr, dir, ClassifyOptions{}).Verdict
		}
	}

	if got["x"] != Accepted {
		t.Errorf("x: verdict %v, want accepted", got["x"])
	}

	if got["z"] != Unresolved {
		t.Errorf("z: verdict %v, want unresolved", got["z"])
	}
}

// TestTheClassifyRulesAreInTheirDocumentedOrder pins the table: the order is
// part of the answer, and the judgement flag is what the strict view turns off.
func TestTheClassifyRulesAreInTheirDocumentedOrder(t *testing.T) {
	type entry struct {
		name      string
		judgement bool
	}

	want := []entry{
		{"untypedHop", true},
		{"undeclaredMember", false},
		{"assignedFromUntypedFunction", false},
		{"sharedMember", true},
		{"namedComponentDeclares", true},
		{"nativeMemberOfUnknown", true},
		{"templateFunction", true},
		{"distinctiveMember", true},
		{"beanAccessorOfUnknown", true},
		{"knownNativeMember", true},
	}

	if len(classifyRules) != len(want) {
		t.Fatalf("%d rules, want %d", len(classifyRules), len(want))
	}

	for i, w := range want {
		if got := (entry{classifyRules[i].name, classifyRules[i].judgement}); got != w {
			t.Errorf("rule %d is %+v, want %+v", i, got, w)
		}
	}
}

// TestTheRuleThatDecidedIsNamed: the trace can say why a call was set aside,
// and the strict view leaves the judgement out.
func TestTheRuleThatDecidedIsNamed(t *testing.T) {
	r := &Resolver{FS: vfs.OS{}, Index: index.New()}
	call := &parser.CallSite{FuncName: "foo", Variable: "svc", Chain: []string{"get"}}
	reason := "method 'get' in Svc has no component return type (chain to 'foo')"

	if v, _, rule := r.ClassifyReasonRule(call, reason, ClassifyOptions{}, nil, ""); v != Accepted || rule != "untypedHop" {
		t.Errorf("open view: %v %q, want accepted by untypedHop", v, rule)
	}

	if v, _, rule := r.ClassifyReasonRule(call, reason, ClassifyOptions{KeepUntypedHops: true}, nil, ""); v != Unresolved || rule != "" {
		t.Errorf("strict view: %v %q, want unresolved by no rule", v, rule)
	}

	bare := &parser.CallSite{FuncName: "len"}
	if _, _, rule := r.ClassifyReasonRule(bare, "function 'len' not found", ClassifyOptions{}, nil, ""); rule != "builtin" {
		t.Errorf("a built-in's rule is %q", rule)
	}
}
