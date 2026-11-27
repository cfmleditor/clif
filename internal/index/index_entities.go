package index

import (
	"path"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

// IndexEntity records the ORM entity a parsed file declares (entityOf),
// replacing what it declared before. Indexing a parse writes the entity with
// the rest of the file (IndexParsedFile); this records it alone, for a test or
// a tool that has only the entity to say. It applies the rule the index
// applies to a parse it is handed,
// so the two cannot disagree about a file: the editor once counted only a
// component inside the Application's cfcLocation, and its reparse dropped what
// its save had registered. A page declares no entity and never held one, so it
// takes no lock.
func (idx *Index) IndexEntity(fileURI uri.URI, pr *parser.ParseResult) {
	idx.indexEntity(fileURI, pr, false)
}

// RefreshEntity is IndexEntity for a file whose entity the index has read
// before, and does nothing for one it has not. IndexParsedFile does the same
// for ParsedFile.EntityIfRead, which the editor uses for a component outside
// the workspace: a buffer opened from another project must not add entities,
// but one the resolver's lazy index took in must follow its edits, since that
// index never reads a file twice.
func (idx *Index) RefreshEntity(fileURI uri.URI, pr *parser.ParseResult) {
	idx.indexEntity(fileURI, pr, true)
}

// indexEntity is IndexEntity, or with onlyIfRead RefreshEntity.
func (idx *Index) indexEntity(fileURI uri.URI, pr *parser.ParseResult, onlyIfRead bool) {
	if !isComponent(fileURI) {
		return
	}

	persistent, entityName := entityOf(fileURI, pr)
	key := uriKey(fileURI)

	idx.mu.Lock()
	defer idx.mu.Unlock()

	if _, read := idx.entityRead[key]; onlyIfRead && !read {
		return
	}

	idx.setFileEntityLocked(fileURI, persistent, entityName)
	idx.entityRead[key] = struct{}{}
}

// entityOf is what a parsed file declares as an ORM entity: whether it is a
// persistent component, and the entityname it gives, if any. The entityname
// is read only for a persistent component, since reading it scans the header.
func entityOf(fileURI uri.URI, pr *parser.ParseResult) (bool, string) {
	if !isComponent(fileURI) {
		return false, ""
	}

	if !pr.Persistent {
		// Mura's bean ORM registers `component entityName="x" table="y"
		// extends="mura.bean.beanORM"` by that name with no persistent
		// attribute (registerModelDir). An entityname alone, without the
		// engine's persistent, is no entity; with a table it is Mura's.
		// Another framework's entity (a Quick model with a table and a name)
		// is not an engine entity. A Mura bean may reach beanORM through a
		// base class of a project's own, and which files are indexed first
		// differs between runs, so the chain is not followed: the bases of
		// the other frameworks are what is turned away.
		// Reading either attribute scans the header, so a file that never
		// spells the word is not scanned: most components are not entities.
		if parser.IndexFold(pr.Content, "entityname") < 0 || otherEntityBase(pr.Extends) {
			return false, ""
		}

		if name := pr.EntityName(); name != "" && pr.DeclaresTable() {
			return true, name
		}

		return false, ""
	}

	return true, pr.EntityName()
}

// otherEntityBase reports whether extends names a base class of an ORM that
// is not the engine's, whose models spell a table and an entity name too. It
// knows the two spellings (`quick.`, `cborm.`): a model that reaches its base
// through an alias or a base class of the project's own is not told from a Mura
// bean that does the same, and the chain is not followed (see entityOf).
func otherEntityBase(extends string) bool {
	lower := strings.ToLower(extends)

	return strings.HasPrefix(lower, "quick.") || strings.HasPrefix(lower, "cborm.")
}

// isComponent reports whether fileURI names a .cfc.
func isComponent(fileURI uri.URI) bool {
	return strings.EqualFold(path.Ext(string(fileURI)), ".cfc")
}

// setFileEntityLocked records what ORM entity fileURI declares, replacing
// what it declared before: entityName when it names one, else an entity
// called after the file, and nothing when the component is not persistent;
// idx.mu is held. Outside the index a file's entity is recorded by indexing
// its parse (IndexParsedFile), or alone through IndexEntity or RefreshEntity;
// each reads both from the parse (entityOf).
func (idx *Index) setFileEntityLocked(fileURI uri.URI, persistent bool, entityName string) {
	key := uriKey(fileURI)
	idx.dropEntitiesLocked(key)

	byFile := strings.ToLower(strings.TrimSuffix(path.Base(string(fileURI)), path.Ext(string(fileURI))))
	if list := slices.DeleteFunc(idx.defaultEntities[byFile], func(u uri.URI) bool { return uriKey(u) == key }); len(list) > 0 {
		idx.defaultEntities[byFile] = list
	} else {
		delete(idx.defaultEntities, byFile)
	}

	switch {
	case !persistent:
	case entityName != "":
		idx.addEntityLocked(entityName, fileURI)
	default:
		idx.defaultEntities[byFile] = append(idx.defaultEntities[byFile], fileURI)
	}
}

// addEntityLocked records that fileURI declares entity name; idx.mu is held.
func (idx *Index) addEntityLocked(name string, fileURI uri.URI) {
	name, key := strings.ToLower(name), uriKey(fileURI)

	if !slices.ContainsFunc(idx.entities[name], func(u uri.URI) bool { return uriKey(u) == key }) {
		idx.entities[name] = append(idx.entities[name], fileURI)
	}

	if !slices.Contains(idx.entityNames[key], name) {
		idx.entityNames[key] = append(idx.entityNames[key], name)
	}
}

// dropEntitiesLocked forgets the entity names the file keyed key declared;
// idx.mu is held.
func (idx *Index) dropEntitiesLocked(key string) {
	for _, name := range idx.entityNames[key] {
		list := slices.DeleteFunc(idx.entities[name], func(u uri.URI) bool { return uriKey(u) == key })
		if len(list) == 0 {
			delete(idx.entities, name)
		} else {
			idx.entities[name] = list
		}
	}

	delete(idx.entityNames, key)
}

// EntityCandidates are the files an ORM entity name may name: those whose
// entityname it is, then every persistent component called after it that
// names none, each group sorted, for the caller to choose the nearest. Both
// groups are offered because each application has its own entities: an
// Author declared by entityname in one must not hide a plain Author.cfc
// nearer the caller in another. Where the two are equally near, the declared
// one comes first and wins.
func (idx *Index) EntityCandidates(name string) []uri.URI {
	_, all := idx.EntityChoices(name)

	return all
}

// EntityChoices is EntityCandidates with the declared ones split out
// (declared is a prefix of all), read under one lock so the two cannot
// disagree when a file is re-indexed between two reads.
func (idx *Index) EntityChoices(name string) (declared, all []uri.URI) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	name = strings.ToLower(name)
	named, byFile := idx.entities[name], idx.defaultEntities[name]

	all = make([]uri.URI, 0, len(named)+len(byFile))
	all = append(all, named...)
	slices.Sort(all)
	all = append(all, byFile...)
	slices.Sort(all[len(named):])

	return all[:len(named):len(named)], all
}

// NamedEntities are the files whose entityname is name (case-insensitive),
// sorted, for the caller to choose the nearest.
func (idx *Index) NamedEntities(name string) []uri.URI {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	out := slices.Clone(idx.entities[strings.ToLower(name)])
	slices.Sort(out)

	return out
}
