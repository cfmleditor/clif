package parser

import "hash/maphash"

// TextFingerprint names a text by its length and two 64-bit hashes of it
// under different seeds, for a memo keyed by a file's text: keyed by the text
// itself, an entry kept the whole file alive for as long as it stood. Two
// texts share a fingerprint with a chance of about one in 2^128.
type TextFingerprint struct {
	n    int
	a, b uint64
}

var fingerprintSeeds = [2]maphash.Seed{maphash.MakeSeed(), maphash.MakeSeed()}

// FingerprintOf is text's TextFingerprint.
func FingerprintOf(text string) TextFingerprint {
	return TextFingerprint{n: len(text), a: maphash.String(fingerprintSeeds[0], text), b: maphash.String(fingerprintSeeds[1], text)}
}
