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

// IsRegexpSpace is a regular expression's \s as Go's regexp reads it: space,
// tab, newline, form feed and carriage return, and not vertical tab.
func IsRegexpSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\f' || c == '\r'
}

// SkipRegexpSpace is the index of the first byte of s at or after i that is
// not IsRegexpSpace, or len(s).
func SkipRegexpSpace(s string, i int) int {
	for i < len(s) && IsRegexpSpace(s[i]) {
		i++
	}

	return i
}

// IsQuote reports whether c is a double or a single quote, either of which
// delimits a CFML string or attribute value.
func IsQuote(c byte) bool {
	return c == '"' || c == '\''
}

// IsWordStart reports whether c can begin a name that is not a number: an
// ASCII letter or '_'. It is IsWord without the digits.
func IsWordStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
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

// NormalizeLineBreaks is s with each bare carriage return, one not followed by
// a line feed, made a line feed: s itself, uncopied, when it has none.
//
// The Language Server Protocol ends a line at \r, \n or \r\n, as editors do,
// and so must every line count the analysis makes. They counted \n alone, and
// a file written with old Mac line endings (66 in the corpus) was one line to
// the parser, the line index and tree-sitter: every position the server gave
// in it was wrong. Replacing the byte keeps every offset, so the text is made
// right once, where it is read, rather than every counter taught a second
// rule. A \r\n is left as it is: its \n already ends the line.
func NormalizeLineBreaks(s string) string {
	i := strings.IndexByte(s, '\r')
	for i >= 0 && i+1 < len(s) && s[i+1] == '\n' {
		next := strings.IndexByte(s[i+1:], '\r')
		if next < 0 {
			return s
		}

		i += 1 + next
	}

	if i < 0 {
		return s
	}

	b := []byte(s)
	for j := i; j < len(b); j++ {
		if b[j] == '\r' && (j+1 == len(b) || b[j+1] != '\n') {
			b[j] = '\n'
		}
	}

	return string(b)
}

// CROnly reports whether s ends its lines with bare carriage returns alone:
// it has one, and no line feed. Text NormalizeLineBreaks changed from such a
// file is written back with them.
func CROnly(s string) bool {
	return strings.IndexByte(s, '\r') >= 0 && strings.IndexByte(s, '\n') < 0
}
