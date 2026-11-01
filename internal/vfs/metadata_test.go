package vfs

import (
	"errors"
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
		t.Fatal("reading through the cache failed", err)
	}
}

type readCountingFS struct {
	OS
	reads atomic.Int64
}

func (c *readCountingFS) ReadFile(path string) ([]byte, error) {
	c.reads.Add(1)

	return c.OS.ReadFile(path)
}

// TestMetadataCacheKeepsFileBytesWithinItsBudget: a batch scan re-reads the
// same sources many times, so the cache keeps their bytes, hands each caller
// its own copy, and stops keeping once the budget is spent.
func TestMetadataCacheKeepsFileBytesWithinItsBudget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.cfc")

	if err := os.WriteFile(path, []byte("component {}"), 0o600); err != nil {
		t.Fatal(err)
	}

	base := &readCountingFS{}
	cached := CacheMetadata(base)

	for range 10 {
		data, err := cached.ReadFile(path)
		if err != nil || string(data) != "component {}" {
			t.Fatalf("ReadFile = %q, %v", data, err)
		}

		data[0] = 'X'
	}

	if base.reads.Load() != 1 {
		t.Errorf("10 reads of one file reached the disk %d times, want 1", base.reads.Load())
	}

	full, ok := cached.(*metadataCache)
	if !ok {
		t.Fatal("not a metadata cache")
	}

	full.held.Store(contentBudget)

	other := filepath.Join(dir, "b.cfc")
	if err := os.WriteFile(other, []byte("component {}"), 0o600); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if _, err := cached.ReadFile(other); err != nil {
			t.Fatal(err)
		}
	}

	if base.reads.Load() != 3 {
		t.Errorf("a file read past the budget was kept: %d disk reads, want 3", base.reads.Load())
	}
}

func (c *readCountingFS) ReadFileString(path string) (string, error) {
	c.reads.Add(1)

	return c.OS.ReadFileString(path)
}

// TestMetadataCacheRemembersAMissingFile: discovery probes the same absent
// files from every application, so a read that found no file is answered
// from the cache, by ReadFile and ReadFileString alike. A read that failed
// for any other reason (here, a directory) still reaches the disk.
func TestMetadataCacheRemembersAMissingFile(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "server.json")

	base := &readCountingFS{}
	cached := CacheMetadata(base)

	for range 5 {
		if _, err := cached.ReadFile(missing); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("ReadFile of a missing file: %v", err)
		}

		if _, err := ReadString(cached, missing); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("ReadFileString of a missing file: %v", err)
		}
	}

	if got := base.reads.Load(); got != 1 {
		t.Errorf("10 reads of a missing file reached the disk %d times, want 1", got)
	}

	for range 3 {
		if _, err := cached.ReadFile(dir); err == nil {
			t.Fatal("reading a directory succeeded")
		}
	}

	if got := base.reads.Load(); got != 4 {
		t.Errorf("a read failing for another reason was remembered: %d disk reads, want 4", got)
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
