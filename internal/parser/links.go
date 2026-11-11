// Document links: the file paths a document names, read by its syntax.

package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
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
// comments.

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

// fileRefFunc receives each path the walk finds: its kind, the offset its
// statement or tag starts at, and the path's bounds, quotes excluded. A
// listing's bounds are its tag's.
type fileRefFunc func(kind fileRefKind, at, start, end int)

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

// tagFileRefs calls emit for each path in text, markup read as a tag region
// is. A markup tag a CF tag interrupts (<a <cfif x>id="y"</cfif> href="z">)
// ends where the CF tag begins (tagEnd), and its attributes after the CF tags
// are read from the text between them, up to its own '>'.
//
// includesOnly reads only what an include needs, <cfinclude> and
// <cfdirectory>: the index asks that of every file, and most tags in one name
// no path. It skips work, not a rule.
func tagFileRefs(text string, includesOnly bool, emit fileRefFunc) {
	ends := newTagEnds(text)

	// outer is the markup tag interrupted by a CF tag, still open, and
	// outerAt where it began; "" for none.
	outer, outerAt := "", 0

	attrs := func(s string, base, at int, name string, cf bool) {
		mode := htmlAttrs
		if cf {
			mode = cfAttrs
		}

		sc := newAttrScanner(s, mode)

		for {
			a, more := sc.next()
			if !more {
				return
			}

			// A path is quoted; an unquoted value is an expression or a word.
			if a.valStart == 0 || !lex.IsQuote(s[a.valStart-1]) {
				continue
			}

			if kind := attrRef(name, s[a.nameStart:a.nameEnd], cf); kind != 0 {
				emit(kind, at, base+a.valStart, base+a.valEnd)
			}
		}
	}

	for pos := 0; pos < len(text); {
		i := strings.IndexByte(text[pos:], '<')
		if i < 0 {
			i = len(text) - pos
		}

		i += pos

		// The text before the next '<' continues an interrupted tag's
		// attributes, up to its '>'.
		if outer != "" && i > pos {
			seg := text[pos:i]
			if gt := tagStartOutsideQuotesEnd(seg); gt >= 0 {
				attrs(seg[:gt], pos, outerAt, outer, false)
				outer = ""
			} else {
				attrs(seg, pos, outerAt, outer, false)
			}
		}

		if i >= len(text) {
			return
		}

		j, k, next := tagNameAt(text, i)
		if k == j {
			pos = next

			continue
		}

		end := tagEnd(&ends, k)
		name := text[j:k]

		// A CF tag's values are read as CF reads them, a custom tag's included
		// (<cf_x>, <p:x>); any other tag's as markup's.
		cf := HasPrefixFold(name, "cf") || strings.IndexByte(name, ':') > 0

		switch {
		case strings.EqualFold(name, "cfdirectory"):
			emit(refListing, i, i, end)
		case includesOnly:
			if strings.EqualFold(name, "cfinclude") {
				attrs(text[i:end], i, i, name, true)
			}
		default:
			if !cf || !expressionTag(name) {
				attrs(text[i:end], i, i, name, cf)
			}

			if !cf && (end >= len(text) || text[end-1] != '>') {
				outer, outerAt = name, i
			}
		}

		pos = end
	}
}

// tagNameAt reads the '<' at i: the bounds of the name of a tag opening
// there, or, when none does (a comment, a closing tag, a stray '<'), an empty
// name and where the walk goes on. A comment and a closing tag are stepped
// over whole: a closing tag's '>' is not an interrupted tag's.
func tagNameAt(text string, i int) (j, k, next int) {
	if strings.HasPrefix(text[i:], "<!---") {
		return 0, 0, SkipCFMLComment(text, i)
	}

	j = i + 1
	if j < len(text) && text[j] == '/' {
		if gt := strings.IndexByte(text[j:], '>'); gt >= 0 {
			return 0, 0, j + gt + 1
		}

		return 0, 0, len(text)
	}

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

// scriptFileRefs calls emit for each path in text, CFScript: the path of an
// include statement (include "x", Lucee's include template="x", the call
// cfinclude(template="x")), the template of a cfmodule( template = "x" ) or a
// module statement as the custom tag scanner reads them, and a CF tag written
// among the statements, read as the tag it is. A template = elsewhere, a
// struct's key, names nothing. includesOnly is tagFileRefs'.
func scriptFileRefs(text string, includesOnly bool, emit fileRefFunc) {
	if !includesOnly {
		calls := scriptCustomTags(text, 0, nil)
		for i := range calls {
			c := &calls[i]
			if c.Kind == CustomTagModuleTemplate && !c.Dynamic {
				emit(refTemplate, c.Start, c.Start, c.End)
			}
		}
	}

	if IndexFold(text, "include") < 0 && IndexFold(text, "<cf") < 0 {
		return
	}

	sc := NewScriptScanner(text)
	prev := Token{Kind: TokSemicolon}

	for {
		t := sc.NextSkipComments()
		if t.Kind == TokEOF {
			return
		}

		before := prev
		prev = t

		if t.Kind == TokLT {
			if next := sc.PeekSkipComments(); next.Kind == TokIdent && HasPrefixFold(next.Value, "cf") {
				if gt := tagStartOutsideQuotesEnd(text[t.Offset:]); gt >= 0 {
					base := t.Offset
					tagFileRefs(text[base:base+gt+1], includesOnly, func(kind fileRefKind, at, start, end int) {
						emit(kind, base+at, base+start, base+end)
					})
				}
			}

			continue
		}

		// After a '<' the keyword is the tag's name, read above; after a '.'
		// it is a member call, arr.include(x).
		if t.Kind != TokIdent || before.Kind == TokDot || before.Kind == TokLT || !identEq(t.Value, "include") && !identEq(t.Value, "cfinclude") {
			continue
		}

		next := sc.PeekSkipComments()
		if next.Kind == TokLParen {
			sc.NextSkipComments()

			next = sc.PeekSkipComments()
		}

		if next.Kind == TokIdent && identEq(next.Value, "template") {
			sc.NextSkipComments()

			if op := sc.PeekSkipComments(); op.Kind == TokEquals || op.Kind == TokColon {
				sc.NextSkipComments()

				next = sc.PeekSkipComments()
			}
		}

		if next.Kind == TokString && len(next.Value) >= 2 {
			sc.NextSkipComments()

			prev = next
			emit(refInclude, t.Offset, next.Offset+1, next.Offset+len(next.Value)-1)
		}
	}
}

// regionFileRefs calls emit for each path in r, a region of content, at
// content's offsets.
func regionFileRefs(r *Region, includesOnly bool, emit fileRefFunc) {
	shifted := func(kind fileRefKind, at, start, end int) {
		emit(kind, r.Offset+at, r.Offset+start, r.Offset+end)
	}

	switch r.Kind {
	case RegionScript:
		scriptFileRefs(r.Text, includesOnly, shifted)
	case RegionTag:
		tagFileRefs(r.Text, includesOnly, shifted)
	case RegionSkip:
	}
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

// regionLinks is the links in text, a region (script or not), with lines
// counted from baseLine and columns from the start of each line of text.
func regionLinks(text string, script bool, baseLine int) []DocumentLink {
	var (
		links []DocumentLink
		idx   LineTable
	)

	emit := func(kind fileRefKind, _, start, end int) {
		if !isLink(kind, text[start:end]) {
			return
		}

		if idx == nil {
			idx = NewLineTable(text)
		}

		line := idx.LineOf(start)
		lineStart := int(idx[line])

		links = append(links, DocumentLink{
			Path:  text[start:end],
			Line:  conv.Uint32(baseLine + line),
			Start: conv.Uint32(start - lineStart),
			End:   conv.Uint32(end - lineStart),
		})
	}

	if script {
		scriptFileRefs(text, false, emit)
	} else {
		tagFileRefs(text, false, emit)
	}

	return links
}

// ExtractLinks is the file path references in content (cfinclude, cfmodule,
// href, action), each region read by its syntax, with lines and columns of
// the whole content.
func ExtractLinks(content string) []DocumentLink {
	regions, idx := ClassifyRegionsIdx(content)

	return linksInRegions(content, regions, idx)
}

// linksInRegions is ExtractLinks over regions already split from content,
// with their line index (nil to build one).
func linksInRegions(content string, regions []Region, idx LineTable) []DocumentLink {
	if idx == nil {
		idx = NewLineTable(content)
	}

	var links []DocumentLink

	emit := func(kind fileRefKind, _, start, end int) {
		if !isLink(kind, content[start:end]) {
			return
		}

		line := idx.LineOf(start)
		lineStart := int(idx[line])

		links = append(links, DocumentLink{
			Path:  content[start:end],
			Line:  conv.Uint32(line),
			Start: conv.Uint32(start - lineStart),
			End:   conv.Uint32(end - lineStart),
		})
	}

	regions = fileRegions(content, regions)
	for i := range regions {
		regionFileRefs(&regions[i], false, emit)
	}

	return links
}
