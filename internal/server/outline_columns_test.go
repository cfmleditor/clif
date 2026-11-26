package server

import (
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/cfmleditor/clif/internal/parser"
)

// A function's range ends at the end of its last line, in UTF-16 units: with
// several functions on a line, and wide characters or a CRLF in it, each end is
// the line's length counted on its own, as the outline once measured it.
func TestOutlineEndColumnsAreTheLineLength(t *testing.T) {
	for name, src := range map[string]string{
		"one line":        "component {\n" + strings.Repeat("function f(){} ", 5) + "\n}\n",
		"wide characters": "component {\nfunction é(){} function 日本(){} /* 😀 */ function z(){}\n}\n",
		"crlf":            "component {\r\nfunction a(){} function b(){}\r\n  function c(){\r\n  }\r\n}\r\n",
	} {
		syms := buildDocumentSymbols("file:///w/S.cfc", parser.Parse("file:///w/S.cfc", src))
		if len(syms) != 1 {
			t.Fatalf("%s: got %d top-level symbols, want the component", name, len(syms))
		}

		lines := strings.Split(src, "\n")

		for _, m := range syms[0].Children {
			text := strings.TrimSuffix(lines[m.Range.End.Line], "\r")

			if want := uint32(len(utf16.Encode([]rune(text)))); m.Range.End.Character != want {
				t.Errorf("%s: %s ends at column %d, want the length of line %d (%d)", name, m.Name, m.Range.End.Character, m.Range.End.Line, want)
			}
		}
	}
}

// Many functions on one line cost the line once: 16,000 of them took 2.3 s when
// each measured the line again.
func TestOutlineOfManyFunctionsOnOneLineIsLinear(t *testing.T) {
	src := "component {\n" + strings.Repeat("function f(){}", 16000) + "\n}\n"
	pr := parser.Parse("file:///w/Long.cfc", src)

	start := time.Now()
	syms := buildDocumentSymbols("file:///w/Long.cfc", pr)

	if took := time.Since(start); took > time.Second {
		t.Errorf("an outline of 16,000 functions on one line took %v, want well under a second", took)
	}

	if len(syms) != 1 || len(syms[0].Children) != 16000 {
		t.Errorf("got %d symbols, want the component with 16000 methods", len(syms))
	}
}
