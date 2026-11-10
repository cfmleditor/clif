// Reading a tag's attributes and the component paths its expressions name.

package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// getAttr extracts an attribute value from a tag string (case-insensitive).
func getAttr(tag, attr string) string {
	start, end, ok := attrSpan(tag, attr)
	if !ok {
		return ""
	}

	return tag[start:end]
}

// attrSpan is where the value of attr stands in tag, quotes excluded, found
// without case as a CF tag's attributes are read (attrScanner); ok is false
// when tag does not set attr, or its quoted value is never closed.
func attrSpan(tag, attr string) (start, end int, ok bool) {
	return tagAttrValue(tag, attr, cfAttrs)
}

// TagAttribute is the value of attr in tag, a CF tag's text, read as the
// parse reads it; "" when the tag does not set it.
func TagAttribute(tag, attr string) string { return getAttr(tag, attr) }

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

// extractAllAttrs is every attribute of tag, a CF tag's text, by name
// lowercased (tagAttrMap).
func extractAllAttrs(tag string) map[string]string {
	return tagAttrMap(tag, cfAttrs)
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
func HasPrefixFold(s, prefix string) bool { return lex.HasPrefixFold(s, prefix) }
