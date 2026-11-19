package index

import "go.lsp.dev/uri"

// FileRevision moves when a file is indexed or its lines are shifted, but not
// when an unrelated file changes. Zero means the file is not indexed. Revisions
// are never reused after removal/recreation; they use the global write counter.
func (idx *Index) FileRevision(fileURI uri.URI) uint64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.fileRevisions[uriKey(fileURI)]
}

// markFileRevision runs under the write lock, after its counter has advanced.
func (idx *Index) markFileRevision(key string) {
	if idx.fileRevisions == nil {
		idx.fileRevisions = make(map[string]uint64)
	}

	idx.fileRevisions[key] = idx.Generation()
}
