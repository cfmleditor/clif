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
