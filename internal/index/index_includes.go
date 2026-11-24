package index

import (
	"hash/maphash"
	"slices"
	"strings"
	"sync"

	"go.lsp.dev/uri"
)

// SetIncludes records the paths a file cfincludes, as parser.ExtractIncludes
// returns them, for a door that indexes from a parse result rather than from
// the content. IndexFile records them itself.
func (idx *Index) SetIncludes(fileURI uri.URI, paths []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.setIncludesLocked(fileURI, paths)
}

// setIncludesLocked stores paths for fileURI and moves the generation only when
// they differ from what was held, so an edit that leaves a file's includes
// alone does not make every reader rebuild. The caller holds mu.
func (idx *Index) setIncludesLocked(fileURI uri.URI, paths []string) {
	key := uriKey(fileURI)
	prev, had := idx.includes[key]

	if len(paths) == 0 {
		if had {
			delete(idx.includes, key)
			idx.includeGen++
		}

		return
	}

	if had && slices.Equal(prev.paths, paths) {
		return
	}

	cloned := make([]string, len(paths))
	for i, p := range paths {
		cloned[i] = strings.Clone(p)
	}

	idx.includes[key] = fileIncludes{uri: uri.URI(strings.Clone(string(fileURI))), paths: cloned}
	idx.includeGen++
}

// IncludeGeneration reports a counter that moves whenever any file's recorded
// includes change.
func (idx *Index) IncludeGeneration() uint64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.includeGen
}

// IncludesForFile returns the cfinclude paths one file writes, as
// parser.ExtractIncludes returns them.
func (idx *Index) IncludesForFile(fileURI uri.URI) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return snapshot(idx.includes[uriKey(fileURI)].paths)
}

// ForEachInclude calls fn with every file that cfincludes something and the
// paths it includes, and returns the generation the walk saw. fn must not call
// back into the index.
func (idx *Index) ForEachInclude(fn func(fileURI uri.URI, paths []string)) uint64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	for _, fi := range idx.includes {
		fn(fi.uri, fi.paths)
	}

	return idx.includeGen
}

// uriKeys remembers uriKey's answers. The key is asked for on every index
// lookup, and for a URI with a capital in it (most workspaces) computing it
// decodes and lowercases a new string each time: 13% of a Lucee scan. The
// answer depends only on the URI, so it is shared by every index; when a shard
// holds too many it starts again.
var uriKeys uriKeyMemo

const (
	uriKeyShards   = 64
	maxURIKeyShard = 2048
)

type uriKeyMemo struct {
	seed   maphash.Seed
	once   sync.Once
	shards [uriKeyShards]struct {
		mu sync.RWMutex
		m  map[string]string
	}
}

func (c *uriKeyMemo) key(s string) string {
	c.once.Do(func() { c.seed = maphash.MakeSeed() })

	sh := &c.shards[maphash.String(c.seed, s)%uriKeyShards]

	sh.mu.RLock()
	k, ok := sh.m[s]
	sh.mu.RUnlock()

	if ok {
		return k
	}

	k = uriKeyUncached(s)

	sh.mu.Lock()
	if sh.m == nil || len(sh.m) >= maxURIKeyShard {
		sh.m = make(map[string]string, 256)
	}

	sh.m[s] = k
	sh.mu.Unlock()

	return k
}
