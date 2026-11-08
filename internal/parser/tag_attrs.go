// Reading a tag's attributes and the component paths its expressions name.

package parser

import (
	"strings"
)

// getAttr extracts an attribute value from a tag string (case-insensitive).
func getAttr(tag, attr string) string {
	// Scanned case-insensitively in place rather than against a lowercased copy
	// of the tag.
	//
	// Two reasons, and the second is a correctness one. A <cffunction> tag is
	// asked for five or six attributes, so the copy was made five or six times
	// for one tag — strings.ToLower was 10% of tag parsing in a profile, the
	// largest single cost in it, and getAttr 13% including it. And the offsets
	// from that copy were then used to index `tag`, which only holds while
	// folding preserves length: it does not universally (U+0130 lowercases to
	// two runes), so a tag carrying one desynchronised every offset after it.
	//
	// Find the attribute name followed by optional whitespace and =
	idx := -1
	searchFrom := 0

	for {
		i := indexFoldFrom(tag, attr, searchFrom)
		if i < 0 {
			break
		}

		// The name has to start a word. Only the text *after* the match used to
		// be checked, so a plain substring hit inside a longer attribute name
		// counted: "name" matched the tail of "displayname=", and
		// <cffunction displayname="Donor Lookup" name="getDonor"> was indexed
		// as a function literally called "Donor Lookup". Same for "type"
		// inside "returntype=" and any other attribute ending in the one being
		// looked up.
		if i > 0 && isAttrNameByte(tag[i-1]) {
			searchFrom = i + 1

			continue
		}

		j := i + len(attr)
		for j < len(tag) && isWhitespace(tag[j]) {
			j++
		}

		if j < len(tag) && tag[j] == '=' {
			idx = i

			break
		}

		searchFrom = i + 1
	}

	if idx < 0 {
		return ""
	}

	// Skip past attr name, whitespace, =, whitespace to reach the value
	valStart := idx + len(attr)
	for valStart < len(tag) && isWhitespace(tag[valStart]) {
		valStart++
	}

	valStart++ // skip =

	for valStart < len(tag) && isWhitespace(tag[valStart]) {
		valStart++
	}

	if valStart >= len(tag) {
		return ""
	}

	ch := tag[valStart]
	if ch == '"' || ch == '\'' {
		valStart++

		end := strings.IndexByte(tag[valStart:], ch)
		if end < 0 {
			return ""
		}

		return tag[valStart : valStart+end]
	}
	// Unquoted value — read until space or >
	end := valStart
	for end < len(tag) && tag[end] != ' ' && tag[end] != '>' && tag[end] != '\t' {
		end++
	}

	return tag[valStart:end]
}

func extractIdent(s string) string {
	i := 0
	for i < len(s) && isIdentPart(s[i]) {
		i++
	}

	if i == 0 {
		return ""
	}

	return s[:i]
}

// extractAllAttrs extracts all attribute key=value pairs from a tag string.
// Keys are lowercased.
func extractAllAttrs(tag string) map[string]string {
	attrs := make(map[string]string)
	// Skip tag name
	i := 1 // past '<'
	for i < len(tag) && tag[i] != ' ' && tag[i] != '\t' && tag[i] != '>' {
		i++
	}

	for i < len(tag) {
		// Skip whitespace
		for i < len(tag) && isWhitespace(tag[i]) {
			i++
		}

		if i >= len(tag) || tag[i] == '>' || tag[i] == '/' {
			break
		}
		// Read attribute name
		start := i

		for i < len(tag) && tag[i] != '=' && tag[i] != ' ' && tag[i] != '>' && tag[i] != '/' {
			i++
		}

		key := strings.ToLower(tag[start:i])

		if i >= len(tag) || tag[i] != '=' {
			continue
		}

		i++ // past '='
		if i >= len(tag) {
			break
		}

		if tag[i] == '"' || tag[i] == '\'' {
			q := tag[i]
			i++
			valStart := i

			for i < len(tag) && tag[i] != q {
				i++
			}

			attrs[key] = tag[valStart:i]

			if i < len(tag) {
				i++ // past closing quote
			}
		} else {
			valStart := i

			for i < len(tag) && tag[i] != ' ' && tag[i] != '>' {
				i++
			}

			attrs[key] = tag[valStart:i]
		}
	}

	return attrs
}

func splitAssign(s string) (name, rhs string) {
	name = extractIdent(s)
	if name == "" {
		return "", ""
	}

	rest := strings.TrimSpace(s[len(name):])
	if rest != "" && rest[0] == '=' {
		return name, strings.TrimSpace(rest[1:])
	}

	return name, ""
}

func extractComponentPath(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	if s[0] == '"' || s[0] == '\'' {
		q := s[0]

		end := strings.IndexByte(s[1:], q)
		if end < 0 {
			return ""
		}

		return s[1 : 1+end]
	}
	// Unquoted dotted path
	i := 0
	for i < len(s) && (isIdentPart(s[i]) || s[i] == '.') {
		i++
	}

	return s[:i]
}

func extractCreateObjectArg(s string) string {
	// Expects: "component", "path") — we're past the opening (
	s = strings.TrimSpace(s)
	if s == "" || (s[0] != '"' && s[0] != '\'') {
		return ""
	}

	q := s[0]

	end := strings.IndexByte(s[1:], q)
	if end < 0 {
		return ""
	}

	first := s[1 : 1+end]
	if !strings.EqualFold(first, "component") {
		return ""
	}

	rest := s[2+end:]

	ci := strings.IndexByte(rest, ',')
	if ci < 0 {
		return ""
	}

	rest = strings.TrimSpace(rest[ci+1:])
	if rest == "" || rest[0] == ')' {
		return ""
	}

	// A path computed at run time — drivernames[ type ], "pkg." & name —
	// names whichever component the program picks, and is dynamic, as the
	// script parser reads it (readCreateObjectComponent).
	if rest[0] != '"' && rest[0] != '\'' {
		return "$any"
	}

	q2 := rest[0]

	end2 := strings.IndexByte(rest[1:], q2)
	if end2 < 0 {
		return ""
	}

	if after := strings.TrimSpace(rest[2+end2:]); after != "" && after[0] != ')' && after[0] != ',' {
		return "$any"
	}

	return rest[1 : 1+end2]
}

func extractEntityNewArg(s string) string {
	// Expects: "EntityName") — we're past the opening (
	s = strings.TrimSpace(s)
	if s == "" || (s[0] != '"' && s[0] != '\'') {
		return ""
	}

	q := s[0]

	end := strings.IndexByte(s[1:], q)
	if end < 0 {
		return ""
	}

	// A computed name, entityNew( "Comp" & nbr ), is no entity called Comp.
	if rest := strings.TrimSpace(s[2+end:]); rest != "" && rest[0] != ')' && rest[0] != ',' {
		return "$any"
	}

	return s[1 : 1+end]
}

// extractMethodCallBase returns the base variable name from a "baseVar.method(" pattern.
// For "variables.jss.getInstance(..." returns "jss". Returns "" if not a method call.
func extractMethodCallBase(rhs string) string {
	prefix, _, ok := strings.Cut(rhs, "(")
	if !ok {
		return ""
	}

	base, _, ok := strings.CutLast(prefix, ".")
	if !ok {
		return ""
	}

	if _, after, ok := strings.CutLast(base, "."); ok {
		return after
	}

	return base
}

func toLowerByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}

	return b
}

// HasPrefixFold is hasPrefixFold, shared with internal/resolve.
func HasPrefixFold(s, prefix string) bool { return hasPrefixFold(s, prefix) }

// hasPrefixFold checks if s starts with prefix (case-insensitive, ASCII only).
func hasPrefixFold(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}

	return strings.EqualFold(s[:len(prefix)], prefix)
}
