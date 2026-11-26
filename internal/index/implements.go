package index

import (
	"slices"
	"strings"

	"go.lsp.dev/uri"
)

// Implementer is a file and the interfaces its implements attribute names, as
// the index records them.
type Implementer struct {
	URI        string
	Interfaces []string
}

// setImplementsLocked records what the file keyed key implements, cloning the
// names; idx.mu is held. An empty list leaves no entry.
func (idx *Index) setImplementsLocked(key string, names []string) {
	if len(names) == 0 {
		delete(idx.implements, key)

		return
	}

	cloned := make([]string, len(names))
	for i, n := range names {
		cloned[i] = strings.Clone(n)
	}

	idx.implements[key] = cloned
}

// SetImplements records the interfaces a file implements, for a caller that
// indexed it from funcs and refs alone (IndexFileFromResult forgets them, as it
// does the file's extends); the doors that have a parse record them with the
// rest of the file. Re-indexing a file drops the record.
func (idx *Index) SetImplements(fileURI uri.URI, names []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.setImplementsLocked(uriKey(fileURI), names)
}

// ImplementsForFile returns the interfaces a file's implements attribute names,
// as the index recorded them.
func (idx *Index) ImplementsForFile(fileURI uri.URI) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return slices.Clone(idx.implements[uriKey(fileURI)])
}

// ImplementersByName groups every file that declares an implements attribute
// by the last segment of each interface it names, lowercased, each group
// sorted by URI. Only files with an implements attribute are held, so the pass
// is over those and not over every component. The caller decides whether each
// name really means the interface it is after.
func (idx *Index) ImplementersByName() map[string][]Implementer {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	out := map[string][]Implementer{}

	for key, names := range idx.implements {
		u, ok := idx.fileURIs[key]
		if !ok {
			continue
		}

		seen := map[string]bool{}

		for _, n := range names {
			last := n
			if _, after, found := strings.CutLast(n, "."); found {
				last = after
			}

			last = strings.ToLower(last)
			if seen[last] {
				continue
			}

			seen[last] = true
			out[last] = append(out[last], Implementer{URI: string(u), Interfaces: names})
		}
	}

	for _, files := range out {
		slices.SortFunc(files, func(a, b Implementer) int { return strings.Compare(a.URI, b.URI) })
	}

	return out
}
