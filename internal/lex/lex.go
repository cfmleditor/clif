// Package lex holds the byte-level character classes and the whitespace skip
// that several packages each wrote for themselves. It imports nothing, so the
// dependency-light parser may use it.
//
// Only definitions that were byte-for-byte alike are here. A package whose
// rule differs keeps its own under a name that says how (the server's
// references.go counts every byte above ASCII as part of a word; the
// formatter's skipSpace skips Unicode space; resolve's also skips \f and \v).
package lex

import "strings"

// IsIdent reports whether c can appear in a CFML identifier: a letter, a
// digit, '_' or '$'. It does not say whether c may start one.
func IsIdent(c byte) bool {
	return c == '_' || c == '$' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// IsWord is a regular expression's \w: IsIdent without '$'.
func IsWord(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// SkipSpace is the index of the first byte of s at or after i that is not a
// space, tab, newline or carriage return, or len(s).
func SkipSpace(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}

	return i
}

// ContainsFold reports whether list holds s, ignoring ASCII and Unicode case
// as strings.EqualFold does.
func ContainsFold(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}

	return false
}

// LowerASCII is c with an ASCII capital lowered and every other byte as it is.
func LowerASCII(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}

	return c
}

// IsASCIILetter reports whether c is a-z or A-Z.
func IsASCIILetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// HasPrefixFold reports whether s begins with prefix, ignoring case as
// strings.EqualFold does.
func HasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

// UTF16Len is the length of s in UTF-16 code units, what an LSP character
// counts: two for a rune outside the Basic Multilingual Plane, one otherwise.
func UTF16Len(s string) int {
	n := 0

	for _, r := range s {
		n++

		if r > 0xFFFF {
			n++
		}
	}

	return n
}
