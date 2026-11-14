package parser

import "testing"

// A string compared with `==` is a piece of an expression, not the argument.
func TestFirstArgumentIsNotAComparedString(t *testing.T) {
	for src, want := range map[string]string{
		`"Provider" )`:          "Provider",
		`name = "Provider" )`:   "Provider",
		`x == "Provider" )`:     "",
		`a = b == "x" )`:        "",
		`url.p != "Provider" )`: "",
		`n <= "Provider" )`:     "",
		`c ? 1 : "Provider" )`:  "",
		`name: "Provider" )`:    "Provider",
	} {
		got, _ := newScriptParser(src, "f.cfc", 0, nil).scanParenBody()
		if got != want {
			t.Errorf("%s: got %q, want %q", src, got, want)
		}
	}
}
