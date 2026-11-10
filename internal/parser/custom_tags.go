package parser

import (
	"cmp"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// The ways CFML calls a custom tag, all of which CustomTagCalls reads. Each
// was checked against Lucee's and RustCFML's test suites and ColdBox:
//
//	<cf_name attr="x">  </cf_name>       a template found by name: the caller's
//	                                     directory, then the custom tag paths
//	cf_name(attr="x");                   the same, from cfscript (Lucee LDEV-1276)
//	<cfmodule template="dir/name.cfm">   a template by path, as cfinclude finds one
//	<cfmodule name="a.b">                a/b.cfm under the custom tag paths
//	cfmodule(template="x.cfm");          either, from cfscript (ColdBox's Renderer)
//	module template="x.cfm";             either, Lucee's tag-in-script form,
//	module template="x.cfm" { … }        with or without a body
//	<cfimport taglib="lib" prefix="p">   then <p:name>: name.cfm in lib, found as
//	                                     a cfinclude path is
//	<cfabridge>                          a tag library named in customTagPaths
//	                                     with a prefix ("cfa" → bridge.cfm), as
//	                                     a Lucee library/tag directory is ("cf")
//
// A template or name holding #...# is computed: the call is returned with
// Dynamic set and names nothing a lookup can follow.

// CustomTagKind is the syntax a custom tag is called with.
type CustomTagKind uint8

// The kinds of custom tag call.
const (
	// CustomTagUnderscore is <cf_name>, </cf_name> or cf_name(...).
	CustomTagUnderscore CustomTagKind = iota + 1
	// CustomTagModuleName is <cfmodule name="a.b">: Name is "a.b".
	CustomTagModuleName
	// CustomTagModuleTemplate is <cfmodule template="x.cfm">: Name is the path.
	CustomTagModuleTemplate
	// CustomTagImported is <p:name> under <cfimport taglib="lib" prefix="p">.
	CustomTagImported
	// CustomTagPrefixed is <cf{rest}> for a configured tag library prefix.
	CustomTagPrefixed
)

// CustomTagCall is one call of a custom tag.
type CustomTagCall struct {
	// Name is the tag's name as written, without its prefix (cf_, the cfimport
	// prefix, or the configured one); for cfmodule it is the attribute value.
	Name string
	// Prefix is the cfimport or configured prefix, lowercased.
	Prefix string
	// Taglib is the cfimport taglib a CustomTagImported prefix names.
	Taglib string
	// Start and End are the byte span in the content of the text naming the
	// tag: the name after its prefix, or the attribute value.
	Start, End int
	// AttrStart and AttrEnd are the span of a tag's attribute text, from its
	// name to its closing >, for TagAttributes; zero for a script call.
	AttrStart, AttrEnd int
	// Line is the 0-based line Start is on.
	Line    int
	Kind    CustomTagKind
	Closing bool // an end tag, </cf_name> or </p:name>
	Script  bool // called from cfscript
	Dynamic bool // Name holds #...#
}

// CustomTagCalls reads every custom tag call in content, in order. prefixes
// are the configured tag library prefixes (config.CustomTagPrefixes), "cf"
// included and lowercased; a tag starting with none of them is not one. A
// caller holding the file's parse asks it instead
// (ParseResult.CustomTagCalls), which keeps its regions and the answer.
func CustomTagCalls(content string, prefixes []string) []CustomTagCall {
	regions, idx := ClassifyRegionsIdx(content)

	return customTagCalls(regions, idx, content, prefixes)
}

// CustomTagCalls is the custom tag calls in the parsed file (see
// parser.CustomTagCalls), read from the regions the parse already split and
// kept for its content and prefixes: go-to-definition and document links
// both ask, once per request, and neither splits or walks the file again.
func (pr *ParseResult) CustomTagCalls(prefixes []string) []CustomTagCall {
	key := strings.Join(prefixes, ",")

	// The walk runs outside pr.mu, which the scoped-variable caches share:
	// a hover on the same document need not wait for it.
	pr.mu.Lock()
	if pr.customTags != nil && pr.customTagsOf == pr.Content && pr.customTagsKey == key {
		calls := pr.customTags
		pr.mu.Unlock()

		return calls
	}

	content, regions, idx := pr.Content, pr.Regions, pr.contentLineIdx
	fresh := pr.regionsOf == content && regions != nil
	pr.mu.Unlock()

	if !fresh {
		regions, idx = pr.classifyRegions()
	}

	calls := customTagCalls(regions, idx, content, prefixes)
	if calls == nil {
		calls = []CustomTagCall{}
	}

	pr.mu.Lock()
	pr.customTags, pr.customTagsOf, pr.customTagsKey = calls, content, key
	pr.mu.Unlock()

	return calls
}

func customTagCalls(regions []Region, idx []int32, content string, prefixes []string) []CustomTagCall {
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
				out = scriptCustomTags(r.Text, r.Offset, out)
			}
		case RegionTag:
			out, imports = tagCustomTags(r.Text, r.Offset, imports, prefixes, out)
		default:
		}
	}

	if len(out) > 0 && idx == nil {
		idx = buildLineIdx(content)
	}

	for i := range out {
		out[i].Line = lineAtOffset(idx, out[i].Start)
	}

	return out
}

// CustomTagCallAt is the custom tag call whose name or path the 0-based line
// and byte column are on.
func CustomTagCallAt(content string, line, char int, prefixes []string) (CustomTagCall, bool) {
	return FindCustomTagCall(CustomTagCalls(content, prefixes), content, line, char)
}

// FindCustomTagCall is the call among calls, read from content, whose name or
// path the 0-based line and byte column are on.
func FindCustomTagCall(calls []CustomTagCall, content string, line, char int) (CustomTagCall, bool) {
	off, ok := offsetAt(content, line, char)
	if !ok {
		return CustomTagCall{}, false
	}

	for i := range calls {
		if calls[i].Start <= off && off <= calls[i].End {
			return calls[i], true
		}
	}

	return CustomTagCall{}, false
}

// tagCustomTags appends the custom tags text, a tag region at base, calls,
// and returns them with imports, the cfimport taglibs by lowercased prefix,
// grown by those the region declares. A prefix applies from its cfimport on,
// as the engines compile it, so it is read in the same walk.
func tagCustomTags(text string, base int, imports map[string]string, prefixes []string, out []CustomTagCall) ([]CustomTagCall, map[string]string) {
	ends := newTagEnds(text)

	for pos := 0; pos < len(text); {
		i := strings.IndexByte(text[pos:], '<')
		if i < 0 {
			break
		}

		i += pos

		if strings.HasPrefix(text[i:], "<!---") {
			pos = skipTagComment(text, i)

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
				out = moduleAttrs(text[k:end], base+k, out)
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

// cfimportTaglib adds what a <cfimport prefix="p" taglib="lib">, attrs its
// attributes, declares to imports. A JSP library (a .jar or .tld) holds no
// CFML templates and is left out.
func cfimportTaglib(attrs string, imports map[string]string) map[string]string {
	prefix, taglib := getAttr(attrs, "prefix"), getAttr(attrs, "taglib")
	if prefix == "" || taglib == "" || isJSPTaglib(taglib) {
		return imports
	}

	if imports == nil {
		imports = map[string]string{}
	}

	imports[strings.ToLower(prefix)] = taglib

	return imports
}

// isJSPTaglib reports whether taglib names a JSP library, a .jar or a .tld.
func isJSPTaglib(taglib string) bool {
	ext := taglib[max(0, len(taglib)-4):]

	return strings.EqualFold(ext, ".jar") || strings.EqualFold(ext, ".tld")
}

// moduleAttrs appends the call a cfmodule's attributes, attrs at base, make:
// by template, else by name.
func moduleAttrs(attrs string, base int, out []CustomTagCall) []CustomTagCall {
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

// TagAttributes is the attributes in attrs, a tag's text after its name (a
// CustomTagCall's AttrStart to AttrEnd), by lowercased name, read as a CF
// tag's are (attrScanner): name="value", name='value' or name=value, space
// allowed around the =, "" an escaped quote. An attribute given twice keeps
// its first value; one with no value is left out.
func TagAttributes(attrs string) map[string]string {
	return tagAttrMap(attrs, cfAttrs)
}

// scriptCustomTags appends the custom tags text, a script region at base,
// calls: cf_name(...), cfmodule(...) and Lucee's module statement.
func scriptCustomTags(text string, base int, out []CustomTagCall) []CustomTagCall {
	sc := NewScanner(text)
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
			out = scriptModuleAttrs(sc, base, TokRParen, out)
			prev = Token{Kind: TokRParen}
		case lower == "module" && statementStart(before) && sc.PeekSkipComments().Kind == TokIdent:
			out = scriptModuleAttrs(sc, base, TokSemicolon, out)
			prev = Token{Kind: TokSemicolon}
		}
	}
}

// statementStart reports whether a token after prev begins a statement:
// after a ; or a brace, the ) closing an if, for or while head with no braces,
// or else.
func statementStart(prev Token) bool {
	switch prev.Kind {
	case TokSemicolon, TokLBrace, TokRBrace, TokRParen:
		return true
	case TokIdent:
		return strings.EqualFold(prev.Value, "else")
	default:
		return false
	}
}

// scriptModuleAttrs reads the name = "value" pairs of a script cfmodule, up
// to stop at depth zero (or a { for the statement form's body), and appends
// the call they make. A key inside a nested struct or call is not one of the
// tag's; a value that is more than its string literal ("a" & b) is computed,
// and the call Dynamic.
func scriptModuleAttrs(sc *Scanner, base int, stop TokenKind, out []CustomTagCall) []CustomTagCall {
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

// endsValue reports whether a token of kind after a string literal ends the
// attribute value: the next argument, the call's end, the statement's end or
// body, or the statement form's next attribute.
func endsValue(kind TokenKind) bool {
	switch kind {
	case TokComma, TokRParen, TokSemicolon, TokLBrace, TokRBrace, TokIdent, TokEOF:
		return true
	default:
		return false
	}
}

// offsetAt is the byte offset of a 0-based line and byte column.
func offsetAt(content string, line, char int) (int, bool) {
	off := 0

	for range line {
		i := strings.IndexByte(content[off:], '\n')
		if i < 0 {
			return 0, false
		}

		off += i + 1
	}

	return min(off+char, len(content)), true
}
