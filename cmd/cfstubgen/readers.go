package main

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

// The readers that find what a stub needs in a framework's CFML source: the
// access, hint and return type a declaration states, its properties, whether
// the file is an interface. They replace the regular expressions that did, and
// answer as each did (readers_test.go holds each to it, and the stubs they
// generate are the same byte for byte).

// isWordAt reports whether s[i] is a word character; false off either end.
func isWordAt(s string, i int) bool { return i >= 0 && i < len(s) && lex.IsWord(s[i]) }

// keywordAt reports whether the word kw stands at i, ignoring case, with no
// word character after it.
func keywordAt(s string, i int, kw string) bool {
	return lex.HasPrefixFold(s[i:], kw) && !isWordAt(s, i+len(kw))
}

// lineStart reports whether i begins a line: the text's start, or after '\n'.
func lineStart(s string, i int) bool { return i == 0 || s[i-1] == '\n' }

// wordRun is the end of the run of word characters at i.
func wordRun(s string, i int) int {
	for i < len(s) && lex.IsWord(s[i]) {
		i++
	}

	return i
}

// integrateTargets are the packages named by each `$integrateComponents( "pkg" )`
// in text, in order: FindAllStringSubmatch of
// `\$integrateComponents\(\s*["']([\w.]+)["']\s*\)`, group one.
func integrateTargets(text string) []string {
	const call = "$integrateComponents("

	var out []string

	for from := 0; ; {
		at := strings.Index(text[from:], call)
		if at < 0 {
			return out
		}

		at += from
		i := lex.SkipRegexpSpace(text, at+len(call))

		if i < len(text) && lex.IsQuote(text[i]) {
			start := i + 1

			end := start
			for end < len(text) && (lex.IsWord(text[end]) || text[end] == '.') {
				end++
			}

			if end > start && end < len(text) && lex.IsQuote(text[end]) {
				if rp := lex.SkipRegexpSpace(text, end+1); rp < len(text) && text[rp] == ')' {
					out = append(out, text[start:end])
					from = rp + 1

					continue
				}
			}
		}

		from = at + 1
	}
}

// validTypeName reports whether t is a type name: `^[A-Za-z_][\w.]*(\[\])?$`.
func validTypeName(t string) bool {
	if t == "" || !lex.IsWordStart(t[0]) {
		return false
	}

	i := 1
	for i < len(t) && (lex.IsWord(t[i]) || t[i] == '.') {
		i++
	}

	return t[i:] == "" || t[i:] == "[]"
}

// identName reports whether s is a variable name as CFML spells one, `$assert`
// included: `^[A-Za-z_$][\w$]*$`.
func identName(s string) bool {
	if s == "" || (!lex.IsWordStart(s[0]) && s[0] != '$') {
		return false
	}

	for i := 1; i < len(s); i++ {
		if !lex.IsIdent(s[i]) {
			return false
		}
	}

	return true
}

// docReturnName is the component a doc comment's `@return` or `@returns` line
// names, a name and the dotted segments after it: the group of
// `(?im)^@returns?\s+([A-Za-z_]\w*(?:\.\w+)*)\b`, which needs one segment or
// more with dotted, `(?:\.\w+)+`. The line begins with the tag.
func docReturnName(doc string, dotted bool) (string, bool) {
	for q := range len(doc) {
		if !lineStart(doc, q) || !lex.HasPrefixFold(doc[q:], "@return") {
			continue
		}

		i := q + len("@return")
		if i < len(doc) && (doc[i] == 's' || doc[i] == 'S') {
			i++
		}

		start := lex.SkipRegexpSpace(doc, i)
		if start == i || start >= len(doc) || !lex.IsWordStart(doc[start]) {
			continue
		}

		end := wordRun(doc, start)
		segments := 0

		for end+1 < len(doc) && doc[end] == '.' && lex.IsWord(doc[end+1]) {
			end = wordRun(doc, end+1)
			segments++
		}

		if dotted && segments == 0 {
			continue
		}

		return doc[start:end], true
	}

	return "", false
}

// delegateName is the function a body that is `{ return f(); }` returns the
// call of: the group of `(?is)^\s*\{\s*return\s+([A-Za-z_]\w*)\s*\(\s*\)\s*;?\s*\}\s*$`.
func delegateName(body string) (string, bool) {
	i := lex.SkipRegexpSpace(body, 0)
	if i >= len(body) || body[i] != '{' {
		return "", false
	}

	i = lex.SkipRegexpSpace(body, i+1)
	if !lex.HasPrefixFold(body[i:], "return") {
		return "", false
	}

	name := lex.SkipRegexpSpace(body, i+len("return"))
	if name == i+len("return") || name >= len(body) || !lex.IsWordStart(body[name]) {
		return "", false
	}

	end := wordRun(body, name)

	i = lex.SkipRegexpSpace(body, end)
	if i >= len(body) || body[i] != '(' {
		return "", false
	}

	i = lex.SkipRegexpSpace(body, i+1)
	if i >= len(body) || body[i] != ')' {
		return "", false
	}

	i = lex.SkipRegexpSpace(body, i+1)
	if i < len(body) && body[i] == ';' {
		i++
	}

	i = lex.SkipRegexpSpace(body, i)
	if i >= len(body) || body[i] != '}' {
		return "", false
	}

	if lex.SkipRegexpSpace(body, i+1) != len(body) {
		return "", false
	}

	return body[name:end], true
}

// declaresInterface reports whether text holds an interface: a line that
// begins, after space, with the word `interface`, or a `<cfinterface`:
// MatchString of `(?im)^\s*interface\b|<cfinterface\b`.
func declaresInterface(text string) bool {
	for i := range len(text) {
		switch {
		case text[i] == '<' && keywordAt(text, i, "<cfinterface"):
			return true
		case lex.IsWordStart(text[i]) && !isWordAt(text, i-1) && keywordAt(text, i, "interface") && onlySpaceBefore(text, i):
			return true
		}
	}

	return false
}

// onlySpaceBefore reports whether only whitespace stands between the start of
// i's line and i.
func onlySpaceBefore(text string, i int) bool {
	for j := i - 1; j >= 0 && text[j] != '\n'; j-- {
		if !lex.IsRegexpSpace(text[j]) {
			return false
		}
	}

	return true
}

// accessWord is the access word of a script declaration, the first of
// public, private, package and remote that stands alone and is followed, before
// any parenthesis, by the word `function`: the group of
// `(?i)\b(public|private|package|remote)\b[^(]*\bfunction\b`.
func accessWord(line string) (string, bool) {
	for i := range len(line) {
		if !lex.IsWordStart(line[i]) || isWordAt(line, i-1) {
			continue
		}

		for _, word := range [...]string{"public", "private", "package", "remote"} {
			if !keywordAt(line, i, word) {
				continue
			}

			rest := i + len(word)

			stop := strings.IndexByte(line[rest:], '(')
			if stop < 0 {
				stop = len(line) - rest
			}

			for at := rest; at+len("function") <= rest+stop; at++ {
				if !isWordAt(line, at-1) && keywordAt(line, at, "function") {
					return line[i:rest], true
				}
			}
		}
	}

	return "", false
}

// tagAccessWord is the access a `<cffunction>` tag states, `access="public"`
// (quotes optional): the group of
// `(?i)\baccess\s*=\s*["']?(public|private|package|remote)`.
func tagAccessWord(tag string) (string, bool) {
	for i := range len(tag) {
		if !lex.HasPrefixFold(tag[i:], "access") || isWordAt(tag, i-1) {
			continue
		}

		j := lex.SkipRegexpSpace(tag, i+len("access"))
		if j >= len(tag) || tag[j] != '=' {
			continue
		}

		j = lex.SkipRegexpSpace(tag, j+1)
		if j < len(tag) && lex.IsQuote(tag[j]) {
			j++
		}

		for _, word := range [...]string{"public", "private", "package", "remote"} {
			if lex.HasPrefixFold(tag[j:], word) {
				return tag[j : j+len(word)], true
			}
		}
	}

	return "", false
}

// hintAttr is the first `hint="…"` or `hint='…'` in a tag, with its quotes
// either kind and holding anything but that quote, a newline included:
// FindStringSubmatch of `(?is)\bhint\s*=\s*"([^"]*)"|\bhint\s*=\s*'([^']*)'`,
// the double-quoted group and the single-quoted one; found is false for none.
func hintAttr(tag string) (double, single string, found bool) {
	for i := range len(tag) {
		if !lex.HasPrefixFold(tag[i:], "hint") || isWordAt(tag, i-1) {
			continue
		}

		j := lex.SkipRegexpSpace(tag, i+len("hint"))
		if j >= len(tag) || tag[j] != '=' {
			continue
		}

		j = lex.SkipRegexpSpace(tag, j+1)
		if j >= len(tag) || (tag[j] != '"' && tag[j] != '\'') {
			continue
		}

		cl := strings.IndexByte(tag[j+1:], tag[j])
		if cl < 0 {
			continue
		}

		if tag[j] == '"' {
			return tag[j+1 : j+1+cl], "", true
		}

		return "", tag[j+1 : j+1+cl], true
	}

	return "", "", false
}

// componentStart is where the component's declaration begins in text, the
// start FindStringIndex of `(?im)^\s*(abstract\s+|final\s+)?component\b|<cfcomponent\b`
// gave: the first line start from which space, an optional `abstract` or
// `final`, and the word `component` follow (space may span lines, so a blank
// line before the declaration is where the match begins), or the first
// `<cfcomponent`, whichever is first.
func componentStart(text string) (int, bool) {
	tag := -1

	for from := 0; from < len(text); {
		at := strings.IndexByte(text[from:], '<')
		if at < 0 {
			break
		}

		if keywordAt(text, from+at, "<cfcomponent") {
			tag = from + at

			break
		}

		from += at + 1
	}

	for q := 0; q < len(text); {
		if !lineStart(text, q) {
			q++

			continue
		}

		if tag >= 0 && q >= tag {
			break
		}

		p := lex.SkipRegexpSpace(text, q)
		if componentWordAt(text, p) {
			return q, true
		}

		// Every line start before p reaches the same p.
		q = max(q+1, p)
	}

	return tag, tag >= 0
}

// componentWordAt reports whether `(abstract\s+|final\s+)?component\b` is at p.
func componentWordAt(text string, p int) bool {
	for _, modifier := range [...]string{"abstract", "final"} {
		if lex.HasPrefixFold(text[p:], modifier) {
			space := lex.SkipRegexpSpace(text, p+len(modifier))

			return space > p+len(modifier) && keywordAt(text, space, "component")
		}
	}

	return keywordAt(text, p, "component")
}

// scriptProperties are the spans of each script property declaration in text,
// from the start of its line to its semicolon: FindAllStringIndex of
// `(?ims)^[ \t]*property\s+[^;{}]*?;`. The declaration holds no brace.
func scriptProperties(text string) [][2]int {
	var out [][2]int

	for q := 0; q < len(text); {
		if !lineStart(text, q) {
			q++

			continue
		}

		i := q
		for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
			i++
		}

		end := -1

		if lex.HasPrefixFold(text[i:], "property") {
			after := i + len("property")
			if lex.SkipRegexpSpace(text, after) > after {
				for j := after; j < len(text); j++ {
					if text[j] == '{' || text[j] == '}' {
						break
					}

					if text[j] == ';' {
						end = j + 1

						break
					}
				}
			}
		}

		if end < 0 {
			q++

			continue
		}

		out = append(out, [2]int{q, end})
		q = end
	}

	return out
}

// tagProperties are the attributes of each `<cfproperty …>` in text, up to its
// `>` and without a `/` just before it: group one of FindAllStringSubmatch of
// `(?is)<cfproperty\b([^>]*?)/?>`.
func tagProperties(text string) []string {
	const open = "<cfproperty"

	var out []string

	for from := 0; ; {
		at := parser.IndexFoldFrom(text, open, from)
		if at < 0 {
			return out
		}

		start := at + len(open)

		if isWordAt(text, start) {
			from = at + 1

			continue
		}

		cl := strings.IndexByte(text[start:], '>')
		if cl < 0 {
			from = at + 1

			continue
		}

		attrs := text[start : start+cl]
		attrs = strings.TrimSuffix(attrs, "/")

		out = append(out, attrs)
		from = start + cl + 1
	}
}
