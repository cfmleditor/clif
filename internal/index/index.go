// Package index maintains a searchable index of CFML function definitions.
package index

import (
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/uri"
)

// Index is a concurrency-safe store of function definitions keyed by name.
type Index struct {
	mu            writeCountingMutex
	funcGen       uint64                                       // protected by mu; advances when function buckets change
	funcs         map[string][]*parser.FunctionDef             // lowercase name -> definitions
	fileFuncs     map[string][]*parser.FunctionDef             // lowercase URI -> definitions in that file
	comprefs      map[string][]*parser.ComponentRef            // lowercase variable -> refs
	fileRefs      map[string][]*parser.ComponentRef            // lowercase URI -> refs in that file
	thisVars      map[string][]string                          // lowercase URI -> this-scoped var names
	scopeRefs     map[string]map[string][]*parser.ComponentRef // lowercase URI -> function scope key -> refs
	extends       map[string]string                            // lowercase URI -> that component's extends, "" for none
	implements    map[string][]string                          // lowercase URI -> the interfaces it implements; kept and dropped with extends
	delegates     map[string][]parser.Delegate                 // lowercase URI -> the WireBox delegations it declares; kept and dropped with extends
	fileURIs      map[string]uri.URI                           // lowercase URI -> the file's URI as indexed, in its real case; kept with fileFuncs
	diskRevisions map[string]uint64                            // disk source revisions; kept with fileURIs
	beans         map[string]string                            // lowercase bean name -> dot-path
	// entities holds every file declaring an entity name: several
	// applications in one workspace may each declare one (Lucee's tests
	// declare Author three times), and which file indexed last is not an
	// answer. entityNames is each file's names, so a re-index or a removal
	// drops exactly its own.
	entities    map[string][]uri.URI // lowercase entity name -> the files declaring it
	entityNames map[string][]string  // lowercase URI -> the lowercase entity names it holds
	// entityRead is every component whose entity the index has read from a parse,
	// persistent or not, so RefreshEntity can follow a file the index took in
	// even after an edit left it no entity for a while.
	entityRead map[string]struct{} // lowercase URI
	// defaultEntities holds persistent components that name no entityname,
	// whose entity is called after the file: several applications in one
	// workspace may each have one, so every candidate is kept.
	defaultEntities map[string][]uri.URI
	includes        map[string]fileIncludes // lowercase URI -> the paths that file cfincludes
	// includeGen counts changes to includes, so a reader that derives something
	// from all of them — the resolver's reverse map — can tell when to rebuild.
	includeGen uint64

	shape atomic.Uint64 // see ShapeGeneration
	// mentions counts, by lowercase bare name, the extends spellings that can
	// resolve through it (mentionedNames); mentionAll counts the computed ones.
	// They decide which files move the shape (shapeMatters).
	mentions   map[string]int
	mentionAll int

	// extenders is ExtendersByName's answer at the generation it was built.
	extenders struct {
		mu     sync.Mutex
		gen    uint64
		byName map[string][]Extender
	}
}

// writeCountingMutex counts every write lock taken, so a reader caching an
// answer derived from the index can tell when anything may have changed. It
// is the lock rather than each writer that counts, so a writer added later
// cannot forget to.
type writeCountingMutex struct {
	sync.RWMutex
	writes atomic.Uint64
}

func (m *writeCountingMutex) Lock() {
	m.RWMutex.Lock()
	m.writes.Add(1)
}

// Generation reports a counter that moves whenever anything in the index may
// have changed. It never moves for reads.
func (idx *Index) Generation() uint64 { return idx.mu.writes.Load() }

// ShapeGeneration moves when a file is added to or dropped from the index, or
// what a file extends or which entity names it declares changes, and for no
// other write: and then only if the file could matter to an answer built from
// the shape (shapeMatters). A file that extends nothing, which no extends
// spells by name, is left out: during a scan most files indexed are of that
// kind, and each used to throw away every leaf-subclass answer. An answer that
// depends only on which components exist and what each extends (the leaf
// subclasses under a base) stays valid across the writes that do not move it:
// variables, includes, functions of files already known, and those files.
func (idx *Index) ShapeGeneration() uint64 { return idx.shape.Load() }

// fileIncludes is one file's cfinclude paths as written, with the file's own
// URI: resolving a relative path needs its directory in its real case.
type fileIncludes struct {
	uri   uri.URI
	paths []string
}

// New creates an empty Index.
func New() *Index {
	return &Index{
		funcs:           make(map[string][]*parser.FunctionDef),
		fileFuncs:       make(map[string][]*parser.FunctionDef),
		comprefs:        make(map[string][]*parser.ComponentRef),
		fileRefs:        make(map[string][]*parser.ComponentRef),
		thisVars:        make(map[string][]string),
		scopeRefs:       make(map[string]map[string][]*parser.ComponentRef),
		extends:         make(map[string]string),
		delegates:       make(map[string][]parser.Delegate),
		implements:      make(map[string][]string),
		fileURIs:        make(map[string]uri.URI),
		beans:           make(map[string]string),
		entities:        make(map[string][]uri.URI),
		entityNames:     make(map[string][]string),
		entityRead:      make(map[string]struct{}),
		defaultEntities: make(map[string][]uri.URI),
		includes:        make(map[string]fileIncludes),
	}
}

// Lookup returns all function definitions matching the given name (case-insensitive).
func (idx *Index) Lookup(name string) []*parser.FunctionDef {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return snapshot(idx.funcs[strings.ToLower(name)])
}

// LookupPreferred answers the question hover, signature help and argument
// completion actually ask of a bare function name: the definition to show,
// preferring the one declared in the requesting file, plus how many candidates
// there were so a caller can tell "the only one" from "the first of several".
//
// It exists so those three do not go through Lookup, which copies the bucket it
// returns (see snapshot). That copy is the right default — a caller walking
// every match pays for what it walks — but these three want one entry, and the
// bucket for a name every component declares holds one entry per file in the
// workspace: 40KB per call, on paths that run while the user is typing.
//
// inFile distinguishes a real preference from a fallback, which hover needs:
// it shows a global match only when it is unambiguous.
func (idx *Index) LookupPreferred(name string, preferURI uri.URI) (def *parser.FunctionDef, inFile bool, total int) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	defs := idx.funcs[strings.ToLower(name)]
	if len(defs) == 0 {
		return nil, false, 0
	}

	// The preference is answered from the file's own definitions rather than by
	// scanning the name's bucket for the file, so the cost is bounded by the
	// open document instead of by the workspace. The two views hold the same
	// entries, so the candidate set is the same either way.
	//
	// One deliberate difference from the callers this replaces, which each
	// compared d.URI == docURI raw: the file is found by uriKey, the same
	// normalisation FunctionsForFile and every other per-file accessor here
	// uses. That makes this agree with the rest of the index on what "the same
	// file" means, where the raw comparison could miss a definition over a
	// percent-escape or a case difference in the URI. It is free — uriKey runs
	// once on the argument, not once per entry.
	for _, d := range idx.fileFuncs[uriKey(preferURI)] {
		if strings.EqualFold(d.Name, name) {
			return d, true, len(defs)
		}
	}

	return nearestTo(defs, preferURI), false, len(defs)
}

// nearestTo picks the definition in the directory closest to preferURI, and the
// lowest URI among equals.
//
// This used to be defs[0] — the order the bucket happened to hold, which is the
// order eight parallel indexing goroutines finished in, so it differed between
// restarts and hovering the same call could name a different component than it
// did yesterday. Nearest is both stable and the better guess: a workspace where
// several components declare the same method name is usually one where the
// relevant one is the near one.
//
// The lowest-URI fallback is not cosmetic. Without it two files the same
// distance away leave the answer decided by bucket order again, which is the
// thing being fixed.
func nearestTo(defs []*parser.FunctionDef, preferURI uri.URI) *parser.FunctionDef {
	ref := string(preferURI)
	best := defs[0]
	bestDist := cfpath.URIDistance(ref, string(best.URI))

	for _, d := range defs[1:] {
		dist := cfpath.URIDistance(ref, string(d.URI))
		if dist < bestDist || (dist == bestDist && d.URI < best.URI) {
			best, bestDist = d, dist
		}
	}

	return best
}

// CountFunctions reports how many definitions are indexed under a name, without
// building the slice Lookup would have to copy to answer the same question.
func (idx *Index) CountFunctions(name string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return len(idx.funcs[strings.ToLower(name)])
}

// AllFunctions returns every indexed function definition.
func (idx *Index) AllFunctions() []*parser.FunctionDef {
	return idx.FunctionsMatching(nil)
}

// FunctionsMatching returns the indexed definitions whose lowercased name
// satisfies match, or every definition when match is nil.
//
// The predicate is given the bucket key, so it runs once per distinct name
// rather than once per definition: a workspace of 5,000 components declaring
// the same eight methods asks it eight times, not forty thousand. It sees only
// a name, which is what keeps it safe to run under the read lock — it has
// nothing to re-enter the index with.
//
// workspace/symbol is the reason this exists. It is issued on every keystroke
// in the symbol picker and it discards all but a handful of matches, but it
// went through AllFunctions, which materialised every definition in the
// workspace into a slice grown from nil: 3.1MB allocated and copied per
// keystroke on a 40,000-definition index, to return a few dozen symbols.
func (idx *Index) FunctionsMatching(match func(loweredName string) bool) []*parser.FunctionDef {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	total := 0

	for name, defs := range idx.funcs {
		if match == nil || match(name) {
			total += len(defs)
		}
	}

	if total == 0 {
		return nil
	}

	all := make([]*parser.FunctionDef, 0, total)

	for name, defs := range idx.funcs {
		if match == nil || match(name) {
			all = append(all, defs...)
		}
	}

	return all
}

const matchingFunctionBatchSize = 256

// VisitMatchingFunctionBatches visits bounded copies of the matching buckets.
// Match runs under the read lock once per name and must not re-enter the index.
// Visit runs without the lock and may retain the immutable definitions. The
// slice is reused. A false result means a definition changed between batches and the
// caller must discard it.
func (idx *Index) VisitMatchingFunctionBatches(match func(loweredName string) bool, visit func([]*parser.FunctionDef)) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	generation := idx.funcGen
	visitUnlocked := func(batch []*parser.FunctionDef) {
		idx.mu.RUnlock()
		defer idx.mu.RLock()

		visit(batch)
	}

	var batch [matchingFunctionBatchSize]*parser.FunctionDef

	used := 0

	for name, defs := range idx.funcs {
		if !match(name) {
			continue
		}

		for len(defs) > 0 {
			copied := copy(batch[used:], defs)
			used += copied
			defs = defs[copied:]

			if used == len(batch) {
				visitUnlocked(batch[:])

				if idx.funcGen != generation {
					return false
				}

				used = 0
			}
		}
	}

	if used > 0 {
		visitUnlocked(batch[:used])
	}

	return true
}

// Every entry below is filed as new(funcs[i]) — a copy the index alone owns —
// and never as &funcs[i].
//
// A pointer into the caller's slice is, for the LSP server, a pointer into the
// live ParseResult for an open document. That gives the entry two writers.
// ParseResult.ApplyEdit shifts line numbers on every in-function keystroke
// (internal/parser/incremental.go), holding the server's per-document lock and
// knowing nothing about this index; meanwhile another editor's connection could
// be reading the same struct through Lookup. In daemon mode one Index is shared
// by every connection, so those are genuinely concurrent, and neither lock
// covers the other's writer.
//
// Copying on the way in makes the index the only writer, which is what lets
// ShiftLines below fix the remaining half. new(x) allocates a fresh variable
// initialised to x's value rather than aliasing it, so it carries the same
// guarantee the ownFunc/ownRef helpers used to.

// snapshot copies a stored slice for a caller.
//
// Every accessor here hands its result out and then releases the read lock, so
// what it returns has to be storage no writer will touch. That used to be
// arranged the other way round — the maps held slices no writer ever mutated in
// place, so a read could hand out the live one for free — and it made writes
// pay for reads. funcs is keyed by lowercased name, so a workspace where every
// component declares init has one bucket holding an entry per file, and
// changing this file's single entry in it rebuilt all 5,000: ~1ms of
// write-locked work and 660KB of garbage per line-changing keystroke, to alter
// one pointer.
//
// Copying on the way out inverts that. A read now pays for the entries it is
// about to iterate anyway, which is a constant factor on work it was already
// doing, while a write pays only for the entries it actually changes. Typing
// is far more frequent than hovering, and a bucket is large in exactly the
// case where the reader wants all of it.
//
// nil for an empty result keeps the accessors' old "nothing indexed" answer
// distinguishable from an empty non-nil slice, which some callers compare
// against nil.
func snapshot[T any](s []T) []T {
	if len(s) == 0 {
		return nil
	}

	return append(make([]T, 0, len(s)), s...)
}

// keepFuncs and keepRefs compact entries in place, returning the prefix that
// satisfied keep.
//
// Writing over the backing array is safe precisely because snapshot exists: no
// caller is holding it. Before that it was not, and this allocated a fresh
// slice for every filtered bucket — the comment here recorded a race that
// `go test -race` had actually reported, where a caller walking the slice it
// had been given read another file's entries shifted into place by a
// concurrent IndexFile.
//
// The tail is cleared so the entries dropped from the bucket can be collected
// rather than pinned by an array the map still points at.
func keepIf[T any](entries []*T, keep func(*T) bool) []*T {
	n := 0

	for _, e := range entries {
		if keep(e) {
			entries[n] = e
			n++
		}
	}

	clear(entries[n:])

	return entries[:n]
}

// uriKey returns a lowercase URI for case-insensitive comparison on case-insensitive filesystems.
func uriKey(u uri.URI) string {
	return uriKeys.key(string(u))
}

func uriKeyUncached(s string) string {
	// Percent-escapes are decoded so that the two ways this codebase builds a
	// file URI land on one key. Indexing goes through uri.File, which escapes:
	// a workspace under "/My Documents" is stored as "file:///My%20Documents/…".
	// Several lookups build "file://" + path by hand instead, so they ask for
	// "file:///My Documents/…" and miss every entry — this-scope completions
	// simply stopped appearing for anyone whose path contained a space, with no
	// error anywhere. Normalising here fixes both directions at once, which the
	// alternative — rewriting all 37 URI conversions in one go — does not.
	if dec, err := url.PathUnescape(s); err == nil {
		s = dec
	}

	return strings.ToLower(s)
}
