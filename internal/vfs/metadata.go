package vfs

import (
	"errors"
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
// file read after the budget is spent stays live. A file that does not exist
// is remembered as missing, as Stat remembers it: discovery probes the same
// absent server.json, .cfconfig.json and framework markers in every directory
// up to the root for each application it looks at, 70,000 failed opens in
// one tassweb scan. Any other failed read stays live.
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
	missing  sync.Map // path -> the fs.ErrNotExist error its read returned
	held     atomic.Int64
}

const (
	contentFileLimit = 2 << 20
	// contentBudget holds the whole of tassweb's 141MB of source: at 128MB the
	// files read last were opened again by every lookup that wanted them, and
	// the larger budget took a scan 0.5s of CPU and about 0.1s of wall time
	// less for 59MB more resident memory.
	contentBudget = 256 << 20
)

// ReadFile returns a copy of the cached bytes, so a caller cannot change
// another reader's file.
func (c *metadataCache) ReadFile(path string) ([]byte, error) {
	if held, ok := c.contents.Load(path); ok {
		text, _ := held.(string)

		return []byte(text), nil
	}

	if err := c.missingErr(path); err != nil {
		return nil, err
	}

	data, err := c.FS.ReadFile(path)
	if err != nil || len(data) > contentFileLimit {
		c.noteMissing(path, err)

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

	if err := c.missingErr(path); err != nil {
		return "", err
	}

	// Through the base's own ReadFileString where it has one (vfs.OS's
	// hands over the bytes it read without copying them).
	text, err := ReadString(c.FS, path)
	if err != nil {
		c.noteMissing(path, err)

		return "", err
	}

	if len(text) > contentFileLimit {
		return text, nil
	}

	if c.held.Add(int64(len(text))) > contentBudget {
		c.held.Add(-int64(len(text)))

		return text, nil
	}

	if held, loaded := c.contents.LoadOrStore(path, text); loaded {
		c.held.Add(-int64(len(text)))
		text, _ = held.(string)
	}

	return text, nil
}

// missingErr is the error an earlier read of path returned when the file did
// not exist, or nil.
func (c *metadataCache) missingErr(path string) error {
	held, ok := c.missing.Load(path)
	if !ok {
		return nil
	}

	err, _ := held.(error)

	return err
}

func (c *metadataCache) noteMissing(path string, err error) {
	if errors.Is(err, fs.ErrNotExist) {
		c.missing.Store(path, err)
	}
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
