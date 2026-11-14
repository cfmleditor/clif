package unresolved

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// The reports read back the reasons the resolver wrote: which method was
// missing, which component, and the shape of a reason with its names and
// counts taken out. These readers replace the regular expressions that did,
// and answer as each did (reason_text_test.go holds them to it).

// quotedName reads `prefix 'name'` at the start of s, a name of one or more
// bytes that are not a quote, and returns the name and what follows its
// closing quote.
func quotedName(s, prefix string) (name, rest string, ok bool) {
	if !strings.HasPrefix(s, prefix) {
		return "", "", false
	}

	s = s[len(prefix):]

	end := strings.IndexByte(s, '\'')
	if end <= 0 {
		return "", "", false
	}

	return s[:end], s[end+1:], true
}

// failedMethodName is the method named by `method 'x' not found…`, as
// `^method '([^']+)' not found` read it.
func failedMethodName(reason string) (string, bool) {
	name, rest, ok := quotedName(reason, "method '")
	if !ok || !strings.HasPrefix(rest, " not found") {
		return "", false
	}

	return name, true
}

// methodMissingFrom is the component named by `method 'x' not found in c`, as
// `^method '[^']+' not found in (.+)$` read it: what follows is not empty and
// holds no newline.
func methodMissingFrom(reason string) (string, bool) {
	_, rest, ok := quotedName(reason, "method '")
	if !ok {
		return "", false
	}

	comp, ok := strings.CutPrefix(rest, " not found in ")
	if !ok || comp == "" || strings.ContainsRune(comp, '\n') {
		return "", false
	}

	return comp, true
}

// missingComponent is the component named by `component 'x' does not exist`,
// as `^component '([^']+)' does not exist` read it.
func missingComponent(reason string) (string, bool) {
	name, rest, ok := quotedName(reason, "component '")
	if !ok || !strings.HasPrefix(rest, " does not exist") {
		return "", false
	}

	return name, true
}

// withoutUncheckedCount is s without each `; N calls not checked` (or
// `; N inherited calls not checked`), as ReplaceAllString of
// `; \d+ (?:inherited )?calls? not checked` with "" did.
func withoutUncheckedCount(s string) string {
	if !strings.Contains(s, "; ") {
		return s
	}

	var b strings.Builder

	for i := 0; i < len(s); {
		if n := uncheckedCountAt(s[i:]); n > 0 {
			i += n

			continue
		}

		b.WriteByte(s[i])

		i++
	}

	return b.String()
}

// uncheckedCountAt is the length of the `; N [inherited ]call[s] not checked`
// that starts s, or 0.
func uncheckedCountAt(s string) int {
	rest, ok := strings.CutPrefix(s, "; ")
	if !ok {
		return 0
	}

	digits := 0
	for digits < len(rest) && rest[digits] >= '0' && rest[digits] <= '9' {
		digits++
	}

	if digits == 0 {
		return 0
	}

	rest = rest[digits:]

	rest, ok = strings.CutPrefix(rest, " ")
	if !ok {
		return 0
	}

	rest = strings.TrimPrefix(rest, "inherited ")

	rest, ok = strings.CutPrefix(rest, "call")
	if !ok {
		return 0
	}

	rest = strings.TrimPrefix(rest, "s")

	rest, ok = strings.CutPrefix(rest, " not checked")
	if !ok {
		return 0
	}

	return len(s) - len(rest)
}

// quotesBlanked is s with each quoted run, 'x' or "x", replaced by '_', as
// ReplaceAllString of `'[^']*'|"[^"]*"` with "'_'" did: a quote with no
// closing one is left as it is.
func quotesBlanked(s string) string {
	if !strings.ContainsAny(s, `'"`) {
		return s
	}

	var b strings.Builder

	for i := 0; i < len(s); {
		if c := s[i]; c == '\'' || c == '"' {
			if end := strings.IndexByte(s[i+1:], c); end >= 0 {
				b.WriteString("'_'")

				i += end + 2

				continue
			}
		}

		b.WriteByte(s[i])

		i++
	}

	return b.String()
}

// numbersBlanked is s with each run of digits that stands alone, with no word
// character before or after it, replaced by N, as ReplaceAllString of
// `\b\d+\b` with "N" did.
func numbersBlanked(s string) string {
	var b strings.Builder

	for i := 0; i < len(s); {
		if s[i] < '0' || s[i] > '9' {
			b.WriteByte(s[i])

			i++

			continue
		}

		end := i
		for end < len(s) && s[end] >= '0' && s[end] <= '9' {
			end++
		}

		if (i == 0 || !lex.IsWord(s[i-1])) && (end == len(s) || !lex.IsWord(s[end])) {
			b.WriteByte('N')
		} else {
			b.WriteString(s[i:end])
		}

		i = end
	}

	return b.String()
}
