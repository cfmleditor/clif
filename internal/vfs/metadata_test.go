package vfs

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

type metadataCountingFS struct {
	OS
	stats atomic.Int64
	dirs  atomic.Int64
}

func (c *metadataCountingFS) Stat(path string) (fs.FileInfo, error) {
	c.stats.Add(1)

	return c.OS.Stat(path)
}

func (c *metadataCountingFS) ReadDir(path string) ([]fs.DirEntry, error) {
	c.dirs.Add(1)

	return c.OS.ReadDir(path)
}

func TestMetadataCacheCoalescesConcurrentLookups(t *testing.T) {
	dir := t.TempDir()
	base := &metadataCountingFS{}

	cached := CacheMetadata(base)
	if CacheMetadata(cached) != cached {
		t.Fatal("nested operation discarded cache")
	}

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			if _, err := cached.Stat(dir); err != nil {
				t.Error(err)
			}

			if _, err := cached.ReadDir(dir); err != nil {
				t.Error(err)
			}

			if _, err := cached.Stat(filepath.Join(dir, "absent")); !os.IsNotExist(err) {
				t.Errorf("missing stat: %v", err)
			}
		})
	}

	wg.Wait()

	if base.stats.Load() != 2 || base.dirs.Load() != 1 {
		t.Fatalf("100 repeated lookups: %d stats, %d directory reads", base.stats.Load(), base.dirs.Load())
	}
}

func TestMetadataCacheLifetimeAndListingOwnership(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.cfc")
	base := OS{}

	cached := CacheMetadata(base)
	if _, err := cached.Stat(path); !os.IsNotExist(err) {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("component {}"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := cached.Stat(path); !os.IsNotExist(err) {
		t.Fatal("cache lifetime changed")
	}

	fresh := CacheMetadata(base)
	if _, err := fresh.Stat(path); err != nil {
		t.Fatal(err)
	}

	entries, err := fresh.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries = %v, error = %v", entries, err)
	}

	entries[0] = nil

	again, err := fresh.ReadDir(dir)
	if err != nil || len(again) != 1 || again[0] == nil {
		t.Fatal("caller mutated cached listing")
	}

	if _, err := cached.ReadFile(path); err != nil {
		t.Fatal("file bytes should remain live", err)
	}
}

func BenchmarkMetadataLookups(b *testing.B) {
	dir := b.TempDir()

	for _, cached := range []bool{false, true} {
		name := "direct"

		var fsys FS = OS{}

		if cached {
			name = "cached"
			fsys = CacheMetadata(fsys)
		}

		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				if _, err := fsys.Stat(dir); err != nil {
					b.Fatal(err)
				}

				if _, err := fsys.ReadDir(dir); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
