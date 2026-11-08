package path

import (
	"bytes"
	"testing"
	"unicode/utf16"
)

func utf16Bytes(s string, bigEndian bool, bom bool) []byte {
	var out []byte

	if bom {
		if bigEndian {
			out = append(out, 0xFE, 0xFF)
		} else {
			out = append(out, 0xFF, 0xFE)
		}
	}

	for _, u := range utf16.Encode([]rune(s)) {
		if bigEndian {
			out = append(out, byte(u>>8), byte(u))
		} else {
			out = append(out, byte(u), byte(u>>8))
		}
	}

	return out
}

func TestIsBinaryReadsTheFirstEightKilobytes(t *testing.T) {
	text := bytes.Repeat([]byte("a"), 12000)

	for name, tc := range map[string]struct {
		nulAt  int
		binary bool
	}{
		"none":                   {-1, false},
		"inside the old window":  {100, true},
		"past the old window":    {600, true}, // a 512-byte check called this text
		"near the end of window": {7990, true},
		"past the window":        {9000, false},
	} {
		data := bytes.Clone(text)
		if tc.nulAt >= 0 {
			data[tc.nulAt] = 0
		}

		if got := IsBinary(data); got != tc.binary {
			t.Errorf("%s: IsBinary = %v, want %v", name, got, tc.binary)
		}

		if _, ok := AsText(data); ok == tc.binary {
			t.Errorf("%s: AsText ok = %v, want %v", name, ok, !tc.binary)
		}
	}
}

func TestUTF16WithAByteOrderMarkIsText(t *testing.T) {
	const src = "<cfset x = 1>\nsvc.save( é )"

	for name, data := range map[string][]byte{
		"little endian": utf16Bytes(src, false, true),
		"big endian":    utf16Bytes(src, true, true),
	} {
		if IsBinary(data) {
			t.Errorf("%s: taken for binary", name)
		}

		got, ok := AsText(data)
		if !ok || string(got) != src {
			t.Errorf("%s: AsText = %q, %v, want the text decoded", name, got, ok)
		}
	}

	// Without a mark it is not told from binary: every other byte is a NUL.
	if !IsBinary(utf16Bytes(src, false, false)) {
		t.Error("UTF-16 without a byte-order mark should read as binary")
	}

	// A mark in front of data that is not text at that width is binary.
	if _, ok := AsText(append([]byte{0xFF, 0xFE}, 'a', 0, 0, 0, 'b', 0)); ok {
		t.Error("a decoded NUL should make the file binary")
	}

	// Plain UTF-8, with or without its mark, is returned untouched.
	plain := []byte("\xef\xbb\xbfcomponent {}")
	if got, ok := AsText(plain); !ok || !bytes.Equal(got, plain) {
		t.Errorf("UTF-8 changed: %q, %v", got, ok)
	}
}
