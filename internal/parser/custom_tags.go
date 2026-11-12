package parser

import (
	"strings"
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
// included and lowercased; a tag starting with none of them is not one. It is
// a reading of a parse that records the file's references and nothing else
// (FileRefsOnly); a caller holding the file's parse asks it instead
// (ParseResult.CustomTagCalls).
func CustomTagCalls(content string, prefixes []string) []CustomTagCall {
	return ParseWithOptions("", content, &ParseOptions{FileRefsOnly: true, ExtractLinks: true}).CustomTagCalls(prefixes)
}

// CustomTagCalls is the custom tag calls in the parsed file (see
// parser.CustomTagCalls), from the ones its walk recorded (file_refs.go), or,
// for a parse that did not record them (ExtractLinks) or whose content an
// edit has changed since, ones read now: go-to-definition and document links
// both ask, once per request, and neither walks the file again.
func (pr *ParseResult) CustomTagCalls(prefixes []string) []CustomTagCall {
	key := strings.Join(prefixes, ",")
	content := pr.Content

	pr.mu.Lock()
	if pr.customTags != nil && pr.customTagsOf == content && pr.customTagsKey == key {
		calls := pr.customTags
		pr.mu.Unlock()

		return calls
	}
	pr.mu.Unlock()

	calls := pr.refRecord(true).customTagCalls(content, pr.lineStartsTable(), prefixes)

	pr.mu.Lock()
	pr.customTags, pr.customTagsOf, pr.customTagsKey = calls, content, key
	pr.mu.Unlock()

	return calls
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
