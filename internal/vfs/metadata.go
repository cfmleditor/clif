package vfs

import (
	"io/fs"
	"slices"
	"sync"
)

// CacheMetadata reuses Stat and ReadDir results during a batch operation that
// assumes its input tree does not change. Create a new wrapper for each operation;
// it is not suitable for a long-lived editor filesystem. File bytes and Walk
// remain live, so this is not an atomic filesystem snapshot. Results, including
// errors, are retained for the operation; retry a failed scan with a new wrapper.
// Wrapping an existing cache shares its lifetime (for nested scan passes).
func CacheMetadata(base FS) FS {
	if _, ok := base.(*metadataCache); ok {
		return base
	}

	return &metadataCache{FS: base}
}

type metadataCache struct {
	FS
	stats sync.Map
	dirs  sync.Map
}

func (c *metadataCache) Stat(path string) (fs.FileInfo, error) {
	value, ok := c.stats.Load(path)
	if !ok {
		value, _ = c.stats.LoadOrStore(path, sync.OnceValues(func() (fs.FileInfo, error) {
			return c.FS.Stat(path)
		}))
	}

	lookup, _ := value.(func() (fs.FileInfo, error))

	return lookup()
}

func (c *metadataCache) ReadDir(path string) ([]fs.DirEntry, error) {
	value, ok := c.dirs.Load(path)
	if !ok {
		value, _ = c.dirs.LoadOrStore(path, sync.OnceValues(func() ([]fs.DirEntry, error) {
			entries, err := c.FS.ReadDir(path)

			return slices.Clone(entries), err
		}))
	}

	lookup, _ := value.(func() ([]fs.DirEntry, error))
	entries, err := lookup()
	// Keep callers from changing another lookup's directory listing.
	return slices.Clone(entries), err
}
