// Document links: the file paths a document names, read by its syntax.

package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
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

// linkAttr reports whether attribute attr of tag name, a CF tag when cf,
// names a file to link: template on a CF tag (cfinclude, cfmodule), href on a
// markup tag, action on a form. A CF tag's action is an operation (cffile
// action="move"), not a path, and was linked on every one.
func linkAttr(name, attr string, cf bool) bool {
	switch {
	case strings.EqualFold(attr, "template"):
		return cf
	case strings.EqualFold(attr, "href"):
		return !cf
	case strings.EqualFold(attr, "action"):
		return strings.EqualFold(name, "form") || strings.EqualFold(name, "cfform")
	default:
		return false
	}
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

// linkable reports whether path is one a link is made for: a literal file
// path, not computed (#...#) and not a URL.
func linkable(path string) bool {
	return strings.TrimSpace(path) != "" && !strings.Contains(path, "#") && !strings.Contains(path, "://")
}

// tagLinkSpans calls emit with the span of each link in text, markup read as
// a tag region is. A markup tag a CF tag interrupts (<a <cfif x>id="y"</cfif>
// href="z">) ends where the CF tag begins (tagEnd), and its attributes after
// the CF tags are read from the text between them, up to its own '>'.
func tagLinkSpans(text string, emit func(start, end int)) {
	ends := newTagEnds(text)

	// outer is the markup tag interrupted by a CF tag, still open; "" for none.
	outer := ""

	attrs := func(s string, base int, name string, cf bool) {
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
			quoted := a.valStart > 0 && isQuoteByte(s[a.valStart-1])
			if quoted && linkAttr(name, s[a.nameStart:a.nameEnd], cf) && linkable(s[a.valStart:a.valEnd]) {
				emit(base+a.valStart, base+a.valEnd)
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
				attrs(seg[:gt], pos, outer, false)
				outer = ""
			} else {
				attrs(seg, pos, outer, false)
			}
		}

		if i >= len(text) {
			return
		}

		if strings.HasPrefix(text[i:], "<!---") {
			pos = skipTagComment(text, i)

			continue
		}

		j := i + 1
		if j < len(text) && text[j] == '/' {
			// A closing tag, stepped over whole: its '>' is not an
			// interrupted tag's.
			if gt := strings.IndexByte(text[j:], '>'); gt >= 0 {
				pos = j + gt + 1
			} else {
				pos = len(text)
			}

			continue
		}

		k := j
		for k < len(text) && isTagNameByte(text[k]) {
			k++
		}

		if k == j {
			pos = i + 1

			continue
		}

		end := tagEnd(&ends, k)
		tag := text[i:end]

		// A CF tag's values are read as CF reads them, a custom tag's included
		// (<cf_x>, <p:x>); any other tag's as markup's.
		name := text[j:k]
		cf := HasPrefixFold(name, "cf") || strings.IndexByte(name, ':') > 0

		if !cf || !expressionTag(name) {
			attrs(tag, i, name, cf)
		}

		if !cf && (end >= len(text) || text[end-1] != '>') {
			outer = name
		}

		pos = end
	}
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
		case isQuoteByte(c):
			quote = c
		case c == '>':
			return i
		}
	}

	return -1
}

// scriptLinkSpans calls emit with the span of each link in text, CFScript:
// the path of an include statement (include "x", Lucee's include
// template="x"), and the template of a cfmodule( template = "x" ) or a module
// statement, as the custom tag scanner reads them. A template = elsewhere, a
// struct's key, is no link.
func scriptLinkSpans(text string, emit func(start, end int)) {
	calls := scriptCustomTags(text, 0, nil)
	for i := range calls {
		c := &calls[i]
		if c.Kind == CustomTagModuleTemplate && !c.Dynamic && linkable(c.Name) {
			emit(c.Start, c.End)
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

		// A CF tag written among the statements (<cfinclude template="x">) is
		// read as the tag it is.
		if t.Kind == TokLT {
			if next := sc.PeekSkipComments(); next.Kind == TokIdent && HasPrefixFold(next.Value, "cf") {
				if gt := tagStartOutsideQuotesEnd(text[t.Offset:]); gt >= 0 {
					base := t.Offset
					tagLinkSpans(text[base:base+gt+1], func(start, end int) { emit(base+start, base+end) })
				}
			}

			continue
		}

		// After a '<' the keyword is the tag's name, read above.
		if t.Kind != TokIdent || before.Kind == TokDot || before.Kind == TokLT || !identEq(t.Value, "include") && !identEq(t.Value, "cfinclude") {
			continue
		}

		// include "x", include template="x", and the call form
		// cfinclude(template="x"), as includeScriptAt reads them.
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

		if next.Kind == TokString {
			sc.NextSkipComments()

			prev = next
			emitString(next, emit)
		}
	}
}

// emitString emits the inside of a string token, when it is a linkable path.
func emitString(t Token, emit func(start, end int)) {
	if len(t.Value) < 2 {
		return
	}

	if inner := t.Value[1 : len(t.Value)-1]; linkable(inner) {
		emit(t.Offset+1, t.Offset+len(t.Value)-1)
	}
}

// regionLinks is the links in text, a region (script or not), with lines
// counted from baseLine and columns from the start of each line of text.
func regionLinks(text string, script bool, baseLine int) []DocumentLink {
	var (
		links []DocumentLink
		idx   []int32
	)

	emit := func(start, end int) {
		if idx == nil {
			idx = buildLineIdx(text)
		}

		line := lineAtOffset(idx, start)
		lineStart := int(idx[line])

		links = append(links, DocumentLink{
			Path:  text[start:end],
			Line:  conv.Uint32(baseLine + line),
			Start: conv.Uint32(start - lineStart),
			End:   conv.Uint32(end - lineStart),
		})
	}

	if script {
		scriptLinkSpans(text, emit)
	} else {
		tagLinkSpans(text, emit)
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
// with their line index (nil to build one). A file with no CF tag is one
// script region to ClassifyRegions, which is the parse's rule; one that
// begins with markup is read as the markup it is, as StructureSpans reads it.
func linksInRegions(content string, regions []Region, idx []int32) []DocumentLink {
	if len(regions) == 1 && regions[0].Kind == RegionScript && len(regions[0].Text) == len(content) && startsWithMarkup(content) {
		regions = []Region{{Kind: RegionTag, Text: content}}
	}

	if idx == nil {
		idx = buildLineIdx(content)
	}

	var links []DocumentLink

	for _, r := range regions {
		if r.Kind == RegionSkip {
			continue
		}

		emit := func(start, end int) {
			start, end = r.Offset+start, r.Offset+end
			line := lineAtOffset(idx, start)
			lineStart := int(idx[line])

			links = append(links, DocumentLink{
				Path:  content[start:end],
				Line:  conv.Uint32(line),
				Start: conv.Uint32(start - lineStart),
				End:   conv.Uint32(end - lineStart),
			})
		}

		if r.Kind == RegionScript {
			scriptLinkSpans(r.Text, emit)
		} else {
			tagLinkSpans(r.Text, emit)
		}
	}

	return links
}
