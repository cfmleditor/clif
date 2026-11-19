package index

import "go.lsp.dev/uri"

// DiskRevision tracks disk indexing. Buffer edits and line shifts leave it
// unchanged. Zero means disk indexing has not established a revision.
func (idx *Index) DiskRevision(fileURI uri.URI) uint64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.diskRevisions[uriKey(fileURI)]
}

// markDiskRevision runs under the write lock, after its counter has advanced.
func (idx *Index) markDiskRevision(key string) {
	if idx.diskRevisions == nil {
		idx.diskRevisions = make(map[string]uint64)
	}

	idx.diskRevisions[key] = idx.Generation()
}
