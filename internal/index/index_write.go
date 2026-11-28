package index

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

// IndexFile parses the given CFC content and updates the index for that file URI.
//
// It parses with no workspace rules: no resolver, mapping or lookup. That is
// right for the index package's own fixtures and wrong for a workspace, where
// the one way to index a component is a parse by the resolver
// ([resolve.Resolver.ParseFile] with IndexSpec) handed to [Index.IndexParsed];
// the editor, the batch commands and the lazy lookup all do that.
func (idx *Index) IndexFile(fileURI uri.URI, content string) {
	pr := parser.Parse(fileURI, content)
	idx.indexParseResult(fileURI, content, pr)
}

// IndexParsed indexes a file whose parse the caller has made, with whatever
// rules the caller parsed it by: the parse a batch run indexes by is the same
// one it then reads calls from.
func (idx *Index) IndexParsed(fileURI uri.URI, content string, pr *parser.ParseResult) {
	idx.indexParseResult(fileURI, content, pr)
}

func (idx *Index) indexParseResult(fileURI uri.URI, content string, pr *parser.ParseResult) {
	// The includes the parse's walk recorded, when it parsed this text.
	var includes []string
	if pr.Content == content {
		includes = pr.Includes()
	} else {
		includes = parser.ExtractIncludes(content)
	}

	idx.indexWhole(fileURI, ParsedFile{PR: pr, Disk: true, DelegatesKnown: true}, includes)
}

func (idx *Index) storeFileFunctionsLocked(fk string, funcs []parser.FunctionDef) {
	if len(funcs) > 0 {
		idx.funcGen++
	}

	fileDefs := make([]*parser.FunctionDef, 0, len(funcs))

	for i := range funcs {
		d := &funcs[i]
		key := strings.ToLower(d.Name)
		idx.funcs[key] = append(idx.funcs[key], d)
		fileDefs = append(fileDefs, d)
	}

	idx.fileFuncs[fk] = fileDefs
}

// ParsedFile is a parse a caller indexes whole: IndexParsedFile writes every
// fact about the file in one step.
type ParsedFile struct {
	PR *parser.ParseResult

	// Disk is a parse of the file as it is on disk, which marks the disk
	// revision source metadata is validated against; false for an editor buffer.
	Disk bool

	// DelegatesKnown is whether PR's delegates are the file's. A shallow parse
	// does not read them, and a file it cannot vouch for is left with none
	// recorded, which a reader takes as "work them out from the file".
	DelegatesKnown bool

	// EntityIfRead registers the file's ORM entity only when the index has read
	// it before (RefreshEntity), instead of always (IndexEntity).
	EntityIfRead bool
}

// IndexParsedFile indexes a file from a parse the caller made: its functions
// and component refs, this-scoped variables, includes, extends, implements,
// delegates and ORM entity, all under one hold of the lock. The writes it
// replaces were one call each, and the first made the file's functions visible
// while the rest were still to come, so a reader that arrived between them
// found a function but none of its this-scoped variables (a completion list
// missing a property), or a file whose extends had been forgotten.
func (idx *Index) IndexParsedFile(fileURI uri.URI, f ParsedFile) {
	idx.indexWhole(fileURI, f, f.PR.Includes())
}

// indexWhole is the one write of a whole file's facts, which every door that
// has a parse goes through (IndexFile, IndexParsed, IndexParsedFile). Adding a
// per-file map or counter means adding it here, once.
func (idx *Index) indexWhole(fileURI uri.URI, f ParsedFile, includes []string) {
	pr := f.PR

	// Every string below is cloned, for the reason indexFileFromResult
	// documents at length: a parsed string is a slice of the file's source,
	// and one retained substring keeps the whole file alive.
	funcs := parser.CompactDefs(pr.Funcs)
	refs := parser.CompactRefs(pr.ComponentRefs)
	thisVars := cloneStrings(pr.ThisVars())

	// An ORM entity is named by its entityname attribute, which is often not
	// its file's name: ContentBox's Author.cfc is cbAuthor, and
	// entityNew( "cbAuthor" ) names it. Reading it scans the header, so it is
	// read only when it will be stored: always, or for a file the index has
	// read the entity of before (EntityIfRead). The peek here only saves the scan
	// when the answer is no; the decision is made again under the lock, where
	// the index's removals and first reads cannot change it.
	component := isComponent(fileURI)
	fk := uriKey(fileURI)
	wantEntity := !f.EntityIfRead || component && idx.entityWasRead(fk)

	var (
		persistent bool
		entityName string
	)

	if wantEntity {
		persistent, entityName = entityOf(fileURI, pr)
	}

	var delegates []parser.Delegate
	if f.DelegatesKnown {
		delegates = cloneDelegates(pr.Delegates)
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	mark := idx.shapeBegin(fk)

	defer idx.shapeEnd(fk, &mark)

	idx.removeFileEntries(fileURI)
	idx.thisVars[fk] = thisVars
	idx.setExtendsLocked(fk, strings.Clone(pr.Extends))
	idx.setImplementsLocked(fk, pr.Implements)

	if f.DelegatesKnown {
		idx.delegates[fk] = delegates
	}

	idx.setIncludesLocked(fileURI, includes)

	_, read := idx.entityRead[fk]
	if want := !f.EntityIfRead || component && read; want {
		if !wantEntity {
			// The file was read since the peek: it is the buffer's entity.
			persistent, entityName = entityOf(fileURI, pr)
		}

		idx.setFileEntityLocked(uri.URI(strings.Clone(string(fileURI))), persistent, entityName)

		if component {
			idx.entityRead[fk] = struct{}{}
		}
	}

	idx.storeFileLocked(fileURI, fk, funcs, refs, f.Disk)
}

// entityWasRead reports whether the index has read this file's ORM entity.
func (idx *Index) entityWasRead(fk string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	_, read := idx.entityRead[fk]

	return read
}

// storeFileLocked stores a file's functions and component refs in both views
// of the index, the name buckets and the per-file lists, which every writer
// must fill together. The caller holds mu, has removed the file's earlier
// entries, and has compacted funcs and refs.
func (idx *Index) storeFileLocked(fileURI uri.URI, fk string, funcs []parser.FunctionDef, refs []parser.ComponentRef, disk bool) {
	idx.storeFileFunctionsLocked(fk, funcs)
	idx.fileURIs[fk] = uri.URI(strings.Clone(string(fileURI)))

	if disk {
		idx.markDiskRevision(fk)
	}

	fileRefsList := make([]*parser.ComponentRef, 0, len(refs))

	for i := range refs {
		r := &refs[i]
		key := strings.ToLower(r.Variable)
		idx.comprefs[key] = append(idx.comprefs[key], r)
		fileRefsList = append(fileRefsList, r)
	}

	idx.fileRefs[fk] = fileRefsList
}

// IndexFileFromResult updates the index using pre-parsed function defs and refs
// read from disk.
func (idx *Index) IndexFileFromResult(fileURI uri.URI, funcs []parser.FunctionDef, refs []parser.ComponentRef) {
	idx.indexFileFromResult(fileURI, funcs, refs, true)
}

// IndexBufferFromResult updates indexed symbols from an editor buffer while
// preserving the revision used to validate disk-backed source metadata.
func (idx *Index) IndexBufferFromResult(fileURI uri.URI, funcs []parser.FunctionDef, refs []parser.ComponentRef) {
	idx.indexFileFromResult(fileURI, funcs, refs, false)
}

func (idx *Index) indexFileFromResult(fileURI uri.URI, funcs []parser.FunctionDef, refs []parser.ComponentRef, disk bool) {
	// The strings a parse produces are slices of the file's source, and a Go
	// substring keeps the whole backing array alive. That is right for a parse
	// result, which dies with the source it came from, and wrong here: the index
	// outlives every one of them, so a single retained function name holds its
	// entire file in memory and an index of a workspace holds the workspace.
	//
	// On a real one — 18,245 files, 322MB of source — the index retained 285MB,
	// 0.9x the source it was built from, for data whose own size is a few
	// megabytes. Copying the strings as they are stored takes it to 55.6MB.
	//
	// Done here rather than at each caller, because a caller that forgot would
	// put the retention back with nothing to show it had. Every other door that
	// stores a parsed string clones it too; TestIndexDoesNotRetainTheSource
	// checks each of them.
	funcs = parser.CompactDefs(funcs)
	refs = parser.CompactRefs(refs)

	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.removeFileEntries(fileURI)
	idx.storeFileLocked(fileURI, uriKey(fileURI), funcs, refs, disk)
}

// RemoveFile drops every entry the index holds for one file — its functions,
// component refs, this-scoped vars, per-function scope refs, and its ORM entity
// registration if it had one. Used when a watched file is deleted on disk.
//
// An entity name may be declared by several files, so only this file's entry
// is dropped from it (entityNames says which names it held), and a name
// another file also declares stays registered to that one.
func (idx *Index) RemoveFile(fileURI uri.URI) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	fk := uriKey(fileURI)
	mark := idx.shapeBegin(fk)

	defer idx.shapeEnd(fk, &mark)

	idx.removeFileEntries(fileURI)
	delete(idx.diskRevisions, fk)
	// Not in removeFileEntries: every re-index runs that first, and a file
	// re-indexed with the includes it already had must not move the generation
	// and make the resolver rebuild its include graph. The writers that
	// re-index set includes afresh, and setIncludesLocked compares.
	idx.setIncludesLocked(fileURI, nil)

	idx.setFileEntityLocked(fileURI, false, "")
	delete(idx.entityRead, uriKey(fileURI))
}

// RemoveFilesUnder removes indexed entries under prefix, except under retained roots.
// Called when a workspace folder is removed, so it is rare and may walk every
// bucket; what it must not do is leave the two views of an entry disagreeing.
//
// It used to clear funcs and comprefs alone. The per-file maps then still
// listed the removed folder's files, which is wrong twice over: FunctionsForFile
// kept answering with definitions no bucket held any more, and HasFile kept
// reporting the file as indexed, so Resolver.EnsureIndexed never re-read it if
// the folder came back. It also leaves removeFileEntries — which reaches the
// name buckets through exactly these maps — reconciling a view no writer
// maintains.
func (idx *Index) RemoveFilesUnder(prefix string, retained ...uri.URI) {
	folderKey := func(root uri.URI) string {
		return strings.TrimRight(uriKey(root), "/") + "/"
	}
	fileKey := folderKey(uri.URI(prefix))
	keepKeys := make([]string, 0, len(retained))

	for _, root := range retained {
		key := folderKey(root)
		switch {
		case strings.HasPrefix(fileKey, key):
			return // The entire removed folder still belongs to a retained root.
		case strings.HasPrefix(key, fileKey):
			keepKeys = append(keepKeys, key)
		}
	}

	remove := func(key string) bool {
		return strings.HasPrefix(key, fileKey) && !slices.ContainsFunc(keepKeys, func(keep string) bool {
			return strings.HasPrefix(key, keep)
		})
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	for key, entries := range idx.funcs {
		filtered := keepIf(entries, func(e *parser.FunctionDef) bool {
			return !remove(uriKey(e.URI))
		})

		if len(filtered) == len(entries) {
			continue
		}

		idx.funcGen++

		if len(filtered) == 0 {
			delete(idx.funcs, key)
		} else {
			idx.funcs[key] = filtered
		}
	}

	for key, entries := range idx.comprefs {
		filtered := keepIf(entries, func(e *parser.ComponentRef) bool {
			return !remove(uriKey(e.URI))
		})

		if len(filtered) == 0 {
			delete(idx.comprefs, key)
		} else {
			idx.comprefs[key] = filtered
		}
	}

	for key := range idx.fileFuncs {
		if remove(key) {
			mark := idx.shapeBegin(key)

			delete(idx.fileFuncs, key)
			delete(idx.fileURIs, key)
			delete(idx.diskRevisions, key)
			idx.shapeEnd(key, &mark)
			delete(idx.fileRefs, key)
			delete(idx.thisVars, key)
			delete(idx.scopeRefs, key)
		}
	}

	// fileRefs can hold a file that declared no functions, so it cannot be
	// swept only alongside fileFuncs.
	for key := range idx.fileRefs {
		if remove(key) {
			delete(idx.fileRefs, key)
			delete(idx.thisVars, key)
			delete(idx.scopeRefs, key)
		}
	}

	for key := range idx.thisVars {
		if remove(key) {
			delete(idx.thisVars, key)
			delete(idx.scopeRefs, key)
		}
	}

	for key := range idx.includes {
		if remove(key) {
			delete(idx.includes, key)
			idx.includeGen++
		}
	}

	// The folder's ORM entities, named and called after their files, which
	// otherwise stay candidates for entityNew() in the folders left.
	var entityFiles []uri.URI

	for _, list := range idx.entities {
		for _, u := range list {
			if remove(uriKey(u)) {
				entityFiles = append(entityFiles, u)
			}
		}
	}

	for _, list := range idx.defaultEntities {
		for _, u := range list {
			if remove(uriKey(u)) {
				entityFiles = append(entityFiles, u)
			}
		}
	}

	for _, u := range entityFiles {
		idx.setFileEntityLocked(u, false, "")
	}

	for key := range idx.entityRead {
		if remove(key) {
			delete(idx.entityRead, key)
		}
	}
}

// removeFileEntries drops one file's functions and component refs from every
// map that holds them.
//
// It reaches the name buckets through fileFuncs/fileRefs, which already hold
// exactly this file's entries, rather than walking every bucket in the index
// and asking each entry which file it came from. That walk is the same shape
// ShiftLines was fixed for and it was left here, where it costs more: every
// index write goes through this function, so re-indexing one file was
// proportional to the whole workspace and the startup scan was quadratic in
// the file count. Measured by BenchmarkIndexFileFromResult on a 5,000-file
// index, one file cost 43ms and 160,020 allocations.
//
// The per-entry uriKey was most of it. It lowercases and percent-decodes, both
// of which allocate on a real workspace path, and it ran once per entry in the
// index per file indexed: 5.9 GB of the 6.6 GB a 5,624-file corpus scan
// allocated, and 86% of its CPU, was this one comparison.
//
// Entries are matched by pointer identity, not by URI, because that is what
// makes the bucket set knowable in advance: a definition lives under its
// lowercased name and a ref under its lowercased variable, and fileFuncs holds
// the very pointers the name buckets do (ownFunc/ownRef mint one pointer per
// entry, so identity is exact). The two views cannot disagree — every writer
// here fills both together, and each removal runs before the write that
// replaces them.
func (idx *Index) removeFileEntries(fileURI uri.URI) {
	key := uriKey(fileURI)
	delete(idx.thisVars, key)
	delete(idx.scopeRefs, key)
	// Dropped with everything else, which is what makes the extends record
	// self-invalidating: a re-index of any kind forgets it, and the next
	// lookup re-establishes it from the file as it now stands.
	idx.dropExtendsLocked(key)
	delete(idx.delegates, key)
	delete(idx.implements, key)

	// Grouped by bucket, not pooled into one set of everything this file
	// declares. Both spellings visit the same bucket entries; the difference is
	// what each visit costs. A pooled set has to be a map, so every entry in a
	// 5,000-entry `init` bucket pays a hash and a probe to be told it is not
	// one of this file's eight — 57% of the time this function spent, measured,
	// was runtime.mapaccess1. Grouped, the candidates for a bucket are the
	// entries of this file that carry that one name, which is one of them
	// except where a file declares the same name twice, so the test is a
	// pointer comparison or two.
	fileFuncs := idx.fileFuncs[key]
	if len(fileFuncs) > 0 {
		idx.funcGen++
	}

	var buf foldScratch

	for _, d := range fileFuncs {
		if b, ok := buf.lowerASCII(d.Name); ok {
			if filtered := dropEntry(idx.funcs[string(b)], d); len(filtered) == 0 {
				delete(idx.funcs, string(b))
			} else {
				idx.funcs[strings.ToLower(d.Name)] = filtered
			}
		} else {
			name := strings.ToLower(d.Name)
			if filtered := dropEntry(idx.funcs[name], d); len(filtered) == 0 {
				delete(idx.funcs, name)
			} else {
				idx.funcs[name] = filtered
			}
		}
	}

	delete(idx.fileFuncs, key)
	delete(idx.fileURIs, key)

	for _, r := range idx.fileRefs[key] {
		if b, ok := buf.lowerASCII(r.Variable); ok {
			if filtered := dropEntry(idx.comprefs[string(b)], r); len(filtered) == 0 {
				delete(idx.comprefs, string(b))
			} else {
				idx.comprefs[strings.ToLower(r.Variable)] = filtered
			}
		} else {
			name := strings.ToLower(r.Variable)
			if filtered := dropEntry(idx.comprefs[name], r); len(filtered) == 0 {
				delete(idx.comprefs, name)
			} else {
				idx.comprefs[name] = filtered
			}
		}
	}

	delete(idx.fileRefs, key)
}

// dropEntry swap-removes target from entries, clearing the vacated slot so a
// dropped entry is not pinned by the backing array.
func dropEntry[T comparable](entries []T, target T) []T {
	i := slices.Index(entries, target)
	if i < 0 {
		return entries
	}

	last := len(entries) - 1
	entries[i] = entries[last]

	clear(entries[last:])

	return entries[:last]
}
