package resolve

import (
	"regexp"
	"testing"
)

var interfaceReference = regexp.MustCompile(`(?im)^\s*interface\b|<cfinterface\b`)

// declaresInterface replaced interfaceReference; it must answer as it did.
func TestDeclaresInterfaceMatchesTheExpression(t *testing.T) {
	for _, src := range []string{
		"interface {}",
		"  \tINTERFACE extends=\"a\" {",
		"/** doc */\ninterface {",
		"component implements=\"interface\" {}",
		"x = 1; interface",
		"interfaces {}",
		"<cfinterface>",
		"<CFINTERFACE displayname=\"x\">",
		"<cfinterfacex>",
		"\n\n   interface",
		"\r\ninterface\r\n",
		"// interface",
		"",
	} {
		if got, want := declaresInterface(src), interfaceReference.MatchString(src); got != want {
			t.Errorf("declaresInterface(%q) = %v, want %v", src, got, want)
		}
	}
}
