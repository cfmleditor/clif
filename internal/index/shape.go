package index

import (
	"path"
	"slices"
	"strings"

	"go.lsp.dev/uri"
)

// fileShape is what one file contributes to the answers ShapeGeneration
// guards: whether the index holds it, under which URI, what it extends and
// which entity names it declares.
type fileShape struct {
	uri      uri.URI
	ext      string
	entities []string
	ok       bool
}

func (a *fileShape) equal(b *fileShape) bool {
	return a.ok == b.ok && a.uri == b.uri && a.ext == b.ext && slices.Equal(a.entities, b.entities)
}

// shapeOfLocked reads the file keyed key; idx.mu is held.
func (idx *Index) shapeOfLocked(key string) fileShape {
	u, ok := idx.fileURIs[key]

	return fileShape{ok: ok, uri: u, ext: idx.extends[key], entities: slices.Clone(idx.entityNames[key])}
}

// shapeMatters reports whether a change to the file keyed key can change what
// a reader of the shape (the leaf subclasses under a base) answers. A file
// that extends nothing matters only as something a name resolves to, and the
// ways a bare name does that are the file-name search (FindFilesByBasename),
// an ORM entity name and a WireBox id, which reads the ModuleConfig and
// WireBox files: so it matters when some extends in the index spells its
// name, or an entity name it declares, or is one of those two files. A new
// extends arrives with its own file's change, which moves the generation, so
// a file skipped here is looked at again the moment a name could reach it.
// idx.mu is held.
func (idx *Index) shapeMatters(key string, fs *fileShape) bool {
	if !fs.ok {
		return false
	}

	if fs.ext != "" || idx.mentionAll > 0 {
		return true
	}

	base := strings.ToLower(strings.TrimSuffix(path.Base(key), path.Ext(key)))

	if base == "moduleconfig" || base == "wirebox" || idx.mentions[base] > 0 {
		return true
	}

	for _, e := range fs.entities {
		if idx.mentions[e] > 0 {
			return true
		}
	}

	return false
}

// shapeBegin is taken before a write to the file keyed key, and shapeEnd after
// it: the generation moves only when the file's shape changed and either
// state could be reached by a name.
type shapeMark struct {
	before  fileShape
	matters bool
}

func (idx *Index) shapeBegin(key string) shapeMark {
	fs := idx.shapeOfLocked(key)

	return shapeMark{before: fs, matters: idx.shapeMatters(key, &fs)}
}

func (idx *Index) shapeEnd(key string, m *shapeMark) {
	now := idx.shapeOfLocked(key)

	if m.before.equal(&now) {
		return
	}

	if m.matters || idx.shapeMatters(key, &now) {
		idx.shape.Add(1)
	}
}

// mentionedNames are the bare names an extends spelling can resolve through:
// the last segment of the path, and of each half of a `Name@module` id.
// wild is true for a spelling computed at run time, which could be any name.
func mentionedNames(ext string) (names []string, wild bool) {
	if strings.Contains(ext, "#") {
		return nil, true
	}

	last := func(s string) string {
		if i := strings.LastIndexAny(s, "./\\"); i >= 0 {
			s = s[i+1:]
		}

		return strings.ToLower(strings.TrimSpace(s))
	}

	names = append(names, last(ext))

	if before, after, ok := strings.Cut(ext, "@"); ok {
		names = append(names, last(before), last(after))
	}

	return names, false
}

// setExtendsLocked records what the file keyed key extends, keeping the
// mentions in step; idx.mu is held.
func (idx *Index) setExtendsLocked(key, ext string) {
	idx.dropExtendsLocked(key)

	idx.extends[key] = ext

	if ext == "" {
		return
	}

	names, wild := mentionedNames(ext)
	if wild {
		idx.mentionAll++

		return
	}

	if idx.mentions == nil {
		idx.mentions = map[string]int{}
	}

	for _, n := range names {
		idx.mentions[n]++
	}
}

// dropExtendsLocked forgets the file's extends; idx.mu is held.
func (idx *Index) dropExtendsLocked(key string) {
	ext, ok := idx.extends[key]
	if !ok {
		return
	}

	delete(idx.extends, key)

	if ext == "" {
		return
	}

	names, wild := mentionedNames(ext)
	if wild {
		idx.mentionAll--

		return
	}

	for _, n := range names {
		idx.mentions[n]--

		if idx.mentions[n] <= 0 {
			delete(idx.mentions, n)
		}
	}
}
