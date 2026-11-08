package formatter

import (
	"bytes"
	"testing"
)

// A binary file has nothing to format and a UTF-16 file would be rewritten as
// UTF-8; both are refused before any parse, UTF-8 (with or without its mark)
// is not.
func TestSourceErrorRefusesBinaryAndUTF16(t *testing.T) {
	binary := append(bytes.Repeat([]byte("a"), 3000), 0, 'b')
	utf16 := []byte{0xFF, 0xFE, '<', 0, 'c', 0, 'f', 0}

	for name, tc := range map[string]struct {
		src     []byte
		refused bool
	}{
		"plain":        {[]byte("component {}"), false},
		"utf8 mark":    {[]byte("\xef\xbb\xbfcomponent {}"), false},
		"empty":        {nil, false},
		"binary":       {binary, true},
		"utf16 le bom": {utf16, true},
	} {
		if err := SourceError(tc.src); (err != nil) != tc.refused {
			t.Errorf("%s: SourceError = %v, want refused=%v", name, err, tc.refused)
		}
	}
}
