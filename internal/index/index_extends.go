package index

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

// FindFilesByBasename returns absolute file paths for all indexed CFC files whose
// filename (without extension) matches name case-insensitively.
func (idx *Index) FindFilesByBasename(name string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	suffix := "/" + strings.ToLower(name) + ".cfc"

	var paths []string

	for key, defs := range idx.fileFuncs {
		if !strings.HasSuffix(key, suffix) {
			continue
		}

		// The file's own URI, in the case it was indexed under. A file with no
		// functions has no definition to recover it from, and its lowercased
		// key names nothing on a case-sensitive filesystem: Lucee's empty
		// LuceeTestCase.cfc came back as luceetestcase.cfc, which the extends
		// walk could not read.
		if u, ok := idx.fileURIs[key]; ok {
			if p := strings.TrimPrefix(string(u), "file://"); p != "" {
				paths = append(paths, p)

				continue
			}
		}

		// Recover the real (correctly-cased) path from a stored definition's URI.
		if len(defs) > 0 {
			if p := strings.TrimPrefix(string(defs[0].URI), "file://"); p != "" {
				paths = append(paths, p)

				continue
			}
		}

		// File has no functions — fall back to the lowercased key (works on case-insensitive FSes).
		if p := strings.TrimPrefix(key, "file://"); p != "" {
			paths = append(paths, p)
		}
	}

	// The loop ranges over a map, so without this the order changed from call
	// to call — and a caller choosing among equals took whichever came first.
	slices.Sort(paths)

	return paths
}

// ExtendsForFile returns the component this file extends, and whether the index
// knows. An empty string with ok true means the component extends nothing; ok
// false means nobody has recorded it yet.
//
// The distinction matters because only one of the two doors into the index can
// fill this in for free. IndexFile parses, so it knows; IndexFileFromResult is
// handed funcs and refs by a caller that already parsed, and threading the
// value through it would touch every call site in the tree for a value almost
// none of them care about. So "unknown" is a real state, and the reader is
// expected to establish the value and record it with SetExtends.
func (idx *Index) ExtendsForFile(fileURI uri.URI) (string, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	ext, ok := idx.extends[uriKey(fileURI)]

	return ext, ok
}

// FilesExtendingName returns the files whose recorded extends ends in name,
// dotted or bare and case-insensitively, sorted. The caller decides whether
// each extends really names the component it means: this is a filter on the
// spelling, so that question is asked of a few files rather than every one.
func (idx *Index) FilesExtendingName(name string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var out []string

	for key, ext := range idx.extends {
		last := ext
		if _, after, ok := strings.CutLast(ext, "."); ok {
			last = after
		}

		if ext != "" && strings.EqualFold(last, name) {
			if u, ok := idx.fileURIs[key]; ok {
				out = append(out, string(u))
			}
		}
	}

	slices.Sort(out)

	return out
}

// Extender is a file and what it extends, as the index records it.
type Extender struct {
	URI     string
	Extends string
}

// ExtendersByName groups every file by the last segment of what it extends,
// lowercased, each group sorted as FilesExtendingName sorts it: the answer to
// FilesExtendingName for every name at once, in one pass over the extends map.
// A walk that asks for a name per component (leafSubclasses over Lucee's
// thousands of test cases) otherwise scans the whole map for each.
//
// The map is kept until a file is added or dropped or an extends changes
// (ShapeGeneration), and shared: callers read it and
// do not edit it.
func (idx *Index) ExtendersByName() map[string][]Extender {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	gen := idx.ShapeGeneration()

	idx.extenders.mu.Lock()
	defer idx.extenders.mu.Unlock()

	if idx.extenders.byName != nil && idx.extenders.gen == gen {
		return idx.extenders.byName
	}

	out := map[string][]Extender{}

	for key, ext := range idx.extends {
		if ext == "" {
			continue
		}

		last := ext
		if _, after, ok := strings.CutLast(ext, "."); ok {
			last = after
		}

		if u, ok := idx.fileURIs[key]; ok {
			name := strings.ToLower(last)
			out[name] = append(out[name], Extender{URI: string(u), Extends: ext})
		}
	}

	for _, files := range out {
		slices.SortFunc(files, func(a, b Extender) int { return strings.Compare(a.URI, b.URI) })
	}

	idx.extenders.byName, idx.extenders.gen = out, gen

	return out
}

// SetExtends records what a file extends, for a reader that had to work it out.
// The value is cloned: see IndexFile.
func (idx *Index) SetExtends(fileURI uri.URI, extends string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	key := uriKey(fileURI)
	mark := idx.shapeBegin(key)

	idx.setExtendsLocked(key, strings.Clone(extends))
	idx.shapeEnd(key, &mark)
}

// DelegatesForFile returns the WireBox delegations a file declares, and
// whether they are known. They are recorded with its extends and forgotten
// with it, by the same rule: a re-index of any kind drops them, and the next
// reader re-establishes them from the file.
func (idx *Index) DelegatesForFile(fileURI uri.URI) ([]parser.Delegate, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	d, ok := idx.delegates[uriKey(fileURI)]

	return d, ok
}

// SetDelegates records a file's delegations, for a reader that had to work
// them out. The strings are cloned: see IndexFile.
func (idx *Index) SetDelegates(fileURI uri.URI, delegates []parser.Delegate) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.delegates[uriKey(fileURI)] = cloneDelegates(delegates)
}

// cloneDelegates copies delegations off the source they were parsed from;
// nil stays nil, which DelegatesForFile's callers read as "none".
func cloneDelegates(ds []parser.Delegate) []parser.Delegate {
	if len(ds) == 0 {
		return nil
	}

	out := make([]parser.Delegate, len(ds))
	for i := range ds {
		d := &ds[i]
		out[i] = parser.Delegate{
			Target: strings.Clone(d.Target), Prefix: strings.Clone(d.Prefix), Suffix: strings.Clone(d.Suffix),
			Includes: cloneStrings(d.Includes), Excludes: cloneStrings(d.Excludes), Line: d.Line,
		}
	}

	return out
}
