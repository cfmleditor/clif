package vfs

import (
	"io/fs"
	"slices"
	"sync"
	"sync/atomic"
)

// CacheMetadata reuses Stat and ReadDir results during a batch operation that
// assumes its input tree does not change. Create a new wrapper for each operation;
// it is not suitable for a long-lived editor filesystem. Walk remains live, so
// this is not an atomic filesystem snapshot. Results, including errors, are
// retained for the operation; retry a failed scan with a new wrapper. Wrapping
// an existing cache shares its lifetime (for nested scan passes).
//
// File bytes are kept too, within a budget (contentFileLimit per file,
// contentBudget in all, first read first kept): resolution re-reads the same
// sources many times over (a framework's bindings, every caller of an
// inferred argument), and those reads were a fifth of a Wheels scan's CPU. A
// file read after the budget is spent, or a failed read, stays live.
func CacheMetadata(base FS) FS {
	if _, ok := base.(*metadataCache); ok {
		return base
	}

	return &metadataCache{FS: base}
}

type metadataCache struct {
	FS
	stats    sync.Map
	dirs     sync.Map
	contents sync.Map // path -> string, within contentBudget
	held     atomic.Int64
}

const (
	contentFileLimit = 2 << 20
	contentBudget    = 128 << 20
)

// ReadFile returns a copy of the cached bytes, so a caller cannot change
// another reader's file.
func (c *metadataCache) ReadFile(path string) ([]byte, error) {
	if held, ok := c.contents.Load(path); ok {
		text, _ := held.(string)

		return []byte(text), nil
	}

	data, err := c.FS.ReadFile(path)
	if err != nil || len(data) > contentFileLimit {
		return data, err
	}

	if c.held.Add(int64(len(data))) > contentBudget {
		c.held.Add(-int64(len(data)))

		return data, nil
	}

	if _, loaded := c.contents.LoadOrStore(path, string(data)); loaded {
		c.held.Add(-int64(len(data)))
	}

	return data, nil
}

// ReadFileString is ReadFile's text, without the copy ReadFile makes: a Go
// string cannot be changed, so readers can share the cached one.
func (c *metadataCache) ReadFileString(path string) (string, error) {
	if held, ok := c.contents.Load(path); ok {
		text, _ := held.(string)

		return text, nil
	}

	data, err := c.FS.ReadFile(path)
	if err != nil {
		return "", err
	}

	text := string(data)
	if len(data) > contentFileLimit {
		return text, nil
	}

	if c.held.Add(int64(len(data))) > contentBudget {
		c.held.Add(-int64(len(data)))

		return text, nil
	}

	if held, loaded := c.contents.LoadOrStore(path, text); loaded {
		c.held.Add(-int64(len(data)))
		text, _ = held.(string)
	}

	return text, nil
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
