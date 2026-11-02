package parser

import "testing"

// The identifier tables must say what the range tests they replaced said, for
// every byte.
func TestIdentTablesMatchTheRangeTests(t *testing.T) {
	for i := range 256 {
		ch := byte(i)
		start := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || ch == '$'
		part := start || (ch >= '0' && ch <= '9')

		if isIdentStart(ch) != start || isIdentPart(ch) != part {
			t.Errorf("byte %#x: start %v part %v, want %v %v", ch, isIdentStart(ch), isIdentPart(ch), start, part)
		}
	}
}

// An identifier ends at the first byte that cannot continue one, at the end of
// the source, and scanning it leaves the scanner just past it.
func TestScanIdentStopsWhereAnIdentifierEnds(t *testing.T) {
	for _, tc := range []struct{ src, first, rest string }{
		{"abc", "abc", ""},
		{"a1$_b;", "a1$_b", ";"},
		{"$x.y", "$x", ".y"},
		{"_9(", "_9", "("},
		{"a", "a", ""},
		{"ab\ncd", "ab", "\ncd"},
		{"é1", "", ""},
	} {
		sc := NewScanner(tc.src)

		tok := sc.NextSkipComments()
		if tc.first == "" {
			if tok.Kind == TokIdent {
				t.Errorf("%q: scanned an identifier %q from a byte that cannot start one", tc.src, tok.Value)
			}

			continue
		}

		if tok.Kind != TokIdent || tok.Value != tc.first || tok.Offset != 0 {
			t.Errorf("%q: token %v %q at %d, want identifier %q at 0", tc.src, tok.Kind, tok.Value, tok.Offset, tc.first)
		}

		if got := sc.Rest(); got != tc.rest {
			t.Errorf("%q: rest %q, want %q", tc.src, got, tc.rest)
		}
	}
}
