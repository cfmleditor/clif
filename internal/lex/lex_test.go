package lex

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf16"
	"unsafe"
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
		"resolve.callerWord": func(c byte) bool {
			return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '$'
		},
		"server.isNameByte": func(b byte) bool {
			return b == '_' || b == '$' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
		},
		"parser.isLineIdentByte": func(b byte) bool {
			return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '$'
		},
	}

	sameOverEveryByte(t, "IsIdent", IsIdent, former)
}

// sameOverEveryByte fails t at each byte where got and one of former disagree.
func sameOverEveryByte(t *testing.T, helper string, got func(byte) bool, former map[string]func(byte) bool) {
	t.Helper()

	for name, f := range former {
		for b := range 256 {
			if g, want := got(byte(b)), f(byte(b)); g != want {
				t.Errorf("%s differs at %#x: %s=%v", name, b, helper, g)
			}
		}
	}
}

func TestIsWordMatchesEveryFormerCopy(t *testing.T) {
	callerWord := func(c byte) bool {
		return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '$'
	}

	sameOverEveryByte(t, "IsWord", IsWord, map[string]func(byte) bool{
		"resolve.isRegexWordByte": func(c byte) bool {
			return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		},
		"resolve.sourceWord": func(c byte) bool { return callerWord(c) && c != '$' },
		"formatter.isTagNameByte": func(b byte) bool {
			return b >= 'a' && b <= 'z' ||
				b >= 'A' && b <= 'Z' ||
				b >= '0' && b <= '9' ||
				b == '_'
		},
	})
}

func TestIsRegexpSpaceMatchesEveryFormerCopy(t *testing.T) {
	s := regexp.MustCompile(`^\s$`)

	sameOverEveryByte(t, "IsRegexpSpace", IsRegexpSpace, map[string]func(byte) bool{
		"parser.isRegexpSpace": func(c byte) bool {
			return c == ' ' || c == '\t' || c == '\n' || c == '\f' || c == '\r'
		},
		"parser.isRESpace": func(c byte) bool {
			return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
		},
		"resolve.isRegexSpace": func(c byte) bool {
			return c == ' ' || c == '\t' || c == '\n' || c == '\f' || c == '\r'
		},
		"resolve.callerSpace": func(c byte) bool {
			return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
		},
		"resolve.assignmentSpace": func(b byte) bool {
			return b == ' ' || b == '\t' || b == '\r' || b == '\n' || b == '\f'
		},
		`regexp \s`: func(c byte) bool { return s.Match([]byte{c}) },
	})
}

func TestSkipRegexpSpace(t *testing.T) {
	for _, c := range []struct {
		s    string
		i    int
		want int
	}{
		{"", 0, 0},
		{"abc", 0, 0},
		{" \t\r\n\fx", 0, 5},
		{"a\vb", 1, 1}, // \v is not \s to Go's regexp
		{"a  b", 1, 3},
		{"   ", 0, 3},
		{"x", 1, 1},
	} {
		if got := SkipRegexpSpace(c.s, c.i); got != c.want {
			t.Errorf("SkipRegexpSpace(%q, %d) = %d, want %d", c.s, c.i, got, c.want)
		}
	}
}

func TestIsQuoteMatchesEveryFormerCopy(t *testing.T) {
	sameOverEveryByte(t, "IsQuote", IsQuote, map[string]func(byte) bool{
		"parser.isQuote":        func(c byte) bool { return c == '"' || c == '\'' },
		"parser.isQuoteByte":    func(c byte) bool { return c == '"' || c == '\'' },
		"formatter.isQuoteByte": func(c byte) bool { return c == '"' || c == '\'' },
	})
}

func TestIsWordStartMatchesEveryFormerCopy(t *testing.T) {
	sameOverEveryByte(t, "IsWordStart", IsWordStart, map[string]func(byte) bool{
		"parser.isAlphaUnder": func(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b == '_' },
		"parser.isTagNameStart": func(c byte) bool {
			return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
		},
		"route.isNameStart": func(c byte) bool {
			return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
		},
		"IsWord without digits": func(c byte) bool { return IsWord(c) && (c < '0' || c > '9') },
	})
}

func TestIsASCIILetterMatchesEveryFormerCopy(t *testing.T) {
	sameOverEveryByte(t, "IsASCIILetter", IsASCIILetter, map[string]func(byte) bool{
		"parser.isAlpha": func(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' },
	})
}

func TestLowerASCIIMatchesEveryFormerCopy(t *testing.T) {
	plus32 := func(c byte) byte {
		if c >= 'A' && c <= 'Z' {
			return c + 32
		}

		return c
	}

	former := map[string]func(byte) byte{
		// resolve.lowerByte, server.lowerASCIIByte, formatter.toLower and
		// parser.toLowerByte were each this, letter for letter.
		"resolve.lowerByte":     plus32,
		"server.lowerASCIIByte": plus32,
		"formatter.toLower":     plus32,
		"parser.toLowerByte":    plus32,
		"path.foldByte": func(c byte) byte {
			if c >= 'A' && c <= 'Z' {
				return c + ('a' - 'A')
			}

			return c
		},
	}

	for name, f := range former {
		for b := range 256 {
			if got, want := LowerASCII(byte(b)), f(byte(b)); got != want {
				t.Errorf("%s differs at %#x: LowerASCII=%#x", name, b, got)
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

func TestContainsFold(t *testing.T) {
	list := []string{"getUser", "SET", "ß"}

	for _, c := range []struct {
		s    string
		want bool
	}{
		{"getuser", true},
		{"set", true},
		{"SET", true},
		{"ß", true},
		{"get", false},
		{"", false},
	} {
		if got := ContainsFold(list, c.s); got != c.want {
			t.Errorf("ContainsFold(%q) = %v, want %v", c.s, got, c.want)
		}
	}

	if ContainsFold(nil, "x") {
		t.Error("an empty list holds nothing")
	}
}

func TestLowerASCIIAndLetters(t *testing.T) {
	for b := range 256 {
		c := byte(b)

		want := c
		if c >= 'A' && c <= 'Z' {
			want = c + 32
		}

		if got := LowerASCII(c); got != want {
			t.Errorf("LowerASCII(%#x) = %#x, want %#x", c, got, want)
		}

		letter := c < 0x80 && (c|0x20 >= 'a' && c|0x20 <= 'z')
		if got := IsASCIILetter(c); got != letter {
			t.Errorf("IsASCIILetter(%#x) = %v, want %v", c, got, letter)
		}
	}
}

func TestHasPrefixFold(t *testing.T) {
	for _, c := range []struct {
		s, prefix string
		want      bool
	}{
		{"CFINCLUDE x", "cfinclude", true},
		{"cf", "cfinclude", false},
		{"anything", "", true},
		{"", "", true},
		{"", "a", false},
		{"Straße", "STRA", true},
	} {
		if got := HasPrefixFold(c.s, c.prefix); got != c.want || got != (len(c.s) >= len(c.prefix) && strings.EqualFold(c.s[:len(c.prefix)], c.prefix)) {
			t.Errorf("HasPrefixFold(%q, %q) = %v, want %v", c.s, c.prefix, got, c.want)
		}
	}
}

func TestUTF16LenCountsCodeUnits(t *testing.T) {
	for _, s := range []string{"", "abc", "é", "日本語", "a😀b", "😀😀", "\xff"} {
		want := len(utf16.Encode([]rune(s)))
		if got := UTF16Len(s); got != want {
			t.Errorf("UTF16Len(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestNormalizeLineBreaks(t *testing.T) {
	for in, want := range map[string]string{
		"a\rb\rc":     "a\nb\nc",
		"a\r\nb\r\n":  "a\r\nb\r\n",
		"a\r\nb\rc\r": "a\r\nb\nc\n",
		"a\nb":        "a\nb",
		"\r":          "\n",
		"":            "",
	} {
		if got := NormalizeLineBreaks(in); got != want {
			t.Errorf("NormalizeLineBreaks(%q) = %q, want %q", in, got, want)
		}
	}

	// Text with nothing to change is the same string, not a copy.
	s := "line\r\nline\n"
	if got := NormalizeLineBreaks(s); unsafe.StringData(got) != unsafe.StringData(s) {
		t.Error("an unchanged text was copied")
	}

	if !CROnly("a\rb") || CROnly("a\r\nb") || CROnly("ab") {
		t.Error("CROnly")
	}
}
