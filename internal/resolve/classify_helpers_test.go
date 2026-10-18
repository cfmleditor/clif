package resolve

import (
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/parser"
)

// TestIsMemberFunctionAgreesWithItsTable derives the positive case from the
// same table the set is built from, so an empty table cannot pass silently.
func TestIsMemberFunctionAgreesWithItsTable(t *testing.T) {
	members := docs.AllMemberFunctions()
	if len(members) == 0 {
		t.Fatal("no documented member functions")
	}

	if !IsMemberFunction(members[0].Name) {
		t.Errorf("IsMemberFunction(%q) = false, want true", members[0].Name)
	}

	if !IsMemberFunction(strings.ToUpper(members[0].Name)) {
		t.Errorf("IsMemberFunction(%q) = false, want the lookup to be case-insensitive", strings.ToUpper(members[0].Name))
	}

	if IsMemberFunction("definitelyNotAMemberFunction12345") {
		t.Fatal("unknown name reported as member function")
	}
}

// TestIsBuiltinCallRequiresNoReceiverOrChain ensures a chained or qualified
// call is never treated as a bare built-in.
func TestIsBuiltinCallRequiresNoReceiverOrChain(t *testing.T) {
	withReceiver := &parser.CallSite{Variable: "x", FuncName: "len"}
	if IsBuiltinCall(withReceiver) {
		t.Fatal("a qualified call reported as a bare built-in")
	}

	chained := &parser.CallSite{FuncName: "len", Chain: []string{"x"}}
	if IsBuiltinCall(chained) {
		t.Fatal("a chained call reported as a bare built-in")
	}
}
