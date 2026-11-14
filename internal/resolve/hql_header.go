package resolve

import (
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

// The entity a literal HQL query selects from is read from its start, and its
// keywords found anywhere in it. These readers replace the three expressions
// that did, and answer as each did (hql_header_test.go holds them to it).

// hqlWordEnd is the index just past the run of word characters at i.
func hqlWordEnd(s string, i int) int {
	for i < len(s) && lex.IsWord(s[i]) {
		i++
	}

	return i
}

// hqlKeywordAt is the index just past the word kw at i, ignoring case, or -1
// when it is not there.
func hqlKeywordAt(s string, i int, kw string) int {
	if !lex.HasPrefixFold(s[i:], kw) {
		return -1
	}

	return i + len(kw)
}

// hqlEntityHeader reads `[SELECT x] FROM Entity [[AS] alias]` at the start of
// query, as FindStringSubmatch of
//
//	(?is)^\s*(?:select\s+(\w+)\s+)?from\s+([A-Za-z_][\w.]*)(?:\s+(?:as\s+)?(\w+))?(\s|$)
//
// answered: selected is the word after SELECT, entity the name after FROM, and
// alias the word after it, which is empty when none follows or the word after
// it is not ended by whitespace or the end (`from Booking b, Other`).
func hqlEntityHeader(query string) (selected, entity, alias string, ok bool) {
	i := lex.SkipRegexpSpace(query, 0)

	if end := hqlKeywordAt(query, i, "select"); end >= 0 {
		space := lex.SkipRegexpSpace(query, end)
		if space == end {
			return "", "", "", false
		}

		word := hqlWordEnd(query, space)
		if word == space {
			return "", "", "", false
		}

		next := lex.SkipRegexpSpace(query, word)
		if next == word {
			return "", "", "", false
		}

		selected, i = query[space:word], next
	}

	end := hqlKeywordAt(query, i, "from")
	if end < 0 {
		return "", "", "", false
	}

	start := lex.SkipRegexpSpace(query, end)
	if start == end || start >= len(query) || !lex.IsWordStart(query[start]) {
		return "", "", "", false
	}

	stop := start + 1
	for stop < len(query) && (lex.IsWord(query[stop]) || query[stop] == '.') {
		stop++
	}

	entity = query[start:stop]

	// The alias is tried first: whitespace, `as` and whitespace, a word.
	if gap := lex.SkipRegexpSpace(query, stop); gap > stop {
		if as := hqlKeywordAt(query, gap, "as"); as >= 0 {
			if name := lex.SkipRegexpSpace(query, as); name > as {
				if word := hqlWordEnd(query, name); word > name && hqlEndsHere(query, word) {
					return selected, entity, query[name:word], true
				}
			}
		}

		if word := hqlWordEnd(query, gap); word > gap && hqlEndsHere(query, word) {
			return selected, entity, query[gap:word], true
		}
	}

	// No alias: the entity itself must end at whitespace or the end.
	if hqlEndsHere(query, stop) {
		return selected, entity, "", true
	}

	return "", "", "", false
}

// hqlEndsHere reports whether i is at whitespace or the end: `(\s|$)`.
func hqlEndsHere(s string, i int) bool {
	return i >= len(s) || lex.IsRegexpSpace(s[i])
}

// hqlKeywordIndex is the index of the first occurrence of the word kw in s,
// ignoring case, with no word character before or after it: the start of a
// match of `(?i)\bkw\b`, or -1.
func hqlKeywordIndex(s, kw string) int {
	for from := 0; from+len(kw) <= len(s); from++ {
		at := parser.IndexFoldFrom(s, kw, from)
		if at < 0 {
			return -1
		}

		if (at == 0 || !lex.IsWord(s[at-1])) && (at+len(kw) == len(s) || !lex.IsWord(s[at+len(kw)])) {
			return at
		}

		from = at
	}

	return -1
}

// hqlHasJoin reports whether the query holds the word `join`: MatchString of
// `(?i)\bjoin\b`.
func hqlHasJoin(query string) bool { return hqlKeywordIndex(query, "join") >= 0 }

// hqlBeforeWhere is the query up to its first `where` word, or all of it: the
// first piece of Split of `(?i)\bwhere\b` into at most two.
func hqlBeforeWhere(query string) string {
	if at := hqlKeywordIndex(query, "where"); at >= 0 {
		return query[:at]
	}

	return query
}
