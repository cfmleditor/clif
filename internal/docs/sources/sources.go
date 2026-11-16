// Package sources reads internal/docs/generated_sources.txt, the record of
// which doc source each generated entry came from. scripts/generate_docs.go
// compares a regeneration with the committed copy, and the internal/docs
// tests check the file against the generated entries; both parse it here, so
// the two cannot disagree about what a valid line is.
//
// It imports nothing from internal/docs, which the generator writes: a broken
// or missing generated file must not stop the generator from running.
package sources

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

const (
	// OrderPrefix starts the comment line that records the source order the
	// file was generated under, lowest precedence first.
	OrderPrefix = "# order: "

	// Retained is the committed source, docs/retained: entries an upstream
	// stopped publishing, kept on purpose.
	Retained = "retained"
)

// File is a parsed sources file.
type File struct {
	// Entries maps an entry key (Key) to the source it came from.
	Entries map[string]string
	// Order is the source order, lowest precedence first; nil when the file
	// has no order line.
	Order []string
}

// Key identifies an entry: a function and a tag may share a name, and losing
// one is not made up for by keeping the other.
func Key(typ, name string) string {
	return typ + " " + strings.ToLower(name)
}

// Lines splits b into lines without a trailing "\r": a checkout with
// core.autocrlf (Windows) writes CRLF unless .gitattributes says otherwise.
func Lines(b []byte) []string {
	out := strings.Split(string(b), "\n")
	for i, l := range out {
		out[i] = strings.TrimSuffix(l, "\r")
	}

	return out
}

// Parse reads a sources file: "function|tag name<TAB>source" lines, blank
// lines and # comments aside, and at most one order line. It is strict,
// because a line it skipped would leave a baseline smaller without a word: a
// bad merge would quietly unprotect every entry on the lines it mangled. name
// labels the errors.
func Parse(name string, b []byte) (File, error) {
	f := File{Entries: map[string]string{}}

	for i, line := range Lines(b) {
		if rest, ok := strings.CutPrefix(line, OrderPrefix); ok {
			if f.Order != nil {
				return File{}, fmt.Errorf("%s:%d: a second order line", name, i+1)
			}

			f.Order = strings.Fields(rest)
			if len(f.Order) == 0 {
				return File{}, fmt.Errorf("%s:%d: an order line naming no sources", name, i+1)
			}

			continue
		}

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, src, ok := strings.Cut(line, "\t")
		typ, entry, _ := strings.Cut(key, " ")

		// A name is an identifier: none is empty or holds whitespace. One
		// with trailing whitespace ("function abs ") would never match the
		// real key, and quietly drop it from the baseline.
		if !ok || src == "" || !ValidName(entry) || (typ != "function" && typ != "tag") {
			return File{}, fmt.Errorf("%s:%d: want \"function|tag name<TAB>source\", have %q", name, i+1, line)
		}

		// Keys are written lowercased (Key), and compared that way: a
		// mixed-case one would never match, read as a removal nobody can
		// make, and slip past the duplicate check beside its lowercase twin.
		if key != Key(typ, entry) {
			return File{}, fmt.Errorf("%s:%d: %q is not lowercase; want %q", name, i+1, key, Key(typ, entry))
		}

		if _, dup := f.Entries[key]; dup {
			return File{}, fmt.Errorf("%s:%d: %q listed twice", name, i+1, key)
		}

		f.Entries[key] = src
	}

	return f, nil
}

// ValidName reports whether name can be an entry's name: not empty, and with
// no whitespace or control character. The generator refuses (or skips) an
// entry whose name is not, and Parse a line holding one, by this one rule, so
// the generator can never write a line Parse would reject.
func ValidName(name string) bool {
	return name != "" && !strings.ContainsFunc(name, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	})
}

// Rank numbers a source by its place in order, lowest precedence first, from
// 1; 0 when order does not name it.
func Rank(order []string, source string) int {
	return slices.Index(order, source) + 1
}
