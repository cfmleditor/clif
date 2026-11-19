package resolve

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

type sourceStatCounter struct {
	vfs.OS
	stats atomic.Int64
}

// Model one edit followed by return inference through many unchanged libraries.
// The check count is deterministic; timings are compared with an isolated base.
func BenchmarkSourceMetadataAfterUnrelatedEdit(b *testing.B) {
	dir := b.TempDir()
	idx := index.New()
	fsys := &sourceStatCounter{}
	r := &Resolver{FS: fsys, Index: idx, IndexTracksFiles: true}

	var files []string

	for i := range 32 {
		file := filepath.Join(dir, fmt.Sprintf("Library%d.cfc", i))
		if err := os.WriteFile(file, []byte("component {}"), 0o600); err != nil {
			b.Fatal(err)
		}

		idx.IndexFile(cfpath.ToURI(file), "component {}")
		files = append(files, file)
	}

	for _, file := range files {
		r.sourceInfo(file)
	}

	edited := cfpath.ToURI(filepath.Join(dir, "Edited.cfc"))

	fsys.stats.Store(0)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		idx.IndexFileFromResult(edited, nil, nil)

		for _, file := range files {
			r.sourceInfo(file)
		}
	}

	b.ReportMetric(float64(fsys.stats.Load())/float64(b.N), "stats/op")
}

func (c *sourceStatCounter) Stat(path string) (fs.FileInfo, error) {
	c.stats.Add(1)

	return c.OS.Stat(path)
}

func TestSourceMetadataSurvivesUnrelatedIndexWrites(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Library.cfc")

	const text = `component {function work(){return this;}}`
	if err := os.WriteFile(file, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}

	idx := index.New()
	idx.IndexFile(cfpath.ToURI(file), text)

	fsys := &sourceStatCounter{}
	r := &Resolver{FS: fsys, Index: idx, IndexTracksFiles: true}
	first := r.sourceInfo(file)

	for range 20 {
		idx.IndexFileFromResult(cfpath.ToURI(filepath.Join(dir, "Edited.cfc")), nil, nil)

		if next := r.sourceInfo(file); next.content != text || next.derived != first.derived {
			t.Fatal("unrelated edit replaced unchanged source facts")
		}
	}

	if got := fsys.stats.Load(); got != 1 {
		t.Fatalf("unchanged library checked %d times; want one check across unrelated edits", got)
	}
}

func TestSourceMetadataRefreshesChangedDeletedAndReplacedFiles(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Library.cfc")
	idx := index.New()
	fsys := &sourceStatCounter{}
	r := &Resolver{FS: fsys, Index: idx, IndexTracksFiles: true}
	write := func(text string) {
		t.Helper()

		if err := os.WriteFile(file, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}

		idx.IndexFile(cfpath.ToURI(file), text)
	}

	write(`component {function a(){}}`)

	first := r.sourceInfo(file)

	write(`component {function changed(){return this;}}`)

	if next := r.sourceInfo(file); next.content == first.content || next.derived == first.derived {
		t.Fatal("changed source reused old facts")
	}

	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}

	idx.RemoveFile(cfpath.ToURI(file))

	if gone := r.sourceInfo(file); gone.content != "" || gone.derived != nil {
		t.Fatal("deleted source reused old facts")
	}

	const recreated = `component {function recreated(){}}`
	write(recreated)

	if next := r.sourceInfo(file); next.content != recreated {
		t.Fatal("recreated source reused old facts")
	}

	// Watched replacements invalidate source caches even if the replacement has
	// the same size and timestamp; a metadata stamp alone cannot detect those.
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}

	const replacement = `component {function replacedx(){}}`
	if len(recreated) != len(replacement) {
		t.Fatal("replacement fixture must have the same size")
	}

	other := filepath.Join(dir, "replacement.cfc")
	if err := os.WriteFile(other, []byte(replacement), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chtimes(other, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(other, file); err != nil {
		t.Fatal(err)
	}

	r.InvalidatePaths()

	if next := r.sourceInfo(file); next.content != replacement {
		t.Fatal("watched replacement reused old facts")
	}
}

func TestSourceMetadataWithoutIndexCoverageRemainsLive(t *testing.T) {
	for _, indexed := range []bool{false, true} {
		dir := t.TempDir()

		file := filepath.Join(dir, "Library.cfc")
		if err := os.WriteFile(file, []byte("component {}"), 0o600); err != nil {
			t.Fatal(err)
		}

		idx := index.New()
		fsys := &sourceStatCounter{}
		r := &Resolver{FS: fsys, Index: idx, IndexTracksFiles: indexed}
		r.sourceInfo(file)
		idx.IndexFileFromResult(cfpath.ToURI(filepath.Join(dir, "Other.cfc")), nil, nil)
		r.sourceInfo(file)

		if got := fsys.stats.Load(); got != 2 {
			t.Fatalf("IndexTracksFiles=%v: unindexed input checked %d times, want 2", indexed, got)
		}
	}
}

func TestSourceMetadataSurvivesEditsToItsBuffer(t *testing.T) {
	const text = "component {function original(){}}"

	file := filepath.Join(t.TempDir(), "Edited.cfc")
	if err := os.WriteFile(file, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}

	u := cfpath.ToURI(file)
	idx := index.New()
	idx.IndexFile(u, text)

	fsys := &sourceStatCounter{}
	r := &Resolver{FS: fsys, Index: idx, IndexTracksFiles: true}
	before := r.sourceInfo(file)

	for range 20 {
		idx.IndexBufferFromResult(u, nil, nil)
		idx.ShiftLines(u, 0, 1)

		if after := r.sourceInfo(file); after.content != text || after.derived != before.derived {
			t.Fatal("buffer edit changed the disk source")
		}
	}

	if got := fsys.stats.Load(); got != 1 {
		t.Fatalf("disk source checked %d times; want one check across all buffer edits", got)
	}
}

type sourceReadHook struct {
	vfs.OS
	afterRead func()
}

func (f *sourceReadHook) ReadFileString(path string) (string, error) {
	text, err := f.OS.ReadFileString(path)
	if f.afterRead != nil {
		f.afterRead()
	}

	return text, err
}

func TestInvalidatedSourceReadCannotRefillCache(t *testing.T) {
	const before = `component { function mode(value="self") { if (arguments.value eq "query") { return q; } else { return this; } } }`

	after := strings.Replace(before, `"self"`, `"none"`, 1)

	file := filepath.Join(t.TempDir(), "Changed.cfc")
	if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}

	u := cfpath.ToURI(file)
	idx := index.New()
	idx.IndexFile(u, before)

	fsys := &sourceReadHook{}
	r := &Resolver{FS: fsys, Index: idx, IndexTracksFiles: true}
	fsys.afterRead = func() {
		fsys.afterRead = nil

		if err := os.WriteFile(file, []byte(after), 0o600); err != nil {
			t.Fatal(err)
		}

		r.InvalidatePaths()
	}

	fd := &parser.FunctionDef{URI: u, Name: "mode", ReturnComponent: "$builtin.struct"}
	if first := r.ReturnComponentOf(fd); first != "" {
		t.Fatalf("in-flight return = %q, want the old dispatch's unknown return", first)
	}

	if got := r.sourceInfo(file).content; got != after {
		t.Fatalf("source after invalidation = %q, want %q", got, after)
	}

	if got := r.ReturnComponentOf(fd); got != "$builtin.struct" {
		t.Fatalf("return after invalidation = %q, want $builtin.struct", got)
	}
}
