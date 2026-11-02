package parser

import (
	"strings"
	"unicode/utf8"

	"github.com/cfmleditor/clif/internal/conv"
)

// collKey names a collection, or a record member, the passes that type them
// track: a name within one function's locals, or within the variables or this
// scope of the component. The passes used to build it as a string,
// "<function>\t<name>" for a local and "variables.<name>" or "this.<name>" for
// a field, for every write and every lookup: a concatenation, and a lowercased
// copy of a camelCase name, each time.
//
// It is now a number packing the kind, the scope and the name, each scope and
// name interned once in a keyTable. A key holds no pointer, so the maps keyed
// by one are hashed as integers and the collector has nothing in them to scan.
// Keyed by a struct of the same strings it was slower than the strings: the
// generic map code, and buckets three times the size, cost more than the
// concatenations they saved.
type collKey uint64

const (
	collLocalBit = 1 << 63

	// The scope words a field's key is in, and the name that stands for a
	// whole scope, are interned first, so their ids are these.
	idVariables  = 0 // "variables."
	idThis       = 1 // "this."
	idScopeLevel = 2 // "*": no identifier is called this
)

func packKey(local bool, scope, name uint32) collKey {
	k := collKey(scope)<<32 | collKey(name)
	if local {
		k |= collLocalBit
	}

	return k
}

func (k collKey) local() bool     { return k&collLocalBit != 0 }
func (k collKey) scopeID() uint32 { return uint32(k >> 32 & 0x7fffffff) }
func (k collKey) nameID() uint32  { return uint32(k & 0xffffffff) }

// keyTable interns the names and scopes of a pass's keys. It is used by one
// goroutine, for one parse.
type keyTable struct {
	ids  map[string]uint32
	strs []string
}

func newKeyTable() *keyTable {
	t := &keyTable{ids: map[string]uint32{}}

	for _, s := range []string{"variables.", "this.", "*"} {
		t.ids[s] = conv.Uint32(len(t.strs))
		t.strs = append(t.strs, s)
	}

	return t
}

// lowerID is the id of s lowercased. A spelling with capitals is remembered
// against the id of its lowercase form, so asking for it again is one lookup.
func (t *keyTable) lowerID(s string) uint32 {
	if id, ok := t.ids[s]; ok {
		return id
	}

	lower := s
	if needsLower(s) {
		lower = strings.ToLower(s)
	}

	id, ok := t.ids[lower]
	if !ok {
		id = conv.Uint32(len(t.strs))
		t.strs = append(t.strs, lower)
		t.ids[lower] = id
	}

	if lower != s {
		t.ids[s] = id
	}

	return id
}

// findLower is lowerID for a lookup that must not add what it asks about: a
// key built from a name nothing has used cannot be in any map.
func (t *keyTable) findLower(s string) (uint32, bool) {
	if id, ok := t.ids[s]; ok {
		return id, true
	}

	if !needsLower(s) {
		return 0, false
	}

	id, ok := t.ids[strings.ToLower(s)]

	return id, ok
}

func needsLower(s string) bool {
	for i := range len(s) {
		c := s[i]
		if c >= utf8.RuneSelf || c >= 'A' && c <= 'Z' {
			return true
		}
	}

	return false
}

// local is the key of name in function's locals.
func (t *keyTable) local(function, name string) collKey {
	return packKey(true, t.lowerID(function), t.lowerID(name))
}

// field is the key of name in the variables or this scope (idVariables or
// idThis).
func (t *keyTable) field(scope uint32, name string) collKey {
	return packKey(false, scope, t.lowerID(name))
}

// scopeLevel is the key standing for k's whole scope, which no collection has.
func scopeLevel(k collKey) collKey { return packKey(k.local(), k.scopeID(), idScopeLevel) }

func (t *keyTable) fieldScope(scope uint32) collKey { return packKey(false, scope, idScopeLevel) }

func (t *keyTable) localScope(function string) collKey {
	return packKey(true, t.lowerID(function), idScopeLevel)
}

// name is k's name, lowercased.
func (t *keyTable) name(k collKey) string { return t.strs[k.nameID()] }

// withName is k with the name s in place of its own.
func (t *keyTable) withName(k collKey, s string) collKey {
	return packKey(k.local(), k.scopeID(), t.lowerID(s))
}

// ownedBy reports whether k is a local of function.
func (t *keyTable) ownedBy(k collKey, function string) bool {
	id, ok := t.findLower(function)

	return k.local() && ok && k.scopeID() == id
}

// text is k as the passes used to spell it, for ordering by.
func (t *keyTable) text(k collKey) string {
	if k.local() {
		return t.strs[k.scopeID()] + "\t" + t.strs[k.nameID()]
	}

	return t.strs[k.scopeID()] + t.strs[k.nameID()]
}

// localSet is the names each function declares: its arguments and the locals
// it writes.
type localSet struct {
	kt *keyTable
	m  map[collKey]bool
}

func (s *localSet) add(function, name string) { s.m[s.kt.local(function, name)] = true }

// has reports whether function declares name. It adds nothing to the table.
func (s *localSet) has(function, name string) bool {
	fn, ok := s.kt.findLower(function)
	if !ok {
		return false
	}

	n, ok := s.kt.findLower(name)

	return ok && s.m[packKey(true, fn, n)]
}
