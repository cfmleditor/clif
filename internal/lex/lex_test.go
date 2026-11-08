package lex

import (
	"regexp"
	"testing"
)

// The reference definitions are the ones the packages carried before they
// shared this one, spelled the way each wrote them.
func TestIsIdentMatchesEveryFormerCopy(t *testing.T) {
	former := map[string]func(byte) bool{
		"parser.isIdentByte": func(c byte) bool {
			return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
		},
		"formatter.isIdentByte": func(c byte) bool {
			return c == '_' || c == '$' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		},
		"resolve.isIdentByte(first=false)": func(c byte) bool {
			return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= '0' && c <= '9'
		},
	}

	for name, f := range former {
		for b := range 256 {
			if got, want := IsIdent(byte(b)), f(byte(b)); got != want {
				t.Errorf("%s differs at %#x: IsIdent=%v", name, b, got)
			}
		}
	}
}

func TestIsWordIsRegexpW(t *testing.T) {
	w := regexp.MustCompile(`^\w$`)

	for b := range 256 {
		if got, want := IsWord(byte(b)), w.Match([]byte{byte(b)}) && b < 0x80; got != want {
			t.Errorf("IsWord(%#x) = %v, want %v", b, got, want)
		}
	}

	if IsWord('$') || !IsIdent('$') {
		t.Error("'$' belongs to IsIdent and not IsWord")
	}
}

func TestSkipSpace(t *testing.T) {
	for _, c := range []struct {
		s    string
		i    int
		want int
	}{
		{"", 0, 0},
		{"abc", 0, 0},
		{"  \t\r\n x", 0, 6},
		{"a  b", 1, 3},
		{"   ", 0, 3},
		{"a\fb", 1, 1}, // \f and \v are not space here; resolve's own skip takes them
		{"a\vb", 1, 1},
		{"x", 1, 1},
	} {
		if got := SkipSpace(c.s, c.i); got != c.want {
			t.Errorf("SkipSpace(%q, %d) = %d, want %d", c.s, c.i, got, c.want)
		}
	}
}
