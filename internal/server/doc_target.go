package server

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// What a doc block documents is read from the text after it, as the
// extension's own patterns read it. These readers replace those patterns and
// answer as each did; doc_target_test.go holds them to the originals.

// docModifiers are the words a function may carry before its return type.
var docModifiers = []string{"private", "package", "public", "remote", "static", "final", "abstract", "default"}

// docToken is the run of non-space bytes at i, and where it ends.
func docToken(s string, i int) (tok string, end int) {
	end = i
	for end < len(s) && !lex.IsRegexpSpace(s[end]) {
		end++
	}

	return s[i:end], end
}

// docIsModifier reports whether tok is one of the words a function may carry.
func docIsModifier(tok string) bool {
	for _, m := range docModifiers {
		if strings.EqualFold(tok, m) {
			return true
		}
	}

	return false
}

// docIsReturnType reports whether tok can be a return type: a word or a dotted
// path, which cannot begin with a dot or a dollar sign (a word boundary has
// to stand before it).
func docIsReturnType(tok string) bool {
	if tok == "" || !lex.IsWord(tok[0]) {
		return false
	}

	for i := range len(tok) {
		if !lex.IsIdent(tok[i]) && tok[i] != '.' {
			return false
		}
	}

	return true
}

// docFunctionEnd is where the declaration of a function ends at the start of
// s: FindStringIndex of
//
//	(?i)^(\s*)(?:\b(?:private|package|public|remote|static|final|abstract|default)\s+)*
//	    (?:\b(?:[A-Za-z0-9_.$]+)\s+)?function\s+([_$a-zA-Z][$\w]*)\s*
//	    (?:\((?:=\s*\{|[^{])*)[{;]
//
// the end, just past the `{` or `;` that closes the header, and whether there
// is one. A modifier or a return type is a word the whitespace ends, so the
// header is words, the word `function`, a name and a `(`.
func docFunctionEnd(s string) (end int, ok bool) {
	i := lex.SkipRegexpSpace(s, 0)

	// The words before the keyword: every one a modifier, or all but the last,
	// which may be any return type. The keyword is a word of its own.
	allModifiers := true

	for prev := ""; i < len(s); {
		tok, next := docToken(s, i)

		if strings.EqualFold(tok, "function") && (prev == "" || allModifiers || docIsReturnType(prev)) {
			if end, ok := docFunctionTail(s, next); ok {
				return end, true
			}
		}

		// A modifier keeps the run going. Any other word is the return type if
		// the keyword comes next, and nothing may stand between them.
		if !allModifiers {
			return 0, false
		}

		if !docIsModifier(tok) {
			allModifiers = false
		}

		prev, i = tok, lex.SkipRegexpSpace(s, next)
	}

	return 0, false
}

// docFunctionTail reads the rest of a header after the word `function` ends at
// i: whitespace, the name, an optional space and the `(` that opens the
// parameters.
func docFunctionTail(s string, i int) (end int, ok bool) {
	name := lex.SkipRegexpSpace(s, i)
	if name == i || name >= len(s) || !(lex.IsWordStart(s[name]) || s[name] == '$') {
		return 0, false
	}

	j := name + 1
	for j < len(s) && lex.IsIdent(s[j]) {
		j++
	}

	j = lex.SkipRegexpSpace(s, j)
	if j >= len(s) || s[j] != '(' {
		return 0, false
	}

	return docParametersEnd(s, j+1)
}

// docParametersEnd is where the parameters of a header end, from just past its
// `(`: the first `{` that does not follow an `=` (a default that is a struct
// literal, `x = {}`); failing that, the last `;` or `{` in the text. A `{`
// after an `=` is part of the parameters unless nothing later closes them,
// which is how the pattern's backtracking ended.
func docParametersEnd(s string, i int) (end int, ok bool) {
	last := -1

	for i < len(s) {
		switch s[i] {
		case '{':
			return i + 1, true
		case ';':
			last = i + 1
		case '=':
			if q := lex.SkipRegexpSpace(s, i+1); q < len(s) && s[q] == '{' {
				last = q + 1
				i = q
			}
		}

		i++
	}

	return last, last >= 0
}

// docPropertyStart reports whether s begins with the word `property` and
// space: `(?i)^(\s*property)\s+`.
func docPropertyStart(s string) bool {
	i := lex.SkipRegexpSpace(s, 0)

	return lex.HasPrefixFold(s[i:], "property") && lex.SkipRegexpSpace(s, i+len("property")) > i+len("property")
}

// docComponentStart is the word `component` or `interface` that begins s and
// opens a body, with a `{` anywhere after it: `(?i)^(\s*(component|interface))\b[^{]*\{`,
// the second group.
func docComponentStart(s string) (word string, ok bool) {
	i := lex.SkipRegexpSpace(s, 0)

	for _, w := range []string{"component", "interface"} {
		if !lex.HasPrefixFold(s[i:], w) {
			continue
		}

		end := i + len(w)
		if end < len(s) && lex.IsWord(s[end]) {
			return "", false
		}

		if !strings.Contains(s[end:], "{") {
			return "", false
		}

		return s[i:end], true
	}

	return "", false
}

// docCFTagStart is the tag name of a `<cffunction`, `<cfproperty`,
// `<cfcomponent` or `<cfinterface` that begins s: `(?i)^\s*<cf(function|property|component|interface)\b`,
// the group.
func docCFTagStart(s string) (name string, ok bool) {
	i := lex.SkipRegexpSpace(s, 0)
	if !lex.HasPrefixFold(s[i:], "<cf") {
		return "", false
	}

	i += len("<cf")

	for _, w := range []string{"function", "property", "component", "interface"} {
		if !lex.HasPrefixFold(s[i:], w) {
			continue
		}

		end := i + len(w)
		if end < len(s) && lex.IsWord(s[end]) {
			return "", false
		}

		return s[i:end], true
	}

	return "", false
}

// docTagWord reads the `@tag` or `@tag.key` the text before the cursor ends
// with, as FindStringSubmatchIndex of `@([\w$]*)(?:\.([\w$]*))?$` did:
// [0, end, tagStart, tagEnd, keyStart, keyEnd], the key's -1 when there is no
// dot, or nil. Nothing but the last `@` can begin it.
func docTagWord(s string) []int {
	at := strings.LastIndexByte(s, '@')
	if at < 0 {
		return nil
	}

	tagStart := at + 1
	tagEnd := tagStart

	for tagEnd < len(s) && lex.IsIdent(s[tagEnd]) {
		tagEnd++
	}

	if tagEnd == len(s) {
		return []int{at, len(s), tagStart, tagEnd, -1, -1}
	}

	if s[tagEnd] != '.' {
		return nil
	}

	keyStart := tagEnd + 1
	keyEnd := keyStart

	for keyEnd < len(s) && lex.IsIdent(s[keyEnd]) {
		keyEnd++
	}

	if keyEnd != len(s) {
		return nil
	}

	return []int{at, len(s), tagStart, tagEnd, keyStart, keyEnd}
}
