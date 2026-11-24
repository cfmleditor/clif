package index

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

// FunctionsForFile returns all indexed function definitions for a specific file.
func (idx *Index) FunctionsForFile(fileURI uri.URI) []*parser.FunctionDef {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return snapshot(idx.fileFuncs[uriKey(fileURI)])
}

// FunctionInFile is the first of fileURI's definitions named name,
// case-insensitively, as a walk of FunctionsForFile finds it, without the
// copy: a component's methods are looked up one name at a time, and copying
// and walking every definition in the file for each was quadratic in a
// component that calls most of its own methods. It walks the shorter of the
// file's list and the name's bucket.
func (idx *Index) FunctionInFile(fileURI uri.URI, name string) *parser.FunctionDef {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	fk := uriKey(fileURI)

	file := idx.fileFuncs[fk]
	if len(file) == 0 {
		return nil
	}

	if bucket := idx.funcs[strings.ToLower(name)]; len(bucket) < len(file) {
		var found *parser.FunctionDef

		n := 0

		for _, d := range bucket {
			if sameFileURI(d.URI, file[0].URI, fk) {
				found = d
				n++
			}
		}

		// Two of the file's: which comes first is the file's order.
		if n < 2 {
			return found
		}
	}

	for _, d := range file {
		if strings.EqualFold(d.Name, name) {
			return d
		}
	}

	return nil
}

// sameFileURI reports whether u names the file whose index key is fk, given
// u0, one URI that file was indexed under. Spelled alike or differing only in
// case it is decided without allocating; it is unescaped only when one of the
// two is escaped.
func sameFileURI(u, u0 uri.URI, fk string) bool {
	if u == u0 || strings.EqualFold(string(u), string(u0)) {
		return true
	}

	escaped := strings.ContainsRune(string(u), '%') || strings.ContainsRune(string(u0), '%')

	return escaped && keyMatches(string(u), fk)
}

// keyMatches is uriKey(u) == fk, decoding and lowering u as it goes rather
// than building its key. A decoded byte beyond ASCII, where strings.ToLower
// may change more than one byte, and a malformed escape go to uriKey itself.
func keyMatches(u, fk string) bool {
	// PathUnescape refuses the whole URI over one malformed escape, and then
	// uriKey compares it undecoded, so decoding any of it would disagree.
	for i := strings.IndexByte(u, '%'); i >= 0; i = nextEscape(u, i) {
		if _, ok := unhexAt(u, i+1); !ok {
			return uriKey(uri.URI(u)) == fk
		}

		if _, ok := unhexAt(u, i+2); !ok {
			return uriKey(uri.URI(u)) == fk
		}
	}

	j := 0

	for i := 0; i < len(u); i++ {
		c := u[i]

		if c == '%' {
			hi, _ := unhexAt(u, i+1)
			lo, _ := unhexAt(u, i+2)
			c = hi<<4 | lo
			i += 2
		}

		if c >= 0x80 {
			return uriKey(uri.URI(u)) == fk
		}

		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}

		if j >= len(fk) || fk[j] != c {
			return false
		}

		j++
	}

	return j == len(fk)
}

// nextEscape is the index of the next '%' after the escape at i, or -1.
func nextEscape(u string, i int) int {
	if n := strings.IndexByte(u[min(i+3, len(u)):], '%'); n >= 0 {
		return min(i+3, len(u)) + n
	}

	return -1
}

// unhexAt is the value of the hex digit at u[i], and whether there is one.
func unhexAt(u string, i int) (byte, bool) {
	if i >= len(u) {
		return 0, false
	}

	switch c := u[i]; {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}

	return 0, false
}

// HasFile reports whether the index holds an entry for this file, which is not
// the same question as whether the file declares any functions: a component
// with only properties or `this` assignments indexes to an empty — but present
// — entry, and callers that use "no functions" as a stand-in for "never
// indexed" re-read such a file from disk on every lookup.
func (idx *Index) HasFile(fileURI uri.URI) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	_, ok := idx.fileFuncs[uriKey(fileURI)]

	return ok
}

// ComponentFiles returns the real-case URIs of every indexed .cfc file, sorted.
func (idx *Index) ComponentFiles() []uri.URI {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var out []uri.URI

	for key, u := range idx.fileURIs {
		if strings.HasSuffix(key, ".cfc") {
			out = append(out, u)
		}
	}

	slices.Sort(out)

	return out
}

// ShiftLines adjusts line numbers for all entries in a file where Line > afterLine.
//
// It replaces the affected entries rather than writing through the pointers it
// handed out. Every accessor here returns those pointers and then releases the
// read lock, so a caller reading def.Line is doing so unsynchronised; writing
// the field in place raced it, which `go test -race` reports as a write in
// ShiftLines against the caller's read. A caller cannot hold the index lock for
// as long as it holds the entry, so the write has to move instead.
//
// The replacement is threaded through all four maps at once: funcs and
// fileFuncs hold the same pointers, as do comprefs and fileRefs, and updating
// one without the other would leave a file's entries disagreeing about where
// its functions are.
//
// A caller that captured an entry before the shift keeps reading the old line
// number, which is what it would have read had it looked a moment earlier. It
// is a snapshot, not a torn value.
func (idx *Index) ShiftLines(fileURI uri.URI, afterLine int, delta int) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	key := uriKey(fileURI)

	// Only the edited file's entries can move, and fileFuncs/fileRefs already
	// index exactly those. This used to walk every bucket of funcs and comprefs
	// to find them and then rebuild all five maps wholesale — on a 5,000-file
	// workspace, 18ms of write-locked work per line-changing keystroke,
	// blocking every other session's lookups for the duration. Measured by
	// BenchmarkShiftLines, that becomes 6µs where method names differ between
	// files.
	//
	// The replacement is then written into the affected buckets in place. What
	// used to remain proportional to workspace size was exactly that step:
	// funcs is keyed by lowercased name, so a workspace where every component
	// declares `init` has one bucket holding an entry per file, and swapping
	// this file's single pointer in it meant rebuilding all 5,000 — because
	// callers held the bucket slice after the read lock was released. Accessors
	// now hand out a copy (see snapshot), so the bucket is the index's alone to
	// write and the worst case is flat rather than ~0.9ms.
	//
	// The entry itself is still replaced rather than written through: a caller
	// holding a *FunctionDef is reading def.Line unsynchronised, so the struct
	// has to be new even though the slot holding it need not be.
	// Replacements are collected per bucket, for the reason removeFileEntries
	// gives: a bucket sweep should only consider the entries that could be in
	// that bucket. Pooling them into one map keyed by pointer made every entry
	// of a 5,000-entry `init` bucket pay a hash and a probe to be told it was
	// not one of this file's few.
	// Slab-allocate shifted copies once per call rather than once per moved
	// function/ref, and update both per-file slices and name buckets in place
	// without intermediate per-name maps or strings.ToLower allocations.
	fileFuncs := idx.fileFuncs[key]
	nShiftedFuncs := 0

	for _, d := range fileFuncs {
		if int(d.Line) > afterLine || (d.HasNameRange() && int(d.NameLine) > afterLine) {
			nShiftedFuncs++
		}
	}

	if nShiftedFuncs > 0 {
		idx.funcGen++

		slab := make([]parser.FunctionDef, nShiftedFuncs)
		si := 0

		var buf foldScratch

		for i, d := range fileFuncs {
			shiftLine, shiftName := int(d.Line) > afterLine, d.HasNameRange() && int(d.NameLine) > afterLine
			if !shiftLine && !shiftName {
				continue
			}

			shifted := &slab[si]
			si++
			*shifted = *d

			if shiftLine {
				shifted.Line = conv.Uint32(int(shifted.Line) + delta)
			}

			if shiftName {
				shifted.NameLine = conv.Uint32(int(shifted.NameLine) + delta)
			}

			fileFuncs[i] = shifted

			var bucket []*parser.FunctionDef
			if b, ok := buf.lowerASCII(d.Name); ok {
				bucket = idx.funcs[string(b)]
			} else {
				bucket = idx.funcs[strings.ToLower(d.Name)]
			}

			if j := slices.Index(bucket, d); j >= 0 {
				bucket[j] = shifted
			}
		}
	}

	fileRefs := idx.fileRefs[key]
	nShiftedRefs := 0

	for _, r := range fileRefs {
		if int(r.Line) > afterLine {
			nShiftedRefs++
		}
	}

	if nShiftedRefs == 0 {
		return
	}

	refSlab := make([]parser.ComponentRef, nShiftedRefs)
	ri := 0

	scopeMap := idx.scopeRefs[key]

	var fileRefReps []refReplacement
	if len(scopeMap) > 0 {
		fileRefReps = make([]refReplacement, 0, nShiftedRefs)
	}

	var buf foldScratch

	for i, r := range fileRefs {
		if int(r.Line) <= afterLine {
			continue
		}

		shifted := &refSlab[ri]
		ri++
		*shifted = *r
		shifted.Line = conv.Uint32(int(shifted.Line) + delta)

		fileRefs[i] = shifted

		var bucket []*parser.ComponentRef
		if b, ok := buf.lowerASCII(r.Variable); ok {
			bucket = idx.comprefs[string(b)]
		} else {
			bucket = idx.comprefs[strings.ToLower(r.Variable)]
		}

		if j := slices.Index(bucket, r); j >= 0 {
			bucket[j] = shifted
		}

		if len(scopeMap) > 0 {
			fileRefReps = append(fileRefReps, refReplacement{old: r, new: shifted})
		}
	}

	for _, refs := range scopeMap {
		remap(refs, fileRefReps)
	}
}

// foldScratch folds an ASCII identifier to lowercase on the stack so map reads
// and deletes (`m[string(b)]`) do not allocate.
type foldScratch [64]byte

func (f *foldScratch) lowerASCII(s string) ([]byte, bool) {
	if len(s) > len(f) {
		return nil, false
	}

	for i := range len(s) {
		c := s[i]
		if c >= 0x80 {
			return nil, false
		}

		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}

		f[i] = c
	}

	return f[:len(s)], true
}

// replacement pairs an entry with the shifted copy that supersedes it.
type replacement[T comparable] struct {
	old, new T
}

type refReplacement = replacement[*parser.ComponentRef]

// remap substitutes replaced entries into a stored slice in place. Safe because
// accessors hand out a copy (see snapshot), so the only slices this is called
// on are the index's own.
//
// It searches the slice for each replacement rather than scanning the
// replacements for each entry, which is the same comparisons in the other
// order and matters because the two are not the same size. For a name bucket,
// reps holds the entries of one file carrying one name — one of them, unless
// that file declares the name twice — while entries is every declaration of
// that name in the workspace. A workspace where 5,000 files declare `init`
// therefore ran 5,000 iterations of an outer loop to do one substitution;
// inverted it is one pass of slices.Index. 120us a line-changing keystroke
// under the write lock, against 6us.
//
// For the per-file slices the two sizes are the same and so is the cost.
func remap[T comparable](entries []T, reps []replacement[T]) {
	for _, r := range reps {
		if i := slices.Index(entries, r.old); i >= 0 {
			entries[i] = r.new
		}
	}
}
