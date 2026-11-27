package index

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

// LookupComponentRef returns component references for the given variable name.
func (idx *Index) LookupComponentRef(variable string) []*parser.ComponentRef {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return snapshot(idx.comprefs[strings.ToLower(variable)])
}

// RefsForFile returns all component references indexed for a specific file.
func (idx *Index) RefsForFile(fileURI uri.URI) []*parser.ComponentRef {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return snapshot(idx.fileRefs[uriKey(fileURI)])
}

// ThisVarsForFile returns the this-scoped variable names for a file.
func (idx *Index) ThisVarsForFile(fileURI uri.URI) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return snapshot(idx.thisVars[uriKey(fileURI)])
}

// SetThisVars stores this-scoped variable names for a file, alone. Indexing a
// parse writes them with the rest of the file (IndexParsedFile): a fact a
// reader can ask for is written there, not in a call of its own after it,
// which would let a reader in between find the file without it.
func (idx *Index) SetThisVars(fileURI uri.URI, vars []string) {
	// Owned, for the reason ownFunc gives: the caller's slice is a live
	// ParseResult's, and the index must be the only writer of what it stores.
	// The strings are copied as well as the slice: they are substrings of the
	// file's source, and copying only the slice left every component with a
	// this-scoped variable pinning its whole file — on the workspace scan, which
	// calls this for every file it indexes.
	owned := cloneStrings(vars)

	idx.mu.Lock()
	idx.thisVars[uriKey(fileURI)] = owned
	idx.mu.Unlock()
}

// cloneStrings copies a slice of strings and every string in it, so nothing
// the result holds shares memory with the source the strings were cut from.
// nil for an empty input, as snapshot answers.
func cloneStrings(ss []string) []string {
	if len(ss) == 0 {
		return nil
	}

	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = strings.Clone(s)
	}

	return out
}

// SetFuncRefs records the component refs found inside one function scope,
// replacing whatever was recorded for that scope before.
//
// These arrive lazily: the server indexes a function's refs the first time a
// hover or a definition lookup lands inside it. The plain append this replaced
// had no way to tell a first indexing from a re-indexing, so every such lookup
// added another copy. Refs are memoised per function and invalidated when that
// function is edited, so an editing session alternating edits and hovers grew
// comprefs without bound — with duplicate refs at identical lines, which
// LookupComponentRefInFile then had to scan through on every subsequent call.
//
// scopeKey identifies the function within the file; the caller's line range is
// the natural choice.
func (idx *Index) SetFuncRefs(fileURI uri.URI, scopeKey string, refs []parser.ComponentRef) {
	// Cloned for the reason indexFileFromResult gives. These come from an open
	// document's ParseResult, so they would pin not the file but the version of
	// it that was current when the function was last hovered — one superseded
	// copy of the document per function scope, surviving every later edit.
	refs = parser.CompactRefs(refs)

	idx.mu.Lock()
	defer idx.mu.Unlock()

	fk := uriKey(fileURI)

	if prev := idx.scopeRefs[fk][scopeKey]; len(prev) > 0 {
		drop := make(map[*parser.ComponentRef]bool, len(prev))
		for _, r := range prev {
			drop[r] = true
		}

		keep := func(e *parser.ComponentRef) bool { return !drop[e] }

		// Sweep each affected bucket once. Several refs in a scope routinely
		// share a variable name or a file, and filtering per ref would rebuild
		// the same bucket once per ref in it.
		varKeys := make(map[string]bool, len(prev))
		fileKeys := make(map[string]bool, len(prev))

		for _, r := range prev {
			varKeys[strings.ToLower(r.Variable)] = true
			fileKeys[uriKey(r.URI)] = true
		}

		for key := range varKeys {
			if kept := keepIf(idx.comprefs[key], keep); len(kept) == 0 {
				delete(idx.comprefs, key)
			} else {
				idx.comprefs[key] = kept
			}
		}

		for rk := range fileKeys {
			if kept := keepIf(idx.fileRefs[rk], keep); len(kept) == 0 {
				delete(idx.fileRefs, rk)
			} else {
				idx.fileRefs[rk] = kept
			}
		}
	}

	added := make([]*parser.ComponentRef, 0, len(refs))

	// Each ref is filed under its own URI, as the append this replaced did;
	// fileURI identifies only the scope whose refs are being replaced.
	for i := range refs {
		r := new(refs[i])
		key := strings.ToLower(r.Variable)
		idx.comprefs[key] = append(idx.comprefs[key], r)
		rk := uriKey(r.URI)
		idx.fileRefs[rk] = append(idx.fileRefs[rk], r)
		added = append(added, r)
	}

	if idx.scopeRefs[fk] == nil {
		idx.scopeRefs[fk] = make(map[string][]*parser.ComponentRef)
	}

	idx.scopeRefs[fk][scopeKey] = added
}

// LookupComponentRefInFile returns the component ref for a variable in a specific file
// that is closest to (but not after) the given line.
//
// It searches the file's own refs for the name, not the name's bucket for the
// file. Both views hold the same entries, so the answer is the same, but the
// sizes are not: comprefs is keyed by variable name, and the names this is
// asked about are the ordinary ones — svc, dao, qry — so that bucket holds one
// entry per file in the workspace. Searching it meant a uriKey per entry, which
// lowercases and percent-decodes and so allocates, on a lookup that hover,
// definition and completion each perform on the keystroke (hover twice).
//
// Measured on a 5,000-file workspace: 0.98ms and 160KB per lookup, against
// 0.29µs and 32 bytes now, flat as the workspace grows. The trade is real but
// small and bounded by the open file: where the old form found the name's
// bucket holding one entry, a file declaring 200 refs now costs 1.6µs against
// the 0.34µs it used to.
func (idx *Index) LookupComponentRefInFile(variable string, fileURI uri.URI, line uint32) *parser.ComponentRef {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var best *parser.ComponentRef

	for _, ref := range idx.fileRefs[uriKey(fileURI)] {
		if ref.Line > line || !strings.EqualFold(ref.Variable, variable) {
			continue
		}

		if ref.Component == "" || strings.HasPrefix(ref.Component, parser.NativeTypePrefix) {
			continue
		}

		if best == nil || ref.Line >= best.Line {
			best = ref
		}
	}

	return best
}

// SetBeans replaces the bean map with the given name→dot-path mapping.
func (idx *Index) SetBeans(beans map[string]string) {
	idx.mu.Lock()
	idx.beans = beans
	idx.mu.Unlock()
}

// LookupBean returns the component dot-path for a bean name (case-insensitive).
func (idx *Index) LookupBean(name string) string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.beans[strings.ToLower(name)]
}
