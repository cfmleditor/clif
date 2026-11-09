// Array and struct literals in CFScript: their items, inline and broken layouts, and keys.

package formatter

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// collectionItem is one entry of an array or struct literal. Comments are
// carried alongside the real elements so they survive formatting, but they
// never take a trailing comma.
type collectionItem struct {
	text      string
	isComment bool
}

// collectionItems renders a literal's children, reporting whether any of them
// is a line comment. A line comment runs to end of line, so a literal holding
// one can never be emitted inline — doing so commented out every element after
// it and destroyed the statement.
func (f *Formatter) collectionItems(n *sitter.Node) (items []collectionItem, hasLineComment bool) {
	items = make([]collectionItem, 0, n.NamedChildCount())

	for i := range n.NamedChildCount() {
		c := n.NamedChild(i)

		switch c.Kind() {
		case "comment":
			hasLineComment = true

			items = append(items, collectionItem{text: trimSpace(f.text(c)), isComment: true})
		case "block_comment", "cf_comment":
			// A CFML comment reaches here from a tag-context struct literal
			// (`<cfset x = { ... }>`). Left in the default branch it counted as
			// an element and was given a trailing comma. Both forms are
			// delimited, so unlike "//" they do not force the literal
			// multi-line.
			items = append(items, collectionItem{text: trimSpace(f.text(c)), isComment: true})
		default:
			items = append(items, collectionItem{text: f.expr(c)})
		}
	}

	return items, hasLineComment
}

// hasTrailingComma reports whether n's last separator is a comma with no
// element after it — `[1, 2, ]`, `{ a: 1, }`, `f(1, 2, )`, `function f(a, )`.
//
// Lucee, Adobe CF and BoxLang all accept the form, and it is common in
// hand-maintained lists because adding an entry then touches one line rather
// than two. Every renderer below builds its output by collecting elements and
// re-joining them with ", ", which reconstructs the separators from scratch and
// so silently dropped the final comma. That is a non-whitespace change, so the
// guard rejected the file and format-on-save did nothing to it at all.
//
// The comma is an anonymous child, and comments may follow it
// (`[1, 2, /* why */]`), so the scan runs backwards over the closing bracket and
// any trailing comments and reports on the first separator it reaches.
func hasTrailingComma(n *sitter.Node) bool {
	if n == nil {
		return false
	}

	for i := int(n.ChildCount()) - 1; i >= 0; i-- {
		switch kind := n.Child(uint(i)).Kind(); kind {
		case "]", "}", ")":
			continue
		case ",":
			return true
		default:
			if isCommentKind(kind) {
				continue
			}

			return false
		}
	}

	return false
}

// joinCollectionInline joins items for a single-line literal. Only safe when no
// item is a line comment, which would swallow the rest of the line.
//
// Commas follow the same rule as the multi-line form: they separate elements,
// and a comment is not one. Comma-joining every item turned an interleaved
// comment into a list entry — `[1, 2, <!--- why --->, 3]`. The comma owed to the
// element before the comment still gets written, so the elements either side
// stay separated.
func joinCollectionInline(items []collectionItem, trailing bool) string {
	lastElement := -1

	for i, it := range items {
		if !it.isComment {
			lastElement = i
		}
	}

	var b strings.Builder

	for i, it := range items {
		if i > 0 {
			b.WriteString(" ")
		}

		b.WriteString(it.text)

		if !it.isComment && (i < lastElement || (trailing && i == lastElement)) {
			b.WriteString(",")
		}
	}

	return b.String()
}

// joinCollectionLines lays items out one per line, giving a trailing comma to
// every element that still has an element after it, and never to a comment.
func joinCollectionLines(items []collectionItem, indent string, trailing bool) string {
	lastElement := -1

	for i, it := range items {
		if !it.isComment {
			lastElement = i
		}
	}

	var b strings.Builder

	for i, it := range items {
		if i > 0 {
			b.WriteString("\n")
			b.WriteString(indent)
		}

		b.WriteString(it.text)

		if !it.isComment && (i < lastElement || (trailing && i == lastElement)) {
			b.WriteString(",")
		}
	}

	return b.String()
}

// deepenItems re-renders, one level deeper, the items of a literal that is
// being laid out one item per line. collectionItems renders them at the
// literal's own level, which is right for the inline form but one level short
// once each item sits on its own indented line: a nested literal that itself
// spanned lines came out with its entries level with its key and its closing
// bracket level with the outer literal's entries. Only an item that spans
// lines depends on the level, so only those are rendered again.
//
// items holds one entry per named child of n, in order — collectionItems
// appends exactly one for each.
func (f *Formatter) deepenItems(n *sitter.Node, items []collectionItem) {
	f.level++
	defer func() { f.level-- }()

	for i := range items {
		if !items[i].isComment && strings.Contains(items[i].text, "\n") {
			items[i].text = f.expr(n.NamedChild(uint(i)))
		}
	}
}

// holdsMultilineClosure reports whether a literal has an entry whose value is a
// closure laid out across lines. Such a literal is always written one entry
// per line: kept inline because its total length was short, it came out as
// `{ a = function() {`, the closure's body, then `}, b = 1 }`, with the next
// entry and the literal's own brace hanging off the closure's.
//
// items holds one entry per named child of n, in order, as collectionItems
// builds it.
func holdsMultilineClosure(n *sitter.Node, items []collectionItem) bool {
	for i, it := range items {
		if it.isComment || !strings.Contains(it.text, "\n") {
			continue
		}

		if isClosure(entryValue(n.NamedChild(uint(i)))) {
			return true
		}
	}

	return false
}

func (f *Formatter) exprArray(n *sitter.Node) string {
	return f.exprCollection(n, "[", "]", "[", "]")
}

func (f *Formatter) exprObject(n *sitter.Node) string {
	return f.exprCollection(n, "{", "}", "{ ", " }")
}

// exprCollection renders an array or struct literal: on one line when it fits,
// else one item per line. open and shut bracket the broken form; inlineOpen and
// inlineClose bracket the one-line form.
func (f *Formatter) exprCollection(n *sitter.Node, open, shut, inlineOpen, inlineClose string) string {
	if n.NamedChildCount() == 0 {
		return open + shut
	}

	items, hasLineComment := f.collectionItems(n)
	trailing := hasTrailingComma(n)

	if !hasLineComment && !holdsMultilineClosure(n, items) {
		inline := inlineOpen + joinCollectionInline(items, trailing) + inlineClose
		if f.lineLen+len(inline) <= f.opts.LineWidth {
			return inline
		}
	}

	f.deepenItems(n, items)
	indent := f.indented() + f.opts.indent(1)

	return open + "\n" + indent + joinCollectionLines(items, indent, trailing) + "\n" + f.indented() + shut
}

// exprPair renders one struct entry as key, separator, value. When the entry
// is too long for the line and its value can break, the value moves onto its
// own line, indented one level under the key.
func (f *Formatter) exprPair(keyStr, sep string, val *sitter.Node) string {
	valStr := f.expr(val)
	result := keyStr + sep + " " + valStr

	if len(result) > f.opts.LineWidth && !strings.Contains(valStr, "\n") {
		f.level++
		valStr = f.expr(val)
		f.level--

		if strings.Contains(valStr, "\n") {
			indent := f.opts.indent(f.level + 1)

			return keyStr + sep + "\n" + indent + valStr
		}
	}

	return result
}

// structKey renders the key of an `=` struct entry. A bare name is a key, not
// a variable reference, so it is written as-is rather than passed through
// scope casing: `{ url = x }` names a key called url, and changing its case
// changes the key wherever the struct's key case is preserved (serializeJSON).
func (f *Formatter) structKey(key *sitter.Node) string {
	if key != nil && key.Kind() == "property_identifier" {
		return f.text(key)
	}

	return f.expr(key)
}

// isStructPattern reports whether an object_pattern is an ordinary `=` struct
// literal that exprObject can lay out: every entry a cf_pair or a bare-name
// object_assignment_pattern, and every comma between two entries or after the
// last one.
//
// The grammar lists entries as commaSep(optional(...)), so `{ a = 1,, b = 2 }`
// and `{ , a = 1 }` parse with an empty slot. Re-joining the entries with ", "
// would drop it — a real change, which the guard would refuse for the whole
// file. Destructuring forms (rest, shorthand, nested patterns) are left alone
// for the same reason: they are not struct literals, and exprObject does not
// know how to write them.
func isStructPattern(n *sitter.Node) bool {
	afterSeparator := true

	for i := range n.ChildCount() {
		c := n.Child(i)
		kind := c.Kind()

		switch {
		case kind == "{" || kind == "}" || isCommentKind(kind):
			continue
		case kind == ",":
			if afterSeparator {
				return false
			}

			afterSeparator = true

			continue
		case kind == "cf_pair":
		case kind == "object_assignment_pattern":
			if left := c.ChildByFieldName("left"); left == nil || left.Kind() != "shorthand_property_identifier_pattern" {
				return false
			}
		default:
			return false
		}

		afterSeparator = false
	}

	return true
}
