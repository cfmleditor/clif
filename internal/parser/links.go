// Document links: the file paths a document names, read by its syntax.

package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// Links were read by scanning each line for `template=`, `include `, `href=`
// and `action=` followed by a quote, wherever they stood. Judged against
// IncludeSites, the include walker, over the corpus, that linked a
// commented-out <cfinclude> (and the code map made it an include edge) and a
// comment's prose (`// This can include "expected" errors`). They are read
// here by syntax instead: in markup, from the attributes of the tags outside
// CFML comments (attrScanner, CF or HTML by the tag); in CFScript, from the
// include statements and template arguments the scanner reads, which skips
// comments. The parse's walk records them (file_refs.go).

// fileRefKind is what a path a file names is to it.
type fileRefKind uint8

const (
	// refInclude is a cfinclude's template or an include statement's path:
	// the file runs in this one's variables scope.
	refInclude fileRefKind = iota + 1
	// refTemplate is the template of another CF tag or call, cfmodule's.
	refTemplate
	// refHref is a markup tag's href.
	refHref
	// refAction is a form's action.
	refAction
	// refListing is a <cfdirectory> tag, whole: a directory listing an
	// include of each name it finds may follow.
	refListing
)

// attrRef is the kind of path attribute attr of tag name, a CF tag when cf,
// names, or 0 for none: template on a CF tag (an include on cfinclude), href
// on a markup tag, action on a form. A CF tag's action is an operation
// (cffile action="move"), not a path, and was linked on every one.
func attrRef(name, attr string, cf bool) fileRefKind {
	switch {
	case strings.EqualFold(attr, "template"):
		if !cf {
			return 0
		}

		if strings.EqualFold(name, "cfinclude") {
			return refInclude
		}

		return refTemplate
	case strings.EqualFold(attr, "href"):
		if cf {
			return 0
		}

		return refHref
	case strings.EqualFold(attr, "action"):
		if strings.EqualFold(name, "form") || strings.EqualFold(name, "cfform") {
			return refAction
		}
	}

	return 0
}

// expressionTag reports whether a CF tag's text after its name is an
// expression rather than attributes: <cfset href = x.getURL()> sets a
// variable, and links nothing.
func expressionTag(name string) bool {
	switch strings.ToLower(name) {
	case "cfset", "cfif", "cfelseif", "cfreturn":
		return true
	default:
		return false
	}
}

// tagNameAt reads the '<' at i, which does not open a closing tag: the
// bounds of the name of a tag opening there, or, when none does (a comment, a
// stray '<'), an empty name and where the walk goes on. A comment is stepped
// over whole.
func tagNameAt(text string, i int) (j, k, next int) {
	if strings.HasPrefix(text[i:], "<!---") {
		return 0, 0, SkipCFMLComment(text, i)
	}

	j = i + 1

	k = j
	for k < len(text) && isTagNameByte(text[k]) {
		k++
	}

	return j, k, i + 1
}

// tagStartOutsideQuotesEnd is the offset of the first '>' in s outside quoted
// values, or -1.
func tagStartOutsideQuotesEnd(s string) int {
	var quote byte

	for i := range len(s) {
		switch c := s[i]; {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case lex.IsQuote(c):
			quote = c
		case c == '>':
			return i
		}
	}

	return -1
}

// fileRegions is regions, split from content, as the path walk reads them. A
// file with no CF tag is one script region to ClassifyRegions, which is the
// parse's rule; one that begins with markup is read as the markup it is, as
// StructureSpans reads it.
func fileRegions(content string, regions []Region) []Region {
	if len(regions) == 1 && regions[0].Kind == RegionScript && len(regions[0].Text) == len(content) && startsWithMarkup(content) {
		return []Region{{Kind: RegionTag, Text: content}}
	}

	return regions
}

// isLink reports whether a path of kind names a file a document links: a
// literal path, not computed (#...#) and not a URL, and not a listing.
func isLink(kind fileRefKind, path string) bool {
	return kind != refListing && strings.TrimSpace(path) != "" && !strings.Contains(path, "#") && !strings.Contains(path, "://")
}

// ExtractLinks is the file path references in content (cfinclude, cfmodule,
// href, action), each region read by its syntax, with lines and columns of
// the whole content: a reading of a parse that records the file's references
// and nothing else (FileRefsOnly). A caller holding the file's parse asks it
// instead (ParseResult.DocumentLinks).
func ExtractLinks(content string) []DocumentLink {
	return ParseWithOptions("", content, &ParseOptions{FileRefsOnly: true, ExtractLinks: true}).DocumentLinks()
}
