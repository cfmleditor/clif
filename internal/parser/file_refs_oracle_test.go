package parser

// The file-reference readers as they were before the parse's walk recorded
// file references (file_refs.go): their own walks over the regions, kept as
// the oracle TestFileRefsAreTheOldReaders' compares the record's readings
// with.

import (
	"cmp"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
)

// includesOnly reads only what an include needs, <cfinclude> and
// <cfdirectory>: the index asks that of every file, and most tags in one name
// no path. It skips work, not a rule.
func oldTagFileRefs(text string, includesOnly bool, emit fileRefFunc) {
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

		j, k, next := oldTagNameAt(text, i)
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

// scriptFileRefs calls emit for each path in text, CFScript: the path of an
// include statement (include "x", Lucee's include template="x", the call
// cfinclude(template="x")), the template of a cfmodule( template = "x" ) or a
// module statement as the custom tag scanner reads them, and a CF tag written
// among the statements, read as the tag it is. A template = elsewhere, a
// struct's key, names nothing. includesOnly is tagFileRefs'.
func oldScriptFileRefs(text string, includesOnly bool, emit fileRefFunc) {
	if !includesOnly {
		calls := oldScriptCustomTags(text, 0, nil)
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

	// For includes alone, a region with neither an include nor a listing tag
	// holds nothing to read: the index asks this of every file.
	if includesOnly && IndexFold(text, "include") < 0 && IndexFold(text, "<cfdirectory") < 0 {
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
					oldTagFileRefs(text[base:base+gt+1], includesOnly, func(kind fileRefKind, at, start, end int) {
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
func oldRegionFileRefs(r *Region, includesOnly bool, emit fileRefFunc) {
	shifted := func(kind fileRefKind, at, start, end int) {
		emit(kind, r.Offset+at, r.Offset+start, r.Offset+end)
	}

	switch r.Kind {
	case RegionScript:
		oldScriptFileRefs(r.Text, includesOnly, shifted)
	case RegionTag:
		oldTagFileRefs(r.Text, includesOnly, shifted)
	case RegionSkip:
	}
}

// ExtractLinks is the file path references in content (cfinclude, cfmodule,
// href, action), each region read by its syntax, with lines and columns of
// the whole content.
func oldExtractLinks(content string) []DocumentLink {
	regions, idx := ClassifyRegionsIdx(content)

	return oldLinksInRegions(content, regions, idx)
}

// linksInRegions is ExtractLinks over regions already split from content,
// with their line index (nil to build one).
func oldLinksInRegions(content string, regions []Region, idx LineTable) []DocumentLink {
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
		oldRegionFileRefs(&regions[i], false, emit)
	}

	return links
}

// IncludeSites is every include ExtractIncludes reads, one per statement and
// with where it is, so a template can be read as its includer sees it at
// that line: a template included inside a function sees the function's locals.
// The script and tag forms come first, in source order, then the directory
// listings' globs.
func oldIncludeSites(content string) []IncludeSite {
	if IndexFold(content, "include") < 0 {
		return nil
	}

	regions, _ := ClassifyRegionsIdx(content)

	return oldIncludeSitesIn(content, regions)
}

// includeSitesIn is IncludeSites over regions already split from content.
func oldIncludeSitesIn(content string, regions []Region) []IncludeSite {
	if IndexFold(content, "include") < 0 {
		return nil
	}

	var (
		out      []IncludeSite
		listed   map[string]string // query name → directory, lowercased name
		fromList []IncludeSite     // includes of a listing's names: Path is the template as written
		wrappers []IncludeSite     // includes of an argument: Path is the argument's name
	)

	emit := func(kind fileRefKind, at, start, end int) {
		switch kind {
		case refListing:
			if dir, name, ok := qualifyingListing(content[start:end]); ok {
				listed = withListing(listed, name, dir)
			}
		case refInclude:
			raw := content[start:end]
			if _, _, ok := listingIncludePath(raw); ok {
				fromList = append(fromList, IncludeSite{Path: raw, Offset: at})
			}

			if site, ok := includeSite(content, raw, at); ok {
				out = append(out, site)
			} else if param, ok := argumentInclude(raw); ok {
				wrappers = append(wrappers, IncludeSite{Path: param, Offset: at})
			}
		case refTemplate, refHref, refAction:
		}
	}

	regions = fileRegions(content, regions)
	for i := range regions {
		oldRegionFileRefs(&regions[i], true, emit)
	}

	if len(wrappers) > 0 {
		out = append(out, oldWrapperIncludes(content, regions, wrappers)...)
	}

	if len(listed) > 0 {
		out = append(out, listingIncludes(fromList, listed)...)
	}

	return out
}

// IncludeWrappers is the functions content declares that include the
// template an argument names, `include "#arguments.template#"` or through one
// call (`#LCase(arguments.template)#`), by lowercased name. Wheels' Global.cfc
// declares $include so; a call of one is an include into the variables scope
// of the component whose method runs it. Finding them takes a parse, which
// only a file with such an include pays for.
func oldIncludeWrappers(content string) map[string]IncludeWrapper {
	if IndexFold(content, "arguments.") < 0 || IndexFold(content, "include") < 0 {
		return nil
	}

	var argIncludes []IncludeSite

	emit := func(kind fileRefKind, at, start, end int) {
		if kind != refInclude {
			return
		}

		if param, ok := argumentInclude(content[start:end]); ok {
			argIncludes = append(argIncludes, IncludeSite{Path: param, Offset: at})
		}
	}

	regions, _ := ClassifyRegionsIdx(content)

	regions = fileRegions(content, regions)
	for i := range regions {
		oldRegionFileRefs(&regions[i], true, emit)
	}

	return oldIncludeWrappersIn(content, argIncludes)
}

// includeWrappersIn is IncludeWrappers from argIncludes, each include of an
// argument in content with the argument's name as its Path.
func oldIncludeWrappersIn(content string, argIncludes []IncludeSite) map[string]IncludeWrapper {
	if len(argIncludes) == 0 {
		return nil
	}

	// One shallow parse says both which function holds each include and what
	// that function's arguments are.
	parsed := ParseWithOptions("", content, &ParseOptions{Shallow: true})
	scopes, defs := parsed.Scopes, parsed.Funcs
	lines := NewLineTable(content)

	var wrappers map[string]IncludeWrapper

	for _, inc := range argIncludes {
		line := lines.LineOf(inc.Offset)

		var fn string

		for _, sc := range scopes {
			if sc.Start <= line && line <= sc.End {
				fn = sc.Name // the innermost, which comes last
			}
		}

		if fn == "" {
			continue
		}

		for i := range defs {
			if !strings.EqualFold(defs[i].Name, fn) {
				continue
			}

			for k, a := range defs[i].Arguments {
				if strings.EqualFold(a.Name, inc.Path) {
					if wrappers == nil {
						wrappers = map[string]IncludeWrapper{}
					}

					wrappers[strings.ToLower(fn)] = IncludeWrapper{Param: inc.Path, Position: k}
				}
			}
		}
	}

	return wrappers
}

// argIncludes holds each include of an argument, the argument's name as its
// Path.
func oldWrapperIncludes(content string, regions []Region, argIncludes []IncludeSite) []IncludeSite {
	params := oldIncludeWrappersIn(content, argIncludes)
	if len(params) == 0 {
		return nil
	}

	var out []IncludeSite

	for i := range regions {
		r := &regions[i]
		if r.Kind != RegionScript {
			continue
		}

		sc := NewScriptScanner(r.Text)
		prev := Token{Kind: TokSemicolon}

		for {
			t := sc.NextSkipComments()
			if t.Kind == TokEOF {
				break
			}

			before := prev
			prev = t

			p, ok := params[strings.ToLower(t.Value)]
			if t.Kind != TokIdent || !ok || before.Kind == TokDot || sc.PeekSkipComments().Kind != TokLParen {
				continue
			}

			sc.NextSkipComments()

			if tmpl, ok := wrapperArgument(sc, p.Param, p.Position); ok {
				if site, ok := includeSite(content, tmpl, r.Offset+t.Offset); ok {
					out = append(out, site)
				}
			}
		}
	}

	return out
}

// CustomTagCalls reads every custom tag call in content, in order. prefixes
// are the configured tag library prefixes (config.CustomTagPrefixes), "cf"
// included and lowercased; a tag starting with none of them is not one. A
// caller holding the file's parse asks it instead
// (ParseResult.CustomTagCalls), which keeps its regions and the answer.
func oldCustomTagCalls(content string, prefixes []string) []CustomTagCall {
	regions, idx := ClassifyRegionsIdx(content)

	return oldCustomTagCallsIn(regions, idx, content, prefixes)
}

func oldCustomTagCallsIn(regions []Region, idx LineTable, content string, prefixes []string) []CustomTagCall {
	// The longest prefix decides: "cfa" before "cf".
	prefixes = slices.SortedFunc(slices.Values(prefixes), func(a, b string) int { return cmp.Compare(len(b), len(a)) })

	var (
		out     []CustomTagCall
		imports map[string]string
	)

	for _, r := range regions {
		switch r.Kind {
		case RegionScript:
			// Every script call spells cf_ or module (cfmodule, Lucee's
			// statement), and most components spell neither.
			if IndexFold(r.Text, "cf_") >= 0 || IndexFold(r.Text, "module") >= 0 {
				out = oldScriptCustomTags(r.Text, r.Offset, out)
			}
		case RegionTag:
			out, imports = oldTagCustomTags(r.Text, r.Offset, imports, prefixes, out)
		default:
		}
	}

	if len(out) > 0 && idx == nil {
		idx = NewLineTable(content)
	}

	for i := range out {
		out[i].Line = idx.LineOf(out[i].Start)
	}

	return out
}

// tagCustomTags appends the custom tags text, a tag region at base, calls,
// and returns them with imports, the cfimport taglibs by lowercased prefix,
// grown by those the region declares. A prefix applies from its cfimport on,
// as the engines compile it, so it is read in the same walk.
func oldTagCustomTags(text string, base int, imports map[string]string, prefixes []string, out []CustomTagCall) ([]CustomTagCall, map[string]string) {
	ends := newTagEnds(text)

	for pos := 0; pos < len(text); {
		i := strings.IndexByte(text[pos:], '<')
		if i < 0 {
			break
		}

		i += pos

		if strings.HasPrefix(text[i:], "<!---") {
			pos = SkipCFMLComment(text, i)

			continue
		}

		j, closing := i+1, false
		if j < len(text) && text[j] == '/' {
			j, closing = j+1, true
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
		name := text[j:k]

		switch {
		case len(name) > 3 && lex.HasPrefixFold(name, "cf_"):
			out = append(out, CustomTagCall{Kind: CustomTagUnderscore, Name: name[3:], Start: base + j + 3, End: base + k, AttrStart: base + k, AttrEnd: base + end, Closing: closing})
		case strings.EqualFold(name, "cfmodule"):
			if !closing {
				out = oldModuleAttrs(text[k:end], base+k, out)
			}
		case strings.EqualFold(name, "cfimport"):
			if !closing {
				imports = cfimportTaglib(text[k:end], imports)
			}
		case strings.IndexByte(name, ':') > 0:
			prefix, local, _ := strings.Cut(name, ":")
			if taglib, ok := imports[strings.ToLower(prefix)]; ok && local != "" {
				out = append(out, CustomTagCall{Kind: CustomTagImported, Name: local, Prefix: strings.ToLower(prefix), Taglib: taglib, Start: base + j + len(prefix) + 1, End: base + k, Closing: closing})
			}
		default:
			for _, p := range prefixes {
				if len(name) > len(p) && lex.HasPrefixFold(name, p) {
					out = append(out, CustomTagCall{Kind: CustomTagPrefixed, Name: name[len(p):], Prefix: p, Start: base + j + len(p), End: base + k, Closing: closing})

					break
				}
			}
		}

		pos = end
	}

	return out, imports
}

// scriptCustomTags appends the custom tags text, a script region at base,
// calls: cf_name(...), cfmodule(...) and Lucee's module statement.
func oldScriptCustomTags(text string, base int, out []CustomTagCall) []CustomTagCall {
	sc := NewScriptScanner(text)
	prev := Token{Kind: TokSemicolon}

	for {
		t := sc.NextSkipComments()
		if t.Kind == TokEOF {
			return out
		}

		before := prev
		prev = t

		if t.Kind != TokIdent || before.Kind == TokDot {
			continue
		}

		lower := strings.ToLower(t.Value)

		switch {
		case strings.HasPrefix(lower, "cf_") && len(lower) > 3 && sc.PeekSkipComments().Kind == TokLParen:
			// `function cf_name()` declares a function, and calls nothing.
			if before.Kind == TokIdent && strings.EqualFold(before.Value, "function") {
				continue
			}

			out = append(out, CustomTagCall{Kind: CustomTagUnderscore, Name: t.Value[3:], Start: base + t.Offset + 3, End: base + t.Offset + len(t.Value), Script: true})
		case lower == "cfmodule" && sc.PeekSkipComments().Kind == TokLParen:
			out = oldScriptModuleAttrs(sc, base, TokRParen, out)
			prev = Token{Kind: TokRParen}
		case lower == "module" && statementStart(before) && sc.PeekSkipComments().Kind == TokIdent:
			out = oldScriptModuleAttrs(sc, base, TokSemicolon, out)
			prev = Token{Kind: TokSemicolon}
		}
	}
}

// moduleAttrs appends the call a cfmodule's attributes, attrs at base, make:
// by template, else by name.
func oldModuleAttrs(attrs string, base int, out []CustomTagCall) []CustomTagCall {
	kind := CustomTagModuleTemplate

	from, to, ok := attrSpan(attrs, "template")
	if !ok {
		kind = CustomTagModuleName

		if from, to, ok = attrSpan(attrs, "name"); !ok {
			return out
		}
	}

	value := attrs[from:to]

	return append(out, CustomTagCall{Kind: kind, Name: value, Start: base + from, End: base + to, AttrStart: base, AttrEnd: base + len(attrs), Dynamic: strings.Contains(value, "#")})
}

// scriptModuleAttrs reads the name = "value" pairs of a script cfmodule, up
// to stop at depth zero (or a { for the statement form's body), and appends
// the call they make. A key inside a nested struct or call is not one of the
// tag's; a value that is more than its string literal ("a" & b) is computed,
// and the call Dynamic.
func oldScriptModuleAttrs(sc *Scanner, base int, stop TokenKind, out []CustomTagCall) []CustomTagCall {
	var (
		template, name *CustomTagCall
		depth          int
		key            string
	)

	for {
		t := sc.NextSkipComments()
		if t.Kind == TokEOF || t.Kind == TokLBrace && depth == 0 {
			break
		}

		switch t.Kind {
		case TokLParen, TokLBracket, TokLBrace:
			depth++
		case TokRParen, TokRBracket, TokRBrace:
			depth--
		default:
		}

		if depth <= 0 && (t.Kind == stop || t.Kind == TokRParen || t.Kind == TokRBrace) {
			break
		}

		switch {
		case t.Kind == TokIdent && depth <= 1:
			key = strings.ToLower(t.Value)
		case t.Kind == TokString && depth <= 1 && (key == "template" || key == "name") && len(t.Value) >= 2:
			value := t.Value[1 : len(t.Value)-1]
			dynamic := strings.Contains(value, "#") || !endsValue(sc.PeekSkipComments().Kind)
			c := &CustomTagCall{Kind: CustomTagModuleTemplate, Name: value, Start: base + t.Offset + 1, End: base + t.Offset + len(t.Value) - 1, Script: true, Dynamic: dynamic}

			if key == "template" {
				template = c
			} else {
				c.Kind = CustomTagModuleName
				name = c
			}

			key = ""
		case t.Kind != TokEquals && t.Kind != TokColon:
			key = ""
		}
	}

	if template != nil {
		return append(out, *template)
	}

	if name != nil {
		return append(out, *name)
	}

	return out
}

// fileRefFunc receives each path the walk finds: its kind, the offset its
// statement or tag starts at, and the path's bounds, quotes excluded. A
// listing's bounds are its tag's.
type fileRefFunc func(kind fileRefKind, at, start, end int)

// regionLinks is the links in text, a region (script or not), with lines
// counted from baseLine and columns from the start of each line of text.
func oldRegionLinks(text string, script bool, baseLine int) []DocumentLink {
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
		oldScriptFileRefs(text, false, emit)
	} else {
		oldTagFileRefs(text, false, emit)
	}

	return links
}

// tagNameAt reads the '<' at i: the bounds of the name of a tag opening
// there, or, when none does (a comment, a closing tag, a stray '<'), an empty
// name and where the walk goes on. A comment and a closing tag are stepped
// over whole: a closing tag's '>' is not an interrupted tag's.
func oldTagNameAt(text string, i int) (j, k, next int) {
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
